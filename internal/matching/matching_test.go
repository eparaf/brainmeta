package matching

import (
	"testing"

	"disci/brain/internal/domain"
)

// TestRouteRespectsCapacityAndValue checks that with 2 leads and a clinic with
// only 1 seat, the higher-value lead wins the seat.
func TestRouteRespectsCapacity(t *testing.T) {
	clinic := domain.Clinic{ID: "c1", Segment: domain.SegmentImplant, CloseRate: 0.4}
	cands := []Candidate{
		{Lead: domain.Lead{ID: "low", Segment: domain.SegmentImplant}, Score: domain.LeadScore{EV: 1000}},
		{Lead: domain.Lead{ID: "high", Segment: domain.SegmentImplant}, Score: domain.LeadScore{EV: 50000}},
	}
	slots := []ClinicSlot{{Clinic: clinic, FreeSeats: 1, SLABias: 1}}
	res := Route(cands, slots)

	routed := map[string]bool{}
	for _, a := range res {
		if a.Routed {
			routed[a.LeadID] = true
		}
	}
	if !routed["high"] {
		t.Fatalf("high-value lead should win the single seat: %+v", res)
	}
	if routed["low"] {
		t.Fatalf("low-value lead should not be routed when capacity is 1")
	}
}

// TestSLABiasRedirectsLead checks a lagging clinic (high SLABias) pulls a lead
// away from an otherwise-equal clinic.
func TestSLABiasRedirectsLead(t *testing.T) {
	a := domain.Clinic{ID: "ahead", Segment: domain.SegmentImplant, CloseRate: 0.4}
	b := domain.Clinic{ID: "behind", Segment: domain.SegmentImplant, CloseRate: 0.4}
	cands := []Candidate{{Lead: domain.Lead{ID: "l1", Segment: domain.SegmentImplant}, Score: domain.LeadScore{EV: 10000}}}
	slots := []ClinicSlot{
		{Clinic: a, FreeSeats: 1, SLABias: 1.0},
		{Clinic: b, FreeSeats: 1, SLABias: 3.0},
	}
	res := Route(cands, slots)
	if len(res) != 1 || res[0].ClinicID != "behind" {
		t.Fatalf("lead should route to the lagging clinic, got %+v", res)
	}
}

// TestIncompatibleSegmentNotRouted ensures an aesthetic-only clinic refuses a
// general lead.
func TestIncompatibleSegmentNotRouted(t *testing.T) {
	clinic := domain.Clinic{ID: "aes", Segment: domain.SegmentAesthetic, CloseRate: 0.3}
	cands := []Candidate{{Lead: domain.Lead{ID: "g", Segment: domain.SegmentGeneral}, Score: domain.LeadScore{EV: 5000}}}
	res := Route(cands, []ClinicSlot{{Clinic: clinic, FreeSeats: 5, SLABias: 1}})
	if res[0].Routed {
		t.Fatalf("general lead should not route to aesthetic-only clinic")
	}
}
