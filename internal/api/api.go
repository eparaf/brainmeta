// Package api exposes the brain over HTTP. These are the endpoints the Go
// serving layer (webhook ingester, clinic dashboard, ad-platform sync) calls.
// The handlers are thin: parse, delegate to the engine, encode. All the
// intelligence is in the motors.
package api

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"disci/brain/internal/agent"
	"disci/brain/internal/auth"
	"disci/brain/internal/consent"
	"disci/brain/internal/domain"
	"disci/brain/internal/engine"
	"disci/brain/internal/httpx"
	"disci/brain/internal/session"
	"disci/brain/internal/store"
	"disci/brain/internal/voice"
	"disci/brain/internal/whatsapp"
)

// indexHTML is the self-contained console served at "/". It needs no build step
// and no Node — open the backend URL in a browser and the UI is right there.
//
//go:embed web/index.html
var indexHTML []byte

// voiceHTML is the FREE browser voice agent (Web Speech API: Turkish STT+TTS in
// the browser → /v1/whatsapp → reply spoken back). No telephony, no creds, no
// cost. Served at /voice. (Paid PSTN calls use internal/voice + Twilio.)
//
//go:embed web/voice.html
var voiceHTML []byte

// widgetJS is the embeddable web-form + appointment-calendar widget clinics drop
// onto their own sites. Served at /embed/widget.js; configured per clinic via the
// public key it carries.
//
//go:embed web/widget.js
var widgetJS []byte

// Version is the build version, set via -ldflags "-X disci/brain/internal/api.Version=...".
var Version = "dev"

// Server wraps the engine with HTTP routing.
type Server struct {
	eng   *engine.Engine
	agent *agent.Agent

	cloud     *whatsapp.Cloud      // live WhatsApp Cloud API (nil → webhook send is skipped)
	consent   *consent.Store       // KVKK opt-out guard
	appSecret string               // Meta app secret for webhook signature verification
	voice     *voice.TwilioHandler // paid PSTN voice (nil → /webhooks/voice off)

	sessions session.Store // conversation state (swap Memory→Redis for HA)

	store       store.Store         // entity store (for dashboard auth/list endpoints)
	authn       *auth.Authenticator // JWT signer/verifier (nil → /v1/auth/* return 503)
	corsOrigins map[string]bool     // allowlist; empty → permissive "*" (dev)
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

// SetVoice wires the (paid) Twilio voice handler; enables /webhooks/voice.
func (s *Server) SetVoice(h *voice.TwilioHandler) { s.voice = h }

// SetIntegrations wires the live WhatsApp client, consent store, and Meta app
// secret (for webhook signature verification). Call before serving.
func (s *Server) SetIntegrations(cloud *whatsapp.Cloud, cons *consent.Store, appSecret string) {
	s.cloud = cloud
	s.appSecret = appSecret
	if cons != nil {
		s.consent = cons
	}
}

// SetStore wires the entity store used by the dashboard auth and list endpoints.
func (s *Server) SetStore(st store.Store) { s.store = st }

// SetAuth wires the JWT authenticator that signs/verifies dashboard tokens.
func (s *Server) SetAuth(a *auth.Authenticator) { s.authn = a }

// SetCORS sets the allowed browser origins. Empty/unset → permissive "*" (dev).
func (s *Server) SetCORS(origins []string) {
	m := map[string]bool{}
	for _, o := range origins {
		if o = strings.TrimSpace(o); o != "" {
			m[o] = true
		}
	}
	s.corsOrigins = m
}

// Routes returns the configured mux.
func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	// Public webhooks (Meta calls these; not auth-gated).
	mux.HandleFunc("GET /webhooks/whatsapp", s.handleWAVerify)     // Meta webhook verification
	mux.HandleFunc("POST /webhooks/whatsapp", s.handleWAInbound)   // inbound WhatsApp → agent
	mux.HandleFunc("POST /webhooks/webform", s.handleWebform)      // website form lead → agent
	mux.HandleFunc("POST /webhooks/meta-leads", s.handleMetaLeads) // Meta Lead Ads → agent
	// Paid PSTN voice (Twilio) — only when configured.
	mux.HandleFunc("POST /webhooks/voice", func(w http.ResponseWriter, r *http.Request) {
		if s.voice == nil {
			w.WriteHeader(503)
			return
		}
		s.voice.Incoming(w, r)
	})
	mux.HandleFunc("POST /webhooks/voice/gather", func(w http.ResponseWriter, r *http.Request) {
		if s.voice == nil {
			w.WriteHeader(503)
			return
		}
		s.voice.Gather(w, r)
	})

	// Dashboard auth (Next.js panel). login/register/refresh are auth-exempt (see
	// auth.ProtectV1); me/logout require a valid token.
	mux.HandleFunc("POST /v1/auth/register", s.handleRegister)
	mux.HandleFunc("POST /v1/auth/login", s.handleLogin)
	mux.HandleFunc("GET /v1/auth/me", s.handleMe)
	mux.HandleFunc("POST /v1/auth/refresh", s.handleRefresh)
	mux.HandleFunc("POST /v1/auth/logout", s.handleLogout)

	mux.HandleFunc("POST /v1/whatsapp", s.handleWhatsApp)  // test console free-text → agent → brain
	mux.HandleFunc("POST /v1/intake", s.handleIntake)      // structured survey answers → brain (deterministic)
	mux.HandleFunc("POST /v1/leads", s.handleLead)         // a pre-qualified lead arrived
	mux.HandleFunc("POST /v1/outcomes", s.handleOutcome)   // a result came back (feedback loop)
	mux.HandleFunc("POST /v1/budget/plan", s.handleBudget) // run a budget allocation cycle
	mux.HandleFunc("GET /v1/sla", s.handleSLA)             // guarantee health
	mux.HandleFunc("GET /v1/arms", s.handleArms)           // learned ad-arm stats
	mux.HandleFunc("GET /v1/templates", s.handleTemplates) // approved templates + drafts

	// Dashboard list/CRUD surface (Next.js panel pages).
	mux.HandleFunc("GET /v1/clinics", s.handleListClinics)
	mux.HandleFunc("GET /v1/leads", s.handleListLeads)
	mux.HandleFunc("GET /v1/appointments", s.handleListAppointments)
	mux.HandleFunc("GET /v1/conversations", s.handleListConversations)
	mux.HandleFunc("GET /v1/conversations/{id}", s.handleConversation)
	mux.HandleFunc("POST /v1/templates", s.handleCreateTemplate)
	mux.HandleFunc("GET /v1/connections", s.handleConnections)
	mux.HandleFunc("POST /v1/connections", s.handleConnections)

	// Embeddable widget: per-clinic config (protected) + public submission surface.
	mux.HandleFunc("GET /v1/widget", s.handleWidgetGet)
	mux.HandleFunc("POST /v1/widget", s.handleWidgetSave)
	mux.HandleFunc("POST /v1/widget/rotate-key", s.handleWidgetRotate)
	// Doctors & services (clinic calendar).
	mux.HandleFunc("GET /v1/doctors", s.handleListDoctors)
	mux.HandleFunc("POST /v1/doctors", s.handleSaveDoctor)
	mux.HandleFunc("DELETE /v1/doctors/{id}", s.handleDeleteDoctor)
	mux.HandleFunc("GET /v1/services", s.handleListServices)
	mux.HandleFunc("POST /v1/services", s.handleSaveService)
	mux.HandleFunc("DELETE /v1/services/{id}", s.handleDeleteService)
	// Public widget surface: form lead + step-by-step calendar booking.
	mux.HandleFunc("GET /public/widget", s.handlePublicWidget)
	mux.HandleFunc("POST /public/widget/lead", s.handlePublicWidgetLead)
	mux.HandleFunc("GET /public/widget/services", s.handlePublicServices)
	mux.HandleFunc("GET /public/widget/doctors", s.handlePublicDoctors)
	mux.HandleFunc("GET /public/widget/availability", s.handlePublicAvailability)
	mux.HandleFunc("GET /public/widget/recommend", s.handlePublicRecommend)
	mux.HandleFunc("POST /public/widget/book", s.handlePublicCalendarBook)
	mux.HandleFunc("GET /embed/widget.js", s.handleWidgetJS)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		agentName := "none"
		if s.agent != nil {
			agentName = s.agent.Provider()
		}
		writeJSON(w, 200, map[string]string{"status": "ok", "agent": agentName})
	})
	// Readiness: dependencies reachable enough to serve. Liveness is /healthz.
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		_ = s.eng.Clinics() // store reachable?
		writeJSON(w, 200, map[string]any{"ready": true})
	})
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"version": Version})
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		httpx.WritePrometheus(w)
	})
	// Embedded console (catch-all GET). More specific routes above win, so this
	// only serves the UI page itself.
	// FREE browser voice agent (Web Speech API) — no telephony/creds/cost.
	mux.HandleFunc("GET /voice", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(voiceHTML)
	})
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexHTML)
	})
	return mux
}

// Handler returns the routes wrapped with CORS so the Next.js dev panel (a
// different origin) can call the API. Origins are an allowlist (BRAIN_CORS_ORIGINS);
// when unset it falls back to permissive "*" for local dev.
func (s *Server) Handler() http.Handler {
	return s.cors(s.Routes())
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		// The embeddable widget + its public submission API are called from arbitrary
		// clinic websites, so they are always wildcard-CORS regardless of allowlist.
		public := strings.HasPrefix(r.URL.Path, "/public/") ||
			strings.HasPrefix(r.URL.Path, "/embed/")
		switch {
		case public || len(s.corsOrigins) == 0: // public widget, or dev default
			w.Header().Set("Access-Control-Allow-Origin", "*")
		case s.corsOrigins[origin]:
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		// Authorization is load-bearing: the panel sends the JWT as a bearer token.
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
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
		Phone      string  `json:"phone"`
		ClinicID   string  `json:"clinicId"`
		ArmID      string  `json:"armId"`
		Message    string  `json:"message"`
		HourOfDay  float64 `json:"hourOfDay"`
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
		Phone      string  `json:"phone"`
		ClinicID   string  `json:"clinicId"`
		ArmID      string  `json:"armId"`
		Segment    string  `json:"segment"`
		Urgency    float64 `json:"urgency"`
		BudgetTRY  float64 `json:"budgetTry"`
		HourOfDay  float64 `json:"hourOfDay"`
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
		"daysInMonth":  days,
		"networkDaily": s.eng.NetworkDailyBudget(days),
		"lambda":       lambda,
		"allocations":  allocs,
	})
}

func (s *Server) handleSLA(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.eng.SLAReport(time.Now()))
}

func (s *Server) handleArms(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.eng.Budget.Snapshot())
}

// handleTemplates returns the static Meta-approved templates merged with any
// clinic-authored drafts the caller can see.
func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	out := []map[string]any{}
	for _, t := range whatsapp.List() {
		out = append(out, map[string]any{
			"id": t.Name + ":" + t.Language, "name": t.Name, "category": t.Category,
			"language": t.Language, "status": t.Status, "body": t.Body, "vars": t.Vars,
		})
	}
	if s.store != nil {
		u, _ := auth.UserFrom(r.Context())
		for _, d := range s.store.ListTemplates("") {
			if !auth.CanAccessClinic(u, d.ClinicID) {
				continue
			}
			out = append(out, map[string]any{
				"id": d.ID, "name": d.Name, "category": d.Category, "language": d.Language,
				"status": d.Status, "body": d.Body, "clinicId": d.ClinicID,
			})
		}
	}
	writeJSON(w, 200, out)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
