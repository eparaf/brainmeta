package engine

import (
	"path/filepath"
	"testing"
	"time"

	"disci/brain/internal/config"
	"disci/brain/internal/domain"
	"disci/brain/internal/persist"
	"disci/brain/internal/store"
)

// TestSnapshotRoundTripPreservesLearning proves the moat survives a restart: a
// brain that has learned from outcomes, saved, and been recreated-from-snapshot
// must report the same learned ticket and guarantee progress — not reset to
// priors.
func TestSnapshotRoundTripPreservesLearning(t *testing.T) {
	clinic := domain.Clinic{
		ID: "umraniye", Segment: domain.SegmentImplant, CloseRate: 0.42,
		DailyCapacity: 10, GuaranteedApptsPerMonth: 80, MonthlyAdBudget: 220_000,
	}

	e1 := New(config.Default(), store.NewMemory())
	e1.RegisterClinic(clinic)

	// Teach it: 50 closed deals at a distinctive ticket size, plus qualified appts.
	yes := true
	feats := domain.LeadFeatures{IntentScore: 0.8, UrgencyScore: 0.7, MessagesExchanged: 6}
	for i := 0; i < 50; i++ {
		e1.Loop.Ingest(domain.Outcome{
			OutcomeID: filepathKey(i), LeadID: filepathKey(i), ClinicID: "umraniye",
			ArmID: "umraniye:meta:implant", Segment: domain.SegmentImplant,
			Qualified: &yes, Booked: &yes, Showed: &yes, Closed: &yes,
			Revenue: 90_000, AdCost: 220,
		}, feats)
	}
	learnedTicket := e1.Scorer.LearnedTicket(domain.SegmentImplant)

	dir := t.TempDir()
	fs := persist.NewFileStore(filepath.Join(dir, "snap.json"))
	if err := e1.SaveSnapshot(fs, time.Now()); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Fresh process: new engine, same clinic registered, load snapshot.
	e2 := New(config.Default(), store.NewMemory())
	e2.RegisterClinic(clinic)
	preLoad := e2.Scorer.LearnedTicket(domain.SegmentImplant)
	ok, err := e2.LoadSnapshot(fs)
	if err != nil || !ok {
		t.Fatalf("load: ok=%v err=%v", ok, err)
	}
	postLoad := e2.Scorer.LearnedTicket(domain.SegmentImplant)

	if postLoad == preLoad {
		t.Fatalf("snapshot did not change learned ticket (still prior %.0f)", preLoad)
	}
	if abs(postLoad-learnedTicket) > 1.0 {
		t.Fatalf("restored ticket %.1f != saved %.1f", postLoad, learnedTicket)
	}

	// Guarantee progress restored too.
	rep := e2.SLAReport(time.Now())
	if len(rep) != 1 || rep[0].Delivered != 50 {
		t.Fatalf("expected 50 delivered restored, got %+v", rep)
	}
}

// TestDedupSurvivesRestart is the regression test for the re-audit's #1 finding:
// after a restart, re-pulling the entire outcome history must NOT double-count.
func TestDedupSurvivesRestart(t *testing.T) {
	clinic := domain.Clinic{ID: "umraniye", Segment: domain.SegmentImplant, CloseRate: 0.42,
		DailyCapacity: 10, GuaranteedApptsPerMonth: 80, MonthlyAdBudget: 220_000}

	e1 := New(config.Default(), store.NewMemory())
	e1.RegisterClinic(clinic)
	yes := true
	outs := make([]domain.Outcome, 40)
	for i := range outs {
		outs[i] = domain.Outcome{OutcomeID: "o-" + itoa(i), LeadID: "l-" + itoa(i),
			ClinicID: "umraniye", ArmID: "umraniye:meta:implant", Segment: domain.SegmentImplant,
			Qualified: &yes, Booked: &yes}
		e1.IngestOutcome(outs[i])
	}
	if got := delivered(e1); got != 40 {
		t.Fatalf("pre-restart delivered = %d, want 40", got)
	}

	dir := t.TempDir()
	fs := persist.NewFileStore(filepath.Join(dir, "s.json"))
	_ = e1.SaveSnapshot(fs, time.Now())

	// Restart: new engine, load, then the sync loop re-pulls ALL outcomes.
	e2 := New(config.Default(), store.NewMemory())
	e2.RegisterClinic(clinic)
	_, _ = e2.LoadSnapshot(fs)
	dup := 0
	for _, o := range outs {
		if e2.IngestOutcome(o) {
			dup++ // counted as fresh — would be a double-count
		}
	}
	if dup != 0 {
		t.Fatalf("re-ingested %d outcomes after restart (double-count); want 0", dup)
	}
	if got := delivered(e2); got != 40 {
		t.Fatalf("post-restart delivered = %d, want 40 (no double-count)", got)
	}
}

func delivered(e *Engine) int {
	tot := 0
	for _, s := range e.SLAReport(time.Now()) {
		tot += s.Delivered
	}
	return tot
}

// TestOverbookingSpreadsAcrossDays is the regression test for the appointment-day
// bucketing fix: bookings to a tiny clinic must spill onto later days, not all
// collapse onto a single day's capacity.
func TestOverbookingSpreadsAcrossDays(t *testing.T) {
	e := New(config.Default(), store.NewMemory())
	e.RegisterClinic(domain.Clinic{ID: "c", Segment: domain.SegmentImplant, CloseRate: 0.4,
		DailyCapacity: 2, GuaranteedApptsPerMonth: 100, MonthlyAdBudget: 50_000})
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	days := map[string]int{}
	booked := 0
	for i := 0; i < 20; i++ {
		dec := e.HandleLead(domain.Lead{ID: "l" + itoa(i), ClinicID: "c", Segment: domain.SegmentImplant,
			Features: domain.LeadFeatures{IntentScore: 0.9, UrgencyScore: 0.8, MessagesExchanged: 6, DistanceKm: 4, HourOfDay: 14}}, now)
		if dec.Booked {
			booked++
			days[dec.ApptTime.Format("2006-01-02")]++
		}
	}
	if booked == 0 {
		t.Fatal("expected some bookings")
	}
	if len(days) < 2 {
		t.Fatalf("bookings should spread across multiple appointment days, got %d day(s)", len(days))
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func filepathKey(i int) string { return "lead-" + itoa(i) }

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}
