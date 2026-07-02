package scenario

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"disci/brain/internal/domain"
	"disci/brain/internal/mathx"
	"disci/brain/internal/priors"
)

// Modeling constants. These are NOT sourced funnel facts (those live in
// internal/priors) — they are Monte-Carlo shape parameters: how much uncertainty
// to put around each prior, and how much impression share a budget can realistically
// capture. Kept here, documented, so the priors stay a clean fact table.
const (
	// betaKappa is the concentration of the Beta drawn around each funnel rate.
	// Higher κ = tighter belief. ~40 gives a realistic ±(few) percentage-point
	// spread on rates like 0.5 while staying in (0,1).
	betaKappa = 40.0

	// maxImpressionShare caps how much of a keyword's monthly searches we can
	// actually win — you never own 100% of impressions. Bounds the search-volume
	// ceiling so tiny budgets are budget-limited and huge budgets are volume-limited.
	maxImpressionShare = 0.55

	// defaultRuns is the Monte-Carlo sample count when a plan doesn't set one.
	defaultRuns = 1000
)

// CampaignPlan is the input: a hypothetical ad plan to stress-test. Keywords is
// optional — when empty the engine pulls them from its KeywordSource.
type CampaignPlan struct {
	ClinicID      string           `json:"clinicId"`
	Segment       domain.Segment   `json:"segment"`
	Platform      domain.Platform  `json:"platform"`
	Audience      priors.Audience  `json:"audience"`
	MonthlyBudget float64          `json:"monthlyBudget"` // TRY
	Keywords      []KeywordMetrics `json:"keywords,omitempty"`
	Runs          int              `json:"runs,omitempty"`
	Seed          int64            `json:"seed,omitempty"`
}

// Band summarises a metric's Monte-Carlo distribution. For appointment/lead
// counts: P10 = pessimistic, P50 = realistic, P90 = optimistic. For costs the
// sense inverts (P90 = expensive = pessimistic) — the numbers are raw percentiles;
// labelling is the caller's job.
type Band struct {
	P10  float64 `json:"p10"`
	P50  float64 `json:"p50"`
	P90  float64 `json:"p90"`
	Mean float64 `json:"mean"`
}

// Assumptions records the priors a run used, so the output is auditable rather
// than a black box.
type Assumptions struct {
	Funnel      priors.Funnel `json:"funnel"`      // qualify/book/show/close
	ClickToLead float64       `json:"clickToLead"` // click → captured lead
	AvgCPCTRY   float64       `json:"avgCpcTRY"`   // volume-weighted mean CPC
	SearchVol   int           `json:"searchVolume"`
	MaxImprShr  float64       `json:"maxImpressionShare"`
}

// Result is the full scenario output.
type Result struct {
	Runs               int         `json:"runs"`
	Budget             float64     `json:"budget"`
	BookedAppointments Band        `json:"bookedAppointments"` // headline: monthly booked appts
	KeptAppointments   Band        `json:"keptAppointments"`   // after no-show
	QualifiedLeads     Band        `json:"qualifiedLeads"`
	Clicks             Band        `json:"clicks"`
	CostPerAppointment Band        `json:"costPerAppointmentTRY"`
	CostPerLead        Band        `json:"costPerLeadTRY"`
	Assumptions        Assumptions `json:"assumptions"`
}

// Engine runs scenarios against a KeywordSource.
type Engine struct {
	Keywords KeywordSource
}

// New returns a scenario engine. Pass nil to use the cold-start PriorKeywordSource.
func New(src KeywordSource) *Engine {
	if src == nil {
		src = PriorKeywordSource{}
	}
	return &Engine{Keywords: src}
}

// Simulate runs the Monte-Carlo forecast for a plan.
func (e *Engine) Simulate(plan CampaignPlan) (Result, error) {
	runs := plan.Runs
	if runs <= 0 {
		runs = defaultRuns
	}
	kws := plan.Keywords
	if len(kws) == 0 {
		var err error
		kws, err = e.Keywords.Keywords(plan.Segment, plan.Audience)
		if err != nil {
			return Result{}, fmt.Errorf("keyword source: %w", err)
		}
	}
	if len(kws) == 0 {
		return Result{}, fmt.Errorf("no keywords for segment %q", plan.Segment)
	}

	// Aggregate keyword economics: total volume, volume-weighted CPC bounds.
	var totalSearch int
	var wLow, wHigh, wSum float64
	for _, k := range kws {
		totalSearch += k.MonthlySearches
		w := float64(k.MonthlySearches)
		if w <= 0 {
			w = 1
		}
		wLow += k.CPCLowTRY * w
		wHigh += k.CPCHighTRY * w
		wSum += w
	}
	cpcLow := wLow / wSum
	cpcHigh := wHigh / wSum
	cpcMid := (cpcLow + cpcHigh) / 2

	f := priors.FunnelFor(plan.Segment)
	c2l := priors.ClickToLeadRate(plan.Segment)

	seed := plan.Seed
	if seed == 0 {
		seed = 42 // deterministic default (repo rule: seeded RNG)
	}
	rng := rand.New(rand.NewSource(seed))

	booked := make([]float64, runs)
	kept := make([]float64, runs)
	qual := make([]float64, runs)
	clicks := make([]float64, runs)
	cpa := make([]float64, runs)
	cpl := make([]float64, runs)

	searchCeil := float64(totalSearch) * maxImpressionShare

	for i := 0; i < runs; i++ {
		cpc := triangular(rng, cpcLow, cpcMid, cpcHigh)
		if cpc <= 0 {
			cpc = cpcMid
		}
		ctr := betaAround(rng, ctrPrior(plan.Platform))
		clickToLead := betaAround(rng, c2l)
		q := betaAround(rng, f.Qualify)
		b := betaAround(rng, f.Book)
		show := betaAround(rng, f.Show)

		clicksBudget := plan.MonthlyBudget / cpc
		clicksSearch := searchCeil * ctr
		cl := math.Min(clicksBudget, clicksSearch)

		leads := cl * clickToLead
		qualified := leads * q
		bk := qualified * b
		kp := bk * show

		clicks[i] = cl
		qual[i] = qualified
		booked[i] = bk
		kept[i] = kp
		cpa[i] = safeCost(plan.MonthlyBudget, bk)
		cpl[i] = safeCost(plan.MonthlyBudget, leads)
	}

	return Result{
		Runs:               runs,
		Budget:             plan.MonthlyBudget,
		BookedAppointments: bandOf(booked),
		KeptAppointments:   bandOf(kept),
		QualifiedLeads:     bandOf(qual),
		Clicks:             bandOf(clicks),
		CostPerAppointment: bandOf(cpa),
		CostPerLead:        bandOf(cpl),
		Assumptions: Assumptions{
			Funnel:      f,
			ClickToLead: c2l,
			AvgCPCTRY:   cpcMid,
			SearchVol:   totalSearch,
			MaxImprShr:  maxImpressionShare,
		},
	}, nil
}

// ctrPrior is the cold-start click-through-rate by platform. Google search CTR
// runs higher than Facebook feed CTR ([CPL-F] gives FB dental CTR ≈ 1.05%).
func ctrPrior(plat domain.Platform) float64 {
	if plat == domain.PlatformMeta {
		return 0.011
	}
	return 0.04 // Google search dental ~4%
}

// betaAround draws from a Beta whose mean is p and concentration is betaKappa,
// so a run samples a plausible rate around the prior. Degenerate p just returns p.
func betaAround(rng *rand.Rand, p float64) float64 {
	p = mathx.Clamp(p, 1e-4, 1-1e-4)
	a := p * betaKappa
	b := (1 - p) * betaKappa
	return mathx.SampleBeta(rng, a, b)
}

// triangular draws from a triangular(low, mode, high) distribution — the standard
// choice for a bounded estimate with a most-likely value (here, CPC).
func triangular(rng *rand.Rand, low, mode, high float64) float64 {
	if high <= low {
		return low
	}
	u := rng.Float64()
	fc := (mode - low) / (high - low)
	if u < fc {
		return low + math.Sqrt(u*(high-low)*(mode-low))
	}
	return high - math.Sqrt((1-u)*(high-low)*(high-mode))
}

// safeCost is budget / count with a guard: zero appointments/leads → 0 (reported
// as "no forecastable cost" rather than +Inf).
func safeCost(budget, count float64) float64 {
	if count < 1e-9 {
		return 0
	}
	return budget / count
}

// bandOf sorts a sample and extracts P10/P50/P90 + mean.
func bandOf(xs []float64) Band {
	n := len(xs)
	if n == 0 {
		return Band{}
	}
	cp := make([]float64, n)
	copy(cp, xs)
	sort.Float64s(cp)
	var sum float64
	for _, x := range xs {
		sum += x
	}
	return Band{
		P10:  percentile(cp, 0.10),
		P50:  percentile(cp, 0.50),
		P90:  percentile(cp, 0.90),
		Mean: sum / float64(n),
	}
}

// percentile returns the p-quantile of an already-sorted slice (linear interp).
func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sorted[0]
	}
	pos := p * float64(n-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return sorted[lo]
	}
	frac := pos - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}
