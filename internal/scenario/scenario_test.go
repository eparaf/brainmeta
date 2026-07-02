package scenario

import (
	"testing"

	"disci/brain/internal/domain"
	"disci/brain/internal/priors"
)

func basePlan() CampaignPlan {
	return CampaignPlan{
		ClinicID:      "umraniye",
		Segment:       domain.SegmentImplant,
		Platform:      domain.PlatformGoogle,
		Audience:      priors.AudienceLocalTR,
		MonthlyBudget: 50_000,
		Runs:          2000,
		Seed:          7,
	}
}

func mustSim(t *testing.T, p CampaignPlan) Result {
	t.Helper()
	r, err := New(nil).Simulate(p)
	if err != nil {
		t.Fatalf("Simulate: %v", err)
	}
	return r
}

// Determinism: same seed → identical result (repo rule 10).
func TestDeterministic(t *testing.T) {
	a := mustSim(t, basePlan())
	b := mustSim(t, basePlan())
	if a.BookedAppointments != b.BookedAppointments {
		t.Fatalf("nondeterministic: %+v vs %+v", a.BookedAppointments, b.BookedAppointments)
	}
	if a.CostPerLead != b.CostPerLead {
		t.Fatalf("nondeterministic cost: %+v vs %+v", a.CostPerLead, b.CostPerLead)
	}
}

// Band ordering: P10 ≤ P50 ≤ P90 for a count metric.
func TestBandOrdering(t *testing.T) {
	r := mustSim(t, basePlan())
	for _, band := range []struct {
		name string
		b    Band
	}{
		{"booked", r.BookedAppointments},
		{"kept", r.KeptAppointments},
		{"qualified", r.QualifiedLeads},
		{"clicks", r.Clicks},
	} {
		if band.b.P10 > band.b.P50 || band.b.P50 > band.b.P90 {
			t.Errorf("%s band not ordered: %+v", band.name, band.b)
		}
	}
	// Kept ≤ booked always (show rate ≤ 1).
	if r.KeptAppointments.P50 > r.BookedAppointments.P50 {
		t.Errorf("kept (%.2f) > booked (%.2f)", r.KeptAppointments.P50, r.BookedAppointments.P50)
	}
}

// Monotone in budget: more money → more expected appointments (while budget-limited).
func TestMoreBudgetMoreAppointments(t *testing.T) {
	lo := basePlan()
	lo.MonthlyBudget = 20_000
	hi := basePlan()
	hi.MonthlyBudget = 80_000
	rl := mustSim(t, lo)
	rh := mustSim(t, hi)
	if rh.BookedAppointments.P50 <= rl.BookedAppointments.P50 {
		t.Errorf("expected more appts with bigger budget: lo=%.2f hi=%.2f",
			rl.BookedAppointments.P50, rh.BookedAppointments.P50)
	}
}

// Higher CPC → fewer clicks → fewer appointments (all else equal).
func TestHigherCPCFewerAppointments(t *testing.T) {
	cheap := basePlan()
	cheap.Keywords = []KeywordMetrics{{Keyword: "k", MonthlySearches: 100000, CompetitionIndex: 0.5, CPCLowTRY: 4, CPCHighTRY: 6}}
	pricey := basePlan()
	pricey.Keywords = []KeywordMetrics{{Keyword: "k", MonthlySearches: 100000, CompetitionIndex: 0.5, CPCLowTRY: 40, CPCHighTRY: 60}}
	rc := mustSim(t, cheap)
	rp := mustSim(t, pricey)
	if rp.BookedAppointments.P50 >= rc.BookedAppointments.P50 {
		t.Errorf("expected fewer appts at higher CPC: cheap=%.2f pricey=%.2f",
			rc.BookedAppointments.P50, rp.BookedAppointments.P50)
	}
}

// Funnel effect: with the SAME clicks (budget-limited, identical keywords), the
// high-funnel general segment yields more appointments-per-click than the
// low-funnel aesthetic segment.
func TestSegmentFunnelEffect(t *testing.T) {
	kws := []KeywordMetrics{{Keyword: "k", MonthlySearches: 200000, CompetitionIndex: 0.5, CPCLowTRY: 8, CPCHighTRY: 12}}
	gen := basePlan()
	gen.Segment = domain.SegmentGeneral
	gen.Keywords = kws
	aes := basePlan()
	aes.Segment = domain.SegmentAesthetic
	aes.Keywords = kws
	rg := mustSim(t, gen)
	ra := mustSim(t, aes)
	if rg.BookedAppointments.P50 <= ra.BookedAppointments.P50 {
		t.Errorf("general funnel should out-convert aesthetic per click: gen=%.2f aes=%.2f",
			rg.BookedAppointments.P50, ra.BookedAppointments.P50)
	}
}

// Cost per appointment should be positive and finite for a normal plan.
func TestCostSane(t *testing.T) {
	r := mustSim(t, basePlan())
	if r.CostPerAppointment.P50 <= 0 {
		t.Errorf("cost per appt should be > 0, got %.2f", r.CostPerAppointment.P50)
	}
	if r.CostPerLead.P50 > r.CostPerAppointment.P50 {
		t.Errorf("cost per lead (%.2f) should be < cost per appt (%.2f)",
			r.CostPerLead.P50, r.CostPerAppointment.P50)
	}
}

// PriorKeywordSource returns keywords for every segment with sane CPC bounds.
func TestPriorKeywordSource(t *testing.T) {
	src := PriorKeywordSource{}
	for _, seg := range domain.AllSegments() {
		kws, err := src.Keywords(seg, priors.AudienceLocalTR)
		if err != nil {
			t.Fatalf("%s: %v", seg, err)
		}
		if len(kws) == 0 {
			t.Fatalf("%s: no keywords", seg)
		}
		for _, k := range kws {
			if k.CPCLowTRY <= 0 || k.CPCHighTRY < k.CPCLowTRY || k.MonthlySearches <= 0 {
				t.Errorf("%s: bad metrics %+v", seg, k)
			}
		}
	}
}
