// Package voice is the VOICE-CALL channel of the agent. A real-time model
// (Gemini Live / OpenAI Realtime) listens and speaks; telephony (Twilio /
// LiveKit / SIP) carries the audio. But the agent ACTS through the same brain
// TOOLS as every other channel, and booking-state speech is DETERMINISTIC — the
// voice model can propose a booking but only the brain creates it (no
// hallucinated appointments on a call, just like on WhatsApp).
//
// Interfaces here are provider-agnostic; plug Gemini Live/OpenAI as ToolLLM +
// Speaker and Twilio/LiveKit as Telephony. The Mock* types let the whole loop
// run and be tested with no audio/creds.
package voice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"disci/brain/internal/agent"
)

// Turn is one utterance in a call.
type Turn struct {
	Role string // "caller" | "agent"
	Text string
}

// ToolLLM is the realtime voice brain: given the conversation + available tools,
// it decides what to SAY and which tools to CALL.
type ToolLLM interface {
	Decide(ctx context.Context, convo []Turn, tools []agent.Tool, clinicID string) (say string, calls []agent.ToolCall, err error)
}

// Speaker renders text to the caller (TTS / realtime audio out).
type Speaker interface {
	Say(ctx context.Context, text string) error
}

// Telephony places outbound calls (inbound arrives via the provider webhook).
type Telephony interface {
	Dial(ctx context.Context, to string) error
}

// Session is per-call state.
type Session struct {
	Phone    string
	ClinicID string
	ArmID    string
	Turns    []Turn
	Booked   bool
	ApptTime time.Time
}

func (s *Session) add(role, text string) { s.Turns = append(s.Turns, Turn{role, text}) }

// Agent drives a voice conversation through the brain tools.
type Agent struct {
	Tools   *agent.BrainTools
	LLM     ToolLLM
	Speaker Speaker
}

// Turn processes one caller utterance: the model decides → tools run against the
// brain → we SPEAK. Booking confirmations are deterministic (brain's real slot).
func (a *Agent) Turn(ctx context.Context, sess *Session, callerText string) (string, error) {
	sess.add("caller", callerText)
	if sess.Booked {
		spoken := replyAlreadyBooked(sess.ApptTime, isAffirmative(callerText))
		_ = a.Speaker.Say(ctx, spoken)
		sess.add("agent", spoken)
		return spoken, nil
	}

	say, calls, err := a.LLM.Decide(ctx, sess.Turns, a.Tools.Definitions(), sess.ClinicID)
	if err != nil {
		return "", err
	}

	spoken := say
	for _, c := range calls {
		res := a.Tools.Execute(ctx, c)
		// Booking-state speech is deterministic, filled with the brain's real data.
		if c.Name == "book_appointment" {
			if booked, _ := res.Output["booked"].(bool); booked {
				if iso, _ := res.Output["apptTime"].(string); iso != "" {
					if t, e := time.Parse(time.RFC3339, iso); e == nil {
						sess.Booked = true
						sess.ApptTime = t
						spoken = replyBooked(t)
					}
				}
			} else if spoken == "" {
				spoken = "Şu an uygun bir yer bulamadım; en yakın boşluğu ayarlayıp sizi geri arayacağız."
			}
		}
	}
	if spoken == "" {
		spoken = "Size nasıl yardımcı olabilirim?"
	}
	if a.Speaker != nil { // telephony renders speech via TwiML; Speaker optional
		_ = a.Speaker.Say(ctx, spoken)
	}
	sess.add("agent", spoken)
	return spoken, nil
}

func replyBooked(t time.Time) string {
	return fmt.Sprintf("Harika, randevunuzu %s için oluşturdum. Onaylıyor musunuz? Hatırlatma göndereceğiz.", t.Format("02 Jan 15:04"))
}
func replyAlreadyBooked(t time.Time, affirmative bool) string {
	if affirmative {
		return fmt.Sprintf("Onaylandı, %s randevunuzda görüşmek üzere.", t.Format("02 Jan 15:04"))
	}
	return fmt.Sprintf("Randevunuz %s için ayarlı. Başka bir sorunuz var mı?", t.Format("02 Jan 15:04"))
}
func isAffirmative(s string) bool {
	l := strings.ToLower(s)
	for _, w := range []string{"evet", "tamam", "olur", "onayl", "yes", "tabii"} {
		if strings.Contains(l, w) {
			return true
		}
	}
	return false
}

// ---- Mock implementations (no audio/creds) for tests & local demo -----------

// MockSpeaker records spoken lines.
type MockSpeaker struct{ Said []string }

func (m *MockSpeaker) Say(ctx context.Context, text string) error {
	m.Said = append(m.Said, text)
	return nil
}

// MockLLM is a deterministic voice brain: if the caller names a treatment it
// books; otherwise it asks. Stands in for Gemini Live / OpenAI Realtime in tests.
type MockLLM struct{}

func (MockLLM) Decide(ctx context.Context, convo []Turn, tools []agent.Tool, clinicID string) (string, []agent.ToolCall, error) {
	last := ""
	for i := len(convo) - 1; i >= 0; i-- {
		if convo[i].Role == "caller" {
			last = strings.ToLower(convo[i].Text)
			break
		}
	}
	seg := segmentOf(last)
	if seg == "" {
		return "Hangi tedaviyle ilgileniyorsunuz — implant, gülüş tasarımı, ortodonti ya da kontrol?", nil, nil
	}
	urgency := 0.3
	if strings.Contains(last, "ağrı") || strings.Contains(last, "acil") {
		urgency = 0.9
	}
	return "", []agent.ToolCall{{Name: "book_appointment", Args: map[string]any{
		"clinicId": clinicID, "segment": seg, "urgency": urgency}}}, nil
}

func segmentOf(t string) string {
	switch {
	case strings.Contains(t, "implant"):
		return "implant"
	case strings.Contains(t, "gülüş") || strings.Contains(t, "gulus") || strings.Contains(t, "beyaz"):
		return "aesthetic"
	case strings.Contains(t, "tel") || strings.Contains(t, "ortodonti"):
		return "ortho"
	case strings.Contains(t, "kontrol") || strings.Contains(t, "temizlik"):
		return "general"
	}
	return ""
}

var _ ToolLLM = MockLLM{}
var _ Speaker = (*MockSpeaker)(nil)
