// Package sim is a ground-truth world model that exercises the whole brain
// end-to-end. It invents clinics and a hidden reality (true conversion rates per
// arm, true latent qualify/book/show/close behaviour per lead), streams leads at
// the engine day by day, feeds realised outcomes back into the learning loop,
// reallocates budget daily, and reports whether the brain (a) learns which ads
// work, (b) hits each clinic's guarantee, and (c) keeps no-shows under control.
//
// This is how we prove the brain "really works" before a single lira of ad spend
// touches a real campaign.
package sim

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"disci/brain/internal/budget"
	"disci/brain/internal/domain"
	"disci/brain/internal/engine"
	"disci/brain/internal/noshow"
	"disci/brain/internal/priors"
	"disci/brain/internal/store"
)

// World is the hidden ground truth the brain does NOT get to see directly.
type World struct {
	rng *rand.Rand

	// trueTheta[armID] = real P(qualified appointment | lead) for that arm.
	trueTheta map[string]float64
	// trueCPL[armID] = real cost per delivered lead.
	trueCPL map[string]float64
	// trueShow[clinicID] base show propensity (clinic neighbourhood effect).
	trueShow map[string]float64
	// trueClose[clinicID] real close rate.
	trueClose map[string]float64
}

// Result is the simulation summary.
type Result struct {
	Days          int
	LeadsHandled  int
	Booked        int
	Showed        int
	Closed        int
	Revenue       float64
	AdSpend       float64
	PerClinic     map[string]*ClinicResult
	FinalArmStats []map[string]any
}

type ClinicResult struct {
	Name              string
	Guaranteed        int
	QualifiedAppts    int
	Showed            int
	Closed            int
	Revenue           float64
	GuaranteeMet      bool
}

// Setup builds a realistic 4-clinic Istanbul network and a matching hidden
// world. Two clinics "advertise well" (high trueTheta), two poorly — the brain
// must discover this and shift budget without being told.
func Setup(eng *engine.Engine, seed int64) *World {
	w := &World{
		rng:       rand.New(rand.NewSource(seed + 7)),
		trueTheta: map[string]float64{},
		trueCPL:   map[string]float64{},
		trueShow:  map[string]float64{},
		trueClose: map[string]float64{},
	}

	clinics := []domain.Clinic{
		{ID: "nisantasi", Name: "Nişantaşı Estetik", District: "Nişantaşı", Side: "europe",
			Segment: domain.SegmentAesthetic, MarginRate: 0.55, CloseRate: 0.32,
			DailyCapacity: 4, GuaranteedApptsPerMonth: 18, MonthlyAdBudget: 300_000,
			LatX: 28.99, LatY: 41.05},
		{ID: "umraniye", Name: "Ümraniye İmplant", District: "Ümraniye", Side: "asia",
			Segment: domain.SegmentImplant, MarginRate: 0.45, CloseRate: 0.42,
			DailyCapacity: 10, GuaranteedApptsPerMonth: 80, MonthlyAdBudget: 220_000,
			LatX: 29.12, LatY: 41.02},
		{ID: "kadikoy", Name: "Kadıköy Diş", District: "Kadıköy", Side: "asia",
			Segment: domain.SegmentOrtho, MarginRate: 0.50, CloseRate: 0.38,
			DailyCapacity: 7, GuaranteedApptsPerMonth: 45, MonthlyAdBudget: 150_000,
			LatX: 29.03, LatY: 40.99},
		{ID: "sisli", Name: "Şişli Aile Diş", District: "Şişli", Side: "europe",
			Segment: domain.SegmentGeneral, MarginRate: 0.35, CloseRate: 0.45,
			DailyCapacity: 9, GuaranteedApptsPerMonth: 48, MonthlyAdBudget: 90_000,
			LatX: 28.98, LatY: 41.06},
	}

	// Hidden truth: real per-lead qualify rate for each (clinic, platform) arm.
	// The brain does NOT see these — it must discover them. nisantasi & umraniye
	// advertise well; kadikoy is mediocre; sisli has a decent Meta arm but a
	// genuinely weak Google arm the brain should starve.
	goodBad := map[string][2]float64{ // [metaTheta, googleTheta]
		"nisantasi": {0.24, 0.16},
		"umraniye":  {0.30, 0.22},
		"kadikoy":   {0.20, 0.14},
		"sisli":     {0.22, 0.06},
	}

	for _, c := range clinics {
		eng.RegisterClinic(c)
		w.trueShow[c.ID] = clamp01(0.62 + w.rng.NormFloat64()*0.05)
		w.trueClose[c.ID] = c.CloseRate
		gb := goodBad[c.ID]
		aud := priors.AudienceLocalTR
		if c.Segment == domain.SegmentAesthetic {
			aud = priors.AudienceTourism // Nişantaşı sells to EU dental tourists
		}
		for i, plat := range []domain.Platform{domain.PlatformMeta, domain.PlatformGoogle} {
			armID := fmt.Sprintf("%s:%s:%s", c.ID, plat, c.Segment)
			w.trueTheta[armID] = gb[i]
			// Real sourced CPL (TRY) for this platform × audience, with noise.
			w.trueCPL[armID] = priors.CPLTRY(plat, aud) * (0.9 + 0.2*w.rng.Float64())
		}
	}
	return w
}

// Run simulates `days` of operation. Each day it: allocates budget, draws leads
// per arm in proportion to that arm's funded budget / true CPL, handles each
// lead through the engine, then resolves outcomes (book/show/close) against the
// hidden world and feeds them back.
func (w *World) Run(eng *engine.Engine, st *store.Memory, days int) *Result {
	res := &Result{Days: days, PerClinic: map[string]*ClinicResult{}}
	for _, c := range st.ListClinics() {
		res.PerClinic[c.ID] = &ClinicResult{Name: c.Name, Guaranteed: c.GuaranteedApptsPerMonth}
	}

	base := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	leadSeq := 0

	for day := 0; day < days; day++ {
		now := base.AddDate(0, 0, day)
		st.ResetSeats()

		allocs, _ := eng.PlanBudget(30)

		for _, a := range allocs {
			if a.DailyBudget <= 0 {
				continue
			}
			cpl := w.trueCPL[a.ArmID]
			if cpl <= 0 {
				cpl = 50
			}
			// Number of leads this arm delivers today for its funded budget.
			nLeads := int(a.DailyBudget / cpl)
			theta := w.trueTheta[a.ArmID]
			clinicID := clinicOfArm(a.ArmID)
			clinic, _ := st.GetClinic(clinicID)

			for i := 0; i < nLeads; i++ {
				leadSeq++
				res.AdSpend += cpl
				lead := w.makeLead(leadSeq, a.ArmID, clinic, theta, now)
				dec := eng.HandleLead(lead, now)
				res.LeadsHandled++

				w.resolveOutcome(eng, st, lead, dec, clinic, res, now)
			}
		}
	}

	res.FinalArmStats = eng.Budget.Snapshot()
	for _, c := range st.ListClinics() {
		cr := res.PerClinic[c.ID]
		cr.GuaranteeMet = cr.QualifiedAppts >= c.GuaranteedApptsPerMonth
	}
	return res
}

// makeLead synthesises a lead whose features correlate with the arm's hidden
// quality, so the scorer has real signal to learn.
func (w *World) makeLead(seq int, armID string, clinic domain.Clinic, theta float64, now time.Time) domain.Lead {
	// Higher-theta arms tend to deliver higher-intent leads.
	intent := clamp01(theta*2 + w.rng.NormFloat64()*0.15)
	return domain.Lead{
		ID:       fmt.Sprintf("lead-%d", seq),
		ArmID:    armID,
		ClinicID: clinic.ID, // single-tenant regime: arm belongs to a clinic
		Segment:  clinic.Segment,
		Platform: platformOfArm(armID),
		CreatedAt: now,
		Status:    domain.LeadNew,
		Features: domain.LeadFeatures{
			FirstResponseSecs: 10 + w.rng.Float64()*120,
			MessagesExchanged: 2 + w.rng.Float64()*8,
			DistanceKm:        w.rng.Float64() * 35,
			HourOfDay:         float64(9 + w.rng.Intn(11)),
			StatedBudgetTRY:   maybeBudget(w.rng, clinic.Segment),
			UrgencyScore:      clamp01(intent + w.rng.NormFloat64()*0.1),
			PastNoShows:       float64(w.rng.Intn(3)),
			IntentScore:       intent,
		},
	}
}

// resolveOutcome plays the lead forward against hidden reality and feeds the
// realised result back into the learning loop.
func (w *World) resolveOutcome(eng *engine.Engine, st *store.Memory, lead domain.Lead, dec engine.LeadDecision, clinic domain.Clinic, res *Result, now time.Time) {
	cr := res.PerClinic[clinic.ID]
	theta := w.trueTheta[lead.ArmID]

	// --- Hidden patient reality (the world), independent of our operations. ---
	// Whether the lead is a genuinely qualified prospect is governed by the arm's
	// hidden quality, modulated by the lead's realised intent. This is observable
	// from the WhatsApp qualification chat for EVERY engaged lead, so we always
	// feed it as a training label.
	qualified := w.rng.Float64() < clamp01(theta*(0.7+0.6*lead.Features.IntentScore))

	o := domain.Outcome{
		LeadID: lead.ID, ClinicID: clinic.ID, ArmID: lead.ArmID,
		Segment: clinic.Segment, At: now,
		AdCost: w.trueCPL[lead.ArmID],
	}
	q := qualified
	o.Qualified = &q

	// --- Operational path: we only observe downstream stages for leads we
	// actually offered an appointment to (dec.Booked). Gated / capacity-rejected
	// leads contribute only the qualification label — exactly the observability
	// you have in production. ---
	if dec.Booked {
		// A qualified prospect accepts the offered slot with high probability;
		// an unqualified one rarely follows through.
		accepts := (qualified && w.rng.Float64() < 0.9) || (!qualified && w.rng.Float64() < 0.15)
		o.Booked = &accepts

		if accepts {
			res.Booked++
			if qualified {
				cr.QualifiedAppts++ // the unit we guarantee: a qualified appointment
			}

			// Show? Hidden neighbourhood propensity blended with the lift from the
			// intervention the brain chose (already baked into dec.PShow).
			showProb := clamp01(w.trueShow[clinic.ID]*0.6 + dec.PShow*0.4)
			showed := w.rng.Float64() < showProb
			o.Showed = &showed

			apptF := noshow.Appt{
				LeadTimeDays: 2, HourOfDay: lead.Features.HourOfDay,
				PastNoShows: lead.Features.PastNoShows, ConfirmedReply: true,
				DistanceKm: lead.Features.DistanceKm, Segment: clinic.Segment,
			}
			eng.Loop.IngestShow(apptF, showed)

			if showed {
				cr.Showed++
				res.Showed++
				closed := w.rng.Float64() < w.trueClose[clinic.ID]
				o.Closed = &closed
				if closed {
					rev := priors.TicketTRY(clinic.Segment) * (0.7 + 0.6*w.rng.Float64())
					o.Revenue = rev
					cr.Closed++
					cr.Revenue += rev
					res.Closed++
					res.Revenue += rev
				}
			}
		}
	}

	eng.Loop.Ingest(o, lead.Features)
}

// ---- helpers -------------------------------------------------------------

func clinicOfArm(armID string) string  { return strings.SplitN(armID, ":", 2)[0] }
func platformOfArm(armID string) domain.Platform {
	parts := strings.Split(armID, ":")
	if len(parts) >= 2 {
		return domain.Platform(parts[1])
	}
	return domain.PlatformMeta
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func maybeBudget(rng *rand.Rand, seg domain.Segment) float64 {
	if rng.Float64() < 0.4 {
		return seg.AvgTicket() * (0.5 + rng.Float64())
	}
	return 0
}

// FormatReport renders a human-readable simulation summary.
func FormatReport(r *Result, arms []budget.Allocation) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n=== BRAIN SIMULATION — %d days ===\n\n", r.Days)
	fmt.Fprintf(&b, "Leads handled : %d\n", r.LeadsHandled)
	fmt.Fprintf(&b, "Booked appts  : %d\n", r.Booked)
	fmt.Fprintf(&b, "Showed up     : %d  (%.0f%% show rate)\n", r.Showed, pct(r.Showed, r.Booked))
	fmt.Fprintf(&b, "Closed deals  : %d\n", r.Closed)
	fmt.Fprintf(&b, "Revenue       : %s TRY\n", money(r.Revenue))
	fmt.Fprintf(&b, "Ad spend      : %s TRY\n", money(r.AdSpend))
	fmt.Fprintf(&b, "ROAS          : %.1fx\n\n", safeDiv(r.Revenue, r.AdSpend))

	fmt.Fprintf(&b, "--- Per-clinic guarantee (qualified appointments delivered) ---\n")
	for _, cr := range r.PerClinic {
		ach := pct(cr.QualifiedAppts, cr.Guaranteed)
		status := "✗ short"
		switch {
		case ach >= 100:
			status = "✓ MET"
		case ach >= 90:
			status = "≈ near"
		}
		fmt.Fprintf(&b, "%-22s appts %3d / %3d (%3.0f%%)  show %3d  closed %3d  rev %-11s  %s\n",
			cr.Name, cr.QualifiedAppts, cr.Guaranteed, ach, cr.Showed, cr.Closed, money(cr.Revenue), status)
	}

	fmt.Fprintf(&b, "\n--- What the brain learned about each ad arm (θ̂ = estimated quality) ---\n")
	for _, a := range r.FinalArmStats {
		fmt.Fprintf(&b, "%-28s θ̂=%.3f  CPL=%.0f  leads=%v  appts=%v  spend=%s\n",
			a["armId"], a["thetaHat"], a["cpl"], a["leads"], a["appts"], money(toF(a["spend"])))
	}
	return b.String()
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}
func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}
func toF(v any) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}
func money(x float64) string {
	s := fmt.Sprintf("%.0f", x)
	// thousands separators
	n := len(s)
	if n <= 3 {
		return s
	}
	var out []byte
	for i, ch := range []byte(s) {
		if i > 0 && (n-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, ch)
	}
	return string(out)
}
