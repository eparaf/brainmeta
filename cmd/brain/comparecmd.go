package main

import (
	"fmt"
	"strings"

	"disci/brain/internal/config"
	"disci/brain/internal/sim"
)

// runCompare is the "prove it before you trust it with real spend" offline
// validation tool: it runs the SAME simulated hidden world through the brain's
// real bandit allocator and a naive equal-split baseline, averaged across many
// seeds (a single seed is noisy enough to flip which strategy looks better), and
// prints the ROAS each strategy achieved. Run this before deploying any change to
// the budget engine's tuning — it costs nothing and touches no live campaign.
func runCompare() {
	cfg := config.Default()
	seeds := []int64{1, 2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 42, 99, 101, 123, 555, 777}
	const days = 30

	fmt.Printf("=== BANDIT vs MANUEL (çevrimdışı doğrulama, %d gün, %d seed, sıfır harcama) ===\n\n", days, len(seeds))
	avg := sim.AverageComparison(cfg, seeds, days, nil)
	fmt.Printf("%-10s harcama %12s   gelir %12s   ROAS %6.2f\n", "Bandit", money(avg.Bandit.AdSpend), money(avg.Bandit.Revenue), avg.Bandit.ROAS)
	fmt.Printf("%-10s harcama %12s   gelir %12s   ROAS %6.2f\n", "Manuel", money(avg.Manual.AdSpend), money(avg.Manual.Revenue), avg.Manual.ROAS)
	fmt.Printf("\nOrtalama fark: bandit ROAS'ı manuelin %+.1f%% %s.\n", avg.UpliftPct, direction(avg.UpliftPct))

	fmt.Println("\n--- Tek-seed uyarısı: neden ortalama şart ---")
	single := sim.CompareBanditVsManual(cfg, 7, days, nil)
	fmt.Printf("seed=7 tek başına: bandit ROAS=%.2f  manuel ROAS=%.2f  (%+.1f%%)\n",
		single.Bandit.ROAS, single.Manual.ROAS, single.UpliftPct)
	fmt.Println("Tek seed yön değiştirebilir — bu yüzden gerçek bir karar öncesi çoklu-seed ortalaması kullanın.")
}

func direction(pct float64) string {
	if pct >= 0 {
		return "üzerinde"
	}
	return "altında"
}

// money formats a TRY amount with thousands separators, no decimals.
func money(x float64) string {
	s := fmt.Sprintf("%.0f", x)
	n := len(s)
	if n <= 3 {
		return s
	}
	var out strings.Builder
	for i, ch := range []byte(s) {
		if i > 0 && (n-i)%3 == 0 {
			out.WriteByte('.')
		}
		out.WriteByte(ch)
	}
	return out.String()
}
