package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"disci/brain/internal/auth"
	"disci/brain/internal/domain"
	"disci/brain/internal/store"
)

// publicUser is the client-safe projection of a user — it never includes the
// password hash. Field names are camelCase for the Next.js panel.
func publicUser(u domain.User) map[string]any {
	return map[string]any{
		"id":        u.ID,
		"email":     u.Email,
		"name":      u.Name,
		"role":      u.Role,
		"clinicIds": u.ClinicIDs,
		"createdAt": u.CreatedAt,
	}
}

// clinicView maps a domain.Clinic to the camelCase shape the panel expects. The
// domain struct has no json tags, so default marshaling would be PascalCase. SLA
// fields (delivered/shadowPrice/status) are enriched later via /v1/clinics+/v1/sla.
func clinicView(c domain.Clinic) map[string]any {
	return map[string]any{
		"id":              c.ID,
		"name":            c.Name,
		"district":        c.District,
		"side":            c.Side,
		"segment":         c.Segment,
		"guarantee":       c.GuaranteedApptsPerMonth,
		"dailyCapacity":   c.DailyCapacity,
		"monthlyAdBudget": c.MonthlyAdBudget,
	}
}

// scopedClinics returns the clinics a user may see (admins / dev-open: all).
func (s *Server) scopedClinics(u *domain.User) []map[string]any {
	all := s.eng.Clinics()
	out := make([]map[string]any, 0, len(all))
	for _, c := range all {
		if auth.CanAccessClinic(u, c.ID) {
			out = append(out, clinicView(c))
		}
	}
	return out
}

// newID returns a short random hex id (16 hex chars).
func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// handleRegister creates an account. Admin-only when authenticated; in dev-open
// mode (no user in ctx) it is allowed so the first admin can bootstrap — though the
// seeded admin (cmd/brain) is the normal path.
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if u, ok := auth.UserFrom(r.Context()); ok && u != nil && u.Role != domain.RoleAdmin {
		writeJSON(w, 403, map[string]string{"error": "forbidden"})
		return
	}
	if s.store == nil {
		writeJSON(w, 503, map[string]string{"error": "store not configured"})
		return
	}
	var b struct {
		Email     string   `json:"email"`
		Password  string   `json:"password"`
		Name      string   `json:"name"`
		Role      string   `json:"role"`
		ClinicIDs []string `json:"clinicIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	b.Email = strings.TrimSpace(strings.ToLower(b.Email))
	if b.Email == "" || len(b.Password) < 8 {
		writeJSON(w, 400, map[string]string{"error": "email and password (min 8 chars) required"})
		return
	}
	role := domain.Role(b.Role)
	if role != domain.RoleAdmin && role != domain.RoleClinic {
		role = domain.RoleClinic
	}
	hash, err := auth.HashPassword(b.Password)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "hash failed"})
		return
	}
	user := domain.User{
		ID:           "user-" + newID(),
		Email:        b.Email,
		Name:         b.Name,
		PasswordHash: hash,
		Role:         role,
		ClinicIDs:    b.ClinicIDs,
		CreatedAt:    time.Now(),
	}
	if err := s.store.CreateUser(user); err != nil {
		if errors.Is(err, store.ErrDuplicate) {
			writeJSON(w, 409, map[string]string{"error": "email_taken"})
			return
		}
		writeJSON(w, 500, map[string]string{"error": "create failed"})
		return
	}
	writeJSON(w, 201, map[string]any{"user": publicUser(user)})
}

// handleLogin verifies credentials and returns {token, user, clinics} — the exact
// shape the Auth.js Credentials provider consumes.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if s.store == nil || s.authn == nil {
		writeJSON(w, 503, map[string]string{"error": "auth not configured"})
		return
	}
	var b struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	u, ok := s.store.GetUserByEmail(strings.ToLower(strings.TrimSpace(b.Email)))
	// Constant generic error avoids leaking whether the email exists.
	if !ok || !auth.VerifyPassword(u.PasswordHash, b.Password) {
		writeJSON(w, 401, map[string]string{"error": "invalid_credentials"})
		return
	}
	token, err := s.authn.Sign(auth.Claims{
		Sub: u.ID, Email: u.Email, Role: string(u.Role), ClinicIDs: u.ClinicIDs,
	})
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "token sign failed"})
		return
	}
	writeJSON(w, 200, map[string]any{
		"token":   token,
		"user":    publicUser(u),
		"clinics": s.scopedClinics(&u),
	})
}

// handleMe returns the current user + their clinics, reloaded for freshness.
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFrom(r.Context())
	if !ok || u == nil {
		writeJSON(w, 401, map[string]string{"error": "unauthorized"})
		return
	}
	full := *u
	if s.store != nil {
		if fresh, ok := s.store.GetUserByID(u.ID); ok {
			full = fresh
		}
	}
	writeJSON(w, 200, map[string]any{
		"user":    publicUser(full),
		"clinics": s.scopedClinics(&full),
	})
}

// handleRefresh issues a fresh token if the presented one is valid and unexpired,
// reloading role/membership from the store. (Path is auth-exempt; it reads the
// token itself.)
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if s.authn == nil {
		writeJSON(w, 503, map[string]string{"error": "auth not configured"})
		return
	}
	c, err := s.authn.Parse(auth.BearerToken(r))
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": "invalid_token"})
		return
	}
	role, clinicIDs := c.Role, c.ClinicIDs
	if s.store != nil {
		if u, ok := s.store.GetUserByID(c.Sub); ok {
			role, clinicIDs = string(u.Role), u.ClinicIDs
		}
	}
	token, err := s.authn.Sign(auth.Claims{Sub: c.Sub, Email: c.Email, Role: role, ClinicIDs: clinicIDs})
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "token sign failed"})
		return
	}
	writeJSON(w, 200, map[string]any{"token": token})
}

// handleLogout is a stateless no-op (tokens expire on their own; the Next.js
// session is cleared client-side). Exists for symmetry and future revocation.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
