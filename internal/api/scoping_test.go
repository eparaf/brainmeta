package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"disci/brain/internal/auth"
	"disci/brain/internal/domain"
)

// scopingFixture wires two clinics (A, B) into a real Server + store, plus a
// doctor/service/connection/lead/appointment/widget for clinic A only — the data
// a cross-tenant request must never see. Mirrors newTestServer (auth_test.go).
type scopingFixture struct {
	s                *Server
	admin, userA     *domain.User // userA is scoped to clinicA only
	clinicA, clinicB string
	leadA            domain.Lead
	doctorA          domain.Doctor
	serviceA         domain.Service
}

func newScopingFixture(t *testing.T) *scopingFixture {
	t.Helper()
	s, st := newTestServer()
	const clinicA, clinicB = "clinicA", "clinicB"
	s.eng.RegisterClinic(domain.Clinic{ID: clinicA, Name: "A", Segment: domain.SegmentGeneral, DailyCapacity: 5})
	s.eng.RegisterClinic(domain.Clinic{ID: clinicB, Name: "B", Segment: domain.SegmentGeneral, DailyCapacity: 5})

	lead := domain.Lead{ID: "leadA1", ClinicID: clinicA, Status: domain.LeadNew, Phone: "+905551110000"}
	st.SaveLead(lead)
	st.SaveAppointment(domain.Appointment{ID: "apptA1", ClinicID: clinicA, When: time.Now()})
	doc := domain.Doctor{ID: "docA1", ClinicID: clinicA, Name: "Dr. A"}
	st.SaveDoctor(doc)
	svc := domain.Service{ID: "svcA1", ClinicID: clinicA, Name: "Muayene"}
	st.SaveService(svc)
	st.UpsertConnection(domain.Connection{ID: clinicA + ":whatsapp", ClinicID: clinicA, Type: "whatsapp"})
	st.SaveWidgetConfig(domain.WidgetConfig{ClinicID: clinicA, PublicKey: "pubA1"})

	return &scopingFixture{
		s:       s,
		admin:   &domain.User{ID: "admin", Role: domain.RoleAdmin},
		userA:   &domain.User{ID: "userA", Role: domain.RoleClinic, ClinicIDs: []string{clinicA}},
		clinicA: clinicA, clinicB: clinicB,
		leadA: lead, doctorA: doc, serviceA: svc,
	}
}

func reqWithUser(method, target string, u *domain.User) *http.Request {
	ctx := context.Background()
	if u != nil {
		ctx = auth.WithUser(ctx, u)
	}
	return httptest.NewRequest(method, target, nil).WithContext(ctx)
}

func reqWithUserJSON(method, target string, u *domain.User, body any) *http.Request {
	b, _ := json.Marshal(body)
	ctx := context.Background()
	if u != nil {
		ctx = auth.WithUser(ctx, u)
	}
	return httptest.NewRequest(method, target, bytes.NewReader(b)).WithContext(ctx)
}

// --- leads / conversations / appointments (data.go) --------------------------

func TestScopingListLeadsCrossTenant(t *testing.T) {
	f := newScopingFixture(t)

	// Clinic-scoped user asking for THEIR OWN clinic sees the lead.
	w := httptest.NewRecorder()
	f.s.handleListLeads(w, reqWithUser("GET", "/v1/leads?clinicId="+f.clinicA, f.userA))
	var out []map[string]any
	json.Unmarshal(w.Body.Bytes(), &out)
	if len(out) != 1 || out[0]["id"] != f.leadA.ID {
		t.Fatalf("clinic user should see their own lead, got %v (status %d)", out, w.Code)
	}

	// Same user asking for the OTHER clinic must be forbidden, not filtered-empty.
	w = httptest.NewRecorder()
	f.s.handleListLeads(w, reqWithUser("GET", "/v1/leads?clinicId="+f.clinicB, f.userA))
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant list should be 403, got %d", w.Code)
	}

	// No clinicId filter (e.g. "all leads"): must be filtered down to the user's
	// own clinic only — clinic B's data must never leak into the response.
	w = httptest.NewRecorder()
	f.s.handleListLeads(w, reqWithUser("GET", "/v1/leads", f.userA))
	out = nil
	json.Unmarshal(w.Body.Bytes(), &out)
	for _, l := range out {
		if l["clinicId"] == f.clinicB {
			t.Fatalf("unfiltered list leaked clinic B's lead to a clinic-A-only user: %v", l)
		}
	}

	// Admin sees it regardless of clinicId.
	w = httptest.NewRecorder()
	f.s.handleListLeads(w, reqWithUser("GET", "/v1/leads?clinicId="+f.clinicA, f.admin))
	if w.Code != http.StatusOK {
		t.Fatalf("admin should access any clinic, got %d", w.Code)
	}
}

func TestScopingGetConversationCrossTenant(t *testing.T) {
	f := newScopingFixture(t)
	r := reqWithUser("GET", "/v1/conversations/"+f.leadA.ID, f.userA)
	r.SetPathValue("id", f.leadA.ID)

	w := httptest.NewRecorder()
	f.s.handleConversation(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("owner should read their own lead's conversation, got %d", w.Code)
	}

	// A user scoped to a DIFFERENT clinic must not read clinic A's conversation.
	userB := &domain.User{ID: "userB", Role: domain.RoleClinic, ClinicIDs: []string{f.clinicB}}
	r2 := reqWithUser("GET", "/v1/conversations/"+f.leadA.ID, userB)
	r2.SetPathValue("id", f.leadA.ID)
	w2 := httptest.NewRecorder()
	f.s.handleConversation(w2, r2)
	if w2.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant conversation read should be 403, got %d", w2.Code)
	}
}

func TestScopingListAppointmentsCrossTenant(t *testing.T) {
	f := newScopingFixture(t)

	w := httptest.NewRecorder()
	f.s.handleListAppointments(w, reqWithUser("GET", "/v1/appointments?clinicId="+f.clinicB, f.userA))
	if w.Code != http.StatusForbidden {
		t.Fatalf("clinic-A user requesting clinic B's appointments should be 403, got %d", w.Code)
	}

	// Unfiltered request must only include clinics the user can access.
	w = httptest.NewRecorder()
	f.s.handleListAppointments(w, reqWithUser("GET", "/v1/appointments", f.userA))
	var out []map[string]any
	json.Unmarshal(w.Body.Bytes(), &out)
	for _, a := range out {
		if a["clinicId"] == f.clinicB {
			t.Fatalf("unfiltered appointments list leaked clinic B's data: %v", a)
		}
	}
}

// --- doctors / services (calendar.go) -----------------------------------------

func TestScopingDoctorCRUDCrossTenant(t *testing.T) {
	f := newScopingFixture(t)

	// List: clinic-A user can only ask for clinic A.
	w := httptest.NewRecorder()
	f.s.handleListDoctors(w, reqWithUser("GET", "/v1/doctors?clinicId="+f.clinicB, f.userA))
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant doctor list should be 403, got %d", w.Code)
	}

	// Save: cannot create a doctor for a clinic you don't belong to.
	other := domain.Doctor{ID: "doc-evil", ClinicID: f.clinicB, Name: "Injected"}
	w = httptest.NewRecorder()
	f.s.handleSaveDoctor(w, reqWithUserJSON("POST", "/v1/doctors", f.userA, other))
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant doctor save should be 403, got %d", w.Code)
	}
	if _, ok := f.s.store.GetDoctor("doc-evil"); ok {
		t.Fatal("forbidden doctor save must not have persisted")
	}

	// Delete: cannot delete a doctor belonging to another clinic (403, not 404 —
	// so a caller can't distinguish "doesn't exist" from "not yours").
	r := reqWithUser("DELETE", "/v1/doctors/"+f.doctorA.ID, &domain.User{Role: domain.RoleClinic, ClinicIDs: []string{f.clinicB}})
	r.SetPathValue("id", f.doctorA.ID)
	w = httptest.NewRecorder()
	f.s.handleDeleteDoctor(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant doctor delete should be 403, got %d", w.Code)
	}
	if _, ok := f.s.store.GetDoctor(f.doctorA.ID); !ok {
		t.Fatal("forbidden delete must not have removed the doctor")
	}
}

func TestScopingServiceCRUDCrossTenant(t *testing.T) {
	f := newScopingFixture(t)
	userB := &domain.User{Role: domain.RoleClinic, ClinicIDs: []string{f.clinicB}}

	other := domain.Service{ID: "svc-evil", ClinicID: f.clinicA, Name: "Injected"}
	w := httptest.NewRecorder()
	f.s.handleSaveService(w, reqWithUserJSON("POST", "/v1/services", userB, other))
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant service save should be 403, got %d", w.Code)
	}

	r := reqWithUser("DELETE", "/v1/services/"+f.serviceA.ID, userB)
	r.SetPathValue("id", f.serviceA.ID)
	w = httptest.NewRecorder()
	f.s.handleDeleteService(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant service delete should be 403, got %d", w.Code)
	}
	if _, ok := f.s.store.GetService(f.serviceA.ID); !ok {
		t.Fatal("forbidden delete must not have removed the service")
	}
}

// --- connections / widget --------------------------------------------------

func TestScopingConnectionsCrossTenant(t *testing.T) {
	f := newScopingFixture(t)
	userB := &domain.User{Role: domain.RoleClinic, ClinicIDs: []string{f.clinicB}}

	w := httptest.NewRecorder()
	f.s.handleConnections(w, reqWithUser("GET", "/v1/connections?clinicId="+f.clinicA, userB))
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant connections read should be 403, got %d", w.Code)
	}

	body := map[string]any{"clinicId": f.clinicA, "type": "whatsapp", "connected": true}
	w = httptest.NewRecorder()
	f.s.handleConnections(w, reqWithUserJSON("POST", "/v1/connections", userB, body))
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant connection write should be 403, got %d", w.Code)
	}
}

func TestScopingWidgetCrossTenant(t *testing.T) {
	f := newScopingFixture(t)
	userB := &domain.User{Role: domain.RoleClinic, ClinicIDs: []string{f.clinicB}}

	w := httptest.NewRecorder()
	f.s.handleWidgetGet(w, reqWithUser("GET", "/v1/widget?clinicId="+f.clinicA, userB))
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant widget read should be 403, got %d", w.Code)
	}

	cfg := domain.WidgetConfig{ClinicID: f.clinicA, FormTitle: "Hijacked"}
	w = httptest.NewRecorder()
	f.s.handleWidgetSave(w, reqWithUserJSON("POST", "/v1/widget", userB, cfg))
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant widget save should be 403, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	f.s.handleWidgetRotate(w, reqWithUser("POST", "/v1/widget/rotate-key?clinicId="+f.clinicA, userB))
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant widget key rotation should be 403, got %d", w.Code)
	}
}

// --- dev-open (no auth) invariant: unauthenticated is full-access, by design --

func TestScopingDevOpenIsFullAccess(t *testing.T) {
	f := newScopingFixture(t)
	w := httptest.NewRecorder()
	f.s.handleListLeads(w, reqWithUser("GET", "/v1/leads?clinicId="+f.clinicB, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("dev-open (nil user) should pass CanAccessClinic, got %d — see auth.CanAccessClinic", w.Code)
	}
}
