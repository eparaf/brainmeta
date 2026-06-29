// Package consent is the KVKK/GDPR guardrail: track who has opted out and never
// message them again. WhatsApp/Meta also require an honoured opt-out. Inbound
// "STOP/DUR" messages opt the contact out; everything checks Allowed() before
// sending. In-memory here; back it with Postgres in production.
package consent

import (
	"strings"
	"sync"
)

type Store struct {
	mu      sync.RWMutex
	optedOut map[string]bool
}

func NewStore() *Store { return &Store{optedOut: map[string]bool{}} }

// IsOptOutKeyword reports whether a message is an opt-out request (TR + EN).
func IsOptOutKeyword(msg string) bool {
	m := strings.ToLower(strings.TrimSpace(msg))
	switch m {
	case "stop", "dur", "iptal", "çık", "cik", "unsubscribe", "abonelikten çık":
		return true
	}
	return false
}

func (s *Store) OptOut(phone string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.optedOut[phone] = true
}

func (s *Store) OptIn(phone string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.optedOut, phone)
}

// Allowed reports whether we may message this contact.
func (s *Store) Allowed(phone string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.optedOut[phone]
}
