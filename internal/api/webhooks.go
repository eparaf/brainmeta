package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"disci/brain/internal/agent"
	"disci/brain/internal/consent"
	"disci/brain/internal/whatsapp"
)

// agentTurn runs one message through the per-phone agent session.
func (s *Server) agentTurn(ctx context.Context, phone, clinicID, armID, msg string) (agent.Result, error) {
	sess, ok := s.sessions.Get(phone)
	if !ok {
		sess = &agent.Session{LeadID: "wa-" + phone, Phone: phone, HourOfDay: 14, DistanceKm: 6}
	}
	res, err := s.agent.Handle(ctx, sess, clinicID, armID, msg)
	s.sessions.Put(phone, sess)
	return res, err
}

// handleWAVerify answers Meta's webhook verification handshake (GET).
func (s *Server) handleWAVerify(w http.ResponseWriter, r *http.Request) {
	if s.cloud != nil {
		if ch, ok := s.cloud.VerifyWebhook(r.URL.Query()); ok {
			w.WriteHeader(200)
			_, _ = w.Write([]byte(ch))
			return
		}
		w.WriteHeader(403)
		return
	}
	w.WriteHeader(200)
}

// handleWAInbound processes inbound WhatsApp messages: opt-out handling, then
// the agent qualifies/decides and we reply via the Cloud API (inside 24h window).
func (s *Server) handleWAInbound(w http.ResponseWriter, r *http.Request) {
	defer w.WriteHeader(200) // ack fast so Meta doesn't retry
	if s.agent == nil {
		return
	}
	body, _ := io.ReadAll(r.Body)
	ins, err := whatsapp.ParseInbound(body)
	if err != nil {
		return
	}
	ctx := r.Context()
	for _, in := range ins {
		if consent.IsOptOutKeyword(in.Text) {
			s.consent.OptOut(in.From)
			if s.cloud != nil {
				_ = s.cloud.SendText(ctx, in.From, "Çıkış işleminiz alındı. Artık mesaj göndermeyeceğiz. Tekrar yazarsanız yeniden yardımcı oluruz.")
			}
			continue
		}
		if !s.consent.Allowed(in.From) {
			continue
		}
		// clinic/arm: in production map the receiving phone-number-id (or the
		// click-to-WhatsApp ad referral) to a clinic+arm. Empty → brain routes.
		res, err := s.agentTurn(ctx, in.From, "", "", in.Text)
		if err != nil || res.Reply == "" {
			continue
		}
		if s.cloud != nil {
			_ = s.cloud.SendText(ctx, in.From, res.Reply)
		}
	}
}

// handleWebform ingests a website form lead and runs it through the agent.
func (s *Server) handleWebform(w http.ResponseWriter, r *http.Request) {
	if s.agent == nil {
		writeJSON(w, 503, map[string]string{"error": "agent not configured"})
		return
	}
	var b struct {
		Phone, ClinicID, ArmID, Name, Message string
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	if b.Message == "" {
		b.Message = "Web formundan randevu talebi"
	}
	res, err := s.agentTurn(r.Context(), b.Phone, b.ClinicID, b.ArmID, b.Message)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	// If we have a live WhatsApp client + phone, also reach out on WhatsApp.
	if s.cloud != nil && b.Phone != "" && s.consent.Allowed(b.Phone) {
		_ = s.cloud.SendText(r.Context(), b.Phone, res.Reply)
	}
	writeJSON(w, 200, map[string]any{"reply": res.Reply, "booked": res.Decision.Booked, "apptTime": res.Decision.ApptTime})
}

// handleMetaLeads accepts Meta Lead Ads webhooks. Meta sends a leadgen id; in
// production you fetch the full lead (name/phone) via the Graph API, then drop
// it into the agent. Here we ack and accept the event.
func (s *Server) handleMetaLeads(w http.ResponseWriter, r *http.Request) {
	// Verification handshake reuse (Meta uses the same hub.* params).
	if r.Method == http.MethodGet && s.cloud != nil {
		if ch, ok := s.cloud.VerifyWebhook(r.URL.Query()); ok {
			_, _ = w.Write([]byte(ch))
			return
		}
	}
	_, _ = io.ReadAll(r.Body)
	writeJSON(w, 200, map[string]string{"status": "received"})
}
