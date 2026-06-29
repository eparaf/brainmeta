package sla

import (
	"testing"
	"time"
)

// TestShadowPriceRisesWithDeficit checks a clinic that's behind schedule gets a
// shadow price > 1, and one that's ahead gets exactly 1.
func TestShadowPriceRisesWithDeficit(t *testing.T) {
	c := NewController(0.90)
	c.Register("behind", 100)
	c.Register("ahead", 100)

	// Mid-month: target is ~50 each.
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	// "ahead" delivered 70 (over target) -> bias 1.
	for i := 0; i < 70; i++ {
		c.RecordQualifiedAppt("ahead")
	}
	// "behind" delivered only 10 (big deficit) -> bias > 1.
	for i := 0; i < 10; i++ {
		c.RecordQualifiedAppt("behind")
	}

	ahead := c.biasAt("ahead", now)
	behind := c.biasAt("behind", now)
	if ahead != 1 {
		t.Fatalf("ahead clinic should have bias 1, got %v", ahead)
	}
	if behind <= 1 {
		t.Fatalf("behind clinic should have bias > 1, got %v", behind)
	}
}

// TestOnTrackReport sanity-checks the report fields.
func TestOnTrackReport(t *testing.T) {
	c := NewController(0.90)
	c.Register("x", 30)
	now := time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC) // ~half month
	for i := 0; i < 20; i++ {
		c.RecordQualifiedAppt("x")
	}
	rep := c.Report(now)
	if len(rep) != 1 || rep[0].Delivered != 20 {
		t.Fatalf("unexpected report: %+v", rep)
	}
	if !rep[0].OnTrack {
		t.Fatalf("clinic ahead of half-month target should be on track")
	}
}
