// Package matching implements Motor 3 — the Lead→Clinic Routing Engine.
//
// In the marketplace regime (the long-term vision: "get the right patient to the
// right clinic"), leads land in a shared pool and the brain decides which clinic
// each one goes to. This is a constrained assignment problem:
//
//	maximize  Σ_ij x_ij · value(lead_i, clinic_j)
//	s.t.      Σ_j x_ij ≤ 1            (each lead to at most one clinic)
//	          Σ_i x_ij ≤ capacity_j  (clinic seat limits)
//	          x_ij = 0 if incompatible (geography / treatment mismatch)
//
// We solve it as a balanced linear assignment problem with the Hungarian
// (Kuhn–Munkres) algorithm. A clinic with capacity C is expanded into C "seats",
// so the bipartite graph is leads × seats. The objective value blends the lead's
// EV with segment fit, distance, the clinic's close-rate, and the clinic's SLA
// shadow price (clinics behind on their guarantee get priority).
package matching

import (
	"math"

	"disci/brain/internal/domain"
)

// Candidate is a lead awaiting routing, with its precomputed EV.
type Candidate struct {
	Lead  domain.Lead
	Score domain.LeadScore
}

// ClinicSlot describes a clinic's routing parameters for this cycle.
type ClinicSlot struct {
	Clinic    domain.Clinic
	FreeSeats int     // remaining new-patient capacity this cycle
	SLABias   float64 // ≥1, from the guarantee controller
}

// Assignment is the routing decision for one lead.
type Assignment struct {
	LeadID   string
	ClinicID string
	Value    float64
	Routed   bool
}

// valueOf computes the objective weight of putting a lead at a clinic. Returns
// (value, compatible). Incompatible pairs are excluded from the assignment.
func valueOf(c Candidate, s ClinicSlot) (float64, bool) {
	clinic := s.Clinic

	// Hard compatibility: a premium-aesthetic-only clinic shouldn't get a pure
	// general-checkup lead, and vice-versa, unless segments are adjacent.
	if !segmentsCompatible(c.Lead.Segment, clinic.Segment) {
		return 0, false
	}
	// Hard distance gate: don't route a patient absurdly far.
	if c.Lead.Features.DistanceKm > 60 {
		return 0, false
	}

	base := c.Score.EV
	// Clinic close-rate scales the realisable value (clinic-side skill matters).
	base *= normalizeCloseRate(clinic.CloseRate)
	// Distance decay.
	base *= math.Exp(-c.Lead.Features.DistanceKm / 25.0)
	// Segment-fit bonus.
	if c.Lead.Segment == clinic.Segment {
		base *= 1.15
	}
	// SLA priority: scale up value for clinics behind on their guarantee.
	base *= s.SLABias

	return base, true
}

func normalizeCloseRate(r float64) float64 {
	if r <= 0 {
		return 0.4
	}
	if r > 1 {
		return 1
	}
	return r
}

func segmentsCompatible(lead, clinic domain.Segment) bool {
	if lead == clinic {
		return true
	}
	// Adjacency: implant clinics can absorb general/ortho; aesthetic stays strict.
	switch clinic {
	case domain.SegmentImplant, domain.SegmentOrtho, domain.SegmentGeneral:
		return lead != domain.SegmentAesthetic
	case domain.SegmentAesthetic:
		return lead == domain.SegmentAesthetic
	}
	return true
}

// Route assigns candidates to clinic seats to maximise total value. It returns
// one Assignment per candidate (Routed=false if the lead couldn't be placed).
func Route(cands []Candidate, slots []ClinicSlot) []Assignment {
	// Expand clinics into seats.
	type seat struct {
		clinicIdx int
		clinicID  string
	}
	var seats []seat
	for ci, s := range slots {
		for k := 0; k < s.FreeSeats; k++ {
			seats = append(seats, seat{clinicIdx: ci, clinicID: s.Clinic.ID})
		}
	}

	n := len(cands)
	m := len(seats)
	results := make([]Assignment, n)
	for i := range cands {
		results[i] = Assignment{LeadID: cands[i].Lead.ID}
	}
	if n == 0 || m == 0 {
		return results
	}

	// Build a square cost matrix for Hungarian (minimisation). We minimise
	// negative value. Incompatible pairs get a large positive cost (never chosen
	// unless forced; we post-filter those out).
	const incompatible = 1e9
	size := n
	if m > size {
		size = m
	}
	cost := make([][]float64, size)
	compat := make([][]bool, size)
	for i := 0; i < size; i++ {
		cost[i] = make([]float64, size)
		compat[i] = make([]bool, size)
		for j := 0; j < size; j++ {
			if i < n && j < m {
				v, ok := valueOf(cands[i], slots[seats[j].clinicIdx])
				if ok {
					cost[i][j] = -v
					compat[i][j] = true
				} else {
					cost[i][j] = incompatible
				}
			} else {
				cost[i][j] = 0 // dummy padding rows/cols
			}
		}
	}

	assign := hungarian(cost)
	for i := 0; i < n; i++ {
		j := assign[i]
		if j < 0 || j >= m || !compat[i][j] {
			continue // unmatched or matched to padding/incompatible
		}
		s := slots[seats[j].clinicIdx]
		v, _ := valueOf(cands[i], s)
		results[i] = Assignment{
			LeadID:   cands[i].Lead.ID,
			ClinicID: seats[j].clinicID,
			Value:    v,
			Routed:   true,
		}
	}
	return results
}

// hungarian solves the square assignment problem (minimisation) via the O(n^3)
// Kuhn–Munkres algorithm with potentials. Returns assign[i] = column for row i.
func hungarian(cost [][]float64) []int {
	n := len(cost)
	const inf = math.MaxFloat64

	u := make([]float64, n+1)
	v := make([]float64, n+1)
	p := make([]int, n+1) // p[j] = row assigned to column j (1-indexed rows)
	way := make([]int, n+1)

	for i := 1; i <= n; i++ {
		p[0] = i
		j0 := 0
		minv := make([]float64, n+1)
		used := make([]bool, n+1)
		for j := 0; j <= n; j++ {
			minv[j] = inf
		}
		for {
			used[j0] = true
			i0 := p[j0]
			delta := inf
			j1 := -1
			for j := 1; j <= n; j++ {
				if used[j] {
					continue
				}
				cur := cost[i0-1][j-1] - u[i0] - v[j]
				if cur < minv[j] {
					minv[j] = cur
					way[j] = j0
				}
				if minv[j] < delta {
					delta = minv[j]
					j1 = j
				}
			}
			for j := 0; j <= n; j++ {
				if used[j] {
					u[p[j]] += delta
					v[j] -= delta
				} else {
					minv[j] -= delta
				}
			}
			j0 = j1
			if p[j0] == 0 {
				break
			}
		}
		for {
			j1 := way[j0]
			p[j0] = p[j1]
			j0 = j1
			if j0 == 0 {
				break
			}
		}
	}

	assign := make([]int, n)
	for i := range assign {
		assign[i] = -1
	}
	for j := 1; j <= n; j++ {
		if p[j] >= 1 && p[j] <= n {
			assign[p[j]-1] = j - 1
		}
	}
	return assign
}
