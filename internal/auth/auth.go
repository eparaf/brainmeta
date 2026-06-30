// Package auth gates the /v1 management endpoints (the surface the Next.js clinic
// dashboard calls). It accepts EITHER a dashboard user's JWT (preferred) or the
// shared BRAIN_API_KEY (back-compat for the embedded console / service callers).
// Webhooks and the health/UI routes stay public. When neither a key is configured
// nor auth is required, /v1 is dev-open — but a presented JWT still populates the
// request context so handlers can scope by clinic.
package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"disci/brain/internal/domain"
)

// Middleware gates protected paths. enforce := requireAuth || apiKey != "".
//   - A presented X-API-Key/Bearer equal to apiKey → synthetic admin (full access).
//   - Otherwise a valid Bearer JWT → the decoded user is injected into the context.
//   - No valid credential: 401 when enforcing, else dev-open passthrough.
func (a *Authenticator) Middleware(apiKey string, requireAuth bool, protect func(path string) bool) func(http.Handler) http.Handler {
	enforce := requireAuth || apiKey != ""
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !protect(r.URL.Path) || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			tok := presented(r)
			// 1) Service/console API key → synthetic admin.
			if apiKey != "" && subtle.ConstantTimeCompare([]byte(tok), []byte(apiKey)) == 1 {
				ctx := WithUser(r.Context(), &domain.User{ID: "service", Role: domain.RoleAdmin})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			// 2) Dashboard user JWT.
			if a != nil && tok != "" {
				if c, err := a.Parse(tok); err == nil {
					u := &domain.User{ID: c.Sub, Email: c.Email, Role: domain.Role(c.Role), ClinicIDs: c.ClinicIDs}
					next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), u)))
					return
				}
			}
			// 3) No valid credential.
			if enforce {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}
			next.ServeHTTP(w, r) // dev-open
		})
	}
}

func presented(r *http.Request) string {
	if k := r.Header.Get("X-API-Key"); k != "" {
		return k
	}
	if a := r.Header.Get("Authorization"); strings.HasPrefix(a, "Bearer ") {
		return strings.TrimPrefix(a, "Bearer ")
	}
	return ""
}

// BearerToken exposes the presented X-API-Key / Bearer token for handlers on
// auth-exempt paths (e.g. /v1/auth/refresh) that need to read it themselves.
func BearerToken(r *http.Request) string { return presented(r) }

// ProtectV1 protects /v1/* management routes, EXCEPT the auth bootstrap endpoints
// (login/register/refresh) — otherwise you could never obtain a token. Webhooks,
// health, metrics and the embedded UI are not under /v1 and stay public.
func ProtectV1(path string) bool {
	switch path {
	case "/v1/auth/login", "/v1/auth/register", "/v1/auth/refresh":
		return false
	}
	return strings.HasPrefix(path, "/v1/")
}
