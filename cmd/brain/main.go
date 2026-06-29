// Command brain is the entry point. Modes:
//
//	brain sim     — run the 30-day end-to-end simulation and print a report.
//	brain chat    — run sample WhatsApp conversations through the agent (mock LLM)
//	                and print the dialogue + booking decisions. No API key needed.
//	brain serve   — start the HTTP API (agent + brain + durable snapshots).
//
// Default (no args) runs the simulation.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"disci/brain/internal/agent"
	"disci/brain/internal/api"
	"disci/brain/internal/auth"
	"disci/brain/internal/config"
	"disci/brain/internal/consent"
	"disci/brain/internal/datasource"
	"disci/brain/internal/engine"
	"disci/brain/internal/meta"
	"disci/brain/internal/persist"
	"disci/brain/internal/reminder"
	"disci/brain/internal/session"
	"disci/brain/internal/sim"
	"disci/brain/internal/store"
	"disci/brain/internal/whatsapp"
)

func main() {
	mode := "sim"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	switch mode {
	case "sim":
		runSim()
	case "chat":
		runChat()
	case "serve":
		runServe()
	default:
		fmt.Println("usage: brain [sim|chat|serve]")
		os.Exit(1)
	}
}

func runSim() {
	cfg := config.Default()
	st := store.NewMemory()
	eng := engine.New(cfg, st)
	world := sim.Setup(eng, cfg.Seed)
	res := world.Run(eng, st, 30)
	allocs, lambda := eng.PlanBudget(30)
	fmt.Print(sim.FormatReport(res, allocs))
	fmt.Printf("\nFinal marginal budget shadow price λ = %.4f (TRY margin per TRY spent)\n", lambda)
}

// runChat demonstrates the conversational agent (mouth) end-to-end with the mock
// LLM — qualify → brain decides → reply references the brain's real slot.
func runChat() {
	cfg := config.Default()
	eng := engine.New(cfg, store.NewMemory())
	sim.Setup(eng, cfg.Seed)
	ag := agent.New(agent.MockLLM{}, eng)

	scripts := []struct {
		phone, clinic, arm string
		msgs               []string
	}{
		{"+90555111", "umraniye", "umraniye:meta:implant", []string{
			"Merhaba, implant yaptırmak istiyorum, ağrım var bütçem 60000 TL"}},
		{"+44777222", "nisantasi", "nisantasi:meta:aesthetic", []string{
			"Hi, how much for a full set of veneers? budget around 4000 GBP"}},
		{"+90555333", "sisli", "sisli:meta:general", []string{
			"merhaba", "diş kontrolü olmak istiyorum"}},
	}

	fmt.Println("=== AGENT DEMO (mock LLM — no API key) ===")
	for _, s := range scripts {
		fmt.Printf("\n--- %s @ %s ---\n", s.phone, s.clinic)
		sess := &agent.Session{LeadID: "wa-" + s.phone, HourOfDay: 14, DistanceKm: 6, FirstResponseSecs: 25}
		for _, m := range s.msgs {
			fmt.Printf("👤 %s\n", m)
			res, err := ag.Handle(context.Background(), sess, s.clinic, s.arm, m)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("🤖 %s\n", res.Reply)
			if res.Acted {
				fmt.Printf("   [brain] segment=%s booked=%v reason=%s appt=%s\n",
					res.Qualification.Segment, res.Decision.Booked, res.Decision.Reason,
					res.Decision.ApptTime.Format("02 Jan 15:04"))
			}
		}
	}
}

// loadEnvFile reads a simple KEY=VALUE file (default ./brain.env, override with
// BRAIN_ENV) and sets any vars not already in the environment. Lets you drop your
// GEMINI_API_KEY / GEMINI_MODEL in one gitignored file instead of exporting it
// every time. Lines starting with # are comments.
func loadEnvFile() {
	path := os.Getenv("BRAIN_ENV")
	if path == "" {
		path = "brain.env"
	}
	f, err := os.Open(path)
	if err != nil {
		return // no file — fine
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if k != "" && os.Getenv(k) == "" {
			_ = os.Setenv(k, v)
		}
	}
	log.Printf("loaded %s", path)
}

func runServe() {
	loadEnvFile()
	cfg := config.Default()

	// Entity store: Postgres if DATABASE_URL is set AND a driver is registered
	// (add `import _ "github.com/jackc/pgx/v5/stdlib"`); otherwise in-memory.
	var st store.Store = store.NewMemory()
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		driver := os.Getenv("DB_DRIVER")
		if driver == "" {
			driver = "pgx"
		}
		if pg, err := store.NewPostgres(driver, dsn); err == nil {
			st = pg
			log.Println("store: postgres")
		} else {
			log.Printf("store: postgres unavailable (%v) — using memory", err)
		}
	}
	eng := engine.New(cfg, st)
	sim.Setup(eng, cfg.Seed) // seed demo clinics so endpoints return data

	// Durable learned state: load on startup, save periodically + on shutdown.
	snapPath := os.Getenv("BRAIN_SNAPSHOT")
	if snapPath == "" {
		snapPath = "./brain-data/brain-snapshot.json" // container overrides to /data
	}
	snaps := persist.NewFileStore(snapPath)
	if loaded, err := eng.LoadSnapshot(snaps); err != nil {
		log.Printf("snapshot load: %v", err)
	} else if loaded {
		log.Printf("restored learned state from %s", snapPath)
	} else {
		log.Printf("no prior snapshot at %s — fresh brain", snapPath)
	}

	// Conversation agent: pick the LLM by which API key is present.
	//   GEMINI_API_KEY     -> Gemini   (GEMINI_MODEL to override, e.g. gemini-2.0-flash)
	//   ANTHROPIC_API_KEY  -> Claude   (BRAIN_AGENT_MODEL to override)
	//   neither            -> deterministic mock (no key needed)
	var llm agent.LLM = agent.MockLLM{}
	switch {
	case os.Getenv("GEMINI_API_KEY") != "":
		llm = agent.NewGemini("", os.Getenv("GEMINI_MODEL"))
		log.Println("agent: using Gemini")
	case os.Getenv("ANTHROPIC_API_KEY") != "":
		llm = agent.NewClaude("", os.Getenv("BRAIN_AGENT_MODEL"))
		log.Println("agent: using Claude")
	default:
		log.Println("agent: using mock LLM (set GEMINI_API_KEY or ANTHROPIC_API_KEY for a real model)")
	}
	ag := agent.New(llm, eng)

	server := api.New(eng, ag)
	cons := consent.NewStore()

	// Horizontal scale: shared session state in Redis (only if built -tags redis).
	if url := os.Getenv("REDIS_URL"); url != "" {
		if session.RedisFactory == nil {
			log.Println("session: REDIS_URL set but binary built without -tags redis — using in-memory")
		} else if rs, err := session.RedisFactory(url); err == nil {
			server.SetSessionStore(rs)
			log.Println("session: redis (multi-instance ready)")
		} else {
			log.Printf("session: redis unavailable (%v) — using in-memory", err)
		}
	}

	// Live WhatsApp Cloud API (Meta) — inbound webhook replies + template sends.
	var cloud *whatsapp.Cloud
	if t := os.Getenv("WHATSAPP_TOKEN"); t != "" && os.Getenv("WHATSAPP_PHONE_NUMBER_ID") != "" {
		cloud = whatsapp.NewCloud(t, os.Getenv("WHATSAPP_PHONE_NUMBER_ID"), os.Getenv("WHATSAPP_VERIFY_TOKEN"))
		log.Println("whatsapp: live Cloud API")
	} else {
		log.Println("whatsapp: not configured (set WHATSAPP_TOKEN + WHATSAPP_PHONE_NUMBER_ID)")
	}
	server.SetIntegrations(cloud, cons)

	// Scheduled no-show reminders: 24h/2h approved-template nudges (needs live
	// WhatsApp to actually send).
	if cloud != nil {
		rem := reminder.New(eng.Clinics, eng.Appointments, cloud, cons.Allowed)
		log.Println("reminders: 24h/2h scheduler (5m tick)")
		go rem.Run(context.Background(), 5*time.Minute)
	}

	addr := os.Getenv("BRAIN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	handler := server.Handler()
	if key := os.Getenv("BRAIN_API_KEY"); key != "" {
		handler = auth.Middleware(key, auth.ProtectV1)(handler)
		log.Println("auth: X-API-Key required on /v1/*")
	}
	srv := &http.Server{Addr: addr, Handler: handler}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Live Meta Marketing ads sync: pull real CPL, push budgets, upload
	// conversions on a schedule.
	if mt := os.Getenv("META_TOKEN"); mt != "" && os.Getenv("META_AD_ACCOUNT_ID") != "" {
		mkt := meta.NewMarketing(mt, os.Getenv("META_AD_ACCOUNT_ID"), os.Getenv("META_PIXEL_ID"))
		sync := &datasource.SyncService{Eng: eng, Ads: mkt, Messenger: cloud}
		log.Println("meta marketing: live ads sync (15m)")
		go func() {
			t := time.NewTicker(15 * time.Minute)
			defer t.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					sync.SyncAds(context.Background())
				}
			}
		}()
	}

	// Periodic snapshot saver (every 60s) + weekly posterior decay.
	go func() {
		t := time.NewTicker(60 * time.Second)
		defer t.Stop()
		ticks := 0
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-t.C:
				if err := eng.SaveSnapshot(snaps, now); err != nil {
					log.Printf("snapshot save: %v", err)
				}
				ticks++
				if ticks%10080 == 0 { // ~weekly at 60s cadence
					eng.Budget.Decay(0.95)
				}
			}
		}
	}()

	go func() {
		log.Printf("brain serving on %s (POST /v1/whatsapp /v1/leads /v1/outcomes /v1/budget/plan; GET /v1/sla /v1/arms)", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutCtx)
	if err := eng.SaveSnapshot(snaps, time.Now()); err != nil {
		log.Printf("final snapshot save: %v", err)
	} else {
		log.Printf("saved learned state to %s", snapPath)
	}
	log.Println("brain stopped")
}
