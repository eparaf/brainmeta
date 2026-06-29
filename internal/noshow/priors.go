package noshow

import "disci/brain/internal/priors"

// baseShowProb is the cold-start P(show) for a new-patient lead with standard
// reminders, from the sourced no-show benchmarks.
func baseShowProb() float64 { return priors.BaseShowProb }

// expectedLift is the additive bump to P(show) an intervention delivers, from
// the sourced reminder-effectiveness benchmarks (Klara 2025, MDPI 2025).
func expectedLift(i Intervention) float64 { return priors.ReminderLift(i.String()) }
