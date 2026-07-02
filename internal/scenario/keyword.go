// Package scenario is the OFFLINE planning layer: "with this budget, how many
// appointments should we expect per month?" It runs a Monte-Carlo simulation over
// the funnel priors and (eventually) real Keyword Planner data, and returns a
// pessimistic / realistic / optimistic band. It spends no money and needs no API
// key — the math is deterministic and the LLM is not involved. It sits ALONGSIDE
// the live decision motors and never touches the brain's learned posteriors.
package scenario

import (
	"disci/brain/internal/domain"
	"disci/brain/internal/priors"
)

// KeywordMetrics is one keyword's search-side economics — the shape Google Ads
// Keyword Planner returns. In Phase A these are synthesised from priors; in Phase
// B a real googleads KeywordPlanIdeaService client fills the same struct.
type KeywordMetrics struct {
	Keyword          string  `json:"keyword"`
	MonthlySearches  int     `json:"monthlySearches"`  // avg monthly search volume
	CompetitionIndex float64 `json:"competitionIndex"` // 0..1 (Keyword Planner LOW/MED/HIGH → 0.25/0.5/0.85)
	CPCLowTRY        float64 `json:"cpcLowTRY"`        // top-of-page bid low estimate
	CPCHighTRY       float64 `json:"cpcHighTRY"`       // top-of-page bid high estimate
}

// KeywordSource supplies keyword economics for a segment × audience. The scenario
// engine depends only on this interface, so the synthetic prior source (now) and
// a live Keyword Planner client (Phase B) are interchangeable — the same
// datasource-style seam the rest of the codebase uses.
type KeywordSource interface {
	Keywords(seg domain.Segment, aud priors.Audience) ([]KeywordMetrics, error)
}

// PriorKeywordSource is the COLD-START (Phase A) keyword source: it synthesises a
// small, plausible keyword set from internal/priors so the scenario engine runs
// end-to-end with zero credentials. These volumes are placeholder estimates, NOT
// real Keyword Planner data — swap this for the live source once a Google Ads test
// account is connected. CPC low/high bracket the priors central CPC by ±35%.
type PriorKeywordSource struct {
	Platform domain.Platform // which platform's CPC benchmark to use (default Google)
}

// searchVolumeBySegment is a rough monthly-search placeholder per segment for the
// clinic's local market. Local implant/general terms search more than niche
// tourism-aesthetic terms. Replaced wholesale by real Keyword Planner volume in
// Phase B; kept here (not in priors) precisely because it is NOT a sourced figure.
var searchVolumeBySegment = map[domain.Segment]int{
	domain.SegmentAesthetic: 8_000, // international tourism demand pool (veneers/smile-design)
	domain.SegmentImplant:   9_000,
	domain.SegmentOrtho:     5_000,
	domain.SegmentGeneral:   14_000,
}

// keywordSeeds gives each segment a couple of representative query stems so the
// output reads like real Keyword Planner rows.
var keywordSeeds = map[domain.Segment][]string{
	domain.SegmentAesthetic: {"gülüş tasarımı", "diş kaplama fiyat", "veneers turkey"},
	domain.SegmentImplant:   {"implant fiyatları", "diş implantı", "dental implant istanbul"},
	domain.SegmentOrtho:     {"şeffaf plak", "ortodonti fiyat", "diş teli"},
	domain.SegmentGeneral:   {"diş hekimi", "diş kontrolü", "diş temizliği fiyat"},
}

// Keywords synthesises a keyword set for the segment. Volume is split across the
// segment's seed terms; CPC low/high bracket the priors central CPC.
func (p PriorKeywordSource) Keywords(seg domain.Segment, aud priors.Audience) ([]KeywordMetrics, error) {
	plat := p.Platform
	if plat == "" {
		plat = domain.PlatformGoogle
	}
	cpc := priors.CPCTRY(plat, aud)
	seeds := keywordSeeds[seg]
	if len(seeds) == 0 {
		seeds = keywordSeeds[domain.SegmentGeneral]
	}
	total := searchVolumeBySegment[seg]
	if total == 0 {
		total = searchVolumeBySegment[domain.SegmentGeneral]
	}
	// Tourism terms are searched abroad (a DIFFERENT, not smaller, demand pool) and
	// cost Western CPCs — so competition is higher but volume is not reduced. CPC is
	// already lifted via priors.CPCTRY(_, AudienceTourism) above.
	comp := 0.5
	if aud == priors.AudienceTourism {
		comp = 0.85
	}
	per := total / len(seeds)
	out := make([]KeywordMetrics, 0, len(seeds))
	for _, kw := range seeds {
		out = append(out, KeywordMetrics{
			Keyword:          kw,
			MonthlySearches:  per,
			CompetitionIndex: comp,
			CPCLowTRY:        cpc * 0.65,
			CPCHighTRY:       cpc * 1.35,
		})
	}
	return out, nil
}
