package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Claims is the JWT payload. ClinicIDs is embedded so the middleware can scope a
// request to the user's clinics without a per-request store lookup.
type Claims struct {
	Sub       string   `json:"sub"` // user ID
	Email     string   `json:"email"`
	Role      string   `json:"role"`
	ClinicIDs []string `json:"clinicIds"`
	Iat       int64    `json:"iat"`
	Exp       int64    `json:"exp"`
}

// Authenticator signs and verifies HS256 tokens. Hand-rolled on the stdlib so the
// default build pulls in no JWT library (rule #9).
type Authenticator struct {
	secret []byte
	ttl    time.Duration
}

// NewAuthenticator builds an Authenticator with the given signing secret and token TTL.
func NewAuthenticator(secret []byte, ttl time.Duration) *Authenticator {
	return &Authenticator{secret: secret, ttl: ttl}
}

// TTL reports the configured access-token lifetime.
func (a *Authenticator) TTL() time.Duration { return a.ttl }

var b64 = base64.RawURLEncoding

// Sign issues a token for the given claims, stamping iat/exp from now.
func (a *Authenticator) Sign(c Claims) (string, error) {
	now := time.Now()
	c.Iat = now.Unix()
	c.Exp = now.Add(a.ttl).Unix()
	header := b64.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	signing := header + "." + b64.EncodeToString(payload)
	return signing + "." + b64.EncodeToString(a.mac(signing)), nil
}

// Parse verifies the signature and expiry and returns the claims.
func (a *Authenticator) Parse(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("malformed token")
	}
	sig, err := b64.DecodeString(parts[2])
	if err != nil {
		return Claims{}, err
	}
	if subtle.ConstantTimeCompare(sig, a.mac(parts[0]+"."+parts[1])) != 1 {
		return Claims{}, errors.New("bad signature")
	}
	payload, err := b64.DecodeString(parts[1])
	if err != nil {
		return Claims{}, err
	}
	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return Claims{}, err
	}
	if time.Now().Unix() >= c.Exp {
		return Claims{}, errors.New("token expired")
	}
	return c, nil
}

func (a *Authenticator) mac(msg string) []byte {
	h := hmac.New(sha256.New, a.secret)
	h.Write([]byte(msg))
	return h.Sum(nil)
}
