// Package persist serializes the brain's LEARNED state so the moat — months of
// accumulated posteriors — survives restarts, deploys, and crashes instead of
// evaporating into process memory. It snapshots all four motors plus the
// guarantee controller as one JSON document.
//
// The Store interface is deliberately tiny so the file-backed implementation
// here can be swapped for Postgres/S3 with no change to the engine. Writes are
// atomic (temp file + rename) so a crash mid-save never corrupts the snapshot.
package persist

import (
	"encoding/json"
	"os"
	"path/filepath"

	"disci/brain/internal/budget"
	"disci/brain/internal/domain"
	"disci/brain/internal/noshow"
	"disci/brain/internal/scoring"
	"disci/brain/internal/sla"
)

// EngineState persists the orchestrator's durable bookkeeping: the outcome dedup
// set and the decision-time feature stores. Without these in the snapshot, a
// restart would re-ingest the entire outcome history (double-counting) and train
// on empty features — the exact regression the re-audit flagged.
type EngineState struct {
	Seen      []string                       `json:"seen"`
	LeadFeats map[string]domain.LeadFeatures `json:"leadFeats"`
	ApptFeats map[string]noshow.Appt         `json:"apptFeats"`
}

// Snapshot is the full learned state of the brain at a point in time.
type Snapshot struct {
	Version int           `json:"version"`
	SavedAt string        `json:"savedAt"`
	Scoring scoring.State `json:"scoring"`
	Budget  budget.State  `json:"budget"`
	NoShow  noshow.State  `json:"noshow"`
	SLA     sla.State     `json:"sla"`
	Engine  EngineState   `json:"engine"`
}

// Store persists and restores brain snapshots.
type Store interface {
	Save(Snapshot) error
	Load() (Snapshot, bool, error)
}

// FileStore writes the snapshot to a single JSON file (mount a Docker volume at
// its directory to make the moat durable across container restarts).
type FileStore struct{ Path string }

// NewFileStore returns a file-backed snapshot store.
func NewFileStore(path string) *FileStore { return &FileStore{Path: path} }

// Save writes the snapshot atomically (temp file + rename).
func (f *FileStore) Save(s Snapshot) error {
	s.Version = 1
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(f.Path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp := f.Path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, f.Path)
}

// Load reads the snapshot. The bool is false (no error) when no snapshot exists
// yet — a fresh brain, not a failure.
func (f *FileStore) Load() (Snapshot, bool, error) {
	var s Snapshot
	b, err := os.ReadFile(f.Path)
	if os.IsNotExist(err) {
		return s, false, nil
	}
	if err != nil {
		return s, false, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return s, false, err
	}
	return s, true, nil
}
