// Package feedback is the flywheel. Every real-world outcome — a reply, a
// booking, a kept appointment, a closed sale — is fanned out to every learning
// component so the next decision is better than the last. This is the part that
// compounds into a moat: competitors can copy the architecture, not the months
// of accumulated posteriors.
package feedback

import (
	"disci/brain/internal/budget"
	"disci/brain/internal/domain"
	"disci/brain/internal/noshow"
	"disci/brain/internal/priors"
	"disci/brain/internal/scoring"
	"disci/brain/internal/sla"
)

// Loop wires outcomes back into the four learning surfaces.
type Loop struct {
	Scorer  *scoring.Engine
	Budget  *budget.Engine
	NoShow  *noshow.Predictor
	SLA     *sla.Controller
}

// Ingest routes a single outcome to every model that can learn from it.
//
//   - The scorer learns each funnel transition (qualify/book/show/close).
//   - The budget arm that produced the lead updates its conversion posterior;
//     "won" means the lead became a qualified appointment.
//   - The SLA controller counts qualified appointments toward the guarantee.
//
// feats are the lead's features (needed for the scorer); apptFeats/showed feed
// the no-show predictor when an appointment outcome is known.
func (l *Loop) Ingest(o domain.Outcome, feats domain.LeadFeatures) {
	if l.Scorer != nil {
		l.Scorer.Learn(o, feats)
	}

	// The ad arm's job is to deliver *qualified* leads — that's its quality
	// signal, independent of our downstream capacity decisions. Rewarding the
	// arm only on booked appointments would unfairly punish good arms whenever a
	// clinic is at capacity.
	if o.ArmID != "" && l.Budget != nil {
		l.Budget.Observe(o.ArmID, truthy(o.Qualified), costOf(o))
	}

	if l.SLA != nil && truthy(o.Qualified) && truthy(o.Booked) {
		l.SLA.RecordQualifiedAppt(o.ClinicID)
	}

	// Close the value loop: once we've closed a real deal, refresh the clinic's
	// arms' expected value-per-appointment from the scorer's LEARNED ticket and
	// close rate — so the budget motor optimises against realised economics, not
	// a frozen registration-time guess.
	if l.Scorer != nil && l.Budget != nil && truthy(o.Closed) {
		v := l.Scorer.LearnedTicket(o.Segment) * priors.MarginFor(o.Segment) * l.Scorer.LearnedClose(o.Segment)
		l.Budget.UpdateValue(o.ClinicID, v)
	}
}

// IngestShow updates the no-show predictor once an appointment's outcome is
// known. Kept separate because show/no-show resolves later than booking.
func (l *Loop) IngestShow(a noshow.Appt, showed bool) {
	if l.NoShow != nil {
		l.NoShow.Learn(a, showed)
	}
}

func truthy(b *bool) bool { return b != nil && *b }

// costOf extracts the ad cost attributable to this lead. In production this is
// the actual spend reported by the ad platform; here it is carried on the
// outcome's revenue-adjacent fields or defaulted.
func costOf(o domain.Outcome) float64 {
	// The ad platform reports the spend attributed to this lead on the outcome.
	return o.AdCost
}
