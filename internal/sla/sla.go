// Package sla implements the Guarantee Controller.
//
// We sell each clinic a monthly guarantee: at least N qualified appointments.
// That is an SLA, and we treat it exactly like an ad-delivery / pacing system
// treats a delivery commitment — as a chance constraint:
//
//	P( appointments_month[k] ≥ N_k ) ≥ serviceLevel
//
// The controller tracks each clinic's running deficit against a time-prorated
// target and converts it into a shadow price (Lagrange multiplier) λ_k ≥ 1. That
// single number is consumed by:
//   - the budget motor   (spend more on lagging clinics' arms), and
//   - the matching motor  (route more leads to lagging clinics),
//
// so the guarantee propagates as gentle, automatic priority pressure rather than
// manual fire-fighting. A clinic comfortably ahead gets λ≈1 (no distortion); a
// clinic dangerously behind gets a large λ that pulls resources toward it.
package sla

import (
	"fmt"
	"sync"
	"time"

	"disci/brain/internal/mathx"
)

// clinicState tracks progress toward one clinic's monthly guarantee.
type clinicState struct {
	guaranteed int
	delivered  int // qualified appointments delivered this month
}

// Controller is the guarantee tracker. Safe for concurrent use.
type Controller struct {
	mu           sync.RWMutex
	clinics      map[string]*clinicState
	maxLamda     float64
	serviceLevel float64 // target P(meet guarantee), e.g. 0.90
	lambdaGain   float64 // how hard the shadow price reacts to breach risk
	month        string  // "YYYY-MM" of the current accounting period
}

// NewController builds a guarantee controller for a target service level
// (P(meet monthly guarantee)). serviceLevel ≤ 0 defaults to 0.90.
func NewController(serviceLevel float64) *Controller {
	if serviceLevel <= 0 || serviceLevel >= 1 {
		serviceLevel = 0.90
	}
	return &Controller{
		clinics:      map[string]*clinicState{},
		maxLamda:     6.0,
		serviceLevel: serviceLevel,
		lambdaGain:   5.0,
	}
}

// Register sets a clinic's monthly guarantee.
func (c *Controller) Register(clinicID string, guaranteedPerMonth int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clinics[clinicID] = &clinicState{guaranteed: guaranteedPerMonth}
}

// RecordQualifiedAppt increments a clinic's delivered count. Called by the
// feedback loop when a lead reaches the "qualified appointment" milestone.
func (c *Controller) RecordQualifiedAppt(clinicID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if s, ok := c.clinics[clinicID]; ok {
		s.delivered++
	}
}

// ResetMonth zeroes delivered counts (call on the 1st of each month).
func (c *Controller) ResetMonth() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, s := range c.clinics {
		s.delivered = 0
	}
}

// MaybeReset zeroes delivered counts when the accounting month rolls over. The
// engine calls this on every lead so the guarantee window is always current —
// this is the caller the audit found missing.
func (c *Controller) MaybeReset(now time.Time) {
	key := fmt.Sprintf("%04d-%02d", now.Year(), int(now.Month()))
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.month == "" {
		c.month = key
		return
	}
	if c.month != key {
		c.month = key
		for _, s := range c.clinics {
			s.delivered = 0
		}
	}
}

// breachProb estimates P(final delivered < guaranteed) by month end. It models
// remaining qualified arrivals as Poisson with mean = expectedRemaining, where
// expectedRemaining blends the PLANNED rate (trusted early) with the OBSERVED
// rate (trusted as the month progresses). This avoids the early-month
// over-reaction a pure run-rate extrapolation suffers.
func (c *Controller) breachProb(s *clinicState, monthFrac float64) float64 {
	need := s.guaranteed - s.delivered
	if need <= 0 {
		return 0
	}
	rem := 1 - monthFrac
	planned := float64(s.guaranteed) * rem // arrivals if we hit plan from here
	observedRate := float64(s.delivered) / monthFrac
	observed := observedRate * rem
	// Confidence in the observed rate grows with elapsed fraction.
	expectedRemaining := monthFrac*observed + (1-monthFrac)*planned
	// P(Poisson(expectedRemaining) ≤ need-1) = P(fall short).
	return mathx.PoissonCDF(need-1, expectedRemaining)
}

// shadowPrice converts a clinic's CHANCE-CONSTRAINT breach risk into a
// multiplier ≥ 1. While P(breach) is within tolerance (1 − serviceLevel) the
// price is 1 (no distortion); past tolerance it rises toward maxLamda.
func (c *Controller) shadowPrice(s *clinicState, monthFrac float64) float64 {
	if s.guaranteed <= 0 {
		return 1
	}
	tol := 1 - c.serviceLevel
	pBreach := c.breachProb(s, monthFrac)
	if pBreach <= tol {
		return 1
	}
	excess := (pBreach - tol) / (1 - tol + 1e-9) // 0..1
	lambda := 1 + c.lambdaGain*excess
	if lambda > c.maxLamda {
		lambda = c.maxLamda
	}
	return lambda
}

// monthFraction returns how far through the current month `now` is, in [0,1].
func monthFraction(now time.Time) float64 {
	year, month, _ := now.Date()
	start := time.Date(year, month, 1, 0, 0, 0, 0, now.Location())
	next := start.AddDate(0, 1, 0)
	total := next.Sub(start).Seconds()
	elapsed := now.Sub(start).Seconds()
	f := elapsed / total
	if f < 1e-3 {
		return 1e-3
	}
	if f > 1 {
		return 1
	}
	return f
}

// BudgetBias implements budget.SLAProvider.
func (c *Controller) BudgetBias(clinicID string) float64 {
	return c.biasAt(clinicID, time.Now())
}

// biasAt is the testable core (injectable clock).
func (c *Controller) biasAt(clinicID string, now time.Time) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.clinics[clinicID]
	if !ok {
		return 1
	}
	return c.shadowPrice(s, monthFraction(now))
}

// MatchBias is the same shadow price exposed for the matching motor. Identical
// signal, different consumer — that's deliberate: one source of truth for "how
// badly does this clinic need patients right now".
func (c *Controller) MatchBias(clinicID string) float64 {
	return c.BudgetBias(clinicID)
}

// Status is a per-clinic guarantee health report for the dashboard.
type Status struct {
	ClinicID    string
	Guaranteed  int
	Delivered   int
	TargetNow   float64
	Deficit     float64
	ShadowPrice float64
	OnTrack     bool
}

// Report returns guarantee health for all clinics at the given time.
func (c *Controller) Report(now time.Time) []Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	mf := monthFraction(now)
	out := make([]Status, 0, len(c.clinics))
	for id, s := range c.clinics {
		target := float64(s.guaranteed) * mf
		deficit := target - float64(s.delivered)
		if deficit < 0 {
			deficit = 0
		}
		out = append(out, Status{
			ClinicID:    id,
			Guaranteed:  s.guaranteed,
			Delivered:   s.delivered,
			TargetNow:   target,
			Deficit:     deficit,
			ShadowPrice: c.shadowPrice(s, mf),
			OnTrack:     deficit <= 0,
		})
	}
	return out
}
