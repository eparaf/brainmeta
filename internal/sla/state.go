package sla

// ClinicSnapshot persists one clinic's guarantee progress.
type ClinicSnapshot struct {
	Guaranteed int `json:"guaranteed"`
	Delivered  int `json:"delivered"`
}

// State is the serializable snapshot of the guarantee controller.
type State struct {
	Month   string                    `json:"month"`
	Clinics map[string]ClinicSnapshot `json:"clinics"`
}

// Export captures every clinic's delivered count for the current month.
func (c *Controller) Export() State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := State{Month: c.month, Clinics: map[string]ClinicSnapshot{}}
	for id, s := range c.clinics {
		out.Clinics[id] = ClinicSnapshot{Guaranteed: s.guaranteed, Delivered: s.delivered}
	}
	return out
}

// Import restores guarantee progress (clinics must already be registered).
func (c *Controller) Import(st State) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.month = st.Month
	for id, snap := range st.Clinics {
		if s, ok := c.clinics[id]; ok {
			s.delivered = snap.Delivered
			if snap.Guaranteed > 0 {
				s.guaranteed = snap.Guaranteed
			}
		}
	}
}
