package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"disci/brain/internal/agent"
	"disci/brain/internal/auth"
	"disci/brain/internal/domain"
	"disci/brain/internal/sla"
	"disci/brain/internal/store"
)

// ---- view mappers (domain → camelCase JSON the Next.js panel expects) --------

func leadStatusTR(s domain.LeadStatus) string {
	switch s {
	case domain.LeadBooked, domain.LeadShowed, domain.LeadClosed:
		return "Randevu"
	case domain.LeadLost:
		return "Düştü"
	case domain.LeadNoShow:
		return "Gelmedi"
	default: // new, qualified
		return "Niteleniyor"
	}
}

func urgencyTR(u float64) string {
	switch {
	case u >= 0.66:
		return "Yüksek"
	case u >= 0.33:
		return "Orta"
	default:
		return "Düşük"
	}
}

func apptTimeStr(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("02 Jan 15:04")
}

func isBooked(s domain.LeadStatus) bool {
	return s == domain.LeadBooked || s == domain.LeadShowed || s == domain.LeadClosed
}

// conversationView maps a lead (+ optional live session transcript) to the
// dashboard's Conversation/lead shape. Used by /v1/leads and /v1/conversations.
func conversationView(l domain.Lead, sess *agent.Session) map[string]any {
	messages := []map[string]any{}
	lastMessage := ""
	if sess != nil {
		for i, t := range sess.Turns {
			messages = append(messages, map[string]any{
				"id":        l.ID + "-" + strconv.Itoa(i),
				"sender":    t.Role, // "patient" | "agent"
				"text":      t.Text,
				"timestamp": "",
			})
			lastMessage = t.Text
		}
	}
	return map[string]any{
		"id":              l.ID,
		"name":            l.Name,
		"phoneNumber":     l.Phone,
		"clinicId":        l.ClinicID,
		"status":          leadStatusTR(l.Status),
		"createdAt":       l.CreatedAt.Format(time.RFC3339),
		"lastMessage":     lastMessage,
		"lastMessageTime": l.CreatedAt.Format("02 Jan 15:04"),
		"messages":        messages,
		"qualification": map[string]any{
			"segment":         string(l.Segment),
			"intentPct":       int(l.Features.IntentScore*100 + 0.5),
			"urgency":         urgencyTR(l.Features.UrgencyScore),
			"budgetTry":       l.Features.StatedBudgetTRY,
			"language":        "TR",
			"booked":          isBooked(l.Status),
			"appointmentTime": apptTimeStr(l.ApptTime),
		},
	}
}

func appointmentView(a domain.Appointment) map[string]any {
	return map[string]any{
		"id":       a.ID,
		"clinicId": a.ClinicID,
		"leadId":   a.LeadID,
		"name":     a.Name,
		"phone":    a.Phone,
		"when":     a.When.Format(time.RFC3339),
		"segment":  string(a.Segment),
		"pShow":    a.PShow,
		"overbook": a.Overbook,
		"doctorId": a.DoctorID,
		"service":  a.Service,
	}
}

// ---- handlers ----------------------------------------------------------------

// handleListClinics returns the user's clinics enriched with SLA stats (delivered,
// guarantee, shadow price λ, on-track/behind) for the Klinikler + Dashboard pages.
func (s *Server) handleListClinics(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	slaByID := map[string]sla.Status{}
	for _, st := range s.eng.SLAReport(time.Now()) {
		slaByID[st.ClinicID] = st
	}
	out := []map[string]any{}
	for _, c := range s.eng.Clinics() {
		if !auth.CanAccessClinic(u, c.ID) {
			continue
		}
		m := clinicView(c)
		if st, ok := slaByID[c.ID]; ok {
			m["delivered"] = st.Delivered
			m["guarantee"] = st.Guaranteed
			m["shadowPrice"] = st.ShadowPrice
			m["deficit"] = st.Deficit
			m["targetNow"] = st.TargetNow
			if st.OnTrack {
				m["status"] = "on-track"
			} else {
				m["status"] = "behind"
			}
		}
		out = append(out, m)
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleListLeads(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeJSON(w, 503, map[string]string{"error": "store not configured"})
		return
	}
	u, _ := auth.UserFrom(r.Context())
	clinicID := r.URL.Query().Get("clinicId")
	if clinicID != "" && !auth.CanAccessClinic(u, clinicID) {
		writeJSON(w, 403, map[string]string{"error": "forbidden"})
		return
	}
	leads := s.store.ListLeads(store.LeadFilter{
		ClinicID: clinicID,
		Status:   domain.LeadStatus(r.URL.Query().Get("status")),
	})
	out := []map[string]any{}
	for _, l := range leads {
		if auth.CanAccessClinic(u, l.ClinicID) {
			out = append(out, conversationView(l, nil))
		}
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleListAppointments(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	clinicID := r.URL.Query().Get("clinicId")
	out := []map[string]any{}
	add := func(cid string) {
		names := map[string]string{}
		if s.store != nil {
			for _, d := range s.store.ListDoctors(cid) {
				names[d.ID] = d.Name
			}
		}
		for _, a := range s.eng.Appointments(cid) {
			v := appointmentView(a)
			if a.DoctorID != "" {
				v["doctor"] = names[a.DoctorID]
			}
			out = append(out, v)
		}
	}
	if clinicID != "" {
		if !auth.CanAccessClinic(u, clinicID) {
			writeJSON(w, 403, map[string]string{"error": "forbidden"})
			return
		}
		add(clinicID)
	} else {
		for _, c := range s.eng.Clinics() {
			if auth.CanAccessClinic(u, c.ID) {
				add(c.ID)
			}
		}
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleListConversations(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeJSON(w, 503, map[string]string{"error": "store not configured"})
		return
	}
	u, _ := auth.UserFrom(r.Context())
	clinicID := r.URL.Query().Get("clinicId")
	if clinicID != "" && !auth.CanAccessClinic(u, clinicID) {
		writeJSON(w, 403, map[string]string{"error": "forbidden"})
		return
	}
	out := []map[string]any{}
	for _, l := range s.store.ListLeads(store.LeadFilter{ClinicID: clinicID}) {
		if auth.CanAccessClinic(u, l.ClinicID) {
			out = append(out, conversationView(l, nil))
		}
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleConversation(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeJSON(w, 503, map[string]string{"error": "store not configured"})
		return
	}
	u, _ := auth.UserFrom(r.Context())
	l, ok := s.store.GetLead(r.PathValue("id"))
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "not_found"})
		return
	}
	if !auth.CanAccessClinic(u, l.ClinicID) {
		writeJSON(w, 403, map[string]string{"error": "forbidden"})
		return
	}
	var sess *agent.Session
	if s.sessions != nil {
		if se, ok := s.sessions.Get(l.Phone); ok {
			sess = se
		}
	}
	writeJSON(w, 200, conversationView(l, sess))
}

func (s *Server) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeJSON(w, 503, map[string]string{"error": "store not configured"})
		return
	}
	u, _ := auth.UserFrom(r.Context())
	var b struct {
		ClinicID string `json:"clinicId"`
		Name     string `json:"name"`
		Category string `json:"category"`
		Language string `json:"language"`
		Body     string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	if b.ClinicID != "" && !auth.CanAccessClinic(u, b.ClinicID) {
		writeJSON(w, 403, map[string]string{"error": "forbidden"})
		return
	}
	if b.Name == "" || b.Body == "" {
		writeJSON(w, 400, map[string]string{"error": "name and body required"})
		return
	}
	if b.Language == "" {
		b.Language = "tr"
	}
	if b.Category == "" {
		b.Category = "UTILITY"
	}
	t := domain.TemplateDraft{
		ID: "tmpl-" + newID(), ClinicID: b.ClinicID, Name: b.Name,
		Category: b.Category, Language: b.Language, Body: b.Body,
		Status: "PENDING", CreatedAt: time.Now(),
	}
	s.store.SaveTemplate(t)
	writeJSON(w, 201, t)
}

// handleConnections lists (GET) or upserts (POST) per-clinic integration status.
// On GET with a clinicId, the four integration types are returned, defaulting to
// env-derived status when not explicitly stored. No secrets are ever returned.
func (s *Server) handleConnections(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeJSON(w, 503, map[string]string{"error": "store not configured"})
		return
	}
	u, _ := auth.UserFrom(r.Context())

	if r.Method == http.MethodPost {
		var b struct {
			ClinicID  string `json:"clinicId"`
			Type      string `json:"type"`
			Connected bool   `json:"connected"`
			Detail    string `json:"detail"`
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		if !auth.CanAccessClinic(u, b.ClinicID) {
			writeJSON(w, 403, map[string]string{"error": "forbidden"})
			return
		}
		c := domain.Connection{
			ID: b.ClinicID + ":" + b.Type, ClinicID: b.ClinicID, Type: b.Type,
			Connected: b.Connected, Detail: b.Detail, UpdatedAt: time.Now(),
		}
		s.store.UpsertConnection(c)
		writeJSON(w, 200, c)
		return
	}

	clinicID := r.URL.Query().Get("clinicId")
	if clinicID != "" && !auth.CanAccessClinic(u, clinicID) {
		writeJSON(w, 403, map[string]string{"error": "forbidden"})
		return
	}
	if clinicID == "" {
		// No clinic specified: return whatever is stored across the user's clinics.
		out := []domain.Connection{}
		for _, c := range s.store.ListConnections("") {
			if auth.CanAccessClinic(u, c.ClinicID) {
				out = append(out, c)
			}
		}
		writeJSON(w, 200, out)
		return
	}
	writeJSON(w, 200, s.connectionsForClinic(clinicID))
}

// connectionsForClinic returns the 4 integration types for a clinic: a stored row
// if present, otherwise a default derived (read-only) from configured env.
func (s *Server) connectionsForClinic(clinicID string) []domain.Connection {
	stored := map[string]domain.Connection{}
	for _, c := range s.store.ListConnections(clinicID) {
		stored[c.Type] = c
	}
	types := []string{"whatsapp", "meta_ads", "google_ads", "web_form"}
	out := make([]domain.Connection, 0, len(types))
	for _, t := range types {
		if c, ok := stored[t]; ok {
			out = append(out, c)
			continue
		}
		out = append(out, domain.Connection{
			ID: clinicID + ":" + t, ClinicID: clinicID, Type: t,
			Connected: envConnected(t), Detail: "",
		})
	}
	return out
}

// envConnected derives a default connected state from configured env vars (no
// secrets exposed — only the boolean).
func envConnected(t string) bool {
	switch t {
	case "whatsapp":
		return os.Getenv("WHATSAPP_TOKEN") != "" && os.Getenv("WHATSAPP_PHONE_NUMBER_ID") != ""
	case "meta_ads":
		return os.Getenv("META_TOKEN") != "" && os.Getenv("META_AD_ACCOUNT_ID") != ""
	case "web_form":
		return true // the webform webhook is always available
	default: // google_ads
		return false
	}
}
