package budget

import (
	"testing"

	"disci/brain/internal/domain"
)

// TestThompsonShiftsBudgetToWinner verifies the core promise: given two arms
// where one truly converts much better, the bandit learns to send it the budget
// without being told which is which.
func TestThompsonShiftsBudgetToWinner(t *testing.T) {
	e := NewEngine(1)
	good := domain.Arm{ID: "good", ClinicID: "c1", Platform: domain.PlatformMeta, AvgCostPerLead: 40}
	bad := domain.Arm{ID: "bad", ClinicID: "c1", Platform: domain.PlatformMeta, AvgCostPerLead: 40}
	e.RegisterArm(good)
	e.RegisterArm(bad)

	// Simulate 400 leads per arm: good converts 30%, bad 5%.
	for i := 0; i < 400; i++ {
		e.Observe("good", i%10 < 3, 40)
		e.Observe("bad", i%20 < 1, 40)
	}

	// Budget can't fully fund both arms, so the allocator must prioritise the
	// better one — that's the decision we're testing.
	allocs, lambda := e.Allocate(map[string]float64{"c1": 5000}, nil)
	var goodB, badB float64
	for _, a := range allocs {
		switch a.ArmID {
		case "good":
			goodB = a.DailyBudget
		case "bad":
			badB = a.DailyBudget
		}
	}
	if goodB <= badB {
		t.Fatalf("expected good arm to win budget: good=%.0f bad=%.0f", goodB, badB)
	}
	if lambda <= 0 {
		t.Fatalf("expected positive shadow price, got %v", lambda)
	}
}

// TestPacerCorrectsUnderspend checks the PID pacer speeds up when behind.
func TestPacerCorrectsUnderspend(t *testing.T) {
	p := NewPacer()
	// Half the day gone but only 10% spent -> should push multiplier > 1.
	m := p.Multiplier(0.5, 0.1)
	if m <= 1.0 {
		t.Fatalf("expected pacer to accelerate, got %v", m)
	}
}

// TestPacerSetIsolatesArms is the regression test for the shared-Pacer bug: each
// arm must keep its own PID state, so driving one arm doesn't contaminate another.
func TestPacerSetIsolatesArms(t *testing.T) {
	ps := NewPacerSet()
	// Hammer arm A as severely behind several times (winds up its PID integral).
	for i := 0; i < 5; i++ {
		ps.Multiplier("A", 0.9, 0.0)
	}
	// Arm B is exactly on pace — with isolated state it should be ~1.0, not
	// dragged up by A's wound-up integral.
	mB := ps.Multiplier("B", 0.5, 0.5)
	if mB > 1.05 || mB < 0.95 {
		t.Fatalf("arm B pacing contaminated by arm A: got %v (want ~1.0)", mB)
	}
}

// TestPerClinicBudgetIsolation verifies the passthrough invariant: each clinic's
// budget funds only that clinic's arms — clinic A's money never reaches clinic B.
func TestPerClinicBudgetIsolation(t *testing.T) {
	e := NewEngine(3)
	e.RegisterArm(domain.Arm{ID: "a1", ClinicID: "clinicA", AvgCostPerLead: 40})
	e.RegisterArm(domain.Arm{ID: "b1", ClinicID: "clinicB", AvgCostPerLead: 40})
	for i := 0; i < 200; i++ {
		e.Observe("a1", i%4 == 0, 40)
		e.Observe("b1", i%4 == 0, 40)
	}
	// Only clinicA is funded today; clinicB has no budget.
	allocs, _ := e.Allocate(map[string]float64{"clinicA": 1000, "clinicB": 0}, nil)
	var aB, bB float64
	for _, a := range allocs {
		if a.ArmID == "a1" {
			aB = a.DailyBudget
		}
		if a.ArmID == "b1" {
			bB = a.DailyBudget
		}
	}
	if aB <= 0 {
		t.Fatalf("clinicA arm should be funded from its budget, got %.0f", aB)
	}
	if bB != 0 {
		t.Fatalf("clinicB has no budget; its arm must get 0, got %.0f", bB)
	}
}
