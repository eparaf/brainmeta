package budget

// ArmSnapshot is one arm's persisted learned state.
type ArmSnapshot struct {
	ArmID      string  `json:"armId"`
	Alpha      float64 `json:"alpha"`
	Beta       float64 `json:"beta"`
	PriorAlpha float64 `json:"priorAlpha"`
	PriorBeta  float64 `json:"priorBeta"`
	CPL        float64 `json:"cpl"`
	Leads      int     `json:"leads"`
	Appts      int     `json:"appts"`
	Spend      float64 `json:"spend"`
}

// State is the serializable snapshot of the budget allocator.
type State struct {
	Arms []ArmSnapshot `json:"arms"`
}

// Export captures every arm's learned posterior + cost estimate.
func (e *Engine) Export() State {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := State{Arms: make([]ArmSnapshot, 0, len(e.arms))}
	for _, a := range e.arms {
		out.Arms = append(out.Arms, ArmSnapshot{
			ArmID: a.arm.ID, Alpha: a.alpha, Beta: a.beta,
			PriorAlpha: a.priorAlpha, PriorBeta: a.priorBeta,
			CPL: a.cpl, Leads: a.leadsDelivered, Appts: a.apptsWon, Spend: a.spend,
		})
	}
	return out
}

// Import restores arm posteriors onto already-registered arms (match by ArmID).
func (e *Engine) Import(s State) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, snap := range s.Arms {
		a, ok := e.arms[snap.ArmID]
		if !ok {
			continue
		}
		a.alpha, a.beta = snap.Alpha, snap.Beta
		if snap.PriorAlpha > 0 {
			a.priorAlpha, a.priorBeta = snap.PriorAlpha, snap.PriorBeta
		}
		if snap.CPL > 0 {
			a.cpl = snap.CPL
		}
		a.leadsDelivered, a.apptsWon, a.spend = snap.Leads, snap.Appts, snap.Spend
	}
}
