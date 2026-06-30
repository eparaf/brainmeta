package auth

import (
	"context"

	"disci/brain/internal/domain"
)

type ctxKey int

const userKey ctxKey = 0

// WithUser attaches the authenticated user to the request context.
func WithUser(ctx context.Context, u *domain.User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// UserFrom returns the authenticated user, if any. In dev-open mode (no creds
// required and none presented) this returns (nil, false) and handlers treat the
// caller as full-access so the embedded console keeps working.
func UserFrom(ctx context.Context) (*domain.User, bool) {
	u, ok := ctx.Value(userKey).(*domain.User)
	return u, ok
}

// CanAccessClinic enforces tenant scoping: admins (and the unauthenticated
// dev-open / service caller) see everything; clinic users only their memberships.
func CanAccessClinic(u *domain.User, clinicID string) bool {
	if u == nil {
		return true // dev-open / service caller
	}
	if u.Role == domain.RoleAdmin {
		return true
	}
	for _, id := range u.ClinicIDs {
		if id == clinicID {
			return true
		}
	}
	return false
}
