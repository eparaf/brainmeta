package agent

import (
	"context"
	"strings"
	"testing"

	"disci/brain/internal/config"
	"disci/brain/internal/domain"
	"disci/brain/internal/engine"
	"disci/brain/internal/store"
)

func newEng() *engine.Engine {
	st := store.NewMemory()
	e := engine.New(config.Default(), st)
	e.RegisterClinic(domain.Clinic{
		ID: "umraniye", Segment: domain.SegmentImplant, CloseRate: 0.42,
		DailyCapacity: 10, GuaranteedApptsPerMonth: 80, MonthlyAdBudget: 220_000,
	})
	return e
}

// TestAgentQualifiesAndBooks: a clear Turkish implant message should qualify in
// one turn, the brain should book it, and the reply must reference the brain's
// AUTHORITATIVE appointment time (not an invented one).
func TestAgentQualifiesAndBooks(t *testing.T) {
	a := New(MockLLM{}, newEng())
	sess := &Session{LeadID: "wa-1", HourOfDay: 14, DistanceKm: 5, FirstResponseSecs: 30}
	res, err := a.Handle(context.Background(), sess, "umraniye", "umraniye:meta:implant",
		"Merhaba, implant yaptırmak istiyorum, bütçem 60000 TL acil ağrım var")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Acted {
		t.Fatalf("agent should have consulted the brain; qualification=%+v", res.Qualification)
	}
	if !res.Decision.Booked {
		t.Fatalf("expected booking, got reason=%s", res.Decision.Reason)
	}
	if res.Qualification.Segment != domain.SegmentImplant {
		t.Fatalf("expected implant segment, got %s", res.Qualification.Segment)
	}
	// Guardrail: the reply must mention the brain's real slot time.
	want := res.Decision.ApptTime.Format("02 Jan 15:04")
	if !strings.Contains(res.Reply, want) {
		t.Fatalf("reply must reference the brain's authoritative slot %q; got %q", want, res.Reply)
	}
}

// TestCurrencyParsedToTRY checks the deterministic multi-currency budget parser
// (the fix for the ~40× tourism currency error).
func TestCurrencyParsedToTRY(t *testing.T) {
	cases := map[string]bool{ // text -> expect budget > 100k TRY (foreign currency)
		"I want veneers, budget 3000 USD": true,
		"bütçem 2500 euro":                true,
		"budget is 2000 GBP":              true,
		"bütçem 60000 TL":                 false, // local; ~60k, below 100k
	}
	for text, big := range cases {
		got := ParseBudgetTRY(text)
		if got <= 0 {
			t.Fatalf("no budget parsed from %q", text)
		}
		if big && got < 100_000 {
			t.Fatalf("%q: expected foreign-currency budget >100k TRY, got %.0f", text, got)
		}
		if !big && got > 100_000 {
			t.Fatalf("%q: expected local budget <100k TRY, got %.0f", text, got)
		}
	}
}

// TestAgentAsksWhenUnclear: a vague first message should NOT book — the agent
// asks a clarifying question and the brain is not consulted.
func TestAgentAsksWhenUnclear(t *testing.T) {
	a := New(MockLLM{}, newEng())
	sess := &Session{LeadID: "wa-2", HourOfDay: 11}
	res, err := a.Handle(context.Background(), sess, "umraniye", "umraniye:meta:implant", "merhaba")
	if err != nil {
		t.Fatal(err)
	}
	if res.Acted {
		t.Fatalf("vague message should not reach the brain yet")
	}
	if res.Reply == "" {
		t.Fatalf("expected a clarifying question")
	}
}
