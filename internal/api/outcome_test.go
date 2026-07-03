package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"disci/brain/internal/domain"
)

// TestHandleOutcomeDedup guards against the historical bug where /v1/outcomes
// called Loop.Ingest directly, bypassing engine.IngestOutcome's durable dedup —
// a retried or duplicate POST (e.g. a PMS sync loop retry, or a double-click in
// the panel) would double-train the models on the same event. The endpoint must
// report fresh=true once and fresh=false on every repeat.
func TestHandleOutcomeDedup(t *testing.T) {
	f := newScopingFixture(t)
	qualified, showed := true, true
	body := map[string]any{
		"outcome": map[string]any{
			"outcomeId": "evt-dedup-1",
			"leadId":    f.leadA.ID,
			"clinicId":  f.clinicA,
			"qualified": qualified,
			"showed":    showed,
		},
	}

	post := func() map[string]any {
		w := httptest.NewRecorder()
		f.s.handleOutcome(w, reqWithUserJSON("POST", "/v1/outcomes", f.admin, body))
		if w.Code != 200 {
			t.Fatalf("handleOutcome: want 200, got %d (%s)", w.Code, w.Body.String())
		}
		var out map[string]any
		json.Unmarshal(w.Body.Bytes(), &out)
		return out
	}

	first := post()
	if fresh, _ := first["fresh"].(bool); !fresh {
		t.Fatalf("first ingest should be fresh, got %v", first)
	}
	second := post()
	if fresh, _ := second["fresh"].(bool); fresh {
		t.Fatalf("duplicate ingest must report fresh=false (dedup), got %v", second)
	}
}

// TestHandleOutcomeCrossTenant: a clinic-scoped user must not be able to inject
// an outcome for a clinic they don't belong to.
func TestHandleOutcomeCrossTenant(t *testing.T) {
	f := newScopingFixture(t)
	userB := &domain.User{Role: domain.RoleClinic, ClinicIDs: []string{f.clinicB}}
	body := map[string]any{
		"outcome": map[string]any{
			"outcomeId": "evt-cross-1",
			"leadId":    f.leadA.ID,
			"clinicId":  f.clinicA, // NOT userB's clinic
			"qualified": true,
		},
	}
	w := httptest.NewRecorder()
	f.s.handleOutcome(w, reqWithUserJSON("POST", "/v1/outcomes", userB, body))
	if w.Code != 403 {
		t.Fatalf("cross-tenant outcome report should be 403, got %d", w.Code)
	}
}
