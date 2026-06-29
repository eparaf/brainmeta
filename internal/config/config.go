// Package config centralises the brain's tunable policy knobs. Keeping them in
// one place (rather than scattered as magic numbers) makes the system auditable
// and lets the offline trainer ship new policies as config, not code.
package config

// Config holds network-wide policy parameters.
type Config struct {
	// MinEVToBook is the expected-value floor (TRY) below which a lead is not
	// worth a scarce appointment slot. Protects capacity for high-value leads.
	MinEVToBook float64

	// MinPQualifyToBook gates obviously-unqualified leads regardless of EV.
	MinPQualifyToBook float64

	// MaxOverbookRisk is the acceptable probability of more arrivals than seats
	// in the overbooking solver.
	MaxOverbookRisk float64

	// ServiceLevel is the target P(meet monthly guarantee) for each clinic.
	ServiceLevel float64

	// Seed makes the whole system deterministic for tests/sim.
	Seed int64
}

// Default returns sensible production starting values.
func Default() Config {
	return Config{
		MinEVToBook:       150,
		MinPQualifyToBook: 0.12,
		MaxOverbookRisk:   0.15,
		ServiceLevel:      0.90,
		Seed:              42,
	}
}
