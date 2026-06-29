package datasource

import (
	"context"
	"sync"
	"time"

	"disci/brain/internal/domain"
)

// This file provides in-memory implementations of every datasource interface.
// They let the full sync pipeline run end-to-end in local dev and tests without
// any external credentials — and they document exactly what a real adapter must
// produce. Swap each for an HTTP-backed client (WhatsApp Cloud API, Meta
// Marketing API, Google Ads API, the clinic's PMS) in production; the
// SyncService doesn't change.

// MemoryLeadSource emits a fixed slice of leads then closes the channel.
type MemoryLeadSource struct{ Events []LeadEvent }

func (m *MemoryLeadSource) Stream(ctx context.Context) (<-chan LeadEvent, error) {
	ch := make(chan LeadEvent)
	go func() {
		defer close(ch)
		for _, e := range m.Events {
			select {
			case <-ctx.Done():
				return
			case ch <- e:
			}
		}
	}()
	return ch, nil
}

// MemoryAdPlatform records budget pushes and serves canned spend data.
type MemoryAdPlatform struct {
	mu          sync.Mutex
	Spend       []ArmSpend
	LastBudgets map[string]float64
	Uploaded    []Conversion
}

func (m *MemoryAdPlatform) PullSpend(ctx context.Context, since time.Time) ([]ArmSpend, error) {
	return m.Spend, nil
}

func (m *MemoryAdPlatform) SetDailyBudgets(ctx context.Context, perArm map[string]float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LastBudgets = perArm
	return nil
}

func (m *MemoryAdPlatform) UploadConversions(ctx context.Context, convs []Conversion) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Uploaded = append(m.Uploaded, convs...)
	return nil
}

// MemoryPMS serves canned capacity/outcomes and records pushed appointments.
type MemoryPMS struct {
	mu           sync.Mutex
	Capacity     map[string]int
	Outcomes     map[string][]domain.Outcome
	Appointments []domain.Appointment
}

func (m *MemoryPMS) PullCapacity(ctx context.Context, clinicID string) (int, error) {
	return m.Capacity[clinicID], nil
}

func (m *MemoryPMS) PushAppointment(ctx context.Context, appt domain.Appointment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Appointments = append(m.Appointments, appt)
	return nil
}

func (m *MemoryPMS) PullOutcomes(ctx context.Context, clinicID string, since time.Time) ([]domain.Outcome, error) {
	return m.Outcomes[clinicID], nil
}

// MemoryMessenger records sent messages.
type MemoryMessenger struct {
	mu   sync.Mutex
	Sent []SentMessage
}

type SentMessage struct {
	Phone, Template string
	Vars            map[string]string
}

func (m *MemoryMessenger) Send(ctx context.Context, phone, templateID string, vars map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Sent = append(m.Sent, SentMessage{Phone: phone, Template: templateID, Vars: vars})
	return nil
}
