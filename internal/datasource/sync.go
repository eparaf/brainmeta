package datasource

import (
	"context"
	"log"
	"time"

	"disci/brain/internal/domain"
	"disci/brain/internal/engine"
	"disci/brain/internal/whatsapp"
)

// SyncService is the conductor that connects the real-world surfaces to the
// brain. It runs three loops:
//
//	leads     — consume inbound LeadEvents → engine.HandleLead → push appt + reminder
//	ads        — periodically: pull real CPL → correct the bandit; push the
//	             brain's budget decisions to the platforms; upload conversions
//	outcomes   — periodically: pull real show/close outcomes → feedback loop
//
// Every external call is behind an interface, so the same SyncService runs
// against live APIs in production and the in-memory adapter in tests/dev.
type SyncService struct {
	Eng       *engine.Engine
	Leads     LeadSource
	Ads       AdPlatform
	PMS       ClinicPMS
	Messenger Messenger

	// Featurizer turns a raw inbound message into model features. In production
	// this is the WhatsApp agent's NLU output; injectable for testing.
	Featurizer func(LeadEvent) domain.LeadFeatures
}

// Run starts all loops and blocks until ctx is cancelled.
func (s *SyncService) Run(ctx context.Context, adEvery, outcomeEvery time.Duration) error {
	if s.Leads != nil {
		ch, err := s.Leads.Stream(ctx)
		if err != nil {
			return err
		}
		go s.consumeLeads(ctx, ch)
	}
	go s.tick(ctx, adEvery, func() { s.SyncAds(ctx) })
	go s.tick(ctx, outcomeEvery, func() { s.SyncOutcomes(ctx, time.Time{}) })
	<-ctx.Done()
	return ctx.Err()
}

func (s *SyncService) tick(ctx context.Context, every time.Duration, fn func()) {
	if every <= 0 {
		return
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			fn()
		}
	}
}

// consumeLeads handles each inbound lead end-to-end.
func (s *SyncService) consumeLeads(ctx context.Context, ch <-chan LeadEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			s.HandleInbound(ctx, ev)
		}
	}
}

// HandleInbound is the per-lead hot path: featurise → decide → act.
func (s *SyncService) HandleInbound(ctx context.Context, ev LeadEvent) engine.LeadDecision {
	feats := domain.LeadFeatures{}
	if s.Featurizer != nil {
		feats = s.Featurizer(ev)
	}
	// The engine stores decision-time features durably (in its snapshot), so we
	// don't keep a parallel copy here.
	lead := domain.Lead{
		ID:        ev.ExternalID,
		ArmID:     ev.ArmID,
		ClinicID:  ev.ClinicID,
		CreatedAt: ev.ReceivedAt,
		Features:  feats,
		Status:    domain.LeadNew,
	}
	dec := s.Eng.HandleLead(lead, time.Now())

	if dec.Booked {
		// Write the appointment to the clinic's calendar. A persistent failure here
		// means the clinic never sees the booking — surface it, don't swallow it.
		if s.PMS != nil {
			if err := retry(3, func() error {
				return s.PMS.PushAppointment(ctx, domain.Appointment{
					ClinicID: dec.ClinicID, LeadID: dec.LeadID,
					When: dec.ApptTime, PShow: dec.PShow,
				})
			}); err != nil {
				log.Printf("DEADLETTER push appointment lead=%s clinic=%s: %v", dec.LeadID, dec.ClinicID, err)
			}
		}
		if s.Messenger != nil && ev.Phone != "" {
			// Business-initiated reminder → Meta-APPROVED WhatsApp template (the
			// intervention the no-show motor chose maps to an approved template).
			tmpl := whatsapp.ForIntervention(dec.Intervention)
			if err := retry(3, func() error {
				return s.Messenger.Send(ctx, ev.Phone, tmpl, map[string]string{
					"appt_time": dec.ApptTime.Format(time.RFC1123),
				})
			}); err != nil {
				log.Printf("DEADLETTER send reminder lead=%s phone=%s: %v", dec.LeadID, ev.Phone, err)
			}
		}
	}
	return dec
}

// SyncAds: pull ground-truth CPL, correct the bandit, PACE & push budgets.
func (s *SyncService) SyncAds(ctx context.Context) {
	if s.Ads == nil {
		return
	}
	spends, err := s.Ads.PullSpend(ctx, time.Now().Add(-24*time.Hour))
	spendByArm := map[string]ArmSpend{}
	if err != nil {
		log.Printf("sync ads: pull spend: %v", err)
	} else {
		for _, sp := range spends {
			s.Eng.Budget.CorrectCPL(sp.ArmID, sp.CostPerLead)
			spendByArm[sp.ArmID] = sp
		}
	}

	allocs, _ := s.Eng.PlanBudget(30)
	// Intraday PACING: scale each arm's daily budget by THAT ARM'S OWN PID pacer
	// (per-arm state — no cross-arm contamination) using how far through the day we
	// are vs how much of the budget is already spent.
	dayFrac := float64(time.Now().Hour()*60+time.Now().Minute()) / (24 * 60)
	perArm := make(map[string]float64, len(allocs))
	for _, a := range allocs {
		mult := 1.0
		if sp, ok := spendByArm[a.ArmID]; ok && a.DailyBudget > 0 {
			spendFrac := sp.Spend / a.DailyBudget
			mult = s.Eng.Pacers.Multiplier(a.ArmID, dayFrac, spendFrac)
		}
		perArm[a.ArmID] = a.DailyBudget * mult
	}
	if err := s.Ads.SetDailyBudgets(ctx, perArm); err != nil {
		log.Printf("sync ads: set budgets: %v", err)
	}
}

// SyncOutcomes pulls realised outcomes from every clinic's PMS and feeds each
// NEW one exactly once (dedup) into the learning loop, training the scorer on
// the lead's real decision-time features and the no-show model on the real show
// outcome, then uploads conversions to the ad platforms.
func (s *SyncService) SyncOutcomes(ctx context.Context, since time.Time) {
	if s.PMS == nil {
		return
	}
	var convs []Conversion
	for _, c := range s.Eng.Clinics() {
		outs, err := s.PMS.PullOutcomes(ctx, c.ID, since)
		if err != nil {
			log.Printf("sync outcomes: %s: %v", c.ID, err)
			continue
		}
		for _, o := range outs {
			// The engine dedups (durably, across restarts) and trains on the real
			// stored features. fresh==false means we've already ingested this one.
			fresh := s.Eng.IngestOutcome(o)
			if fresh && o.Closed != nil && *o.Closed {
				convs = append(convs, Conversion{
					ArmID: o.ArmID, ExternalID: o.LeadID, EventName: "closed",
					Value: o.Revenue, At: o.At,
				})
			}
		}
	}
	if s.Ads != nil && len(convs) > 0 {
		if err := s.Ads.UploadConversions(ctx, convs); err != nil {
			log.Printf("sync outcomes: upload conversions: %v", err)
		}
	}
}

// retry runs fn up to n times with a tiny backoff, returning the last error.
func retry(n int, fn func() error) error {
	var err error
	for i := 0; i < n; i++ {
		if err = fn(); err == nil {
			return nil
		}
	}
	return err
}
