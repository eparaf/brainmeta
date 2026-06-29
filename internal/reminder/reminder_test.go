package reminder

import (
	"context"
	"testing"
	"time"

	"disci/brain/internal/domain"
)

type mockNotifier struct{ calls []string }

func (m *mockNotifier) SendTemplate(ctx context.Context, to, name, lang string, params ...string) error {
	m.calls = append(m.calls, to+"|"+name)
	return nil
}

func TestScheduler24hAnd2hAndConsent(t *testing.T) {
	now := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC)
	clinics := []domain.Clinic{{ID: "c1", Name: "Klinik"}}
	appts := []domain.Appointment{
		{ID: "a1", ClinicID: "c1", Phone: "+90111", When: now.Add(23 * time.Hour)},   // → 24h
		{ID: "a2", ClinicID: "c1", Phone: "+90222", When: now.Add(90 * time.Minute)}, // → 2h
		{ID: "a3", ClinicID: "c1", Phone: "+90333", When: now.Add(30 * time.Minute)}, // opted out
		{ID: "a4", ClinicID: "c1", Phone: "", When: now.Add(time.Hour)},              // no phone → skip
	}
	n := &mockNotifier{}
	allowed := func(p string) bool { return p != "+90333" }
	s := New(func() []domain.Clinic { return clinics }, func(string) []domain.Appointment { return appts }, n, allowed)

	sent := s.Tick(context.Background(), now)
	if sent != 2 {
		t.Fatalf("expected 2 reminders sent, got %d (%v)", sent, n.calls)
	}
	// Idempotent: a second tick at the same time sends nothing new.
	if again := s.Tick(context.Background(), now); again != 0 {
		t.Fatalf("expected no duplicate sends, got %d", again)
	}
	// Right templates to the right people.
	want := map[string]bool{"+90111|reminder_24h": true, "+90222|reminder_2h": true}
	for _, c := range n.calls {
		if !want[c] {
			t.Fatalf("unexpected reminder call: %s", c)
		}
	}
}
