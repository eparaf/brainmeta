package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// Iterations is the PBKDF2 work factor. It's a var (not const) so tests can lower
// it — 600k is the OWASP 2024 floor for PBKDF2-HMAC-SHA256 and is deliberately slow.
var Iterations = 600_000

// HashPassword returns a self-describing PHC-style string:
//
//	pbkdf2$sha256$<iter>$<base64 salt>$<base64 hash>
//
// Stdlib-only (crypto/pbkdf2 landed in Go 1.24) so the default build stays
// dependency-free — bcrypt would pull in golang.org/x/crypto (rule #9).
func HashPassword(plain string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	dk, err := pbkdf2.Key(sha256.New, plain, salt, Iterations, 32)
	if err != nil {
		return "", err
	}
	enc := base64.RawStdEncoding.EncodeToString
	return fmt.Sprintf("pbkdf2$sha256$%d$%s$%s", Iterations, enc(salt), enc(dk)), nil
}

// VerifyPassword constant-time compares plain against a stored PHC string. Returns
// false on any malformed input rather than erroring.
func VerifyPassword(stored, plain string) bool {
	p := strings.Split(stored, "$") // ["pbkdf2","sha256",iter,salt,hash]
	if len(p) != 5 || p[0] != "pbkdf2" || p[1] != "sha256" {
		return false
	}
	iter, err := strconv.Atoi(p[2])
	if err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(p[3])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(p[4])
	if err != nil {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, plain, salt, iter, len(want))
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}
