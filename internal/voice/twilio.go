package voice

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Twilio Voice integration (turn-based via TwiML). Twilio carries the call and
// does Turkish STT (<Gather input="speech">) + TTS (<Say>), so there's no audio
// streaming to manage — production-proven for appointment bots and live the
// moment you point a Twilio number's voice webhook at /webhooks/voice. Outbound
// calls (no-show recovery) use the REST API and need creds; INBOUND needs only a
// public URL. The conversation still runs through the brain tools.

// TwilioHandler serves the voice webhooks and keeps per-call sessions.
type TwilioHandler struct {
	Agent           *Agent
	ActionBase      string                               // public base for callback URLs, e.g. https://x.ngrok.io
	ClinicForNumber func(to string) (clinic, arm string) // map the dialed number → clinic

	mu       sync.Mutex
	sessions map[string]*Session // keyed by Twilio CallSid
}

func NewTwilioHandler(a *Agent, actionBase string, resolver func(string) (string, string)) *TwilioHandler {
	return &TwilioHandler{Agent: a, ActionBase: actionBase, ClinicForNumber: resolver, sessions: map[string]*Session{}}
}

func (h *TwilioHandler) clinic(to string) (string, string) {
	if h.ClinicForNumber != nil {
		return h.ClinicForNumber(to)
	}
	return "umraniye", "umraniye:meta:implant" // demo default
}

// Incoming handles the inbound-call webhook: greet + open the mic.
func (h *TwilioHandler) Incoming(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	callSid := r.FormValue("CallSid")
	clinicID, armID := h.clinic(r.FormValue("To"))
	h.mu.Lock()
	h.sessions[callSid] = &Session{Phone: r.FormValue("From"), ClinicID: clinicID, ArmID: armID}
	h.mu.Unlock()
	writeTwiML(w, say("Merhaba, randevu asistanına hoş geldiniz. Size nasıl yardımcı olabilirim?")+
		gather(h.ActionBase+"/webhooks/voice/gather"))
}

// Gather handles each speech turn: run the agent over the transcript, speak the
// reply, then either continue listening or hang up after a booking.
func (h *TwilioHandler) Gather(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	callSid := r.FormValue("CallSid")
	speech := r.FormValue("SpeechResult")
	h.mu.Lock()
	sess := h.sessions[callSid]
	h.mu.Unlock()
	if sess == nil {
		writeTwiML(w, say("Bir sorun oluştu, sizi geri arayacağız.")+"<Hangup/>")
		return
	}
	if speech == "" {
		writeTwiML(w, say("Sizi duyamadım, tekrar eder misiniz?")+gather(h.ActionBase+"/webhooks/voice/gather"))
		return
	}
	reply, err := h.Agent.Turn(r.Context(), sess, speech)
	if err != nil {
		writeTwiML(w, say("Şu an bağlantıda sorun var, sizi geri arayacağız.")+"<Hangup/>")
		return
	}
	body := say(reply)
	if sess.Booked {
		body += say("Görüşmek üzere, iyi günler.") + "<Hangup/>"
	} else {
		body += gather(h.ActionBase + "/webhooks/voice/gather")
	}
	writeTwiML(w, body)
}

// ---- TwiML helpers ----------------------------------------------------------

func say(text string) string {
	return fmt.Sprintf(`<Say language="tr-TR" voice="Polly.Filiz">%s</Say>`, xmlEscape(text))
}
func gather(action string) string {
	return fmt.Sprintf(`<Gather input="speech" language="tr-TR" speechTimeout="auto" action="%s" method="POST"/>`,
		xmlEscape(action))
}
func writeTwiML(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?><Response>`+body+`</Response>`)
}
func xmlEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// ---- Outbound calls (REST) — for no-show recovery --------------------------

// TwilioClient places outbound calls via the REST API. Needs account creds.
type TwilioClient struct {
	AccountSID string
	AuthToken  string
	From       string // Twilio number to call from
	VoiceURL   string // public webhook the call connects to (TwiML)
	HTTP       *http.Client
}

func NewTwilioClient(sid, token, from, voiceURL string) *TwilioClient {
	return &TwilioClient{AccountSID: sid, AuthToken: token, From: from, VoiceURL: voiceURL,
		HTTP: &http.Client{Timeout: 15 * time.Second}}
}

// Dial places an outbound call to `to`, connecting it to VoiceURL (which serves
// TwiML driving the agent). Implements voice.Telephony.
func (c *TwilioClient) Dial(ctx context.Context, to string) error {
	if c.AccountSID == "" || c.AuthToken == "" {
		return fmt.Errorf("twilio: missing creds")
	}
	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Calls.json", c.AccountSID)
	form := url.Values{"To": {to}, "From": {c.From}, "Url": {c.VoiceURL}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.AccountSID, c.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twilio dial: status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

var _ Telephony = (*TwilioClient)(nil)
