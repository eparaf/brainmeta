package scoring

import (
	"testing"

	"disci/brain/internal/domain"
)

// TestColdStartIsSane checks the very first lead (zero data) gets a plausible
// EV from the priors, not zero or NaN.
func TestColdStartIsSane(t *testing.T) {
	e := NewEngine()
	lead := domain.Lead{Segment: domain.SegmentImplant, Features: domain.LeadFeatures{
		IntentScore: 0.7, UrgencyScore: 0.6, DistanceKm: 5, MessagesExchanged: 6,
	}}
	s := e.Score(lead)
	if s.EV <= 0 || s.PQualify <= 0 || s.PQualify >= 1 {
		t.Fatalf("cold-start score implausible: %+v", s)
	}
}

// TestLearningSeparatesGoodFromBadLeads feeds many positive outcomes for
// high-intent leads and negatives for low-intent ones, then checks the scorer
// rates a high-intent lead above a low-intent one.
func TestLearningSeparatesGoodFromBadLeads(t *testing.T) {
	e := NewEngine()
	hi := domain.LeadFeatures{IntentScore: 0.9, UrgencyScore: 0.9, MessagesExchanged: 9, DistanceKm: 3}
	lo := domain.LeadFeatures{IntentScore: 0.1, UrgencyScore: 0.1, MessagesExchanged: 1, DistanceKm: 30}
	yes, no := true, false
	for i := 0; i < 300; i++ {
		e.Learn(domain.Outcome{Segment: domain.SegmentImplant, Qualified: &yes, Booked: &yes}, hi)
		e.Learn(domain.Outcome{Segment: domain.SegmentImplant, Qualified: &no, Booked: &no}, lo)
	}
	sHi := e.Score(domain.Lead{Segment: domain.SegmentImplant, Features: hi})
	sLo := e.Score(domain.Lead{Segment: domain.SegmentImplant, Features: lo})
	if sHi.EV <= sLo.EV {
		t.Fatalf("expected high-intent EV > low-intent: hi=%.0f lo=%.0f", sHi.EV, sLo.EV)
	}
}
