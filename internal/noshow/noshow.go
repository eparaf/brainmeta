// Package noshow implements Motor 4 — the Show-up Engine.
//
// A booked appointment is not a kept appointment. Dental no-show rates run
// 20-30%, and unmanaged no-shows are the single biggest threat to the
// "qualified appointments delivered" guarantee. This motor does three things:
//
//  1. Predicts P(show | booked) per appointment from its features (online
//     logistic, concurrency-safe).
//  2. Chooses an intervention tier (reminder cadence / deposit / call) to lift
//     low-probability appointments.
//  3. Solves the overbooking problem EXACTLY (Poisson-binomial convolution):
//     how many appointments to accept against a fixed seat capacity so expected
//     arrivals fill the seats without exceeding a double-booking risk budget.
package noshow

import (
	"sort"
	"sync"

	"disci/brain/internal/domain"
	"disci/brain/internal/mathx"
)

// Predictor estimates show probability. Online logistic with a sourced prior
// intercept so it works on day one. Safe for concurrent use.
type Predictor struct {
	mu   sync.RWMutex
	w    []float64
	b    float64
	lr   float64
	seen int

	// Platt calibration on the raw logit: pCal = sigmoid(calA·z + calB). Starts as
	// the identity (1, 0) so it's a no-op on day one, then is learned online. This
	// matters because the calibrated show probabilities feed the Poisson-binomial
	// overbooking solver: if the model is systematically over/under-confident, the
	// tail-risk math (P(arrivals > seats)) is wrong and we over- or under-book.
	calA  float64
	calB  float64
	calLR float64
}

// feature order (dim 9):
// leadTimeDays, isWeekend, hourSin, hourCos, pastNoShows, deposit,
// confirmedReply, distanceKm, segmentImplant
const showDim = 9

func NewPredictor() *Predictor {
	return &Predictor{
		w: make([]float64, showDim), b: mathx.Logit(baseShowProb()), lr: 0.05,
		calA: 1, calB: 0, calLR: 0.02, // identity calibration until data says otherwise
	}
}

func showFeatures(a Appt) []float64 {
	return []float64{
		mathx.Clamp(1.0-a.LeadTimeDays/14.0, -1, 1), // booking far out -> more no-show
		boolf(a.IsWeekend),
		mathx.SinHour(a.HourOfDay), // sin+cos pair removes the 3am≡9am aliasing
		mathx.CosHour(a.HourOfDay),
		mathx.Clamp(-a.PastNoShows/3.0, -1, 0),
		boolf(a.DepositPaid) * 1.5, // deposits strongly raise show rate
		boolf(a.ConfirmedReply),
		mathx.Clamp(1.0-a.DistanceKm/40.0, -1, 1),
		boolf(a.Segment == domain.SegmentImplant),
	}
}

func boolf(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// Appt carries the features needed to predict show probability.
type Appt struct {
	LeadTimeDays   float64
	IsWeekend      bool
	HourOfDay      float64
	PastNoShows    float64
	DepositPaid    bool
	ConfirmedReply bool
	DistanceKm     float64
	Segment        domain.Segment
}

// rawLogit is the base model's score (pre-calibration). Callers hold the lock.
func (p *Predictor) rawLogit(x []float64) float64 { return mathx.Dot(p.w, x) + p.b }

// predict is the base model's UNCALIBRATED probability (used for training).
func (p *Predictor) predict(x []float64) float64 { return mathx.Sigmoid(p.rawLogit(x)) }

// calibrated applies Platt scaling to a raw logit → a calibrated probability.
func (p *Predictor) calibrated(z float64) float64 { return mathx.Sigmoid(p.calA*z + p.calB) }

// PShow returns the CALIBRATED probability the patient shows up — this is what the
// overbooking solver consumes.
func (p *Predictor) PShow(a Appt) float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.calibrated(p.rawLogit(showFeatures(a)))
}

// Learn folds in a realised show/no-show: one SGD step on the base logistic, then
// one on the Platt calibrator (lightly regularised toward the identity so it stays
// a no-op until enough outcomes justify a correction).
func (p *Predictor) Learn(a Appt, showed bool) {
	x := showFeatures(a)
	p.mu.Lock()
	defer p.mu.Unlock()
	z := p.rawLogit(x)
	y := boolf(showed)
	err := mathx.Sigmoid(z) - y
	for i := range p.w {
		p.w[i] -= p.lr * (err*x[i] + 1e-4*p.w[i])
	}
	p.b -= p.lr * err
	// Online Platt calibration on the raw logit z, regularised toward (calA=1, calB=0).
	ec := p.calibrated(z) - y
	p.calA -= p.calLR * (ec*z + 1e-3*(p.calA-1))
	p.calB -= p.calLR * (ec + 1e-3*p.calB)
	p.seen++
}

// State is the serializable snapshot of the predictor (for persistence).
type State struct {
	W    []float64 `json:"w"`
	B    float64   `json:"b"`
	Seen int       `json:"seen"`
	CalA float64   `json:"calA,omitempty"`
	CalB float64   `json:"calB,omitempty"`
}

// Export returns a copy of the learned state.
func (p *Predictor) Export() State {
	p.mu.RLock()
	defer p.mu.RUnlock()
	w := make([]float64, len(p.w))
	copy(w, p.w)
	return State{W: w, B: p.b, Seen: p.seen, CalA: p.calA, CalB: p.calB}
}

// Import restores learned state (e.g. on startup). Ignores dimension mismatch.
func (p *Predictor) Import(s State) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(s.W) == len(p.w) {
		copy(p.w, s.W)
		p.b = s.B
		p.seen = s.Seen
	}
	// Older snapshots carry no calibration (both zero) → keep the identity set in
	// NewPredictor. A real calibrator always has calA≈1 (non-zero), so this is safe.
	if s.CalA != 0 || s.CalB != 0 {
		p.calA = s.CalA
		p.calB = s.CalB
	}
}

// Intervention tiers, escalating in cost and effectiveness.
type Intervention int

const (
	InterventionStandard Intervention = iota // 24h + 2h WhatsApp reminder
	InterventionEnhanced                     // + personal message / easy reschedule link
	InterventionDeposit                      // request a small refundable deposit
	InterventionCall                         // human/AI voice call to confirm
)

func (i Intervention) String() string {
	switch i {
	case InterventionEnhanced:
		return "enhanced_reminders"
	case InterventionDeposit:
		return "request_deposit"
	case InterventionCall:
		return "confirmation_call"
	default:
		return "standard_reminders"
	}
}

// ChooseIntervention escalates based on risk and the appointment's value. We
// only spend on expensive interventions (deposit/call) when the appointment is
// both risky and valuable.
func ChooseIntervention(pShow, apptValue float64) Intervention {
	risk := 1 - pShow
	switch {
	case risk > 0.45 && apptValue > 50_000:
		return InterventionCall
	case risk > 0.45:
		return InterventionDeposit
	case risk > 0.25:
		return InterventionEnhanced
	default:
		return InterventionStandard
	}
}

// OverbookPlan is the output of the overbooking solver.
type OverbookPlan struct {
	Capacity         int
	Booked           int       // how many appointments to accept
	ExpectedArrivals float64   // sum of (possibly lifted) P(show)
	OverbookRisk     float64   // P(arrivals > capacity), EXACT
	PShowAfter       []float64 // per-appointment show prob after interventions
}

// PlanOverbooking decides how many candidate appointments to accept against a
// seat capacity. It accepts most-likely-to-show first and keeps accepting while
// the EXACT double-booking risk P(arrivals > capacity) stays within maxRisk.
// Because each patient shows with prob < 1, this books MORE than `capacity`.
//
// The risk uses the exact Poisson-binomial distribution maintained by
// convolution — O(n²) total — which is correct in the small-n upper tail where
// the normal approximation is worst (and where dental capacities live).
func PlanOverbooking(capacity int, pShows []float64, maxRisk float64) OverbookPlan {
	idx := make([]int, len(pShows))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return pShows[idx[a]] > pShows[idx[b]] })

	dist := []float64{1} // P(#arrivals = k); starts certain at 0
	accepted := make([]float64, 0, len(pShows))
	var mean float64
	booked := 0
	for _, i := range idx {
		p := mathx.Clamp(pShows[i], 0, 1)
		trial := convolveBernoulli(dist, p)
		if overflow(trial, capacity) <= maxRisk {
			dist = trial
			mean += p
			accepted = append(accepted, p)
			booked++
		} else {
			break
		}
	}
	return OverbookPlan{
		Capacity:         capacity,
		Booked:           booked,
		ExpectedArrivals: mean,
		OverbookRisk:     overflow(dist, capacity),
		PShowAfter:       accepted,
	}
}

// convolveBernoulli folds one more independent Bernoulli(p) show event into the
// arrivals distribution: dist'[k] = dist[k]·(1−p) + dist[k−1]·p.
func convolveBernoulli(dist []float64, p float64) []float64 {
	out := make([]float64, len(dist)+1)
	for k := 0; k < len(dist); k++ {
		out[k] += dist[k] * (1 - p)
		out[k+1] += dist[k] * p
	}
	return out
}

// overflow returns P(arrivals > capacity) from an arrivals distribution.
func overflow(dist []float64, capacity int) float64 {
	var s float64
	for k := capacity + 1; k < len(dist); k++ {
		s += dist[k]
	}
	return s
}

// ExpectedShows is the expected number of arrivals for a set of show probs.
func ExpectedShows(pShows []float64) float64 { return mathx.Sum(pShows) }

// AcceptableOverbook answers the ONLINE overbooking question: given the show
// probabilities of appointments already accepted for a clinic-day, would adding
// one more with show prob p keep the exact double-booking risk
// P(arrivals > capacity) within maxRisk? This is how the overbooking motor is
// consulted on the live per-lead booking path.
func AcceptableOverbook(accepted []float64, p float64, capacity int, maxRisk float64) bool {
	dist := []float64{1}
	for _, q := range accepted {
		dist = convolveBernoulli(dist, mathx.Clamp(q, 0, 1))
	}
	dist = convolveBernoulli(dist, mathx.Clamp(p, 0, 1))
	return overflow(dist, capacity) <= maxRisk
}

// ApplyBestIntervention returns the lifted show probability and the chosen
// intervention for one appointment.
func ApplyBestIntervention(pShow, apptValue float64) (float64, Intervention) {
	iv := ChooseIntervention(pShow, apptValue)
	lifted := mathx.Clamp(pShow+expectedLift(iv), 0, 0.98)
	return lifted, iv
}
