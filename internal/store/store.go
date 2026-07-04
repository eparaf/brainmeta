// Package store defines the persistence interfaces and an in-memory
// implementation. The engine depends only on the interfaces, so swapping the
// in-memory store for a Postgres-backed one is a drop-in change — no decision
// logic touches SQL. (In production: Postgres for entities, Redis for the hot
// online feature cache.)
package store

import (
	"errors"
	"strings"
	"sync"

	"disci/brain/internal/domain"
)

// ErrDuplicate is returned by CreateUser when the email already exists.
var ErrDuplicate = errors.New("store: duplicate")

// LeadFilter narrows a ListLeads query. Empty fields match anything.
type LeadFilter struct {
	ClinicID string
	Status   domain.LeadStatus
}

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

	// Users (dashboard auth). CreateUser is the one method that returns an error,
	// because registration must reject a duplicate email (→ ErrDuplicate).
	CreateUser(domain.User) error
	GetUserByEmail(email string) (domain.User, bool)
	GetUserByID(id string) (domain.User, bool)
	ListUsers() []domain.User

	// Dashboard list/CRUD surface (panel pages).
	ListLeads(LeadFilter) []domain.Lead
	UpsertConnection(domain.Connection)
	ListConnections(clinicID string) []domain.Connection
	SaveTemplate(domain.TemplateDraft)
	ListTemplates(clinicID string) []domain.TemplateDraft

	// OAuth secrets for ad-platform sync (refresh tokens). Kept apart from
	// Connection so secrets never leak through the status surface. Listed so the
	// server can spin up a live sync per clinic that has one.
	UpsertOAuthToken(domain.OAuthToken)
	// GetOAuthToken looks up by (clinicID, key) where key is the connection TYPE
	// used at write time (e.g. "whatsapp", "meta_ads", "google_ads") — falls back
	// to matching by Provider for tokens saved before Type existed. Currently
	// unused by any caller; kept for future per-connection-type lookups.
	GetOAuthToken(clinicID, key string) (domain.OAuthToken, bool)
	ListOAuthTokens(provider string) []domain.OAuthToken
	// ResolveClinicByPhoneNumberID maps a WhatsApp Cloud API phone_number_id
	// (from an inbound webhook's metadata) back to the clinic that connected it
	// via Embedded Signup — the per-clinic routing resolver. ok=false means no
	// clinic has claimed that number yet (caller falls back to unscoped routing).
	ResolveClinicByPhoneNumberID(phoneNumberID string) (clinicID string, ok bool)

	// Embeddable widget config (web form + calendar). Keyed by clinic, and looked
	// up by the public embed key on the public endpoints.
	SaveWidgetConfig(domain.WidgetConfig)
	GetWidgetConfig(clinicID string) (domain.WidgetConfig, bool)
	GetWidgetConfigByKey(publicKey string) (domain.WidgetConfig, bool)

	// Doctors & services (clinic calendar / appointment booking).
	SaveDoctor(domain.Doctor)
	ListDoctors(clinicID string) []domain.Doctor
	GetDoctor(id string) (domain.Doctor, bool)
	DeleteDoctor(id string)
	SaveService(domain.Service)
	ListServices(clinicID string) []domain.Service
	GetService(id string) (domain.Service, bool)
	DeleteService(id string)
}

// Memory is a thread-safe in-memory Store for tests, the simulator, and early
// production before Postgres is wired in.
type Memory struct {
	mu           sync.RWMutex
	clinics      map[string]domain.Clinic
	leads        map[string]domain.Lead
	appts        map[string][]domain.Appointment
	seatUsed     map[string]int
	users        map[string]domain.User       // keyed by ID
	usersByEmail map[string]string            // lowercased email → ID
	connections  map[string]domain.Connection // keyed by connection ID
	oauthTokens  map[string]domain.OAuthToken // keyed by "<clinicID>:<provider>"
	templates    map[string]domain.TemplateDraft
	widgets      map[string]domain.WidgetConfig // keyed by clinic ID
	widgetByKey  map[string]string              // public key → clinic ID
	doctors      map[string]domain.Doctor       // keyed by doctor ID
	services     map[string]domain.Service      // keyed by service ID
}

func NewMemory() *Memory {
	return &Memory{
		clinics:      map[string]domain.Clinic{},
		leads:        map[string]domain.Lead{},
		appts:        map[string][]domain.Appointment{},
		seatUsed:     map[string]int{},
		users:        map[string]domain.User{},
		usersByEmail: map[string]string{},
		connections:  map[string]domain.Connection{},
		oauthTokens:  map[string]domain.OAuthToken{},
		templates:    map[string]domain.TemplateDraft{},
		widgets:      map[string]domain.WidgetConfig{},
		widgetByKey:  map[string]string{},
		doctors:      map[string]domain.Doctor{},
		services:     map[string]domain.Service{},
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

func (m *Memory) CreateUser(u domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	email := strings.ToLower(u.Email)
	if _, exists := m.usersByEmail[email]; exists {
		return ErrDuplicate
	}
	u.Email = email
	m.users[u.ID] = u
	m.usersByEmail[email] = u.ID
	return nil
}

func (m *Memory) GetUserByEmail(email string) (domain.User, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.usersByEmail[strings.ToLower(email)]
	if !ok {
		return domain.User{}, false
	}
	u, ok := m.users[id]
	return u, ok
}

func (m *Memory) GetUserByID(id string) (domain.User, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	return u, ok
}

func (m *Memory) ListUsers() []domain.User {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.User, 0, len(m.users))
	for _, u := range m.users {
		out = append(out, u)
	}
	return out
}

func (m *Memory) ListLeads(f LeadFilter) []domain.Lead {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []domain.Lead
	for _, l := range m.leads {
		if f.ClinicID != "" && l.ClinicID != f.ClinicID {
			continue
		}
		if f.Status != "" && l.Status != f.Status {
			continue
		}
		out = append(out, l)
	}
	return out
}

func (m *Memory) UpsertConnection(c domain.Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connections[c.ID] = c
}

func (m *Memory) ListConnections(clinicID string) []domain.Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []domain.Connection
	for _, c := range m.connections {
		if clinicID == "" || c.ClinicID == clinicID {
			out = append(out, c)
		}
	}
	return out
}

// oauthTokenKey derives the storage key: Type when present (distinguishes
// "whatsapp" from "meta_ads" — both Provider="meta"), else Provider for
// backward compatibility with rows saved before Type existed.
func oauthTokenKey(clinicID, provider, typ string) string {
	key := typ
	if key == "" {
		key = provider
	}
	return clinicID + ":" + key
}

func (m *Memory) UpsertOAuthToken(t domain.OAuthToken) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.oauthTokens[oauthTokenKey(t.ClinicID, t.Provider, t.Type)] = t
}

func (m *Memory) GetOAuthToken(clinicID, key string) (domain.OAuthToken, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.oauthTokens[clinicID+":"+key]
	return t, ok
}

func (m *Memory) ListOAuthTokens(provider string) []domain.OAuthToken {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []domain.OAuthToken
	for _, t := range m.oauthTokens {
		if provider == "" || t.Provider == provider {
			out = append(out, t)
		}
	}
	return out
}

// ResolveClinicByPhoneNumberID linear-scans the (small, per-clinic) oauth token
// set — fine at this scale; Postgres uses an indexed column instead.
func (m *Memory) ResolveClinicByPhoneNumberID(phoneNumberID string) (string, bool) {
	if phoneNumberID == "" {
		return "", false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.oauthTokens {
		if t.PhoneNumberID == phoneNumberID {
			return t.ClinicID, true
		}
	}
	return "", false
}

func (m *Memory) SaveTemplate(t domain.TemplateDraft) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.templates[t.ID] = t
}

func (m *Memory) ListTemplates(clinicID string) []domain.TemplateDraft {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []domain.TemplateDraft
	for _, t := range m.templates {
		if clinicID == "" || t.ClinicID == clinicID {
			out = append(out, t)
		}
	}
	return out
}

func (m *Memory) SaveWidgetConfig(c domain.WidgetConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if old, ok := m.widgets[c.ClinicID]; ok && old.PublicKey != c.PublicKey {
		delete(m.widgetByKey, old.PublicKey)
	}
	m.widgets[c.ClinicID] = c
	if c.PublicKey != "" {
		m.widgetByKey[c.PublicKey] = c.ClinicID
	}
}

func (m *Memory) GetWidgetConfig(clinicID string) (domain.WidgetConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.widgets[clinicID]
	return c, ok
}

func (m *Memory) GetWidgetConfigByKey(publicKey string) (domain.WidgetConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.widgetByKey[publicKey]
	if !ok {
		return domain.WidgetConfig{}, false
	}
	c, ok := m.widgets[id]
	return c, ok
}

func (m *Memory) SaveDoctor(d domain.Doctor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.doctors[d.ID] = d
}

func (m *Memory) ListDoctors(clinicID string) []domain.Doctor {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []domain.Doctor{}
	for _, d := range m.doctors {
		if clinicID == "" || d.ClinicID == clinicID {
			out = append(out, d)
		}
	}
	return out
}

func (m *Memory) GetDoctor(id string) (domain.Doctor, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.doctors[id]
	return d, ok
}

func (m *Memory) DeleteDoctor(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.doctors, id)
}

func (m *Memory) SaveService(svc domain.Service) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.services[svc.ID] = svc
}

func (m *Memory) ListServices(clinicID string) []domain.Service {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []domain.Service{}
	for _, svc := range m.services {
		if clinicID == "" || svc.ClinicID == clinicID {
			out = append(out, svc)
		}
	}
	return out
}

func (m *Memory) GetService(id string) (domain.Service, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	svc, ok := m.services[id]
	return svc, ok
}

func (m *Memory) DeleteService(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.services, id)
}
