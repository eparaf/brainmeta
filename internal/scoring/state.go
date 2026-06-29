package scoring

import (
	"disci/brain/internal/domain"
	"disci/brain/internal/priors"
)

// headState is one funnel-stage model's persisted parameters.
type headState struct {
	W     []float64 `json:"w"`
	B     float64   `json:"b"`
	Seen  int       `json:"seen"`
	Alpha float64   `json:"alpha"`
	Beta  float64   `json:"beta"`
}

// segState bundles the four stage heads + the learned ticket size for a segment.
type segState struct {
	Heads  [numStages]headState `json:"heads"`
	Ticket float64              `json:"ticket"`
}

// State is the serializable snapshot of the whole scorer.
type State struct {
	Segments map[string]segState `json:"segments"`
}

// Export captures all learned scorer state for persistence.
func (e *Engine) Export() State {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := State{Segments: map[string]segState{}}
	for _, seg := range domain.AllSegments() {
		var ss segState
		for s := stage(0); s < numStages; s++ {
			m := e.models[seg][s]
			p := e.priors[seg][s]
			w := make([]float64, len(m.w))
			copy(w, m.w)
			ss.Heads[s] = headState{W: w, B: m.b, Seen: m.seen, Alpha: p.alpha, Beta: p.beta}
		}
		ss.Ticket = e.ticket[seg].value()
		out.Segments[string(seg)] = ss
	}
	return out
}

// Import restores learned scorer state (dimension-safe).
func (e *Engine) Import(st State) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, seg := range domain.AllSegments() {
		ss, ok := st.Segments[string(seg)]
		if !ok {
			continue
		}
		for s := stage(0); s < numStages; s++ {
			h := ss.Heads[s]
			m := e.models[seg][s]
			if len(h.W) == len(m.w) {
				copy(m.w, h.W)
				m.b = h.B
				m.seen = h.Seen
			}
			if h.Alpha > 0 || h.Beta > 0 {
				e.priors[seg][s].alpha = h.Alpha
				e.priors[seg][s].beta = h.Beta
			}
		}
		if ss.Ticket > 0 {
			e.ticket[seg] = newEWMA(ss.Ticket, 0.05)
		}
	}
}

// LearnedTicket exposes the current learned ticket EWMA for a segment so the
// budget motor can keep arm ExpectedValuePerAppt in sync with reality.
func (e *Engine) LearnedTicket(seg domain.Segment) float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if t, ok := e.ticket[seg]; ok {
		return t.value()
	}
	return priors.TicketTRY(seg)
}

// LearnedClose exposes the learned P(close | show) (Beta posterior mean) for a
// segment, so the expected value per appointment reflects realised close rates.
func (e *Engine) LearnedClose(seg domain.Segment) float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if ps, ok := e.priors[seg]; ok {
		return ps[stageClose].mean()
	}
	return priors.FunnelFor(seg).Close
}
