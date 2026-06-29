// Package scoring implements Motor 1 — the Lead Value Engine.
//
// For every lead it estimates the chain of conditional probabilities that turn
// a click into revenue:
//
//	EV = P(qualify) · P(book|qualify) · P(show|book) · P(close|show) · ticket · margin
//
// Each probability comes from a logistic model over the lead's feature vector.
// The weights are learned online with SGD. Before we have data (cold start) the
// models fall back to per-segment Bayesian priors (Beta posteriors), so the very
// first lead still gets a sane score. As real outcomes arrive, the priors are
// blended out and the logistic models take over.
package scoring

import (
	"sync"

	"disci/brain/internal/domain"
	"disci/brain/internal/mathx"
	"disci/brain/internal/priors"
)

// stage identifies one funnel transition we model independently.
type stage int

const (
	stageQualify stage = iota
	stageBook
	stageShow
	stageClose
	numStages
)

// logisticModel is a single online-learned logistic regression head.
type logisticModel struct {
	w    []float64 // weights, aligned with featureVector()
	b    float64   // bias
	lr   float64   // learning rate
	l2   float64   // L2 regularisation strength
	seen int       // number of SGD updates applied
}

func newLogistic(dim int, biasLogit float64) *logisticModel {
	return &logisticModel{
		w:  make([]float64, dim),
		b:  biasLogit,
		lr: 0.05,
		l2: 1e-4,
	}
}

func (m *logisticModel) predict(x []float64) float64 {
	return mathx.Sigmoid(mathx.Dot(m.w, x) + m.b)
}

// update applies one SGD step on the log-loss for a single (x, y) example.
func (m *logisticModel) update(x []float64, y float64) {
	p := m.predict(x)
	err := p - y // gradient of log-loss wrt logit
	for i := range m.w {
		grad := err*x[i] + m.l2*m.w[i]
		m.w[i] -= m.lr * grad
	}
	m.b -= m.lr * err
	m.seen++
}

// betaCounter is the Bayesian cold-start fallback: a Beta posterior over a
// stage's base rate, maintained per segment.
type betaCounter struct {
	alpha, beta float64
}

func (b betaCounter) mean() float64 { return mathx.BetaMean(b.alpha, b.beta) }

func (b *betaCounter) observe(success bool) {
	if success {
		b.alpha++
	} else {
		b.beta++
	}
}

// confidence returns a 0..1 weight reflecting how much real data we have. Used
// to blend the prior (early) with the logistic model (later).
func (b betaCounter) confidence() float64 {
	n := b.alpha + b.beta
	// Reaches ~0.9 around 40 observations.
	return n / (n + 8)
}

// Engine is the public scorer. Safe for concurrent use.
type Engine struct {
	mu     sync.RWMutex
	dim    int
	models map[domain.Segment][numStages]*logisticModel
	priors map[domain.Segment][numStages]*betaCounter

	// Ticket estimate per segment, updated from realised revenue.
	ticket map[domain.Segment]*ewma
}

// priorRatesFor returns the cold-start funnel rates for a segment as a stage
// array, sourced from the real 2025–2026 benchmarks in the priors package.
func priorRatesFor(seg domain.Segment) [numStages]float64 {
	f := priors.FunnelFor(seg)
	return [numStages]float64{f.Qualify, f.Book, f.Show, f.Close}
}

const featureDim = 10

// NewEngine builds a scorer seeded with the per-segment priors above.
func NewEngine() *Engine {
	e := &Engine{
		dim:    featureDim,
		models: map[domain.Segment][numStages]*logisticModel{},
		priors: map[domain.Segment][numStages]*betaCounter{},
		ticket: map[domain.Segment]*ewma{},
	}
	for _, seg := range domain.AllSegments() {
		var ms [numStages]*logisticModel
		var ps [numStages]*betaCounter
		rates := priorRatesFor(seg)
		for s := stage(0); s < numStages; s++ {
			ms[s] = newLogistic(e.dim, mathx.Logit(rates[s]))
			// Seed the Beta with a weak pseudo-count matching the prior rate.
			const pseudo = 6.0
			ps[s] = &betaCounter{alpha: rates[s] * pseudo, beta: (1 - rates[s]) * pseudo}
		}
		e.models[seg] = ms
		e.priors[seg] = ps
		e.ticket[seg] = newEWMA(priors.TicketTRY(seg), 0.05)
	}
	return e
}

// featureVector turns raw LeadFeatures into the normalised vector both the
// logistic models and the offline trainer consume. Normalisation keeps SGD
// stable and the feature order MUST stay fixed (train/serve parity).
func featureVector(f domain.LeadFeatures) []float64 {
	// Budget is reported in TRY by the agent (currency-normalised upstream). A
	// missing budget (0) is encoded with an explicit indicator so the model
	// doesn't confuse "unknown budget" with "zero-budget tyre-kicker".
	budgetKnown := 0.0
	budgetNorm := 0.0
	if f.StatedBudgetTRY > 0 {
		budgetKnown = 1.0
		budgetNorm = clampNorm(f.StatedBudgetTRY/100_000.0, 0, 1)
	}
	return []float64{
		clampNorm(1.0-f.FirstResponseSecs/300.0, -1, 1), // faster reply -> higher
		clampNorm(f.MessagesExchanged/10.0, 0, 1),
		clampNorm(1.0-f.DistanceKm/40.0, -1, 1), // closer -> higher
		mathx.SinHour(f.HourOfDay),               // cyclic time: sin AND cos, no aliasing
		mathx.CosHour(f.HourOfDay),
		budgetNorm,
		budgetKnown,
		clampNorm(f.UrgencyScore, 0, 1),
		clampNorm(-f.PastNoShows/3.0, -1, 0), // prior no-shows hurt
		clampNorm(f.IntentScore, 0, 1),
	}
}

func clampNorm(x, lo, hi float64) float64 { return mathx.Clamp(x, lo, hi) }

// blended returns the stage probability, mixing the Beta prior with the logistic
// model by the prior's confidence. confidence≈0 -> all logistic-with-prior-bias;
// as data accrues the logistic model (which itself learns) dominates.
func (e *Engine) blended(seg domain.Segment, s stage, x []float64) float64 {
	m := e.models[seg][s]
	p := e.priors[seg][s]
	modelP := m.predict(x)
	priorMean := p.mean()
	c := p.confidence()
	// The logistic head already starts at the prior bias, so this blend mostly
	// stabilises the first few dozen leads against noisy SGD steps.
	return c*modelP + (1-c)*priorMean
}

// Score computes the full EV decomposition for a lead.
func (e *Engine) Score(lead domain.Lead) domain.LeadScore {
	e.mu.RLock()
	defer e.mu.RUnlock()

	seg := lead.Segment
	if _, ok := e.models[seg]; !ok {
		seg = domain.SegmentGeneral
	}
	x := featureVector(lead.Features)

	pq := e.blended(seg, stageQualify, x)
	pb := e.blended(seg, stageBook, x)
	ps := e.blended(seg, stageShow, x)
	pc := e.blended(seg, stageClose, x)

	ticket := e.ticket[seg].value()
	// If the patient stated a budget, nudge the ticket estimate toward it.
	if lead.Features.StatedBudgetTRY > 0 {
		ticket = 0.7*ticket + 0.3*lead.Features.StatedBudgetTRY
	}
	margin := ticket * marginFor(seg)
	ev := pq * pb * ps * pc * margin

	return domain.LeadScore{
		PQualify: pq, PBook: pb, PShow: ps, PClose: pc,
		Ticket: ticket, Margin: margin, EV: ev,
	}
}

func marginFor(seg domain.Segment) float64 { return priors.MarginFor(seg) }

// Learn folds a realised outcome back into the models. This is called by the
// feedback loop for every lead that reaches a terminal state.
func (e *Engine) Learn(o domain.Outcome, f domain.LeadFeatures) {
	e.mu.Lock()
	defer e.mu.Unlock()

	seg := o.Segment
	if _, ok := e.models[seg]; !ok {
		seg = domain.SegmentGeneral
	}
	x := featureVector(f)

	apply := func(s stage, label *bool) {
		if label == nil {
			return
		}
		y := 0.0
		if *label {
			y = 1.0
		}
		e.models[seg][s].update(x, y)
		e.priors[seg][s].observe(*label)
	}
	apply(stageQualify, o.Qualified)
	apply(stageBook, o.Booked)
	apply(stageShow, o.Showed)
	apply(stageClose, o.Closed)

	if o.Closed != nil && *o.Closed && o.Revenue > 0 {
		e.ticket[seg].observe(o.Revenue)
	}
}

// Stats exposes a snapshot for the dashboard / debugging.
func (e *Engine) Stats() map[string]any {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := map[string]any{}
	for _, seg := range domain.AllSegments() {
		out[string(seg)] = map[string]any{
			"pQualify": e.priors[seg][stageQualify].mean(),
			"pBook":    e.priors[seg][stageBook].mean(),
			"pShow":    e.priors[seg][stageShow].mean(),
			"pClose":   e.priors[seg][stageClose].mean(),
			"ticket":   e.ticket[seg].value(),
			"samples":  e.priors[seg][stageQualify].alpha + e.priors[seg][stageQualify].beta,
		}
	}
	return out
}

// ewma is an exponentially weighted moving average used for online ticket-size
// estimation.
type ewma struct {
	v       float64
	alpha   float64
	started bool
}

func newEWMA(init, alpha float64) *ewma { return &ewma{v: init, alpha: alpha, started: true} }

func (e *ewma) observe(x float64) {
	if !e.started {
		e.v = x
		e.started = true
		return
	}
	e.v = e.alpha*x + (1-e.alpha)*e.v
}

func (e *ewma) value() float64 { return e.v }
