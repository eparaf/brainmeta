// Package store defines the persistence interfaces and an in-memory
// implementation. The engine depends only on the interfaces, so swapping the
// in-memory store for a Postgres-backed one is a drop-in change — no decision
// logic touches SQL. (In production: Postgres for entities, Redis for the hot
// online feature cache.)
package store

import (
	"sync"

	"disci/brain/internal/domain"
)

// Store is the full persistence surface the brain needs.
type Store interface {
	UpsertClinic(domain.Clinic)
	GetClinic(id string) (domain.Clinic, bool)
	ListClinics() []domain.Clinic

	SaveLead(domain.Lead)
	GetLead(id string) (domain.Lead, bool)
	UpdateLead(domain.Lead)
	LeadsByStatus(domain.LeadStatus) []domain.Lead

	SaveAppointment(domain.Appointment)
	AppointmentsForClinic(clinicID string) []domain.Appointment

	// SeatUsage returns how many new-patient appointments a clinic already holds
	// for the current planning cycle (used to compute free capacity).
	SeatUsage(clinicID string) int
}

// Memory is a thread-safe in-memory Store for tests, the simulator, and early
// production before Postgres is wired in.
type Memory struct {
	mu       sync.RWMutex
	clinics  map[string]domain.Clinic
	leads    map[string]domain.Lead
	appts    map[string][]domain.Appointment
	seatUsed map[string]int
}

func NewMemory() *Memory {
	return &Memory{
		clinics:  map[string]domain.Clinic{},
		leads:    map[string]domain.Lead{},
		appts:    map[string][]domain.Appointment{},
		seatUsed: map[string]int{},
	}
}

func (m *Memory) UpsertClinic(c domain.Clinic) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clinics[c.ID] = c
}

func (m *Memory) GetClinic(id string) (domain.Clinic, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.clinics[id]
	return c, ok
}

func (m *Memory) ListClinics() []domain.Clinic {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.Clinic, 0, len(m.clinics))
	for _, c := range m.clinics {
		out = append(out, c)
	}
	return out
}

func (m *Memory) SaveLead(l domain.Lead) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.leads[l.ID] = l
}

func (m *Memory) GetLead(id string) (domain.Lead, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	l, ok := m.leads[id]
	return l, ok
}

func (m *Memory) UpdateLead(l domain.Lead) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.leads[l.ID] = l
}

func (m *Memory) LeadsByStatus(s domain.LeadStatus) []domain.Lead {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []domain.Lead
	for _, l := range m.leads {
		if l.Status == s {
			out = append(out, l)
		}
	}
	return out
}

func (m *Memory) SaveAppointment(a domain.Appointment) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.appts[a.ClinicID] = append(m.appts[a.ClinicID], a)
	if !a.Overbook {
		m.seatUsed[a.ClinicID]++
	}
}

func (m *Memory) AppointmentsForClinic(clinicID string) []domain.Appointment {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]domain.Appointment(nil), m.appts[clinicID]...)
}

func (m *Memory) SeatUsage(clinicID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.seatUsed[clinicID]
}

// ResetSeats clears per-cycle seat usage (call at the start of each planning
// cycle / day).
func (m *Memory) ResetSeats() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seatUsed = map[string]int{}
}
