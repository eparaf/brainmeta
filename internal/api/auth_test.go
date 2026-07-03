package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"disci/brain/internal/auth"
	"disci/brain/internal/config"
	"disci/brain/internal/domain"
	"disci/brain/internal/engine"
	"disci/brain/internal/store"
)

func newTestServer() (*Server, store.Store) {
	st := store.NewMemory()
	eng := engine.New(config.Default(), st)
	s := New(eng, nil)
	s.SetStore(st)
	return s, st
}

func registerReq(ctx context.Context, body map[string]any) *http.Request {
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(b))
	return r.WithContext(ctx)
}

// Registration is admin-only: an unauthenticated caller (dev-open, nil user) must
// NOT be able to create an account — least of all an admin (privilege escalation).
func TestRegisterRequiresAdmin(t *testing.T) {
	s, st := newTestServer()
	body := map[string]any{"email": "evil@x.com", "password": "password123", "role": "admin"}

	// 1) No user in context → 401, nothing created.
	w := httptest.NewRecorder()
	s.handleRegister(w, registerReq(context.Background(), body))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated register: want 401, got %d", w.Code)
	}
	if _, ok := st.GetUserByEmail("evil@x.com"); ok {
		t.Fatal("unauthenticated register created a user")
	}

	// 2) Non-admin (clinic) user → 403.
	clinicCtx := auth.WithUser(context.Background(), &domain.User{Role: domain.RoleClinic})
	w = httptest.NewRecorder()
	s.handleRegister(w, registerReq(clinicCtx, body))
	if w.Code != http.StatusForbidden {
		t.Fatalf("clinic-user register: want 403, got %d", w.Code)
	}

	// 3) Admin user → 201, user created.
	adminCtx := auth.WithUser(context.Background(), &domain.User{Role: domain.RoleAdmin})
	w = httptest.NewRecorder()
	s.handleRegister(w, registerReq(adminCtx, map[string]any{
		"email": "new@clinic.com", "password": "password123", "role": "clinic",
	}))
	if w.Code != http.StatusCreated {
		t.Fatalf("admin register: want 201, got %d (%s)", w.Code, w.Body.String())
	}
	if _, ok := st.GetUserByEmail("new@clinic.com"); !ok {
		t.Fatal("admin register did not persist the user")
	}
}

// Clinic scoping: a clinic-scoped user must only pass CanAccessClinic for their own
// clinics; an admin passes for any. Guards the tenant-isolation invariant handlers
// rely on.
func TestClinicScopingEnforced(t *testing.T) {
	clinicUser := &domain.User{Role: domain.RoleClinic, ClinicIDs: []string{"umraniye"}}
	if !auth.CanAccessClinic(clinicUser, "umraniye") {
		t.Error("clinic user denied their own clinic")
	}
	if auth.CanAccessClinic(clinicUser, "nisantasi") {
		t.Error("clinic user allowed a clinic they don't belong to")
	}
	admin := &domain.User{Role: domain.RoleAdmin}
	if !auth.CanAccessClinic(admin, "any-clinic") {
		t.Error("admin denied access")
	}
}
