// Package auth is a minimal API-key gate for the /v1 management endpoints (the
// surface your Next.js clinic dashboard calls). Webhooks and the health/UI
// routes stay public. If no key is configured it's a no-op (dev mode). Swap for
// per-tenant JWT when you add real multi-tenant accounts.
package auth

import (
	"net/http"
	"strings"
)

// Middleware returns a wrapper that requires X-API-Key (or Bearer) == apiKey on
// protected paths. apiKey == "" disables the check (dev).
func Middleware(apiKey string, protect func(path string) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKey == "" || !protect(r.URL.Path) || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			if presented(r) == apiKey {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
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

// ProtectV1 protects /v1/* management routes but leaves webhooks/health/UI open.
func ProtectV1(path string) bool {
	return strings.HasPrefix(path, "/v1/")
}
