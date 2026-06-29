package voice

import (
	"context"
	"strings"
	"testing"

	"disci/brain/internal/agent"
	"disci/brain/internal/config"
	"disci/brain/internal/domain"
	"disci/brain/internal/engine"
	"disci/brain/internal/store"
)

func newAgent() (*Agent, *MockSpeaker) {
	e := engine.New(config.Default(), store.NewMemory())
	e.RegisterClinic(domain.Clinic{ID: "umraniye", Segment: domain.SegmentImplant, CloseRate: 0.42,
		DailyCapacity: 10, GuaranteedApptsPerMonth: 80, MonthlyAdBudget: 220_000})
	sp := &MockSpeaker{}
	return &Agent{Tools: agent.NewBrainTools(e), LLM: MockLLM{}, Speaker: sp}, sp
}

// TestVoiceBooksThroughTools: a caller naming a treatment with pain → the voice
// model calls book_appointment → the brain books → we SPEAK a deterministic
// confirmation containing the brain's real slot (not invented).
func TestVoiceBooksThroughTools(t *testing.T) {
	a, sp := newAgent()
	sess := &Session{Phone: "+90555", ClinicID: "umraniye"}

	reply, err := a.Turn(context.Background(), sess, "implant istiyorum acil ağrım var")
	if err != nil {
		t.Fatal(err)
	}
	if !sess.Booked {
		t.Fatalf("expected booking via tools, reply=%q", reply)
	}
	want := sess.ApptTime.Format("02 Jan 15:04")
	if !strings.Contains(reply, want) {
		t.Fatalf("spoken reply must contain brain's real slot %q; got %q", want, reply)
	}
	if len(sp.Said) != 1 || sp.Said[0] != reply {
		t.Fatalf("speaker should have spoken the reply once")
	}

	// Follow-up after booking → no re-book, deterministic ack.
	r2, _ := a.Turn(context.Background(), sess, "evet")
	if !strings.Contains(r2, "Onaylandı") {
		t.Fatalf("affirmative follow-up should confirm, got %q", r2)
	}
}

// TestVoiceAsksWhenUnclear: vague opener → no tool call, asks a question.
func TestVoiceAsksWhenUnclear(t *testing.T) {
	a, _ := newAgent()
	sess := &Session{Phone: "+90555", ClinicID: "umraniye"}
	reply, _ := a.Turn(context.Background(), sess, "merhaba")
	if sess.Booked || reply == "" || !strings.Contains(strings.ToLower(reply), "tedavi") {
		t.Fatalf("vague opener should ask a treatment question, got %q", reply)
	}
}
