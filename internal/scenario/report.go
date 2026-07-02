package scenario

import (
	"fmt"
	"strings"
)

// FormatReport renders a scenario Result as a human-readable console block. Used
// by `brain scenario`. Appointment/lead bands are labelled pessimistic/realistic/
// optimistic (P10/P50/P90); cost bands invert (P90 = pricier).
func FormatReport(plan CampaignPlan, r Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "=== SENARYO: %s / %s / %s ===\n", plan.Segment, plan.Platform, plan.Audience)
	fmt.Fprintf(&b, "Aylık bütçe: %s TRY   •   %d Monte-Carlo koşusu   •   arama hacmi: %d/ay\n",
		money(plan.MonthlyBudget), r.Runs, r.Assumptions.SearchVol)
	fmt.Fprintf(&b, "Varsayımlar: CTR/funnel prior'ları (qualify %.0f%% · book %.0f%% · show %.0f%%), CPC≈%s TRY, click→lead %.1f%%\n",
		r.Assumptions.Funnel.Qualify*100, r.Assumptions.Funnel.Book*100, r.Assumptions.Funnel.Show*100,
		money(r.Assumptions.AvgCPCTRY), r.Assumptions.ClickToLead*100)
	b.WriteString("\n")
	fmt.Fprintf(&b, "%-24s %12s %12s %12s\n", "", "PESİMİST", "GERÇEKÇİ", "OPTİMİST")
	fmt.Fprintf(&b, "%-24s %12s %12s %12s\n", "", "(P10)", "(P50)", "(P90)")
	row := func(label string, band Band) {
		fmt.Fprintf(&b, "%-24s %12.1f %12.1f %12.1f\n", label, band.P10, band.P50, band.P90)
	}
	row("Tıklama / ay", r.Clicks)
	row("Nitelikli lead / ay", r.QualifiedLeads)
	row("Randevu (booked) / ay", r.BookedAppointments)
	row("Gelen randevu (kept)/ay", r.KeptAppointments)
	b.WriteString("\n")
	// Cost rows invert: cheap = optimistic.
	fmt.Fprintf(&b, "%-24s %12s %12s %12s\n", "Maliyet", "UCUZ(P10)", "ORTA(P50)", "PAHALI(P90)")
	fmt.Fprintf(&b, "%-24s %12s %12s %12s\n", "Lead başı (TRY)",
		money(r.CostPerLead.P10), money(r.CostPerLead.P50), money(r.CostPerLead.P90))
	fmt.Fprintf(&b, "%-24s %12s %12s %12s\n", "Randevu başı (TRY)",
		money(r.CostPerAppointment.P10), money(r.CostPerAppointment.P50), money(r.CostPerAppointment.P90))
	b.WriteString("\n")
	fmt.Fprintf(&b, "SONUÇ: Bu bütçeyle ayda ~%.0f–%.0f randevu beklenir (gerçekçi: %.0f).\n",
		r.BookedAppointments.P10, r.BookedAppointments.P90, r.BookedAppointments.P50)
	return b.String()
}

// money formats a TRY amount with thousands separators, no decimals.
func money(x float64) string {
	neg := x < 0
	if neg {
		x = -x
	}
	s := fmt.Sprintf("%.0f", x)
	// insert dots every 3 digits from the right
	n := len(s)
	if n <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	var out strings.Builder
	pre := n % 3
	if pre > 0 {
		out.WriteString(s[:pre])
		if n > pre {
			out.WriteString(".")
		}
	}
	for i := pre; i < n; i += 3 {
		out.WriteString(s[i : i+3])
		if i+3 < n {
			out.WriteString(".")
		}
	}
	if neg {
		return "-" + out.String()
	}
	return out.String()
}
