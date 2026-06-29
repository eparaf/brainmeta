// Package datasource is how the brain reaches REAL data. The motors are pure
// decision logic; this package is the adapter ring around them that talks to the
// outside world. Every external system is hidden behind a narrow interface, so
// the engine never knows whether a lead came from WhatsApp or a test, or whether
// spend numbers came from Meta or a CSV.
//
// Four real-world surfaces feed/drain the brain:
//
//	1. LeadSource   — inbound prospects        (Meta/Google "Click-to-WhatsApp",
//	                                             WhatsApp Cloud API webhooks)
//	2. AdPlatform   — spend & conversions      (Meta Marketing API, Google Ads API)
//	3. ClinicPMS    — capacity, appts, outcomes(clinic practice-mgmt system / CRM)
//	4. Messenger    — outbound reminders       (WhatsApp Cloud API send)
//
// The SyncService wires them to the engine on a schedule. Concrete adapters
// (HTTP clients for each API) implement these interfaces; the ones here are
// documented contracts plus a runnable file/in-memory adapter for local dev.
package datasource

import (
	"context"
	"time"

	"disci/brain/internal/domain"
)

// ---- Inbound: where leads come from -------------------------------------

// LeadEvent is a normalised inbound lead from any channel. The concrete adapter
// is responsible for mapping (e.g.) a WhatsApp Cloud API webhook payload or a
// Meta lead-ad form into this shape, including attributing the ArmID from the
// ad's UTM / campaign metadata so the bandit can credit the right arm.
type LeadEvent struct {
	ExternalID string
	ArmID      string // resolved from ad click metadata (utm/campaign → arm)
	ClinicID   string
	Phone      string
	RawMessage string
	ReceivedAt time.Time
}

// LeadSource delivers inbound leads. In production this is an HTTP webhook
// handler (WhatsApp Cloud API) that pushes events; for batch channels it can be
// a poller. Either way it emits normalised LeadEvents on the channel.
type LeadSource interface {
	// Stream returns a channel of inbound leads until ctx is cancelled.
	Stream(ctx context.Context) (<-chan LeadEvent, error)
}

// ---- Ads: spend, cost, and conversion feedback --------------------------

// ArmSpend is one arm's realised performance over a window, pulled from the ad
// platform. CostPerLead here is GROUND TRUTH (what we actually paid), which the
// budget motor uses to correct its learned CPL.
type ArmSpend struct {
	ArmID       string
	Spend       float64 // TRY spent in the window
	Leads       int     // leads delivered
	CostPerLead float64 // Spend / Leads
	Impressions int
	Clicks      int
}

// AdPlatform is the two-way ads integration.
type AdPlatform interface {
	// PullSpend reports realised spend/CPL per arm since `since` (Meta Marketing
	// API insights / Google Ads API reports).
	PullSpend(ctx context.Context, since time.Time) ([]ArmSpend, error)

	// SetDailyBudgets pushes the brain's allocation back to the platform so the
	// campaigns actually spend where the brain decided (Meta ad-set daily_budget /
	// Google campaign budget mutate).
	SetDailyBudgets(ctx context.Context, perArm map[string]float64) error

	// UploadConversions sends realised qualified-appointment / closed-deal events
	// back to the platform's optimiser (Meta Conversions API / Google offline
	// conversion import). This is what makes the platform's ML compound with ours.
	UploadConversions(ctx context.Context, convs []Conversion) error
}

// Conversion is an offline conversion event uploaded to an ad platform.
type Conversion struct {
	ArmID     string
	ExternalID string  // platform click id (gclid / fbclid) for attribution
	EventName string  // "qualified_appointment" | "showed" | "closed"
	Value     float64 // TRY
	At        time.Time
}

// ---- Clinic side: capacity, appointments, outcomes ----------------------

// ClinicPMS is the clinic's practice-management system / CRM. It tells us real
// capacity and confirms real-world outcomes (did the patient show? did the case
// close?), which are the labels the whole flywheel learns from.
type ClinicPMS interface {
	// PullCapacity returns the clinic's current bookable new-patient capacity.
	PullCapacity(ctx context.Context, clinicID string) (int, error)

	// PushAppointment writes a booked appointment into the clinic's calendar.
	PushAppointment(ctx context.Context, appt domain.Appointment) error

	// PullOutcomes returns realised outcomes (show / no-show / closed + revenue)
	// since `since`, e.g. from the PMS ledger or a nightly export.
	PullOutcomes(ctx context.Context, clinicID string, since time.Time) ([]domain.Outcome, error)
}

// ---- Outbound: reminders / confirmations --------------------------------

// Messenger sends outbound WhatsApp messages (reminders, confirmation requests,
// deposit links) — the no-show interventions Motor 4 selects.
type Messenger interface {
	Send(ctx context.Context, phone, templateID string, vars map[string]string) error
}
