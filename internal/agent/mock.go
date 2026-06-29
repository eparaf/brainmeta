package agent

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"disci/brain/internal/domain"
)

// MockLLM is a deterministic stand-in for the real Claude agent. It runs the
// qualification heuristics and templated replies so the ENTIRE pipeline
// (WhatsApp → qualify → brain → reply) runs end-to-end with no API key — for
// tests, local dev, and the simulator. A real ClaudeLLM (claude.go) implements
// the same interface for production.
type MockLLM struct{}

// Name identifies this provider in diagnostics.
func (MockLLM) Name() string { return "mock" }

// Qualify infers structured qualification from the latest patient message using
// keyword + currency heuristics (Turkish & English).
func (MockLLM) Qualify(ctx context.Context, convo []Turn) (Qualification, error) {
	last := lastPatient(convo)
	all := allPatientText(convo)
	low := strings.ToLower(all)

	seg := SegmentFromText(all)
	urgency := 0.4
	if containsAny(low, "ağrı", "agri", "acil", "pain", "urgent", "kırıldı", "kirildi", "swelling", "şiş") {
		urgency = 0.85
	}
	budget := ParseBudgetTRY(all)

	intent := 0.45
	if seg != domain.SegmentGeneral {
		intent += 0.2
	}
	if budget > 0 {
		intent += 0.2
	}
	if urgency > 0.6 {
		intent += 0.1
	}
	if intent > 0.95 {
		intent = 0.95
	}

	locale := "tr"
	if isEnglish(low) {
		locale = "en"
	}

	patientTurns := countRole(convo, "patient")
	// Enough to qualify once we know the treatment, OR after a couple of turns.
	done := seg != domain.SegmentGeneral || patientTurns >= 2

	q := Qualification{
		Segment: seg, Treatment: last, Urgency: urgency, IntentScore: intent,
		BudgetTRY: budget, Locale: locale, Messages: patientTurns, Done: done,
	}
	if !done {
		q.AskNext = askNext(locale)
	}
	return q, nil
}

// Compose writes the patient-facing reply around the brain's AUTHORITATIVE
// decision. It never invents a slot — ApptTime comes from the engine.
func (MockLLM) Compose(ctx context.Context, convo []Turn, rc ReplyContext) (string, error) {
	tr := rc.Locale != "en"
	// Follow-up turn after a booking already exists: confirm or acknowledge, never
	// create a new appointment.
	if rc.AlreadyBooked {
		when := rc.ApptTime.Format("02 Jan 15:04")
		if isAffirmative(lastPatient(convo)) {
			if tr {
				return fmt.Sprintf("Onaylandı ✓ %s randevunuzda görüşmek üzere. Hatırlatma göndereceğiz, iyi günler!", when), nil
			}
			return fmt.Sprintf("Confirmed ✓ — see you at your %s appointment. We'll send a reminder. Take care!", when), nil
		}
		if tr {
			return fmt.Sprintf("Randevunuz %s için ayarlı ✓. Başka bir sorunuz var mı?", when), nil
		}
		return fmt.Sprintf("Your appointment is set for %s ✓. Anything else I can help with?", when), nil
	}
	if rc.Booked {
		when := rc.ApptTime.Format("02 Jan 15:04")
		if tr {
			return fmt.Sprintf("Harika! Randevunuzu %s için oluşturdum. Onaylıyor musunuz? Hatırlatma göndereceğiz. 🦷", when), nil
		}
		return fmt.Sprintf("Great! I've reserved your appointment for %s. Shall I confirm? We'll send a reminder.", when), nil
	}
	switch rc.Reason {
	case "clinic_at_capacity", "no_capacity_anywhere":
		if tr {
			return "Şu an bu güne uygun yer kalmadı; en yakın uygun günü ayarlayıp size hemen döneceğiz.", nil
		}
		return "We're fully booked for that day — let me find you the next available slot and get right back to you.", nil
	default:
		if tr {
			return "Size en uygun seçeneği sunabilmem için birkaç kısa soru sorabilir miyim?", nil
		}
		return "Could I ask a couple of quick questions so I can offer you the best option?", nil
	}
}

func isAffirmative(s string) bool {
	return containsAny(strings.ToLower(s), "evet", "tamam", "olur", "onayl", "yes", "ok", "okay", "tabii", "kesinlikle")
}

func askNext(locale string) string {
	if locale == "en" {
		return "What treatment are you interested in — implants, smile design, braces, or a checkup?"
	}
	return "Hangi tedaviyle ilgileniyorsunuz — implant, gülüş tasarımı, ortodonti ya da kontrol?"
}

// ParseBudgetTRY extracts a stated budget and normalises it to TRY. It handles
// attached symbols ($1500, 1500₺), trailing words (1500 dolar/euro/tl), and
// thousand separators. Returns 0 if no budget is stated — distinct from a
// real zero, which the scorer encodes with a missing-value indicator.
func ParseBudgetTRY(text string) float64 {
	toks := strings.Fields(strings.ToLower(text))
	for i, tok := range toks {
		cur, num, ok := splitAmount(tok)
		if !ok {
			continue
		}
		// Currency may be attached, or in the next/previous token.
		if cur == "" && i+1 < len(toks) {
			cur = currencyOf(toks[i+1])
		}
		if cur == "" && i > 0 {
			cur = currencyOf(toks[i-1])
		}
		if cur == "" {
			cur = "TRY" // a bare number is assumed local currency
		}
		rate, ok := fxToTRY[cur]
		if !ok {
			rate = 1
		}
		return num * rate
	}
	return 0
}

// splitAmount parses a token that may carry an attached currency symbol/word and
// a number with separators, e.g. "$1,500", "1500tl", "150.000".
func splitAmount(tok string) (currency string, amount float64, ok bool) {
	cur := ""
	// Leading symbol.
	for sym := range fxToTRY {
		if strings.HasPrefix(tok, sym) && !isDigit(rune(sym[0])) {
			cur = canonCurrency(sym)
			tok = strings.TrimPrefix(tok, sym)
			break
		}
	}
	// Trailing currency word/symbol.
	for _, suf := range []string{"tl", "try", "₺", "usd", "$", "eur", "€", "gbp", "£", "dolar", "euro", "avro", "sterlin"} {
		if strings.HasSuffix(tok, suf) {
			if c := currencyOf(suf); c != "" {
				cur = c
			}
			tok = strings.TrimSuffix(tok, suf)
			break
		}
	}
	digits := strings.Map(func(r rune) rune {
		if isDigit(r) {
			return r
		}
		return -1 // drop separators (.,/space already split)
	}, tok)
	if digits == "" {
		return "", 0, false
	}
	n, err := strconv.ParseFloat(digits, 64)
	if err != nil || n <= 0 {
		return "", 0, false
	}
	return cur, n, true
}

func currencyOf(w string) string {
	switch w {
	case "tl", "try", "₺", "lira":
		return "TRY"
	case "usd", "$", "dolar", "dollar", "dollars":
		return "USD"
	case "eur", "€", "euro", "avro":
		return "EUR"
	case "gbp", "£", "sterlin", "pound", "pounds":
		return "GBP"
	}
	return ""
}

func canonCurrency(sym string) string {
	if c := currencyOf(strings.ToLower(sym)); c != "" {
		return c
	}
	return ""
}

func isDigit(r rune) bool { return r >= '0' && r <= '9' }

func isEnglish(low string) bool {
	// Crude: presence of common English words and absence of Turkish-specific ones.
	en := containsAny(low, "i want", "i need", "hello", "hi ", "implant cost", "how much", "appointment", "teeth")
	tr := containsAny(low, "istiyorum", "merhaba", "randevu", "fiyat", "diş", "dis", "ağrı")
	return en && !tr
}

func lastPatient(convo []Turn) string {
	for i := len(convo) - 1; i >= 0; i-- {
		if convo[i].Role == "patient" {
			return convo[i].Text
		}
	}
	return ""
}

func allPatientText(convo []Turn) string {
	var b strings.Builder
	for _, t := range convo {
		if t.Role == "patient" {
			b.WriteString(t.Text)
			b.WriteString(" ")
		}
	}
	return b.String()
}

func countRole(convo []Turn, role string) int {
	n := 0
	for _, t := range convo {
		if t.Role == role {
			n++
		}
	}
	return n
}

// compile-time check that MockLLM satisfies LLM.
var _ LLM = MockLLM{}
