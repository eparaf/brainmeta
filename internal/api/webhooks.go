package api

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"disci/brain/internal/agent"
	"disci/brain/internal/consent"
	"disci/brain/internal/meta"
	"disci/brain/internal/whatsapp"
)

// leadFetcher fetches the full lead (name/phone) for a Meta Lead Ads
// leadgen_id. An interface so tests can inject a fake instead of hitting the
// real Graph API; *meta.LeadAdsClient satisfies it.
type leadFetcher interface {
	FetchLead(ctx context.Context, leadgenID string) (meta.Lead, error)
}

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
	body, _ := io.ReadAll(r.Body)
	// Verify the payload is genuinely from Meta (HMAC-SHA256 over the raw body).
	if !whatsapp.VerifySignature(s.appSecret, r.Header.Get("X-Hub-Signature-256"), body) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.WriteHeader(200) // ack fast so Meta doesn't retry
	if s.agent == nil {
		return
	}
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
		// Resolve which clinic owns the RECEIVING number (set when that clinic
		// completed WhatsApp Embedded Signup — see handleOAuthToken). No match
		// (number not yet claimed by any clinic, or store unset) → clinic="" and
		// the brain routes across the marketplace, same as before this existed.
		clinicID := ""
		if s.store != nil && in.PhoneNumberID != "" {
			if cid, ok := s.store.ResolveClinicByPhoneNumberID(in.PhoneNumberID); ok {
				clinicID = cid
			}
		}
		res, err := s.agentTurn(ctx, in.From, clinicID, "", in.Text)
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

// metaLeadsPayload is the Meta Lead Ads webhook shape: an entry per Page, each
// with a "leadgen" change carrying only the leadgen_id — the full answers
// (name/phone) require a follow-up Graph API fetch (see leadFetcher).
type metaLeadsPayload struct {
	Entry []struct {
		Changes []struct {
			Field string `json:"field"`
			Value struct {
				LeadgenID string `json:"leadgen_id"`
				FormID    string `json:"form_id"`
				AdID      string `json:"ad_id"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

// handleMetaLeads accepts Meta Lead Ads webhooks. The payload carries only a
// leadgen_id; we verify the signature, ack immediately (Meta retries on any
// non-2xx or slow response), then fetch each lead's real name/phone via the
// Graph API and run it through the agent — the same path handleWebform uses,
// so a Lead Ads submission gets a WhatsApp follow-up exactly like a website form.
func (s *Server) handleMetaLeads(w http.ResponseWriter, r *http.Request) {
	// Verification handshake reuse (Meta uses the same hub.* params).
	if r.Method == http.MethodGet && s.cloud != nil {
		if ch, ok := s.cloud.VerifyWebhook(r.URL.Query()); ok {
			_, _ = w.Write([]byte(ch))
			return
		}
	}
	body, _ := io.ReadAll(r.Body)
	if !whatsapp.VerifySignature(s.appSecret, r.Header.Get("X-Hub-Signature-256"), body) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.WriteHeader(200) // ack fast so Meta doesn't retry
	if s.leadAds == nil {
		return // no Graph API token configured — nothing more we can do
	}
	var payload metaLeadsPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("meta leads: bad payload: %v", err)
		return
	}
	ctx := r.Context()
	for _, e := range payload.Entry {
		for _, ch := range e.Changes {
			if ch.Field != "leadgen" || ch.Value.LeadgenID == "" {
				continue
			}
			s.handleOneMetaLead(ctx, ch.Value.LeadgenID)
		}
	}
}

// handleOneMetaLead fetches one lead's real data and routes it into the agent
// (clinic/arm unresolved here — same "brain routes" convention as WhatsApp
// inbound; map ad_id/form_id → clinic+arm once that config surface exists).
func (s *Server) handleOneMetaLead(ctx context.Context, leadgenID string) {
	lead, err := s.leadAds.FetchLead(ctx, leadgenID)
	if err != nil {
		log.Printf("meta leads: fetch %s: %v", leadgenID, err)
		return
	}
	if lead.Phone == "" {
		log.Printf("meta leads: %s had no phone number in field_data", leadgenID)
		return
	}
	if s.agent == nil {
		return
	}
	msg := "Meta reklamından randevu talebi"
	if lead.Name != "" {
		msg = lead.Name + " — " + msg
	}
	res, err := s.agentTurn(ctx, lead.Phone, "", "", msg)
	if err != nil || res.Reply == "" {
		return
	}
	if s.cloud != nil && s.consent.Allowed(lead.Phone) {
		_ = s.cloud.SendText(ctx, lead.Phone, res.Reply)
	}
}
