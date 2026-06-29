// Package agent is the brain's MOUTH — the conversational layer that qualifies
// inbound WhatsApp leads and books them. It is built on one hard principle:
//
//	The LLM PROPOSES, the deterministic brain DISPOSES.
//
// The LLM runs the Turkish/English qualification dialogue and emits a STRUCTURED
// result (intent, treatment, urgency, budget). Deterministic Go converts that to
// model features and calls engine.HandleLead — the brain alone decides whether
// and where to book, and which slot. The LLM never invents a slot, never bypasses
// the SLA/capacity logic; it only phrases the reply around the brain's decision.
// This is what lets us sell a guarantee on top of a generative agent.
package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"disci/brain/internal/domain"
	"disci/brain/internal/engine"
	"disci/brain/internal/priors"
)

// Turn is one message in the conversation.
type Turn struct {
	Role string // "patient" | "agent"
	Text string
}

// Qualification is the STRUCTURED output the LLM must produce. This is the
// contract between the generative layer and the deterministic brain — frozen and
// versioned so train/serve stay consistent.
type Qualification struct {
	Segment    domain.Segment `json:"segment"`
	Treatment  string         `json:"treatment"`
	Urgency    float64        `json:"urgency"` // 0..1
	IntentScore float64       `json:"intent"`  // 0..1, how serious/ready
	BudgetTRY  float64        `json:"budgetTry"` // currency-normalised; 0 if unknown
	Locale     string         `json:"locale"`    // "tr" | "en"
	Messages   int            `json:"messages"`
	Done       bool           `json:"done"` // enough gathered to qualify & book
	AskNext    string         `json:"askNext"` // next question if not done
}

// ReplyContext is what the LLM gets to phrase a patient-facing reply around the
// brain's decision. The slot/booking facts are AUTHORITATIVE — supplied by the
// brain, not the model.
type ReplyContext struct {
	Booked       bool
	AlreadyBooked bool // session already has a booking; this is a follow-up turn
	ApptTime     time.Time
	Locale       string
	Reason       string // engine decision reason (for non-booked paths)
}

// LLM abstracts a tool-use / structured-output model (Claude/Gemini in
// production, a deterministic mock in tests/dev).
type LLM interface {
	// Qualify runs qualification reasoning over the conversation and returns the
	// structured result.
	Qualify(ctx context.Context, convo []Turn) (Qualification, error)
	// Compose writes the patient-facing reply around the brain's decision.
	Compose(ctx context.Context, convo []Turn, rc ReplyContext) (string, error)
	// Name is a short identifier shown in health/diagnostics (e.g. "gemini:...").
	Name() string
}

// Provider returns the active LLM's name (for /healthz and the UI).
func (a *Agent) Provider() string {
	if a == nil || a.LLM == nil {
		return "none"
	}
	return a.LLM.Name()
}

// Agent drives a conversation to a booking decision.
type Agent struct {
	LLM LLM
	Eng *engine.Engine
	Now func() time.Time
}

// New builds an agent.
func New(llm LLM, eng *engine.Engine) *Agent {
	return &Agent{LLM: llm, Eng: eng, Now: time.Now}
}

// Result is the outcome of handling one inbound turn.
type Result struct {
	Reply         string
	Qualification Qualification
	Decision      engine.LeadDecision
	Acted         bool // true if the brain made a booking decision this turn
}

// Handle processes one inbound patient message within a session. If the LLM
// judges enough has been gathered (Done), it qualifies → hands to the brain →
// composes a reply around the brain's authoritative decision. Otherwise it asks
// the next question.
func (a *Agent) Handle(ctx context.Context, sess *Session, clinicID, armID, patientMsg string) (Result, error) {
	sess.add("patient", patientMsg)

	// 1) The LLM ONLY extracts structured info from the conversation. We MERGE it
	//    into the session so qualification ACCUMULATES across turns (it never
	//    resets — the fix for "nitelik değişmiyor").
	q, err := a.LLM.Qualify(ctx, sess.Turns)
	if err != nil {
		return Result{}, err
	}
	sess.Qual = mergeQual(sess.Qual, q)
	if sess.Locale == "" {
		sess.Locale = q.Locale
	}
	tr := sess.Locale != "en"
	res := Result{Qualification: sess.Qual} // always surface the accumulated state

	// 2) Booking-state replies are DETERMINISTIC templates filled with the brain's
	//    real data — the LLM is never allowed to assert/invent/cancel an
	//    appointment (the fix for the hallucinated "you have an appointment").
	if sess.Booked {
		res.Reply = replyAlreadyBooked(tr, sess.ApptTime, isAffirmative(patientMsg))
		sess.add("agent", res.Reply)
		return res, nil
	}

	// 3) Enough gathered → the BRAIN decides (book or not). Deterministic reply.
	if sess.Qual.Done {
		lead := domain.Lead{
			ID: sess.LeadID, Phone: sess.Phone, Name: sess.Name,
			ClinicID: clinicID, ArmID: armID, Segment: sess.Qual.Segment,
			CreatedAt: a.Now(), Features: sess.Qual.ToFeatures(sess), Status: domain.LeadNew,
		}
		dec := a.Eng.HandleLead(lead, a.Now())
		res.Decision = dec
		res.Acted = true
		if dec.Booked {
			sess.Booked = true
			sess.ApptTime = dec.ApptTime
			res.Reply = replyBooked(tr, dec.ApptTime)
		} else {
			res.Reply = replyNoSlot(tr)
		}
		sess.add("agent", res.Reply)
		return res, nil
	}

	// 4) Need more info → ask the next question. This is the ONLY place the LLM's
	//    free text is used, and a question cannot fabricate a booking.
	reply := strings.TrimSpace(q.AskNext)
	if reply == "" {
		reply = replyDefaultQuestion(tr)
	}
	sess.add("agent", reply)
	res.Reply = reply
	return res, nil
}

// mergeQual accumulates qualification across turns: keep known fields, update
// with any new non-empty info. A specific segment is never overwritten by a
// later vague "general".
func mergeQual(acc, q Qualification) Qualification {
	if q.Segment != "" && (acc.Segment == "" || q.Segment != domain.SegmentGeneral) {
		acc.Segment = q.Segment
	}
	if q.Urgency > 0 {
		acc.Urgency = q.Urgency
	}
	if q.IntentScore > 0 {
		acc.IntentScore = q.IntentScore
	}
	if q.BudgetTRY > 0 {
		acc.BudgetTRY = q.BudgetTRY
	}
	if q.Locale != "" {
		acc.Locale = q.Locale
	}
	if q.Treatment != "" {
		acc.Treatment = q.Treatment
	}
	acc.Done = q.Done
	acc.AskNext = q.AskNext
	acc.Messages = q.Messages
	return acc
}

func replyBooked(tr bool, t time.Time) string {
	when := t.Format("02 Jan 15:04")
	if tr {
		return fmt.Sprintf("Harika! Randevunuzu %s için oluşturdum. Onaylıyor musunuz? Hatırlatma göndereceğiz.", when)
	}
	return fmt.Sprintf("Great! I've reserved your appointment for %s. Shall I confirm? We'll send a reminder.", when)
}
func replyAlreadyBooked(tr bool, t time.Time, affirmative bool) string {
	when := t.Format("02 Jan 15:04")
	if affirmative {
		if tr {
			return fmt.Sprintf("Onaylandı ✓ %s randevunuzda görüşmek üzere. Hatırlatma göndereceğiz.", when)
		}
		return fmt.Sprintf("Confirmed ✓ — see you on %s. We'll send a reminder.", when)
	}
	if tr {
		return fmt.Sprintf("Randevunuz %s için ayarlı ✓. Başka bir sorunuz var mı?", when)
	}
	return fmt.Sprintf("Your appointment is set for %s ✓. Anything else I can help with?", when)
}
func replyNoSlot(tr bool) string {
	if tr {
		return "Şu an uygun bir yer kalmadı; en yakın boşluğu ayarlayıp sizi en kısa sürede arayacağız."
	}
	return "We're fully booked right now — we'll find the next available slot and get back to you shortly."
}
func replyDefaultQuestion(tr bool) string {
	if tr {
		return "Size en uygun randevuyu ayarlayabilmem için hangi tedaviyle ilgilendiğinizi ve ne zaman müsait olduğunuzu öğrenebilir miyim?"
	}
	return "To find you the best slot — which treatment are you interested in, and when are you available?"
}

// ToFeatures maps the LLM's structured qualification into the scorer's feature
// struct deterministically. Response speed is measured from the session.
func (q Qualification) ToFeatures(sess *Session) domain.LeadFeatures {
	return domain.LeadFeatures{
		FirstResponseSecs: sess.FirstResponseSecs,
		MessagesExchanged: float64(sess.Count()),
		DistanceKm:        sess.DistanceKm,
		HourOfDay:         sess.HourOfDay,
		StatedBudgetTRY:   q.BudgetTRY,
		UrgencyScore:      norm01(q.Urgency),
		PastNoShows:       sess.PastNoShows,
		IntentScore:       norm01(q.IntentScore),
	}
}

// norm01 maps a score onto [0,1] defensively. We ask the model for 0..1, but
// different models sometimes answer on a 0..5 or 0..10 scale — so rather than
// clamping everything above 1 down to 1 (losing all signal), we infer the scale.
func norm01(x float64) float64 {
	switch {
	case x <= 1:
		if x < 0 {
			return 0
		}
		return x
	case x <= 5:
		return x / 5
	default:
		v := x / 10
		if v > 1 {
			v = 1
		}
		return v
	}
}

// Session holds per-conversation state.
type Session struct {
	LeadID            string
	Phone             string
	Name              string
	Turns             []Turn
	FirstResponseSecs float64
	DistanceKm        float64
	HourOfDay         float64
	PastNoShows       float64

	// Accumulated qualification across turns (never reset — merged each message).
	Qual Qualification

	// Set once the brain books this session, so follow-up turns don't re-book.
	Booked   bool
	ApptTime time.Time
	Locale   string
}

func (s *Session) add(role, text string) { s.Turns = append(s.Turns, Turn{Role: role, Text: text}) }

// Count returns the number of patient turns so far.
func (s *Session) Count() int {
	n := 0
	for _, t := range s.Turns {
		if t.Role == "patient" {
			n++
		}
	}
	return n
}

// ---- Deterministic helpers shared by the mock and any real adapter ----------

// fxToTRY converts a stated amount in a given currency to TRY. The lira moves
// fast; rates are anchored to the central priors.USDTRY constant.
var fxToTRY = map[string]float64{
	"TRY": 1,
	"TL":  1,
	"USD": priors.USDTRY,
	"$":   priors.USDTRY,
	"EUR": priors.USDTRY * 1.08,
	"€":   priors.USDTRY * 1.08,
	"GBP": priors.USDTRY * 1.27,
	"£":   priors.USDTRY * 1.27,
}

// SegmentFromText infers the dental segment from free text (TR + EN keywords).
func SegmentFromText(text string) domain.Segment {
	t := strings.ToLower(text)
	switch {
	case containsAny(t, "gülüş", "gulus", "veneer", "lamina", "zirkon", "estetik", "smile", "hollywood"):
		return domain.SegmentAesthetic
	case containsAny(t, "implant", "implant", "diş eksik", "dis eksik", "missing tooth"):
		return domain.SegmentImplant
	case containsAny(t, "ortodonti", "tel", "braces", "aligner", "invisalign", "şeffaf plak"):
		return domain.SegmentOrtho
	default:
		return domain.SegmentGeneral
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
