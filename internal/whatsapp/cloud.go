package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Cloud is a real WhatsApp Cloud API (Meta) client. It both RECEIVES inbound
// messages (via the webhook handlers in the api package, which call ParseInbound
// / VerifyWebhook here) and SENDS messages:
//   - SendText      → free-form reply inside the 24h customer-service window
//   - SendTemplate  → business-initiated, Meta-APPROVED template (reminders etc.)
//
// Configure with the phone-number id + a permanent access token from Meta, and a
// verify token you choose (must match what you enter in the Meta webhook setup).
type Cloud struct {
	Token         string // permanent access token (Bearer)
	PhoneNumberID string // WhatsApp phone number id
	VerifyToken   string // your chosen webhook verify token
	GraphVersion  string // e.g. "v21.0"
	HTTP          *http.Client
}

func NewCloud(token, phoneNumberID, verifyToken string) *Cloud {
	return &Cloud{
		Token: token, PhoneNumberID: phoneNumberID, VerifyToken: verifyToken,
		GraphVersion: "v21.0", HTTP: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Cloud) base() string {
	v := c.GraphVersion
	if v == "" {
		v = "v21.0"
	}
	return fmt.Sprintf("https://graph.facebook.com/%s/%s/messages", v, c.PhoneNumberID)
}

// SendText sends a free-form text message (valid only inside the 24h window).
func (c *Cloud) SendText(ctx context.Context, to, body string) error {
	return c.post(ctx, map[string]any{
		"messaging_product": "whatsapp", "to": to, "type": "text",
		"text": map[string]any{"body": body},
	})
}

// SendTemplate sends an approved template message with body parameters.
func (c *Cloud) SendTemplate(ctx context.Context, to, name, lang string, params ...string) error {
	comps := []any{}
	if len(params) > 0 {
		ps := make([]any, len(params))
		for i, p := range params {
			ps[i] = map[string]any{"type": "text", "text": p}
		}
		comps = append(comps, map[string]any{"type": "body", "parameters": ps})
	}
	return c.post(ctx, map[string]any{
		"messaging_product": "whatsapp", "to": to, "type": "template",
		"template": map[string]any{"name": name, "language": map[string]any{"code": lang}, "components": comps},
	})
}

// Send implements datasource.Messenger (structurally): a business-initiated
// reminder via an approved template. templateID is the WhatsApp template name.
func (c *Cloud) Send(ctx context.Context, phone, templateID string, vars map[string]string) error {
	lang := "tr"
	// Pass the appointment time as the body parameter when present.
	var params []string
	if v, ok := vars["appt_time"]; ok {
		params = []string{v}
	}
	return c.SendTemplate(ctx, phone, templateID, lang, params...)
}

func (c *Cloud) post(ctx context.Context, body map[string]any) error {
	if c.Token == "" || c.PhoneNumberID == "" {
		return fmt.Errorf("whatsapp cloud: missing token/phone-number-id")
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base(), bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("whatsapp cloud: status %d: %s", resp.StatusCode, string(data))
	}
	return nil
}

// VerifyWebhook handles Meta's GET verification handshake. Return (challenge,
// true) to echo back when the verify token matches.
func (c *Cloud) VerifyWebhook(q url.Values) (string, bool) {
	if q.Get("hub.mode") == "subscribe" && q.Get("hub.verify_token") == c.VerifyToken {
		return q.Get("hub.challenge"), true
	}
	return "", false
}

// Inbound is a normalised inbound WhatsApp message.
type Inbound struct {
	From string // sender phone (wa id)
	Text string
	Name string // profile name if present
}

// ParseInbound extracts inbound text messages from a webhook payload.
func ParseInbound(body []byte) ([]Inbound, error) {
	var p struct {
		Entry []struct {
			Changes []struct {
				Value struct {
					Contacts []struct {
						Profile struct{ Name string `json:"name"` } `json:"profile"`
						WaID    string `json:"wa_id"`
					} `json:"contacts"`
					Messages []struct {
						From string `json:"from"`
						Type string `json:"type"`
						Text struct{ Body string `json:"body"` } `json:"text"`
					} `json:"messages"`
				} `json:"value"`
			} `json:"changes"`
		} `json:"entry"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, err
	}
	var out []Inbound
	for _, e := range p.Entry {
		for _, ch := range e.Changes {
			name := ""
			if len(ch.Value.Contacts) > 0 {
				name = ch.Value.Contacts[0].Profile.Name
			}
			for _, m := range ch.Value.Messages {
				if m.Type == "text" {
					out = append(out, Inbound{From: m.From, Text: m.Text.Body, Name: name})
				}
			}
		}
	}
	return out, nil
}
