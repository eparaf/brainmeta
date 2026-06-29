package engine

import (
	"time"

	"disci/brain/internal/domain"
	"disci/brain/internal/noshow"
	"disci/brain/internal/persist"
)

// SaveSnapshot captures every motor's learned state through the given store.
// Call it periodically (and on shutdown) so the moat is durable.
func (e *Engine) SaveSnapshot(store persist.Store, now time.Time) error {
	snap := persist.Snapshot{
		SavedAt: now.Format(time.RFC3339),
		Scoring: e.Scorer.Export(),
		Budget:  e.Budget.Export(),
		NoShow:  e.NoShow.Export(),
		SLA:     e.SLA.Export(),
		Engine:  e.exportState(),
	}
	return store.Save(snap)
}

// exportState copies the durable orchestrator bookkeeping for the snapshot.
func (e *Engine) exportState() persist.EngineState {
	e.featMu.Lock()
	defer e.featMu.Unlock()
	seen := make([]string, 0, len(e.seen))
	for k := range e.seen {
		seen = append(seen, k)
	}
	lf := make(map[string]domain.LeadFeatures, len(e.leadFeats))
	for k, v := range e.leadFeats {
		lf[k] = v
	}
	af := make(map[string]noshow.Appt, len(e.apptFeats))
	for k, v := range e.apptFeats {
		af[k] = v
	}
	return persist.EngineState{Seen: seen, LeadFeats: lf, ApptFeats: af}
}

// LoadSnapshot restores learned state on startup. Clinics/arms must already be
// registered (Import matches by ID). Returns false if there was no snapshot yet.
func (e *Engine) LoadSnapshot(store persist.Store) (bool, error) {
	snap, ok, err := store.Load()
	if err != nil || !ok {
		return false, err
	}
	e.Scorer.Import(snap.Scoring)
	e.Budget.Import(snap.Budget)
	e.NoShow.Import(snap.NoShow)
	e.SLA.Import(snap.SLA)
	e.importState(snap.Engine)
	return true, nil
}

// importState restores the dedup set + feature stores so a restart never
// re-ingests history and can still train on the real decision-time features.
func (e *Engine) importState(s persist.EngineState) {
	e.featMu.Lock()
	defer e.featMu.Unlock()
	for _, k := range s.Seen {
		e.seen[k] = true
	}
	for k, v := range s.LeadFeats {
		e.leadFeats[k] = v
	}
	for k, v := range s.ApptFeats {
		e.apptFeats[k] = v
	}
}
