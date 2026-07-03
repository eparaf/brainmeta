package sim

import (
	"disci/brain/internal/config"
	"disci/brain/internal/engine"
	"disci/brain/internal/store"
)

// ClinicRunSummary is one strategy's aggregate result across a full simulation.
// ROAS (Revenue/AdSpend), not appointment COUNT per TRY, is the metric that
// matches what the budget engine actually optimises: each arm's efficiency is
// value-weighted (ticket × margin × close-rate), so a strategy can book FEWER
// but higher-value appointments and be strictly better for the network — raw
// appointments-per-TRY would call that a loss.
type ClinicRunSummary struct {
	QualifiedAppts int
	AdSpend        float64
	Revenue        float64
	ROAS           float64 // Revenue / AdSpend — the objective the brain optimises
}

// Comparison is the offline "bandit vs manual" validation report: the SAME
// hidden world (including any drift schedule) run through two allocation
// strategies, compared on ROAS. This is the "prove it before you trust it with
// real spend" check — run this before deploying a config/policy change, exactly
// as MABWiser's simulation utility and sony/ABA's manual-baseline comparison do
// in the open-source bandit-budgeting literature.
//
// IMPORTANT — this is a measurement tool, not a guarantee: averaged across many
// seeds with the current tuning, the bandit does NOT reliably beat a naive equal
// split on ROAS in this simulated world (measured ~-3% on average over 20 seeds,
// 30 days). That is exactly the kind of fact this harness exists to surface
// BEFORE trusting the bandit with real ad spend, rather than assuming a bandit is
// automatically better. Investigating why (capacity-capped arms leaving a
// clinic's budget partially unspent; segment value-weighting interacting with
// the SLA booking gate) is follow-up work, not something this harness should
// paper over with a hardcoded "bandit wins" assertion.
type Comparison struct {
	Bandit    ClinicRunSummary
	Manual    ClinicRunSummary
	UpliftPct float64 // bandit's ROAS vs manual's, as a percentage gain (can be negative)
}

// CompareBanditVsManual runs two INDEPENDENT brains (fresh engine + store)
// against the same hidden-world seed and drift schedule — one allocated by the
// real bandit (BanditAllocator), one by a naive equal split
// (EqualSplitAllocator) — and reports each strategy's realised ROAS.
func CompareBanditVsManual(cfg config.Config, seed int64, days int, drifts []Drift) Comparison {
	run := func(alloc AllocatorFunc) ClinicRunSummary {
		st := store.NewMemory()
		eng := engine.New(cfg, st)
		w := Setup(eng, seed)
		w.SetDrifts(drifts)
		res := w.RunWithAllocator(eng, st, days, alloc)

		roas := 0.0
		if res.AdSpend > 0 {
			roas = res.Revenue / res.AdSpend
		}
		return ClinicRunSummary{QualifiedAppts: res.Booked, AdSpend: res.AdSpend, Revenue: res.Revenue, ROAS: roas}
	}

	bandit := run(BanditAllocator)
	manual := run(EqualSplitAllocator)
	uplift := 0.0
	if manual.ROAS > 0 {
		uplift = (bandit.ROAS - manual.ROAS) / manual.ROAS * 100
	}
	return Comparison{Bandit: bandit, Manual: manual, UpliftPct: uplift}
}

// AverageComparison runs CompareBanditVsManual across several seeds and averages
// the ROAS uplift — a single seed is noisy enough to flip sign (measured: seed=7
// alone shows manual beating bandit by ~30%, while other seeds show the reverse),
// so any real go/no-go decision on a policy change should look at the average
// over a spread of seeds, not one run. This mirrors why A/B tests need enough
// samples before you trust the result.
func AverageComparison(cfg config.Config, seeds []int64, days int, drifts []Drift) Comparison {
	var bSpend, bRev, mSpend, mRev float64
	var bAppts, mAppts int
	for _, seed := range seeds {
		c := CompareBanditVsManual(cfg, seed, days, drifts)
		bSpend += c.Bandit.AdSpend
		bRev += c.Bandit.Revenue
		bAppts += c.Bandit.QualifiedAppts
		mSpend += c.Manual.AdSpend
		mRev += c.Manual.Revenue
		mAppts += c.Manual.QualifiedAppts
	}
	bandit := ClinicRunSummary{QualifiedAppts: bAppts, AdSpend: bSpend, Revenue: bRev, ROAS: safeDivROAS(bRev, bSpend)}
	manual := ClinicRunSummary{QualifiedAppts: mAppts, AdSpend: mSpend, Revenue: mRev, ROAS: safeDivROAS(mRev, mSpend)}
	uplift := 0.0
	if manual.ROAS > 0 {
		uplift = (bandit.ROAS - manual.ROAS) / manual.ROAS * 100
	}
	return Comparison{Bandit: bandit, Manual: manual, UpliftPct: uplift}
}

func safeDivROAS(revenue, spend float64) float64 {
	if spend <= 0 {
		return 0
	}
	return revenue / spend
}
