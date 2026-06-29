// Package api exposes the brain over HTTP. These are the endpoints the Go
// serving layer (webhook ingester, clinic dashboard, ad-platform sync) calls.
// The handlers are thin: parse, delegate to the engine, encode. All the
// intelligence is in the motors.
package api

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"time"

	"disci/brain/internal/agent"
	"disci/brain/internal/consent"
	"disci/brain/internal/domain"
	"disci/brain/internal/engine"
	"disci/brain/internal/session"
	"disci/brain/internal/whatsapp"
)

// indexHTML is the self-contained console served at "/". It needs no build step
// and no Node — open the backend URL in a browser and the UI is right there.
//
//go:embed web/index.html
var indexHTML []byte

// Server wraps the engine with HTTP routing.
type Server struct {
	eng   *engine.Engine
	agent *agent.Agent

	cloud   *whatsapp.Cloud // live WhatsApp Cloud API (nil → webhook send is skipped)
	consent *consent.Store  // KVKK opt-out guard

	sessions session.Store // conversation state (swap Memory→Redis for HA)
}

// New builds the HTTP server. The agent may be nil (then /v1/whatsapp is off).
func New(eng *engine.Engine, ag *agent.Agent) *Server {
	return &Server{eng: eng, agent: ag, sessions: session.NewMemory(), consent: consent.NewStore()}
}

// SetSessionStore swaps the conversation-state store (e.g. a Redis-backed one
// for multi-instance deployments). Call before serving.
func (s *Server) SetSessionStore(store session.Store) {
	if store != nil {
		s.sessions = store
	}
}

// SetIntegrations wires the live WhatsApp client (and shares a consent store) so
// the webhooks can send replies and honour opt-outs. Call before serving.
func (s *Server) SetIntegrations(cloud *whatsapp.Cloud, cons *consent.Store) {
	s.cloud = cloud
	if cons != nil {
		s.consent = cons
	}
}

// Routes returns the configured mux.
func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	// Public webhooks (Meta calls these; not auth-gated).
	mux.HandleFunc("GET /webhooks/whatsapp", s.handleWAVerify)   // Meta webhook verification
	mux.HandleFunc("POST /webhooks/whatsapp", s.handleWAInbound) // inbound WhatsApp → agent
	mux.HandleFunc("POST /webhooks/webform", s.handleWebform)    // website form lead → agent
	mux.HandleFunc("POST /webhooks/meta-leads", s.handleMetaLeads) // Meta Lead Ads → agent

	mux.HandleFunc("POST /v1/whatsapp", s.handleWhatsApp)   // test console free-text → agent → brain
	mux.HandleFunc("POST /v1/intake", s.handleIntake)       // structured survey answers → brain (deterministic)
	mux.HandleFunc("POST /v1/leads", s.handleLead)         // a pre-qualified lead arrived
	mux.HandleFunc("POST /v1/outcomes", s.handleOutcome)   // a result came back (feedback loop)
	mux.HandleFunc("POST /v1/budget/plan", s.handleBudget) // run a budget allocation cycle
	mux.HandleFunc("GET /v1/sla", s.handleSLA)             // guarantee health
	mux.HandleFunc("GET /v1/arms", s.handleArms)           // learned ad-arm stats
	mux.HandleFunc("GET /v1/templates", s.handleTemplates) // Meta-approved WhatsApp templates
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		agentName := "none"
		if s.agent != nil {
			agentName = s.agent.Provider()
		}
		writeJSON(w, 200, map[string]string{"status": "ok", "agent": agentName})
	})
	// Embedded console (catch-all GET). More specific routes above win, so this
	// only serves the UI page itself.
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexHTML)
	})
	return mux
}

// Handler returns the routes wrapped with permissive CORS so the Vite/Tailwind
// dev UI (a different origin) can call the API directly without auth. This is a
// dev/demo convenience — lock it down before exposing publicly.
func (s *Server) Handler() http.Handler {
	return cors(s.Routes())
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handleWhatsApp runs one inbound WhatsApp turn through the conversation agent.
// Sessions are kept in memory keyed by phone for multi-turn dialogue.
func (s *Server) handleWhatsApp(w http.ResponseWriter, r *http.Request) {
	if s.agent == nil {
		writeJSON(w, 503, map[string]string{"error": "agent not configured"})
		return
	}
	var body struct {
		Phone    string  `json:"phone"`
		ClinicID string  `json:"clinicId"`
		ArmID    string  `json:"armId"`
		Message  string  `json:"message"`
		HourOfDay float64 `json:"hourOfDay"`
		DistanceKm float64 `json:"distanceKm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	sess, ok := s.sessions.Get(body.Phone)
	if !ok {
		sess = &agent.Session{LeadID: "wa-" + body.Phone, Phone: body.Phone, HourOfDay: body.HourOfDay, DistanceKm: body.DistanceKm}
	}
	res, err := s.agent.Handle(r.Context(), sess, body.ClinicID, body.ArmID, body.Message)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	s.sessions.Put(body.Phone, sess)
	writeJSON(w, 200, map[string]any{
		"reply":         res.Reply,
		"qualification": res.Qualification,
		"booked":        res.Decision.Booked,
		"apptTime":      res.Decision.ApptTime,
		"reason":        res.Decision.Reason,
	})
}

// handleIntake takes STRUCTURED survey answers (segment, urgency, budget) and
// runs them through the brain deterministically — no LLM, so the flow is
// consistent and cannot hallucinate appointments. This is the "anket" path.
func (s *Server) handleIntake(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Phone     string  `json:"phone"`
		ClinicID  string  `json:"clinicId"`
		ArmID     string  `json:"armId"`
		Segment   string  `json:"segment"`
		Urgency   float64 `json:"urgency"`
		BudgetTRY float64 `json:"budgetTry"`
		HourOfDay float64 `json:"hourOfDay"`
		DistanceKm float64 `json:"distanceKm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	// A completed survey is a genuinely engaged lead → solid intent, lifted by
	// urgency and a known budget.
	intent := 0.55 + b.Urgency*0.2
	if b.BudgetTRY > 0 {
		intent += 0.15
	}
	if intent > 0.95 {
		intent = 0.95
	}
	hour := b.HourOfDay
	if hour == 0 {
		hour = 14
	}
	lead := domain.Lead{
		ID:       "intake-" + b.Phone,
		Phone:    b.Phone,
		ClinicID: b.ClinicID,
		ArmID:    b.ArmID,
		Segment:  domain.Segment(b.Segment),
		Features: domain.LeadFeatures{
			FirstResponseSecs: 20, MessagesExchanged: 4, DistanceKm: b.DistanceKm,
			HourOfDay: hour, StatedBudgetTRY: b.BudgetTRY, UrgencyScore: b.Urgency,
			IntentScore: intent,
		},
	}
	dec := s.eng.HandleLead(lead, time.Now())
	writeJSON(w, 200, map[string]any{
		"booked": dec.Booked, "apptTime": dec.ApptTime, "reason": dec.Reason,
		"qualification": map[string]any{"segment": b.Segment, "intent": intent, "urgency": b.Urgency, "budgetTry": b.BudgetTRY},
	})
}

func (s *Server) handleLead(w http.ResponseWriter, r *http.Request) {
	var lead domain.Lead
	if err := json.NewDecoder(r.Body).Decode(&lead); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	if lead.CreatedAt.IsZero() {
		lead.CreatedAt = time.Now()
	}
	dec := s.eng.HandleLead(lead, time.Now())
	writeJSON(w, 200, dec)
}

func (s *Server) handleOutcome(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Outcome  domain.Outcome      `json:"outcome"`
		Features domain.LeadFeatures `json:"features"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	s.eng.Loop.Ingest(body.Outcome, body.Features)
	writeJSON(w, 200, map[string]string{"status": "ingested"})
}

func (s *Server) handleBudget(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DaysInMonth float64 `json:"daysInMonth"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	days := body.DaysInMonth
	if days <= 0 {
		days = 30
	}
	allocs, lambda := s.eng.PlanBudget(days)
	writeJSON(w, 200, map[string]any{
		"daysInMonth": days,
		"networkDaily": s.eng.NetworkDailyBudget(days),
		"lambda":      lambda,
		"allocations": allocs,
	})
}

func (s *Server) handleSLA(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.eng.SLAReport(time.Now()))
}

func (s *Server) handleArms(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.eng.Budget.Snapshot())
}

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, whatsapp.List())
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
