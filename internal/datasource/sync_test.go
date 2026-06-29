package datasource

import (
	"context"
	"testing"
	"time"

	"disci/brain/internal/config"
	"disci/brain/internal/domain"
	"disci/brain/internal/engine"
	"disci/brain/internal/store"
)

// TestEndToEndSyncPipeline wires the in-memory adapters to a real engine and
// proves the whole loop runs: an inbound lead is decided, booked into the PMS, a
// reminder is sent, budget decisions are pushed to the ad platform, and a
// realised closed-deal outcome is fed back and uploaded as a conversion.
func TestEndToEndSyncPipeline(t *testing.T) {
	st := store.NewMemory()
	eng := engine.New(config.Default(), st)
	eng.RegisterClinic(domain.Clinic{
		ID: "umraniye", Segment: domain.SegmentImplant, CloseRate: 0.42,
		DailyCapacity: 10, GuaranteedApptsPerMonth: 80, MonthlyAdBudget: 220_000,
	})

	ads := &MemoryAdPlatform{Spend: []ArmSpend{
		{ArmID: "umraniye:meta:implant", Spend: 10_000, Leads: 50, CostPerLead: 200},
	}}
	pms := &MemoryPMS{
		Capacity: map[string]int{"umraniye": 10},
		Outcomes: map[string][]domain.Outcome{"umraniye": {
			closedOutcome("umraniye", "umraniye:meta:implant"),
		}},
	}
	msgr := &MemoryMessenger{}

	svc := &SyncService{
		Eng: eng, Ads: ads, PMS: pms, Messenger: msgr,
		Featurizer: func(LeadEvent) domain.LeadFeatures {
			return domain.LeadFeatures{IntentScore: 0.8, UrgencyScore: 0.7, MessagesExchanged: 6, DistanceKm: 4, HourOfDay: 14}
		},
	}
	ctx := context.Background()

	// 1. Inbound lead → decision → PMS + reminder.
	dec := svc.HandleInbound(ctx, LeadEvent{
		ExternalID: "wa-1", ArmID: "umraniye:meta:implant", ClinicID: "umraniye",
		Phone: "+90555", ReceivedAt: time.Now(),
	})
	if !dec.Booked {
		t.Fatalf("expected high-intent implant lead to book; reason=%s", dec.Reason)
	}
	if len(pms.Appointments) != 1 {
		t.Fatalf("expected appointment pushed to PMS, got %d", len(pms.Appointments))
	}
	if len(msgr.Sent) != 1 {
		t.Fatalf("expected a reminder sent, got %d", len(msgr.Sent))
	}

	// 2. Ad sync → CPL corrected + budgets pushed.
	svc.SyncAds(ctx)
	if ads.LastBudgets == nil || ads.LastBudgets["umraniye:meta:implant"] <= 0 {
		t.Fatalf("expected budget pushed to ad platform, got %+v", ads.LastBudgets)
	}

	// 3. Outcome sync → feedback ingested + conversion uploaded.
	svc.SyncOutcomes(ctx, time.Time{})
	if len(ads.Uploaded) != 1 || ads.Uploaded[0].EventName != "closed" {
		t.Fatalf("expected a closed conversion uploaded, got %+v", ads.Uploaded)
	}
}

func closedOutcome(clinic, arm string) domain.Outcome {
	yes := true
	return domain.Outcome{
		LeadID: "wa-1", ClinicID: clinic, ArmID: arm, Segment: domain.SegmentImplant,
		Qualified: &yes, Booked: &yes, Showed: &yes, Closed: &yes,
		Revenue: 45_000, AdCost: 200, At: time.Now(),
	}
}
