// Package budget implements Motor 2 — the Budget Allocation Engine.
//
// This is the answer to "some clinics advertise well, some don't". We never
// hand-tune that. Each advertising lever — an Arm = (clinic, platform, campaign,
// creative, segment) — is a bandit arm with an unknown probability of producing
// a *qualified appointment* per lead, and a learned cost-per-lead. The engine:
//
//  1. Thompson-samples a plausible conversion rate for every arm from its Beta
//     posterior (exploration ∝ uncertainty).
//  2. Computes each arm's sampled "appointments per TRY" efficiency.
//  3. Water-fills the monthly budget toward the most efficient arms until the
//     marginal efficiency across funded arms equalises — the Lagrangian optimum,
//     where λ is the shadow price of one TRY of budget.
//  4. Applies a guarantee bias: arms belonging to clinics behind on their SLA
//     get their efficiency scaled up by the clinic's shadow price.
//  5. Paces daily spend with a PID controller so budget isn't dumped early.
//
// A clinic that "advertises well" has high-θ, low-CPL arms → they win budget
// automatically. A poor one gets throttled, unless its SLA shadow price forces a
// floor. No manual intervention.
package budget

import (
	"math"
	"math/rand"
	"sort"
	"sync"

	"disci/brain/internal/domain"
	"disci/brain/internal/mathx"
	"disci/brain/internal/priors"
)

// armState is the learned posterior for one arm.
type armState struct {
	arm domain.Arm

	// Beta posterior over P(qualified-appointment | lead) for this arm.
	alpha, beta float64

	// Prior the posterior decays toward (non-stationarity handling).
	priorAlpha, priorBeta float64

	// Online estimate of cost per lead (TRY), EWMA.
	cpl float64

	leadsDelivered int
	apptsWon       int
	spend          float64 // cumulative spend this period
}

// valueWeight is the expected margin (TRY) per qualified appointment for the
// arm, with a backward-compatible floor of 1 (so arms with no value set behave
// as a pure appointments-per-TRY allocator — used by unit tests).
func (a *armState) valueWeight() float64 {
	if a.arm.ExpectedValuePerAppt > 0 {
		return a.arm.ExpectedValuePerAppt
	}
	return 1
}

// efficiency = sampled expected VALUE per TRY for an arm = (θ/CPL)·value·bias.
func (a *armState) efficiency(rng *rand.Rand, slaBias float64) float64 {
	theta := mathx.SampleBeta(rng, a.alpha, a.beta) // P(qualified | lead)
	cpl := a.cpl
	if cpl <= 0 {
		cpl = 50 // bootstrap CPL before any data
	}
	return (theta / cpl) * a.valueWeight() * slaBias
}

// Allocation is the engine's decision for one arm in one planning cycle.
type Allocation struct {
	ArmID         string
	ClinicID      string
	DailyBudget   float64
	SampledTheta  float64
	ExpectedAppts float64
	SLABias       float64
}

// Engine is the budget allocator. Safe for concurrent use.
type Engine struct {
	mu   sync.Mutex
	rng  *rand.Rand
	arms map[string]*armState

	cplAlpha float64 // EWMA factor for CPL learning
}

// NewEngine constructs the allocator with a seeded RNG (deterministic tests).
func NewEngine(seed int64) *Engine {
	return &Engine{
		rng:      rand.New(rand.NewSource(seed)),
		arms:     map[string]*armState{},
		cplAlpha: 0.1,
	}
}

// RegisterArm adds an arm whose Beta prior is SEEDED from the segment's sourced
// qualify-rate benchmark (with a small pseudo-count + slight optimism), instead
// of a flat 1.5/1.0. This gives every new creative/campaign a realistic, segment-
// aware cold start that matches what the scorer already assumes.
func (e *Engine) RegisterArm(arm domain.Arm) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.arms[arm.ID]; ok {
		return
	}
	cpl := arm.AvgCostPerLead
	if cpl <= 0 {
		cpl = 50
	}
	const pseudo = 8.0
	q := priors.FunnelFor(arm.Segment).Qualify
	alpha := q*pseudo + 0.5 // +0.5 = mild optimism to encourage early exploration
	beta := (1 - q) * pseudo
	if beta < 0.5 {
		beta = 0.5
	}
	e.arms[arm.ID] = &armState{
		arm: arm, alpha: alpha, beta: beta,
		priorAlpha: alpha, priorBeta: beta, cpl: cpl,
	}
}

// Decay shrinks every arm's posterior toward its prior by factor gamma∈(0,1].
// Called once per period so the allocator keeps exploring and reacts to creative
// fatigue / seasonality instead of freezing on stale evidence (discounted
// Thompson sampling). gamma=1 → no decay; smaller → faster forgetting.
func (e *Engine) Decay(gamma float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	g := mathx.Clamp(gamma, 0, 1)
	for _, a := range e.arms {
		a.alpha = a.priorAlpha + g*(a.alpha-a.priorAlpha)
		a.beta = a.priorBeta + g*(a.beta-a.priorBeta)
	}
}

// UpdateValue refreshes the expected margin per qualified appointment for all
// arms of a clinic — fed from the scorer's learned ticket size so the budget
// motor optimises against realised value, not a frozen registration-time guess.
func (e *Engine) UpdateValue(clinicID string, valuePerAppt float64) {
	if valuePerAppt <= 0 {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, a := range e.arms {
		if a.arm.ClinicID == clinicID {
			a.arm.ExpectedValuePerAppt = valuePerAppt
		}
	}
}

// Observe folds one lead's result into the arm's posterior and cost estimate.
// won = the lead became a qualified appointment.
func (e *Engine) Observe(armID string, won bool, costPaid float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	a, ok := e.arms[armID]
	if !ok {
		return
	}
	a.leadsDelivered++
	a.spend += costPaid
	if won {
		a.alpha++
		a.apptsWon++
	} else {
		a.beta++
	}
	if costPaid > 0 {
		if a.cpl <= 0 {
			a.cpl = costPaid
		} else {
			a.cpl = e.cplAlpha*costPaid + (1-e.cplAlpha)*a.cpl
		}
	}
}

// CorrectCPL overrides an arm's learned cost-per-lead with ground truth pulled
// from the ad platform (Meta/Google reports). Platform-reported spend is more
// accurate than our per-lead estimate, so we trust it when available.
func (e *Engine) CorrectCPL(armID string, cpl float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if a, ok := e.arms[armID]; ok && cpl > 0 {
		a.cpl = cpl
	}
}

// SLAProvider lets the budget engine ask the guarantee controller how badly a
// clinic needs more appointments right now. Returns a bias multiplier ≥ 1.
type SLAProvider interface {
	BudgetBias(clinicID string) float64
}

// noBias is the default when no SLA controller is wired in.
type noBias struct{}

func (noBias) BudgetBias(string) float64 { return 1 }

// Allocate runs one planning cycle. Budgets are PER-CLINIC: in the passthrough
// model each clinic funds its own ad budget, so clinic A's money can never be
// spent on clinic B. Within each clinic's daily budget we Thompson-sample and
// water-fill across that clinic's arms (platforms/creatives), funding the most
// value-efficient ones first.
//
// clinicDaily maps clinicID → that clinic's daily ad budget (TRY). Guarantee
// pressure does NOT redirect budget across clinics here — it acts through lead
// routing and the booking gate. Returns per-arm daily budgets plus lambda, the
// marginal value-per-TRY at the optimum (a saturation health metric).
func (e *Engine) Allocate(clinicDaily map[string]float64, sla SLAProvider) ([]Allocation, float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if sla == nil {
		sla = noBias{}
	}

	// Group arms by clinic.
	byClinic := map[string][]*armState{}
	for _, a := range e.arms {
		byClinic[a.arm.ClinicID] = append(byClinic[a.arm.ClinicID], a)
	}

	allocs := make([]Allocation, 0, len(e.arms))
	var lambdaNum, lambdaDen float64

	for clinicID, arms := range byClinic {
		budget := clinicDaily[clinicID]
		bias := sla.BudgetBias(clinicID)

		type scored struct {
			st    *armState
			eff   float64
			theta float64
		}
		scoredArms := make([]scored, 0, len(arms))
		for _, a := range arms {
			theta := mathx.SampleBeta(e.rng, a.alpha, a.beta)
			cpl := a.cpl
			if cpl <= 0 {
				cpl = 50
			}
			// Value per TRY. bias is constant within a clinic so it doesn't change
			// the within-clinic ranking, but it scales lambda for reporting.
			eff := (theta / cpl) * a.valueWeight() * bias
			scoredArms = append(scoredArms, scored{st: a, eff: eff, theta: theta})
		}
		sort.Slice(scoredArms, func(i, j int) bool { return scoredArms[i].eff > scoredArms[j].eff })

		remaining := budget
		// CLINIC-LEVEL qualified-arrival headroom, SHARED across the clinic's arms.
		// This is the fix for the N×capacity overbuy: capping each arm at the full
		// clinic capacity independently let a clinic with N arms buy N× its seats.
		qRemaining := math.Inf(1)
		if cap := clinicCapacity(arms); cap > 0 {
			qRemaining = 1.3 * float64(cap) // headroom for no-shows / unqualified slippage
		}
		var clinicMarginalEff float64
		for _, s := range scoredArms {
			give := 0.0
			theta := s.theta
			if theta < 0.01 {
				theta = 0.01
			}
			if remaining > 0 && qRemaining > 0 {
				cpl := clampCPL(s.st.cpl)
				leadCap := 120.0 // audience-fatigue ceiling
				if capLeads := qRemaining / theta; capLeads < leadCap {
					leadCap = capLeads // shared clinic-capacity ceiling
				}
				give = math.Min(cpl*leadCap, remaining)
				remaining -= give
				qRemaining -= give / cpl * theta
				clinicMarginalEff = s.eff
			}
			allocs = append(allocs, Allocation{
				ArmID: s.st.arm.ID, ClinicID: clinicID,
				DailyBudget:   round2(give),
				SampledTheta:  s.theta,
				ExpectedAppts: give / clampCPL(s.st.cpl) * s.theta,
				SLABias:       bias,
			})
		}
		// Budget-weighted network shadow price: the marginal value-per-TRY of the
		// last funded arm in each clinic, averaged by clinic budget. (Previously a
		// single var clobbered across map iteration → returned a random clinic's.)
		if budget > 0 {
			lambdaNum += clinicMarginalEff * budget
			lambdaDen += budget
		}
	}
	if len(allocs) == 0 {
		return nil, 0
	}
	lambda := 0.0
	if lambdaDen > 0 {
		lambda = lambdaNum / lambdaDen
	}
	return allocs, lambda
}

// clinicCapacity returns the shared daily new-patient seat count for a clinic's
// arms (they all carry the same ClinicCapacity).
func clinicCapacity(arms []*armState) int {
	for _, a := range arms {
		if a.arm.ClinicCapacity > 0 {
			return a.arm.ClinicCapacity
		}
	}
	return 0
}

func clampCPL(c float64) float64 {
	if c < 5 {
		return 5
	}
	return c
}

func round2(x float64) float64 { return math.Round(x*100) / 100 }

// ---- PID pacing controller ----------------------------------------------

// Pacer keeps actual spend tracking a smooth target curve across the day, so an
// arm doesn't exhaust its daily budget in the first hour (which wrecks CPA).
type Pacer struct {
	kp, ki, kd float64
	integral   float64
	prevErr    float64
}

func NewPacer() *Pacer { return &Pacer{kp: 0.6, ki: 0.05, kd: 0.1} }

// PacerSet holds one independent, locked Pacer per arm. A single shared Pacer is
// a correctness bug: PID integral/derivative state would bleed across distinct
// arms (and races under concurrent callers). Each arm gets its own.
type PacerSet struct {
	mu     sync.Mutex
	pacers map[string]*Pacer
}

// NewPacerSet builds an empty per-arm pacer set.
func NewPacerSet() *PacerSet { return &PacerSet{pacers: map[string]*Pacer{}} }

// Multiplier returns the pacing multiplier for one arm, using that arm's own
// PID state. Safe for concurrent use.
func (ps *PacerSet) Multiplier(armID string, dayFrac, spendFrac float64) float64 {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	p, ok := ps.pacers[armID]
	if !ok {
		p = NewPacer()
		ps.pacers[armID] = p
	}
	return p.Multiplier(dayFrac, spendFrac)
}

// Multiplier returns a bid multiplier given the fraction of the day elapsed and
// the fraction of the daily budget already spent. >1 means "spend faster".
func (p *Pacer) Multiplier(dayFrac, spendFrac float64) float64 {
	target := dayFrac // linear pacing target
	err := target - spendFrac
	p.integral += err
	deriv := err - p.prevErr
	p.prevErr = err
	out := 1.0 + p.kp*err + p.ki*p.integral + p.kd*deriv
	return mathx.Clamp(out, 0.2, 3.0)
}

// Snapshot reports per-arm learned stats for the dashboard.
func (e *Engine) Snapshot() []map[string]any {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]map[string]any, 0, len(e.arms))
	for _, a := range e.arms {
		out = append(out, map[string]any{
			"armId":    a.arm.ID,
			"clinicId": a.arm.ClinicID,
			"segment":  a.arm.Segment,
			"thetaHat": mathx.BetaMean(a.alpha, a.beta),
			"cpl":      round2(a.cpl),
			"leads":    a.leadsDelivered,
			"appts":    a.apptsWon,
			"spend":    round2(a.spend),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i]["thetaHat"].(float64) > out[j]["thetaHat"].(float64)
	})
	return out
}
