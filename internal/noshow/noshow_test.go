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

// TestCalibrationIdentityOnDayOne: a fresh predictor's calibration is the identity,
// so PShow equals the base show prior — no behavioural change until data arrives.
func TestCalibrationIdentityOnDayOne(t *testing.T) {
	p := NewPredictor()
	got := p.PShow(Appt{LeadTimeDays: 2, HourOfDay: 14, DistanceKm: 5})
	if math.Abs(got-baseShowProb()) > 1e-9 {
		t.Fatalf("day-one PShow should equal base prior %.4f, got %.4f", baseShowProb(), got)
	}
	st := p.Export()
	if st.CalA != 1 || st.CalB != 0 {
		t.Fatalf("fresh calibration should be identity (1,0), got (%.3f,%.3f)", st.CalA, st.CalB)
	}
}

// TestCalibrationApplied: a non-identity calibrator changes PShow, and survives an
// Export→Import round-trip. Also checks old snapshots (no calA/calB) stay identity.
func TestCalibrationApplied(t *testing.T) {
	p := NewPredictor()
	base := p.PShow(Appt{LeadTimeDays: 2})
	// A calibrator that sharpens toward 1 should raise the calibrated prob.
	p.Import(State{W: make([]float64, showDim), B: p.Export().B, CalA: 2, CalB: 1})
	lifted := p.PShow(Appt{LeadTimeDays: 2})
	if lifted <= base {
		t.Fatalf("calibrator (2,1) should raise PShow above %.3f, got %.3f", base, lifted)
	}
	// Round-trip preserves calibration.
	q := NewPredictor()
	q.Import(p.Export())
	if math.Abs(q.PShow(Appt{LeadTimeDays: 2})-lifted) > 1e-9 {
		t.Fatalf("calibration did not survive round-trip: %.4f vs %.4f", q.PShow(Appt{LeadTimeDays: 2}), lifted)
	}
	// Old snapshot (no calibration fields) → identity retained.
	r := NewPredictor()
	r.Import(State{W: make([]float64, showDim), B: 0.94})
	if st := r.Export(); st.CalA != 1 || st.CalB != 0 {
		t.Fatalf("old snapshot should keep identity calibration, got (%.3f,%.3f)", st.CalA, st.CalB)
	}
}

// TestCalibratorLearnsFromBias: feeding a stream where everyone shows drives the
// online calibrator off the identity — proof the calibration is live, not frozen.
func TestCalibratorLearnsFromBias(t *testing.T) {
	p := NewPredictor()
	a := Appt{LeadTimeDays: 2, HourOfDay: 14, DistanceKm: 5}
	for i := 0; i < 500; i++ {
		p.Learn(a, true) // always shows
	}
	st := p.Export()
	if st.CalA == 1 && st.CalB == 0 {
		t.Fatal("calibrator stayed at identity despite 500 biased outcomes")
	}
	// Calibrated prob should have risen toward 1 (everyone showed).
	if p.PShow(a) <= baseShowProb() {
		t.Fatalf("PShow should rise above base after all-show stream, got %.3f", p.PShow(a))
	}
}
