package auth

import (
	"testing"
	"time"

	"disci/brain/internal/domain"
)

// Keep PBKDF2 cheap in tests — the production work factor (600k) would slow the
// suite. Iterations is a package var precisely so tests can do this.
func init() { Iterations = 4096 }

func TestPasswordRoundTrip(t *testing.T) {
	h, err := HashPassword("s3cret-pw")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(h, "s3cret-pw") {
		t.Fatal("correct password rejected")
	}
	if VerifyPassword(h, "wrong") {
		t.Fatal("wrong password accepted")
	}
	if VerifyPassword("garbage", "x") {
		t.Fatal("malformed hash accepted")
	}
}

func TestJWTSignParse(t *testing.T) {
	a := NewAuthenticator([]byte("test-secret"), time.Hour)
	tok, err := a.Sign(Claims{Sub: "user-1", Email: "a@b.c", Role: "admin", ClinicIDs: []string{"x", "y"}})
	if err != nil {
		t.Fatal(err)
	}
	c, err := a.Parse(tok)
	if err != nil {
		t.Fatal(err)
	}
	if c.Sub != "user-1" || c.Role != "admin" || len(c.ClinicIDs) != 2 {
		t.Fatalf("claims round-trip mismatch: %+v", c)
	}
}

func TestJWTExpired(t *testing.T) {
	a := NewAuthenticator([]byte("k"), -time.Second) // already expired
	tok, _ := a.Sign(Claims{Sub: "u"})
	if _, err := a.Parse(tok); err == nil {
		t.Fatal("expected expired token to fail")
	}
}

func TestJWTTampered(t *testing.T) {
	a := NewAuthenticator([]byte("k"), time.Hour)
	tok, _ := a.Sign(Claims{Sub: "u"})
	other := NewAuthenticator([]byte("different"), time.Hour)
	if _, err := other.Parse(tok); err == nil {
		t.Fatal("expected signature mismatch under a different secret")
	}
}

func TestCanAccessClinic(t *testing.T) {
	admin := &domain.User{Role: domain.RoleAdmin}
	if !CanAccessClinic(admin, "any") {
		t.Fatal("admin should access any clinic")
	}
	clinic := &domain.User{Role: domain.RoleClinic, ClinicIDs: []string{"a"}}
	if !CanAccessClinic(clinic, "a") || CanAccessClinic(clinic, "b") {
		t.Fatal("clinic scoping wrong")
	}
	if !CanAccessClinic(nil, "a") {
		t.Fatal("dev-open (nil) should be allowed")
	}
}
