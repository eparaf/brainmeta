package noshow

import (
	"math"
	"testing"
)

// TestOverbookingFillsCapacity verifies the solver books MORE than capacity when
// show probability is below 1, so expected arrivals approach capacity.
func TestOverbookingFillsCapacity(t *testing.T) {
	capacity := 10
	ps := make([]float64, 30)
	for i := range ps {
		ps[i] = 0.7 // each patient shows 70% of the time
	}
	plan := PlanOverbooking(capacity, ps, 0.15)
	if plan.Booked <= capacity {
		t.Fatalf("expected overbooking beyond %d seats, booked %d", capacity, plan.Booked)
	}
	if math.Abs(plan.ExpectedArrivals-float64(capacity)) > 2.5 {
		t.Fatalf("expected arrivals near capacity, got %.1f", plan.ExpectedArrivals)
	}
	if plan.OverbookRisk > 0.30 {
		t.Fatalf("overbook risk too high: %.2f", plan.OverbookRisk)
	}
}

// TestInterventionEscalatesWithRisk checks risky, valuable appointments get the
// expensive interventions.
func TestInterventionEscalatesWithRisk(t *testing.T) {
	if ChooseIntervention(0.95, 200000) != InterventionStandard {
		t.Fatal("safe appt should get standard reminders")
	}
	if ChooseIntervention(0.4, 200000) != InterventionCall {
		t.Fatal("risky high-value appt should get a confirmation call")
	}
	if ChooseIntervention(0.4, 5000) != InterventionDeposit {
		t.Fatal("risky low-value appt should get a deposit request")
	}
}

// TestPredictorLearnsDepositLift confirms a deposit raises predicted show prob.
func TestPredictorLearnsDepositLift(t *testing.T) {
	p := NewPredictor()
	withDep := Appt{LeadTimeDays: 3, DepositPaid: true, ConfirmedReply: true}
	without := Appt{LeadTimeDays: 3, DepositPaid: false, ConfirmedReply: true}
	// Teach it reality: patients who paid a deposit show, those who didn't often
	// don't. The model should then predict a higher show prob with a deposit.
	for i := 0; i < 200; i++ {
		p.Learn(withDep, true)
		p.Learn(without, i%2 == 0)
	}
	if p.PShow(withDep) <= p.PShow(without) {
		t.Fatalf("deposit should raise show prob: %.2f vs %.2f", p.PShow(withDep), p.PShow(without))
	}
}
