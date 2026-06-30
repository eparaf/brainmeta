package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"disci/brain/internal/auth"
	"disci/brain/internal/domain"
	"disci/brain/internal/engine"
)

func newKey() string {
	b := make([]byte, 18)
	_, _ = rand.Read(b)
	return "pk_" + hex.EncodeToString(b)
}

// ensureWidget returns the clinic's widget config, creating a default (with a fresh
// public key) on first access.
func (s *Server) ensureWidget(clinicID string) domain.WidgetConfig {
	if c, ok := s.store.GetWidgetConfig(clinicID); ok {
		// Backfill fields added after this config was first persisted (schema evolution).
		def := domain.DefaultWidgetConfig(clinicID, c.PublicKey)
		changed := false
		set := func(dst *string, val string) {
			if *dst == "" {
				*dst = val
				changed = true
			}
		}
		set(&c.PrimaryColor, def.PrimaryColor)
		set(&c.CalendarColor, def.CalendarColor)
		set(&c.CalendarTitle, def.CalendarTitle)
		set(&c.CalendarSubtitle, def.CalendarSubtitle)
		set(&c.ConfirmText, def.ConfirmText)
		if changed {
			c.UpdatedAt = time.Now()
			s.store.SaveWidgetConfig(c)
		}
		return c
	}
	c := domain.DefaultWidgetConfig(clinicID, newKey())
	c.UpdatedAt = time.Now()
	s.store.SaveWidgetConfig(c)
	return c
}

// widgetPublicView is the safe projection returned to the embed (no internal ids).
func widgetPublicView(cfg domain.WidgetConfig, clinicName string) map[string]any {
	fields := []domain.WidgetField{}
	for _, f := range cfg.Fields {
		if f.Enabled {
			fields = append(fields, f)
		}
	}
	theme := cfg.Theme
	if theme == "" {
		theme = "dark"
	}
	return map[string]any{
		"clinicName":       clinicName,
		"primaryColor":     cfg.PrimaryColor,
		"formTitle":        cfg.FormTitle,
		"formSubtitle":     cfg.FormSubtitle,
		"successText":      cfg.SuccessText,
		"fields":           fields,
		"calendarColor":    cfg.CalendarColor,
		"calendarTitle":    cfg.CalendarTitle,
		"calendarSubtitle": cfg.CalendarSubtitle,
		"confirmText":      cfg.ConfirmText,
		"theme":            theme,
		"recommend":        cfg.Recommend,
	}
}

// widgetLead funnels a widget submission through the brain (same path as a webform
// lead). intent/urgency are higher for a calendar booking than a plain form lead.
func (s *Server) widgetLead(
	cfg domain.WidgetConfig, name, phone, message string, intent, urgency float64,
) engine.LeadDecision {
	clinic, _ := s.eng.Clinic(cfg.ClinicID)
	now := time.Now()
	lead := domain.Lead{
		ID:        "web-" + newID(),
		Phone:     phone,
		Name:      name,
		ClinicID:  cfg.ClinicID,
		ArmID:     cfg.ClinicID + ":webform:" + string(clinic.Segment),
		Segment:   clinic.Segment,
		Platform:  domain.PlatformOrganic,
		CreatedAt: now,
		Features: domain.LeadFeatures{
			FirstResponseSecs: 30, MessagesExchanged: 1, HourOfDay: float64(now.Hour()),
			IntentScore: intent, UrgencyScore: urgency,
		},
		Status: domain.LeadNew,
	}
	return s.eng.HandleLead(lead, now)
}

// ---- protected (/v1) handlers ------------------------------------------------

func (s *Server) handleWidgetGet(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeJSON(w, 503, map[string]string{"error": "store not configured"})
		return
	}
	u, _ := auth.UserFrom(r.Context())
	clinicID := r.URL.Query().Get("clinicId")
	if clinicID == "" {
		writeJSON(w, 400, map[string]string{"error": "clinicId required"})
		return
	}
	if !auth.CanAccessClinic(u, clinicID) {
		writeJSON(w, 403, map[string]string{"error": "forbidden"})
		return
	}
	writeJSON(w, 200, s.ensureWidget(clinicID))
}

func (s *Server) handleWidgetSave(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeJSON(w, 503, map[string]string{"error": "store not configured"})
		return
	}
	u, _ := auth.UserFrom(r.Context())
	var in domain.WidgetConfig
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	if in.ClinicID == "" {
		writeJSON(w, 400, map[string]string{"error": "clinicId required"})
		return
	}
	if !auth.CanAccessClinic(u, in.ClinicID) {
		writeJSON(w, 403, map[string]string{"error": "forbidden"})
		return
	}
	// The key is never changed via save — only via rotate-key.
	in.PublicKey = s.ensureWidget(in.ClinicID).PublicKey
	in.UpdatedAt = time.Now()
	s.store.SaveWidgetConfig(in)
	writeJSON(w, 200, in)
}

func (s *Server) handleWidgetRotate(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeJSON(w, 503, map[string]string{"error": "store not configured"})
		return
	}
	u, _ := auth.UserFrom(r.Context())
	clinicID := r.URL.Query().Get("clinicId")
	if !auth.CanAccessClinic(u, clinicID) {
		writeJSON(w, 403, map[string]string{"error": "forbidden"})
		return
	}
	c := s.ensureWidget(clinicID)
	c.PublicKey = newKey()
	c.UpdatedAt = time.Now()
	s.store.SaveWidgetConfig(c)
	writeJSON(w, 200, c)
}

// ---- public (/public) handlers — no auth, permissive CORS --------------------

func (s *Server) handlePublicWidget(w http.ResponseWriter, r *http.Request) {
	cfg, ok := s.widgetByKey(r)
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "invalid_key"})
		return
	}
	clinic, _ := s.eng.Clinic(cfg.ClinicID)
	writeJSON(w, 200, widgetPublicView(cfg, clinic.Name))
}

func (s *Server) handlePublicWidgetLead(w http.ResponseWriter, r *http.Request) {
	cfg, ok := s.widgetByKey(r)
	if !ok {
		writeJSON(w, 401, map[string]string{"error": "invalid_key"})
		return
	}
	var b struct {
		Name    string `json:"name"`
		Phone   string `json:"phone"`
		Message string `json:"message"`
		Email   string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	if b.Phone == "" {
		writeJSON(w, 400, map[string]string{"error": "phone required"})
		return
	}
	dec := s.widgetLead(cfg, b.Name, b.Phone, b.Message, 0.6, 0.4)
	writeJSON(w, 200, map[string]any{"ok": true, "booked": dec.Booked, "apptTime": dec.ApptTime})
}

// widgetByKey resolves the widget config from the ?key= query param.
func (s *Server) widgetByKey(r *http.Request) (domain.WidgetConfig, bool) {
	if s.store == nil {
		return domain.WidgetConfig{}, false
	}
	return s.store.GetWidgetConfigByKey(r.URL.Query().Get("key"))
}

// handleWidgetJS serves the embeddable widget script.
func (s *Server) handleWidgetJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(widgetJS)
}
