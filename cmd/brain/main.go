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
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"disci/brain/internal/agent"
	"disci/brain/internal/api"
	"disci/brain/internal/auth"
	"disci/brain/internal/config"
	"disci/brain/internal/consent"
	"disci/brain/internal/datasource"
	"disci/brain/internal/domain"
	"disci/brain/internal/engine"
	"disci/brain/internal/googleads"
	"disci/brain/internal/httpx"
	"disci/brain/internal/meta"
	"disci/brain/internal/persist"
	"disci/brain/internal/priors"
	"disci/brain/internal/reminder"
	"disci/brain/internal/scenario"
	"disci/brain/internal/session"
	"disci/brain/internal/sim"
	"disci/brain/internal/store"
	"disci/brain/internal/voice"
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
	case "scenario":
		runScenario()
	case "google-oauth":
		runGoogleOAuth()
	case "google-ads-test":
		runGoogleAdsTest()
	case "compare":
		runCompare()
	default:
		fmt.Println("usage: brain [sim|chat|serve|scenario|google-oauth|google-ads-test|compare]")
		os.Exit(1)
	}
}

// runScenario prints a Monte-Carlo appointment forecast for each demo clinic —
// "with this budget, how many appointments per month?" — spending no money and
// calling no LLM. Proves the scenario engine end-to-end from the CLI.
func runScenario() {
	cfg := config.Default()
	eng := engine.New(cfg, store.NewMemory())
	sim.Setup(eng, cfg.Seed)
	sc := scenario.New(nil) // cold-start PriorKeywordSource (no credentials)
	for _, c := range eng.Clinics() {
		plat := domain.PlatformGoogle
		aud := priors.AudienceLocalTR
		if c.Segment == domain.SegmentAesthetic {
			aud = priors.AudienceTourism
		}
		plan := scenario.CampaignPlan{
			ClinicID:      c.ID,
			Segment:       c.Segment,
			Platform:      plat,
			Audience:      aud,
			MonthlyBudget: c.MonthlyAdBudget,
			Seed:          cfg.Seed,
		}
		res, err := sc.Simulate(plan)
		if err != nil {
			log.Printf("scenario %s: %v", c.ID, err)
			continue
		}
		fmt.Printf("\n--- %s (%s) ---\n", c.Name, c.ID)
		fmt.Print(scenario.FormatReport(plan, res))
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

// seedAdmin creates a default admin account (idempotent) so the Next.js panel can
// log in immediately on a fresh in-memory OR Postgres store. Override the
// credentials with BRAIN_ADMIN_EMAIL / BRAIN_ADMIN_PASSWORD.
func seedAdmin(st store.Store, eng *engine.Engine) {
	email := envOr("BRAIN_ADMIN_EMAIL", "admin@disci.local")
	pass := envOr("BRAIN_ADMIN_PASSWORD", "admin1234")
	if _, ok := st.GetUserByEmail(email); ok {
		return // already seeded (e.g. persisted in Postgres)
	}
	// Never seed the well-known default password when auth is enforced.
	if pass == "admin1234" && strings.EqualFold(os.Getenv("BRAIN_REQUIRE_AUTH"), "true") {
		log.Fatal("auth: refusing to seed the DEFAULT admin password with BRAIN_REQUIRE_AUTH=true — set BRAIN_ADMIN_PASSWORD")
	}
	hash, err := auth.HashPassword(pass)
	if err != nil {
		log.Printf("seed admin: hash failed: %v", err)
		return
	}
	var ids []string
	for _, c := range eng.Clinics() {
		ids = append(ids, c.ID)
	}
	u := domain.User{
		ID: "user-admin", Email: email, Name: "Admin",
		PasswordHash: hash, Role: domain.RoleAdmin, ClinicIDs: ids,
		CreatedAt: time.Now(),
	}
	if err := st.CreateUser(u); err != nil {
		log.Printf("seed admin: %v", err)
		return
	}
	if pass == "admin1234" {
		log.Printf("auth: seeded admin %s with the DEFAULT password — set BRAIN_ADMIN_PASSWORD", email)
	} else {
		log.Printf("auth: seeded admin %s", email)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// seedDemoLeads runs a handful of synthetic leads through the brain (only when the
// store has no leads yet) so the panel's lead/appointment/conversation pages have
// data to show. Booked leads also count toward the clinic's SLA. Disable with
// BRAIN_SEED_DEMO=false.
func seedDemoLeads(eng *engine.Engine, st store.Store) {
	if os.Getenv("BRAIN_SEED_DEMO") == "false" {
		return
	}
	if len(st.ListLeads(store.LeadFilter{})) > 0 {
		return // already seeded / has real leads
	}
	clinics := eng.Clinics()
	if len(clinics) == 0 {
		return
	}
	now := time.Now()
	names := []string{
		"Ahmet Yılmaz", "Zeynep Kaya", "Mert Demir", "Sarah Jenkins", "Caner Özkan",
		"Buse Yurt", "Kadir Bulut", "Elif Şahin", "Deniz Arslan", "Ümit Çelik",
		"Fatma Koç", "John Smith", "Aylin Toprak", "Burak Aydın", "Naz Güneş", "Onur Yıldız",
	}
	budgets := []float64{15000, 45000, 8000, 220000, 60000, 4000}
	booked := 0
	for i, name := range names {
		c := clinics[i%len(clinics)]
		lead := domain.Lead{
			ID:        "demo-" + strconv.Itoa(i),
			Phone:     "+90555" + strconv.Itoa(1000000+i*54321),
			Name:      name,
			ClinicID:  c.ID,
			ArmID:     fmt.Sprintf("%s:meta:%s", c.ID, c.Segment),
			Segment:   c.Segment,
			Platform:  domain.PlatformMeta,
			CreatedAt: now.Add(-time.Duration(i) * time.Hour),
			Features: domain.LeadFeatures{
				FirstResponseSecs: 25, MessagesExchanged: 4, DistanceKm: 5, HourOfDay: 14,
				StatedBudgetTRY: budgets[i%len(budgets)],
				UrgencyScore:    float64((i*13)%100) / 100.0,
				IntentScore:     0.45 + float64((i*7)%45)/100.0,
			},
			Status: domain.LeadNew,
		}
		if dec := eng.HandleLead(lead, now); dec.Booked {
			eng.SLA.RecordQualifiedAppt(dec.ClinicID)
			booked++
		}
	}
	log.Printf("seeded %d demo leads (%d booked)", len(names), booked)
}

// seedDemoCalendar gives each clinic a couple of doctors + services so the
// appointment-calendar widget has real data to drive. Skipped if the clinic
// already has doctors, or when BRAIN_SEED_DEMO=false.
func seedDemoCalendar(st store.Store, eng *engine.Engine) {
	if os.Getenv("BRAIN_SEED_DEMO") == "false" {
		return
	}
	specialty := map[domain.Segment]string{
		domain.SegmentImplant:   "İmplantoloji",
		domain.SegmentAesthetic: "Estetik Diş Hekimliği",
		domain.SegmentOrtho:     "Ortodonti",
		domain.SegmentGeneral:   "Genel Diş Hekimliği",
	}
	service2 := map[domain.Segment]string{
		domain.SegmentImplant:   "İmplant Muayenesi",
		domain.SegmentAesthetic: "Gülüş Tasarımı Konsültasyonu",
		domain.SegmentOrtho:     "Ortodonti Muayenesi",
		domain.SegmentGeneral:   "Diş Temizliği",
	}
	first := []string{"Elif", "Mehmet", "Selin", "Caner"}
	last := []string{"Demir", "Yıldız", "Aksoy", "Kaya"}
	for i, c := range eng.Clinics() {
		if len(st.ListDoctors(c.ID)) > 0 {
			continue
		}
		d1 := domain.Doctor{
			ID: "doc-" + c.ID + "-1", ClinicID: c.ID,
			Name: first[i%len(first)] + " " + last[i%len(last)], Title: "Dt.",
			Specialty: specialty[c.Segment], Active: true,
			Days: []int{1, 2, 3, 4, 5}, StartHour: 9, EndHour: 17, SlotMins: 30,
		}
		d2 := domain.Doctor{
			ID: "doc-" + c.ID + "-2", ClinicID: c.ID,
			Name: first[(i+1)%len(first)] + " " + last[(i+2)%len(last)], Title: "Uzm. Dt.",
			Specialty: specialty[c.Segment], Active: true,
			Days: []int{1, 2, 3, 4, 6}, StartHour: 10, EndHour: 18, SlotMins: 30,
		}
		st.SaveDoctor(d1)
		st.SaveDoctor(d2)
		st.SaveService(domain.Service{
			ID: "svc-" + c.ID + "-1", ClinicID: c.ID, Name: "Muayene & Konsültasyon",
			DurationMins: 30, DoctorIDs: []string{d1.ID, d2.ID}, Active: true,
		})
		st.SaveService(domain.Service{
			ID: "svc-" + c.ID + "-2", ClinicID: c.ID, Name: service2[c.Segment],
			DurationMins: 45, DoctorIDs: []string{d1.ID}, Active: true,
		})
	}
	log.Println("seeded demo doctors + services for calendar")
}

func runServe() {
	loadEnvFile()
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
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
	server.SetStore(st)

	// Dashboard auth (JWT for the Next.js panel). Secret from BRAIN_JWT_SECRET; if
	// empty, a random per-process secret is generated (dev — tokens invalidate on
	// restart). TTL from BRAIN_JWT_TTL (default 24h).
	// Production readiness gate: when auth is enforced, insecure defaults must not
	// silently pass. Computed once here and reused for the middleware below.
	requireAuth := strings.EqualFold(os.Getenv("BRAIN_REQUIRE_AUTH"), "true")

	jwtSecret := []byte(os.Getenv("BRAIN_JWT_SECRET"))
	if len(jwtSecret) == 0 {
		if requireAuth {
			log.Fatal("auth: BRAIN_REQUIRE_AUTH=true requires a stable BRAIN_JWT_SECRET — refusing to start with a throwaway secret")
		}
		jwtSecret = make([]byte, 32)
		_, _ = rand.Read(jwtSecret)
		log.Println("auth: BRAIN_JWT_SECRET unset — using a random per-process secret (tokens invalidate on restart)")
	}
	ttl := 24 * time.Hour
	if v := os.Getenv("BRAIN_JWT_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			ttl = d
		}
	}
	authn := auth.NewAuthenticator(jwtSecret, ttl)
	server.SetAuth(authn)
	if origins := os.Getenv("BRAIN_CORS_ORIGINS"); origins != "" {
		server.SetCORS(strings.Split(origins, ","))
	}
	seedAdmin(st, eng)
	seedDemoLeads(eng, st)
	seedDemoCalendar(st, eng)

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
	server.SetIntegrations(cloud, cons, os.Getenv("META_APP_SECRET"))

	// FREE browser voice agent is always on at /voice (Web Speech API, no creds).
	// PAID PSTN voice (Twilio) turns on when a public callback URL is set.
	if base := os.Getenv("VOICE_PUBLIC_URL"); base != "" {
		vAgent := &voice.Agent{Tools: agent.NewBrainTools(eng), LLM: voice.MockLLM{}}
		server.SetVoice(voice.NewTwilioHandler(vAgent, base, nil))
		log.Println("voice: Twilio PSTN webhooks enabled (paid) at /webhooks/voice")
	} else {
		log.Println("voice: free browser agent at /voice (set VOICE_PUBLIC_URL for paid PSTN calls)")
	}

	// Scheduled no-show reminders: 24h/2h approved-template nudges (needs live
	// WhatsApp to actually send).
	if cloud != nil {
		rem := reminder.New(eng.Clinics, eng.Appointments, cloud, cons.Allowed)
		log.Println("reminders: 24h/2h scheduler (5m tick)")
		go rem.Run(context.Background(), 5*time.Minute)
	}

	// Live Keyword Planner for /v1/scenario: when Google Ads creds + a refresh token
	// are present, the scenario engine forecasts on REAL search-volume/CPC data
	// instead of cold-start priors. Read-only (KeywordPlanIdeaService) — no spend.
	if refresh := os.Getenv("GOOGLE_ADS_REFRESH_TOKEN"); refresh != "" &&
		os.Getenv("GOOGLE_ADS_DEVELOPER_TOKEN") != "" && os.Getenv("GOOGLE_CLIENT_ID") != "" {
		kc := googleads.New(
			os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"), refresh,
			os.Getenv("GOOGLE_ADS_DEVELOPER_TOKEN"), os.Getenv("GOOGLE_ADS_CUSTOMER_ID"),
			os.Getenv("GOOGLE_ADS_LOGIN_CUSTOMER_ID"))
		server.SetKeywordSource(googleads.LiveKeywordSource{Client: kc})
		log.Println("scenario: live Google Ads Keyword Planner wired")
	}

	addr := os.Getenv("BRAIN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	handler := server.Handler()
	// Auth gate is ALWAYS installed: it injects the user/clinic scope from a JWT
	// when present. Enforcement turns on with BRAIN_REQUIRE_AUTH=true or a set
	// BRAIN_API_KEY; otherwise /v1/* is dev-open (embedded console keeps working).
	apiKey := os.Getenv("BRAIN_API_KEY")
	handler = authn.Middleware(apiKey, requireAuth, auth.ProtectV1)(handler)
	switch {
	case requireAuth:
		log.Println("auth: JWT required on /v1/* (X-API-Key also accepted)")
	case apiKey != "":
		log.Println("auth: X-API-Key or JWT required on /v1/*")
	default:
		log.Println("auth: dev-open /v1/* (set BRAIN_REQUIRE_AUTH=true to enforce)")
	}
	// Production middleware chain (outermost first): recover → request-id →
	// structured logging → security headers → rate limit → body cap.
	handler = httpx.Chain(handler,
		httpx.Recover, httpx.RequestID, httpx.Logger, httpx.SecurityHeaders,
		httpx.RateLimit(50, 100), httpx.MaxBody(1<<20),
	)
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      35 * time.Second, // headroom for synchronous LLM calls
		IdleTimeout:       90 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

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

	// Live Google Ads sync: for every clinic that completed the OAuth flow (a
	// stored refresh token), pull real CPL, push budgets, upload conversions on a
	// schedule. Needs an account-level developer token; without it Google rejects
	// every call, so we stay off (no mock) and log why.
	if devTok := os.Getenv("GOOGLE_ADS_DEVELOPER_TOKEN"); devTok != "" {
		gClientID := os.Getenv("GOOGLE_CLIENT_ID")
		gSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
		loginCust := os.Getenv("GOOGLE_ADS_LOGIN_CUSTOMER_ID")
		convAction := os.Getenv("GOOGLE_ADS_CONVERSION_ACTION")
		started := 0
		for _, tok := range st.ListOAuthTokens("google") {
			gc := googleads.New(gClientID, gSecret, tok.RefreshToken, devTok, tok.CustomerID, loginCust)
			gc.ConvAction = convAction
			// Auto-discover the customer id when the panel didn't capture one.
			if gc.CustomerID == "" {
				if ids, err := gc.ListAccessibleCustomers(ctx); err == nil && len(ids) > 0 {
					gc.CustomerID = ids[0]
				} else if err != nil {
					log.Printf("google ads %s: cannot resolve customer id: %v", tok.ClinicID, err)
					continue
				}
			}
			sync := &datasource.SyncService{Eng: eng, Ads: gc, Messenger: cloud}
			started++
			go func(clinic string) {
				t := time.NewTicker(15 * time.Minute)
				defer t.Stop()
				sync.SyncAds(context.Background()) // prime immediately
				for {
					select {
					case <-ctx.Done():
						return
					case <-t.C:
						sync.SyncAds(context.Background())
					}
				}
			}(tok.ClinicID)
		}
		if started > 0 {
			log.Printf("google ads: live sync for %d clinic(s) (15m)", started)
		} else {
			log.Println("google ads: developer token set but no clinic has an OAuth refresh token yet")
		}
	}

	// Realised outcomes (show/no-show/closed) are NOT auto-pulled: datasource.
	// ClinicPMS is a generic interface, but there is no single real-world practice-
	// management system to integrate against (every clinic uses different
	// software) — a fake/mock PMS adapter would silently teach the flywheel on
	// invented data, which is worse than no automation. Until a specific clinic's
	// PMS adapter is wired in (implement datasource.ClinicPMS and pass it as
	// SyncService.PMS above), outcomes must be reported via the dedup-safe
	// POST /v1/outcomes (e.g. clinic staff marking a result in the panel).
	log.Println("outcomes: no PMS wired — report realised show/close via POST /v1/outcomes (panel or API)")

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
