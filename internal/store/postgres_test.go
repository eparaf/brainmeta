//go:build pgx

// Integration tests for StorePG against a REAL Postgres. Build with -tags pgx
// (registers the pgx stdlib driver via pgx_driver.go) and point DATABASE_URL at a
// disposable database — CI spins one up as a service container (see
// .github/workflows/ci.yml). Skips gracefully if DATABASE_URL is unset or the
// database is unreachable, so `go test -tags pgx ./...` stays safe to run
// without a live Postgres on a dev machine.
//
// Every entity uses a unique ID per test run (via a run-scoped prefix) rather
// than TRUNCATE, so these tests are safe to run repeatedly against a shared,
// persistent database without wiping anyone else's data.
package store

import (
	"fmt"
	"os"
	"testing"
	"time"

	"disci/brain/internal/domain"
)

// testStore opens (or skips) a StorePG for the test, and returns a unique run
// prefix to namespace IDs so parallel/repeated runs never collide.
func testStore(t *testing.T) (*StorePG, string) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping Postgres integration test")
	}
	s, err := NewPostgres("pgx", dsn)
	if err != nil {
		t.Skipf("Postgres unreachable, skipping: %v", err)
	}
	prefix := fmt.Sprintf("t%d-", time.Now().UnixNano())
	return s, prefix
}

func TestPostgresClinicRoundTrip(t *testing.T) {
	s, p := testStore(t)
	c := domain.Clinic{ID: p + "clinic1", Name: "Test Clinic", Segment: domain.SegmentGeneral, DailyCapacity: 5}
	s.UpsertClinic(c)

	got, ok := s.GetClinic(c.ID)
	if !ok || got.Name != c.Name || got.DailyCapacity != c.DailyCapacity {
		t.Fatalf("GetClinic round-trip mismatch: %+v", got)
	}

	// Upsert again with a changed field — must update, not duplicate.
	c.DailyCapacity = 9
	s.UpsertClinic(c)
	got, _ = s.GetClinic(c.ID)
	if got.DailyCapacity != 9 {
		t.Fatalf("UpsertClinic did not update existing row: got %d", got.DailyCapacity)
	}

	found := false
	for _, lc := range s.ListClinics() {
		if lc.ID == c.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("ListClinics did not include the upserted clinic")
	}
}

func TestPostgresLeadRoundTripAndFilters(t *testing.T) {
	s, p := testStore(t)
	clinicID := p + "clinic1"
	lead := domain.Lead{ID: p + "lead1", ClinicID: clinicID, Status: domain.LeadNew, Phone: "+905551234567"}
	s.SaveLead(lead)

	got, ok := s.GetLead(lead.ID)
	if !ok || got.Phone != lead.Phone {
		t.Fatalf("GetLead round-trip mismatch: %+v", got)
	}

	lead.Status = domain.LeadBooked
	s.UpdateLead(lead)
	got, _ = s.GetLead(lead.ID)
	if got.Status != domain.LeadBooked {
		t.Fatalf("UpdateLead did not persist status change: got %s", got.Status)
	}

	byStatus := s.LeadsByStatus(domain.LeadBooked)
	if !containsLeadID(byStatus, lead.ID) {
		t.Fatal("LeadsByStatus did not return the updated lead")
	}

	filtered := s.ListLeads(LeadFilter{ClinicID: clinicID})
	if !containsLeadID(filtered, lead.ID) {
		t.Fatal("ListLeads(clinicID filter) did not return the lead")
	}
	filteredWrong := s.ListLeads(LeadFilter{ClinicID: p + "other-clinic"})
	if containsLeadID(filteredWrong, lead.ID) {
		t.Fatal("ListLeads leaked a lead across clinic filter")
	}
}

func containsLeadID(leads []domain.Lead, id string) bool {
	for _, l := range leads {
		if l.ID == id {
			return true
		}
	}
	return false
}

func TestPostgresAppointmentAndSeatUsage(t *testing.T) {
	s, p := testStore(t)
	clinicID := p + "clinic1"
	now := time.Now()
	appt := domain.Appointment{ID: p + "appt1", ClinicID: clinicID, When: now, Overbook: false}
	s.SaveAppointment(appt)

	list := s.AppointmentsForClinic(clinicID)
	if len(list) != 1 || list[0].ID != appt.ID {
		t.Fatalf("AppointmentsForClinic mismatch: %+v", list)
	}

	if n := s.SeatUsage(clinicID); n < 1 {
		t.Fatalf("SeatUsage should count today's non-overbook appointment, got %d", n)
	}

	// An overbook appointment must NOT count toward seat usage.
	over := domain.Appointment{ID: p + "appt2", ClinicID: clinicID, When: now, Overbook: true}
	s.SaveAppointment(over)
	before := s.SeatUsage(clinicID)
	s.SaveAppointment(over) // re-save, idempotent
	if after := s.SeatUsage(clinicID); after != before {
		t.Fatalf("overbook appointment changed seat usage unexpectedly: before=%d after=%d", before, after)
	}
}

func TestPostgresUserCreateDuplicateEmail(t *testing.T) {
	s, p := testStore(t)
	u := domain.User{ID: p + "user1", Email: p + "person@example.com", Role: domain.RoleAdmin}
	if err := s.CreateUser(u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	got, ok := s.GetUserByID(u.ID)
	if !ok || got.Email != u.Email {
		t.Fatalf("GetUserByID mismatch: %+v", got)
	}
	byEmail, ok := s.GetUserByEmail(u.Email)
	if !ok || byEmail.ID != u.ID {
		t.Fatalf("GetUserByEmail mismatch: %+v", byEmail)
	}

	// Duplicate email (even different ID) must be rejected with ErrDuplicate.
	dup := domain.User{ID: p + "user2", Email: u.Email, Role: domain.RoleClinic}
	if err := s.CreateUser(dup); err != ErrDuplicate {
		t.Fatalf("expected ErrDuplicate for duplicate email, got %v", err)
	}

	found := false
	for _, lu := range s.ListUsers() {
		if lu.ID == u.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("ListUsers did not include the created user")
	}
}

func TestPostgresConnectionsAndOAuthTokens(t *testing.T) {
	s, p := testStore(t)
	clinicID := p + "clinic1"

	conn := domain.Connection{ID: clinicID + ":google", ClinicID: clinicID, Type: "google"}
	s.UpsertConnection(conn)
	list := s.ListConnections(clinicID)
	if len(list) != 1 || list[0].Type != "google" {
		t.Fatalf("ListConnections mismatch: %+v", list)
	}

	tok := domain.OAuthToken{ClinicID: clinicID, Provider: "google", RefreshToken: "refresh-abc"}
	s.UpsertOAuthToken(tok)
	got, ok := s.GetOAuthToken(clinicID, "google")
	if !ok || got.RefreshToken != "refresh-abc" {
		t.Fatalf("GetOAuthToken mismatch: %+v", got)
	}

	all := s.ListOAuthTokens("google")
	found := false
	for _, tt := range all {
		if tt.ClinicID == clinicID {
			found = true
		}
	}
	if !found {
		t.Fatal("ListOAuthTokens did not include the upserted token")
	}
}

func TestPostgresTemplates(t *testing.T) {
	s, p := testStore(t)
	clinicID := p + "clinic1"
	tmpl := domain.TemplateDraft{ID: p + "tmpl1", ClinicID: clinicID, Status: "pending", Name: "reminder"}
	s.SaveTemplate(tmpl)

	list := s.ListTemplates(clinicID)
	if len(list) != 1 || list[0].Name != "reminder" {
		t.Fatalf("ListTemplates mismatch: %+v", list)
	}
}

func TestPostgresWidgetConfig(t *testing.T) {
	s, p := testStore(t)
	clinicID := p + "clinic1"
	cfg := domain.WidgetConfig{ClinicID: clinicID, PublicKey: p + "pubkey1"}
	s.SaveWidgetConfig(cfg)

	got, ok := s.GetWidgetConfig(clinicID)
	if !ok || got.PublicKey != cfg.PublicKey {
		t.Fatalf("GetWidgetConfig mismatch: %+v", got)
	}
	byKey, ok := s.GetWidgetConfigByKey(cfg.PublicKey)
	if !ok || byKey.ClinicID != clinicID {
		t.Fatalf("GetWidgetConfigByKey mismatch: %+v", byKey)
	}
}

func TestPostgresDoctorsAndServices(t *testing.T) {
	s, p := testStore(t)
	clinicID := p + "clinic1"

	doc := domain.Doctor{ID: p + "doc1", ClinicID: clinicID, Name: "Dr. Test", Active: true}
	s.SaveDoctor(doc)
	if got, ok := s.GetDoctor(doc.ID); !ok || got.Name != doc.Name {
		t.Fatalf("GetDoctor mismatch: %+v", got)
	}
	if list := s.ListDoctors(clinicID); len(list) != 1 {
		t.Fatalf("ListDoctors expected 1, got %d", len(list))
	}
	s.DeleteDoctor(doc.ID)
	if _, ok := s.GetDoctor(doc.ID); ok {
		t.Fatal("DeleteDoctor did not remove the row")
	}

	svc := domain.Service{ID: p + "svc1", ClinicID: clinicID, Name: "Muayene", DurationMins: 30}
	s.SaveService(svc)
	if got, ok := s.GetService(svc.ID); !ok || got.Name != svc.Name {
		t.Fatalf("GetService mismatch: %+v", got)
	}
	if list := s.ListServices(clinicID); len(list) != 1 {
		t.Fatalf("ListServices expected 1, got %d", len(list))
	}
	s.DeleteService(svc.ID)
	if _, ok := s.GetService(svc.ID); ok {
		t.Fatal("DeleteService did not remove the row")
	}
}

// TestPostgresMigrateIsIdempotent confirms calling Migrate() again (e.g. on
// every process restart, as NewPostgres does) doesn't error on an existing schema.
func TestPostgresMigrateIsIdempotent(t *testing.T) {
	s, _ := testStore(t)
	if err := s.Migrate(); err != nil {
		t.Fatalf("re-running Migrate() on an existing schema should be a no-op, got: %v", err)
	}
}
