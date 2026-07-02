// Package priors holds the brain's COLD-START knowledge: real, sourced 2025–2026
// industry benchmarks that seed every model before we have first-party data.
// These are aggregate public statistics (not personal data), so they're free to
// use and make the system useful on day one. As real outcomes arrive, the
// feedback loop blends these out — our own data always wins.
//
// Every number below carries its source. Update the figures as fresher reports
// land; the code reads from here, so there are no magic numbers in the motors.
//
// SOURCES (accessed 2026-06):
//
//	[CPL-G]  WordStream Google Ads Benchmarks 2026 — dental CPC ≈ $8.00,
//	         dental CPL ≈ $50–$95.  wordstream.com/blog/2026-google-ads-benchmarks
//	[CPL-F]  WordStream/LocaliQ Facebook Ads Benchmarks — dental CPL ≈ $76.71,
//	         CPC ≈ $9.78, CTR ≈ 1.05%.  localiq.com/blog/facebook-advertising-benchmarks
//	[CONV]   First Page Sage 2025 / InfluxMD / Unbounce — healthcare paid-search
//	         prospect→patient ≈ 64%, paid-social ≈ 66%; healthcare PPC lead
//	         conversion 2.4–11%; phone leads convert 25–40% vs 2% for web forms.
//	[NOSHOW] Henry Schein One 2024 Industry Report (dental no-show 7%→4%);
//	         broad practices 5–10%, cold new-patient leads ~20–30%.
//	[REM]    Klara 2025 (SMS reminders −38% no-shows); MDPI Appl.Sci. 2025
//	         (automated system 18.55%→7.01%); reminder-only ≈ −25%,
//	         confirmation-request ≈ −40–60%.
//	[PRICE]  Turkey dental tourism 2026 price guides (MedicalTourismCo,
//	         TurkeyTravelPlanner): single implant $300–$1,500; All-on-4
//	         $3,300–$6,600/arch; full veneer set $3,200–$8,000.
//	[SPEED]  Invoca/InfluxMD 2025 — optimal lead response <10 min; after 30 min
//	         leads are ~21× less likely to convert.
package priors

import "disci/brain/internal/domain"

// USDTRY is the exchange rate used to convert sourced USD prices/costs to TRY.
// Marked explicitly and kept in one place because the lira moves fast — update
// this single constant rather than every figure. (~mid-2026 estimate.)
const USDTRY = 41.0

// Funnel holds the four cold-start conversion rates for a segment:
// qualify (ad-lead → genuinely qualified), book (→ appointment),
// show (→ shows up, WITH standard reminders), close (→ treatment accepted).
type Funnel struct {
	Qualify, Book, Show, Close float64
}

// funnelBySegment is calibrated from [CONV] + [NOSHOW]. Aesthetic/tourism leads
// qualify less (more price-shoppers) but close at high value; implant leads are
// higher-intent; general checkups qualify easily but are low value.
var funnelBySegment = map[domain.Segment]Funnel{
	//                          qualify  book  show  close
	domain.SegmentAesthetic: {0.35, 0.45, 0.72, 0.30}, // tourism: long deliberation, high ticket
	domain.SegmentImplant:   {0.45, 0.55, 0.74, 0.40}, // high-intent, motivated by pain/function
	domain.SegmentOrtho:     {0.40, 0.50, 0.76, 0.45},
	domain.SegmentGeneral:   {0.55, 0.60, 0.78, 0.55}, // easy to qualify, low value
}

// FunnelFor returns the cold-start funnel for a segment.
func FunnelFor(seg domain.Segment) Funnel {
	if f, ok := funnelBySegment[seg]; ok {
		return f
	}
	return funnelBySegment[domain.SegmentGeneral]
}

// ticketUSD is the expected realised case value in USD, from [PRICE]. These are
// per-CASE (not per-tooth): an implant patient typically buys multiple units; an
// aesthetic patient buys a full veneer set.
var ticketUSD = map[domain.Segment]float64{
	domain.SegmentAesthetic: 5_400, // full veneer/smile set, mid of $3.2k–$8k
	domain.SegmentImplant:   1_100, // multi-unit implant case
	domain.SegmentOrtho:     1_500, // clear-aligner / ortho course
	domain.SegmentGeneral:   100,   // checkup + minor work
}

// TicketTRY returns the expected case value in TRY for a segment.
func TicketTRY(seg domain.Segment) float64 {
	if u, ok := ticketUSD[seg]; ok {
		return u * USDTRY
	}
	return ticketUSD[domain.SegmentGeneral] * USDTRY
}

// MarginRate is the share of the ticket the clinic keeps as margin (and thus the
// basis of what they'll pay us). Aesthetic carries the fattest margins.
var marginBySegment = map[domain.Segment]float64{
	domain.SegmentAesthetic: 0.55,
	domain.SegmentImplant:   0.45,
	domain.SegmentOrtho:     0.50,
	domain.SegmentGeneral:   0.35,
}

// MarginFor returns the margin rate for a segment.
func MarginFor(seg domain.Segment) float64 {
	if m, ok := marginBySegment[seg]; ok {
		return m
	}
	return marginBySegment[domain.SegmentGeneral]
}

// Audience distinguishes who we're advertising to — it dominates CPL. Selling to
// European dental-tourism patients costs Western CPLs; local Turkish patients are
// far cheaper per lead in TRY.
type Audience string

const (
	AudienceLocalTR Audience = "local_tr" // Turkish patients (e.g. Ümraniye implant)
	AudienceTourism Audience = "tourism"  // EU/UK patients (e.g. Nişantaşı aesthetic)
)

// CPLTRY returns the cold-start cost-per-lead in TRY for a platform × audience,
// from [CPL-G]/[CPL-F]. Tourism CPLs use the Western dental benchmarks directly;
// local-TR CPLs are a fraction of that (lower competition, lira-denominated).
func CPLTRY(plat domain.Platform, aud Audience) float64 {
	// Western benchmark CPLs (USD): Google ~$70 mid, Facebook ~$77.
	var usd float64
	switch plat {
	case domain.PlatformGoogle:
		usd = 70
	default: // Meta/Facebook
		usd = 77
	}
	full := usd * USDTRY
	if aud == AudienceTourism {
		return full
	}
	// Local Turkish dental CPLs run roughly an order of magnitude lower in TRY.
	if plat == domain.PlatformGoogle {
		return full * 0.14 // ~400 TRY
	}
	return full * 0.07 // ~220 TRY
}

// ReminderLift is the additive bump to P(show) from a no-show intervention,
// derived from [REM]. Mapping no-show reduction → show-prob lift assumes a base
// new-patient no-show around 25% (show ≈ 0.75):
//
//	reminder-only   ≈ −25% no-shows → show +0.06
//	confirmation    ≈ −45% no-shows → show +0.11
//	deposit         ≈ strongest commitment device → show +0.16
//	human/AI call   ≈ −38% no-shows → show +0.10
var reminderLift = map[string]float64{
	"enhanced_reminders": 0.06,
	"request_deposit":    0.16,
	"confirmation_call":  0.10,
	"standard_reminders": 0.0,
}

// ReminderLift returns the show-probability lift for a named intervention.
func ReminderLift(name string) float64 { return reminderLift[name] }

// BaseShowProb is the cold-start P(show) for a NEW-patient cold ad lead WITH
// standard reminders, from [NOSHOW]. Lower than established-patient figures
// because first-time ad leads no-show far more often.
const BaseShowProb = 0.72

// CPCTRY returns the cold-start cost-per-CLICK (not per-lead) in TRY for a
// platform × audience, from [CPL-G]/[CPL-F] (Google dental CPC ≈ $8.00, Facebook
// CPC ≈ $9.78). Used by the scenario engine to turn a budget into a click volume.
// Local-TR clicks run far cheaper than the Western tourism benchmark (same ratio
// as CPLTRY). CPC is the per-click cost; CPLTRY above is the downstream per-lead
// cost — they are different points in the funnel.
func CPCTRY(plat domain.Platform, aud Audience) float64 {
	var usd float64
	switch plat {
	case domain.PlatformGoogle:
		usd = 8.00 // [CPL-G]
	default: // Meta/Facebook
		usd = 9.78 // [CPL-F]
	}
	full := usd * USDTRY
	if aud == AudienceTourism {
		return full
	}
	// Local Turkish dental CPCs run roughly an order of magnitude lower in TRY
	// (lower competition, lira-denominated) — mirror the CPLTRY local scaling.
	if plat == domain.PlatformGoogle {
		return full * 0.14
	}
	return full * 0.07
}

// ClickToLead is the ad-click → captured-lead (someone who actually contacts the
// clinic) conversion, from [CONV] (healthcare PPC lead conversion 2.4–11%). This
// is the funnel step BEFORE the qualify/book/show rates in Funnel. The Low/High
// bound the Monte Carlo spread; ClickToLeadRate is the per-segment central value
// (higher-intent segments convert a click to contact more readily).
const (
	ClickToLeadLow  = 0.024
	ClickToLeadHigh = 0.11
)

var clickToLeadBySegment = map[domain.Segment]float64{
	domain.SegmentAesthetic: 0.05, // tourism: more price-shopping before contact
	domain.SegmentImplant:   0.07, // pain/function-driven, contacts readily
	domain.SegmentOrtho:     0.06,
	domain.SegmentGeneral:   0.08, // low-friction checkup enquiries
}

// ClickToLeadRate returns the cold-start click→lead rate for a segment.
func ClickToLeadRate(seg domain.Segment) float64 {
	if r, ok := clickToLeadBySegment[seg]; ok {
		return r
	}
	return clickToLeadBySegment[domain.SegmentGeneral]
}

// FastResponseSecs is the lead-response time below which conversion is maximised,
// from [SPEED]. The WhatsApp agent must reply within this window.
const FastResponseSecs = 600 // 10 minutes
