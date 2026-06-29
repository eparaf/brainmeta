// Package engine is the orchestrator — the brain proper. It owns the four motors
// and the feedback loop, and exposes the two decisions the rest of the system
// asks for:
//
//	HandleLead   — a lead just arrived: score it, route it to a clinic, and
//	               decide whether/how to book it (capacity + no-show aware).
//	PlanBudget   — a planning cycle ticked: reallocate ad spend across arms.
//
// The orchestrator is deliberately thin: all the hard math lives in the motors.
// Its job is sequencing and enforcing the cross-cutting policies in config.
package engine

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"disci/brain/internal/budget"
	"disci/brain/internal/config"
	"disci/brain/internal/domain"
	"disci/brain/internal/feedback"
	"disci/brain/internal/matching"
	"disci/brain/internal/noshow"
	"disci/brain/internal/priors"
	"disci/brain/internal/scoring"
	"disci/brain/internal/sla"
	"disci/brain/internal/store"
)

// Engine bundles every component.
type Engine struct {
	cfg    config.Config
	store  store.Store // interface → swap Memory for Postgres without touching logic
	Scorer *scoring.Engine
	Budget *budget.Engine
	NoShow *noshow.Predictor
	SLA    *sla.Controller
	Loop   *feedback.Loop
	Pacers *budget.PacerSet // per-arm PID pacers (no shared state across arms)

	// Booking ledger: accepted show probabilities bucketed by (clinic, APPOINTMENT
	// day) — not booking day — so the exact overbooking math is applied to the day
	// the patients will actually arrive. Guarded by bookMu; past days are pruned.
	bookMu   sync.Mutex
	dayShows map[string][]float64 // key = clinicID|YYYY-MM-DD (appointment day)

	// Decision-time feature stores, persisted in the snapshot so learning survives
	// restarts: leadFeats (scorer features) and apptFeats (no-show features), keyed
	// by lead ID; plus the outcome dedup set so a restart never re-ingests history.
	featMu    sync.Mutex
	leadFeats map[string]domain.LeadFeatures
	apptFeats map[string]noshow.Appt
	seen      map[string]bool

	apptSeq uint64
}

// New constructs a fully-wired brain. st is any store.Store (Memory or Postgres).
func New(cfg config.Config, st store.Store) *Engine {
	scorer := scoring.NewEngine()
	bud := budget.NewEngine(cfg.Seed)
	ns := noshow.NewPredictor()
	guard := sla.NewController(cfg.ServiceLevel)
	return &Engine{
		cfg:       cfg,
		store:     st,
		Scorer:    scorer,
		Budget:    bud,
		NoShow:    ns,
		SLA:       guard,
		Pacers:    budget.NewPacerSet(),
		dayShows:  map[string][]float64{},
		leadFeats: map[string]domain.LeadFeatures{},
		apptFeats: map[string]noshow.Appt{},
		seen:      map[string]bool{},
		Loop: &feedback.Loop{
			Scorer: scorer, Budget: bud, NoShow: ns, SLA: guard,
		},
	}
}

// RegisterClinic onboards a tenant: stores it, registers its SLA, and creates
// its advertising arms (one per platform for its primary segment to start).
func (e *Engine) RegisterClinic(c domain.Clinic) {
	e.store.UpsertClinic(c)
	e.SLA.Register(c.ID, c.GuaranteedApptsPerMonth)
	for _, plat := range []domain.Platform{domain.PlatformMeta, domain.PlatformGoogle} {
		e.Budget.RegisterArm(domain.Arm{
			ID:             fmt.Sprintf("%s:%s:%s", c.ID, plat, c.Segment),
			ClinicID:       c.ID,
			Platform:       plat,
			Campaign:       "default",
			Creative:       "v1",
			Segment:        c.Segment,
			AvgCostPerLead: defaultCPL(plat, c.Segment),
			ClinicCapacity: c.DailyCapacity,
			// Expected realised margin per qualified appointment: ticket × margin ×
			// the clinic's close rate. This is what makes the budget motor value a
			// premium tourism appointment above a cheap local one.
			ExpectedValuePerAppt: priors.TicketTRY(c.Segment) * priors.MarginFor(c.Segment) * closeOrDefault(c.CloseRate),
		})
	}
}

// defaultCPL seeds an arm's cost-per-lead from the sourced 2025–2026 benchmarks.
// Aesthetic clinics sell to the European dental-tourism audience (Western CPLs);
// the rest target local Turkish patients (far cheaper per lead in TRY).
func closeOrDefault(r float64) float64 {
	if r <= 0 {
		return 0.4
	}
	return r
}

func defaultCPL(plat domain.Platform, seg domain.Segment) float64 {
	aud := priors.AudienceLocalTR
	if seg == domain.SegmentAesthetic {
		aud = priors.AudienceTourism
	}
	return priors.CPLTRY(plat, aud)
}

// LeadDecision is the result of HandleLead.
type LeadDecision struct {
	LeadID       string
	Booked       bool
	ClinicID     string
	Score        domain.LeadScore
	Reason       string
	PShow        float64
	Intervention string
	ApptTime     time.Time
}

// HandleLead scores a freshly-captured lead, routes it to the best compatible
// clinic, and decides whether to book it. This is the real-time hot path that
// fires the moment the WhatsApp agent has gathered enough to qualify.
func (e *Engine) HandleLead(lead domain.Lead, now time.Time) LeadDecision {
	// Roll the guarantee window if the month changed (the previously-missing
	// ResetMonth caller).
	e.SLA.MaybeReset(now)

	score := e.Scorer.Score(lead)
	lead.Score = score

	// Persist the decision-time scorer features (durable across restarts) so the
	// outcome can later train the scorer on the REAL features, not empty ones.
	e.recordLeadFeatures(lead.ID, lead.Features)

	dec := LeadDecision{LeadID: lead.ID, Score: score}

	// Junk filter first — clinic-independent. A lead that isn't even plausibly
	// qualified never consumes a slot, regardless of which clinic.
	if score.PQualify < e.cfg.MinPQualifyToBook {
		lead.Status = domain.LeadLost
		dec.Reason = "below_qualify_threshold"
		e.store.SaveLead(lead)
		return dec
	}

	// Route: explicit clinic (single-tenant regime) or marketplace routing.
	clinicID := lead.ClinicID
	if clinicID == "" {
		clinicID = e.routeOne(lead, score, now)
	}
	if clinicID == "" {
		lead.Status = domain.LeadQualified
		dec.Reason = "no_capacity_anywhere"
		e.store.SaveLead(lead)
		return dec
	}
	clinic, _ := e.store.GetClinic(clinicID)

	// EV floor — but SLA-aware. The value of booking a lead is its treatment
	// margin PLUS the contractual value of moving a clinic toward its guarantee.
	// We fold that in via the same shadow price the budget/matching motors use:
	// a clinic that's behind gets its effective EV scaled up, so low-ticket leads
	// (e.g. general checkups) still get booked when the guarantee demands volume.
	slaBias := e.SLA.BudgetBias(clinicID)
	if score.EV*slaBias < e.cfg.MinEVToBook {
		lead.Status = domain.LeadLost
		dec.Reason = "below_ev_floor"
		dec.ClinicID = clinicID
		e.store.SaveLead(lead)
		return dec
	}

	// Provisional no-show estimate (assume a 2-day lead) to choose the
	// intervention tier and the show prob used for the overbooking reservation.
	prov := noshow.Appt{
		LeadTimeDays: 2, HourOfDay: lead.Features.HourOfDay,
		PastNoShows: lead.Features.PastNoShows, ConfirmedReply: true,
		DistanceKm: lead.Features.DistanceKm, Segment: lead.Segment,
	}
	lifted, iv := noshow.ApplyBestIntervention(e.NoShow.PShow(prov), score.Margin)

	// Reserve the earliest APPOINTMENT day whose EXACT Poisson-binomial overbooking
	// risk still admits this booking. Buckets are keyed by appointment day, so the
	// risk math is applied to the population that actually arrives that day (the
	// re-audit's fix), and the reservation is atomic (race-free under bookMu).
	apptTime, ok := e.reserveSlot(clinic, lifted, now)
	if !ok {
		lead.Status = domain.LeadQualified
		dec.Reason = "clinic_at_capacity"
		dec.ClinicID = clinicID
		e.store.SaveLead(lead)
		return dec
	}
	leadDays := apptTime.Sub(now).Hours() / 24.0
	iv = noshow.ChooseIntervention(lifted, score.Margin) // tier consistent with lifted

	// Book it.
	lead.Status = domain.LeadBooked
	lead.ClinicID = clinicID
	lead.BookedAt = now
	lead.ApptTime = apptTime
	e.store.SaveLead(lead)

	// Remember the FINAL decision-time no-show features (real lead time) so we can
	// train the no-show model on the real outcome later.
	e.recordApptFeatures(lead.ID, noshow.Appt{
		LeadTimeDays: leadDays, HourOfDay: lead.Features.HourOfDay,
		PastNoShows: lead.Features.PastNoShows, ConfirmedReply: true,
		DistanceKm: lead.Features.DistanceKm, Segment: lead.Segment,
		IsWeekend: apptTime.Weekday() == time.Saturday || apptTime.Weekday() == time.Sunday,
	})

	e.store.SaveAppointment(domain.Appointment{
		ID:       e.nextApptID(),
		ClinicID: clinicID,
		LeadID:   lead.ID,
		Phone:    lead.Phone,
		Name:     lead.Name,
		When:     apptTime,
		Segment:  lead.Segment,
		PShow:    lifted,
	})

	dec.Booked = true
	dec.ClinicID = clinicID
	dec.PShow = lifted
	dec.Intervention = iv.String()
	dec.ApptTime = apptTime
	dec.Reason = "booked"
	return dec
}

const bookingHorizonDays = 21

func dayBucketKey(clinicID, day string) string { return clinicID + "|" + day }

// pruneLocked drops appointment-day buckets for days already in the past. Caller
// must hold bookMu. Keeps the ledger bounded for a long-running server.
func (e *Engine) pruneLocked(now time.Time) {
	today := now.Format("2006-01-02")
	for k := range e.dayShows {
		if i := strings.LastIndex(k, "|"); i >= 0 && k[i+1:] < today {
			delete(e.dayShows, k)
		}
	}
}

// reserveSlot finds the earliest future appointment day whose EXACT overbooking
// risk still admits a booking at show prob p, atomically records it in that
// day's bucket, and returns the appointment time. ok=false if no day within the
// horizon has room. Buckets are keyed by appointment day (not booking day).
func (e *Engine) reserveSlot(clinic domain.Clinic, p float64, now time.Time) (time.Time, bool) {
	cap := clinic.DailyCapacity
	if cap <= 0 {
		cap = 1
	}
	e.bookMu.Lock()
	defer e.bookMu.Unlock()
	e.pruneLocked(now)
	for off := 1; off <= bookingHorizonDays; off++ {
		day := now.Truncate(24*time.Hour).AddDate(0, 0, off)
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			continue
		}
		key := dayBucketKey(clinic.ID, day.Format("2006-01-02"))
		bucket := e.dayShows[key]
		if noshow.AcceptableOverbook(bucket, p, cap, e.cfg.MaxOverbookRisk) {
			slot := len(bucket) % cap
			e.dayShows[key] = append(bucket, p)
			return day.Add(time.Duration(9)*time.Hour + time.Duration(slot*30)*time.Minute), true
		}
	}
	return time.Time{}, false
}

// clinicHasRoom checks (without reserving) whether a clinic could accept a
// typical additional booking on some day in the horizon — filters routing
// candidates.
func (e *Engine) clinicHasRoom(clinicID string, capacity int, now time.Time) bool {
	cap := capacity
	if cap <= 0 {
		cap = 1
	}
	e.bookMu.Lock()
	defer e.bookMu.Unlock()
	e.pruneLocked(now)
	for off := 1; off <= bookingHorizonDays; off++ {
		day := now.Truncate(24*time.Hour).AddDate(0, 0, off)
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			continue
		}
		key := dayBucketKey(clinicID, day.Format("2006-01-02"))
		if noshow.AcceptableOverbook(e.dayShows[key], priors.BaseShowProb, cap, e.cfg.MaxOverbookRisk) {
			return true
		}
	}
	return false
}

// routeOne picks the best compatible clinic for a single lead (marketplace
// regime). It reuses the matching motor's value function via a 1×N assignment.
func (e *Engine) routeOne(lead domain.Lead, score domain.LeadScore, now time.Time) string {
	cands := []matching.Candidate{{Lead: lead, Score: score}}
	var slots []matching.ClinicSlot
	for _, c := range e.store.ListClinics() {
		if !e.clinicHasRoom(c.ID, c.DailyCapacity, now) {
			continue
		}
		slots = append(slots, matching.ClinicSlot{
			Clinic:    c,
			FreeSeats: 1, // routing one lead; the overbooking gate enforces real capacity
			SLABias:   e.SLA.MatchBias(c.ID),
		})
	}
	if len(slots) == 0 {
		return ""
	}
	res := matching.Route(cands, slots)
	if len(res) > 0 && res[0].Routed {
		return res[0].ClinicID
	}
	return ""
}

// RouteBatch routes a batch of pooled leads optimally (full Hungarian
// assignment). Used when leads are buffered and routed in waves rather than
// one-at-a-time — yields a globally better allocation than greedy.
func (e *Engine) RouteBatch(leads []domain.Lead, now time.Time) []matching.Assignment {
	cands := make([]matching.Candidate, len(leads))
	for i, l := range leads {
		cands[i] = matching.Candidate{Lead: l, Score: e.Scorer.Score(l)}
	}
	var slots []matching.ClinicSlot
	for _, c := range e.store.ListClinics() {
		if !e.clinicHasRoom(c.ID, c.DailyCapacity, now) {
			continue
		}
		slots = append(slots, matching.ClinicSlot{
			Clinic: c, FreeSeats: c.DailyCapacity, SLABias: e.SLA.MatchBias(c.ID),
		})
	}
	return matching.Route(cands, slots)
}

// PlanBudget runs a budget-allocation cycle. Each clinic's monthly budget is
// prorated to a daily figure and allocated across that clinic's own arms.
func (e *Engine) PlanBudget(daysInMonth float64) ([]budget.Allocation, float64) {
	if daysInMonth <= 0 {
		daysInMonth = 30
	}
	clinicDaily := map[string]float64{}
	for _, c := range e.store.ListClinics() {
		clinicDaily[c.ID] = c.MonthlyAdBudget / daysInMonth
	}
	return e.Budget.Allocate(clinicDaily, e.SLA)
}

// NetworkDailyBudget sums clinics' monthly budgets prorated to a daily figure
// (reporting convenience).
func (e *Engine) NetworkDailyBudget(daysInMonth float64) float64 {
	var total float64
	for _, c := range e.store.ListClinics() {
		total += c.MonthlyAdBudget / daysInMonth
	}
	return total
}

// SLAReport surfaces guarantee health.
func (e *Engine) SLAReport(now time.Time) []sla.Status { return e.SLA.Report(now) }

// Clinics returns all onboarded clinics (used by the datasource sync loops).
func (e *Engine) Clinics() []domain.Clinic { return e.store.ListClinics() }

// Appointments returns a clinic's appointments (used by the reminder scheduler).
func (e *Engine) Appointments(clinicID string) []domain.Appointment {
	return e.store.AppointmentsForClinic(clinicID)
}

// Clinic returns a clinic by id.
func (e *Engine) Clinic(id string) (domain.Clinic, bool) { return e.store.GetClinic(id) }

// NextSlots previews the next n bookable appointment times for a clinic WITHOUT
// reserving them — so the agent (text or voice) can offer concrete options to the
// patient. Read-only; the actual hold happens in HandleLead/reserveSlot.
func (e *Engine) NextSlots(clinicID string, n int, now time.Time) []time.Time {
	c, ok := e.store.GetClinic(clinicID)
	cap := 1
	if ok && c.DailyCapacity > 0 {
		cap = c.DailyCapacity
	}
	e.bookMu.Lock()
	defer e.bookMu.Unlock()
	e.pruneLocked(now)
	out := make([]time.Time, 0, n)
	for off := 1; off <= bookingHorizonDays && len(out) < n; off++ {
		day := now.Truncate(24*time.Hour).AddDate(0, 0, off)
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			continue
		}
		used := len(e.dayShows[dayBucketKey(clinicID, day.Format("2006-01-02"))])
		for slot := used; slot < cap && len(out) < n; slot++ {
			out = append(out, day.Add(9*time.Hour+time.Duration(slot*30)*time.Minute))
		}
	}
	return out
}

// recordApptFeatures stores the booking-time no-show features for a lead.
func (e *Engine) recordApptFeatures(leadID string, a noshow.Appt) {
	e.featMu.Lock()
	defer e.featMu.Unlock()
	e.apptFeats[leadID] = a
}

// recordLeadFeatures stores the decision-time scorer features for a lead.
func (e *Engine) recordLeadFeatures(leadID string, f domain.LeadFeatures) {
	e.featMu.Lock()
	defer e.featMu.Unlock()
	e.leadFeats[leadID] = f
}

// ApptFeatures returns the booking-time no-show features for a lead, if known.
func (e *Engine) ApptFeatures(leadID string) (noshow.Appt, bool) {
	e.featMu.Lock()
	defer e.featMu.Unlock()
	a, ok := e.apptFeats[leadID]
	return a, ok
}

// IngestOutcome is the single, dedup-guarded entry point for realised outcomes.
// It ensures every outcome trains the models EXACTLY ONCE — even across restarts,
// because the dedup set is part of the persisted snapshot. It looks up the lead's
// stored decision-time features so the scorer and no-show model learn on the real
// features, and returns false if the outcome was already ingested.
func (e *Engine) IngestOutcome(o domain.Outcome) bool {
	key := OutcomeKey(o)
	e.featMu.Lock()
	if e.seen[key] {
		e.featMu.Unlock()
		return false
	}
	e.seen[key] = true
	feats := e.leadFeats[o.LeadID]
	apptF, hasAppt := e.apptFeats[o.LeadID]
	e.featMu.Unlock()

	e.Loop.Ingest(o, feats)
	if o.Showed != nil && hasAppt {
		e.Loop.IngestShow(apptF, *o.Showed)
	}
	return true
}

// OutcomeKey is the dedup key for an outcome (stable across restarts).
func OutcomeKey(o domain.Outcome) string {
	if o.OutcomeID != "" {
		return o.OutcomeID
	}
	b := func(p *bool) string {
		if p == nil {
			return "-"
		}
		if *p {
			return "1"
		}
		return "0"
	}
	return fmt.Sprintf("%s|%s|%s|%s|%s", o.LeadID, b(o.Qualified), b(o.Booked), b(o.Showed), b(o.Closed))
}

func (e *Engine) nextApptID() string {
	n := atomic.AddUint64(&e.apptSeq, 1)
	return fmt.Sprintf("appt-%d", n)
}
