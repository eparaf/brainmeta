// Package session stores in-flight conversation state behind an interface so it
// can move OUT of process memory for horizontal scale. Default is in-memory
// (single instance). For multi-instance, implement Store with Redis (Get =
// JSON GET, Put = JSON SET with TTL) — a ~30-line drop-in; nothing else changes.
//
// Note on scale: the conversational FRONT-END (webhooks/sessions) scales
// horizontally with a shared session Store. The LEARNING BRAIN (posteriors,
// budget bandit) is a single-writer by design — scale it vertically or SHARD by
// clinic; don't run two writers against the same clinic's posteriors.
package session

import (
	"sync"

	"disci/brain/internal/agent"
)

// Store persists per-conversation sessions, keyed by phone.
type Store interface {
	Get(id string) (*agent.Session, bool)
	Put(id string, s *agent.Session)
}

// RedisFactory builds a Redis-backed Store. It is nil unless the binary is built
// with `-tags redis` (see redis.go), which registers it in init(). main checks
// it when REDIS_URL is set, so the default build stays dependency-free.
var RedisFactory func(url string) (Store, error)

// Memory is the default in-process Store.
type Memory struct {
	mu sync.RWMutex
	m  map[string]*agent.Session
}

func NewMemory() *Memory { return &Memory{m: map[string]*agent.Session{}} }

func (s *Memory) Get(id string) (*agent.Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[id]
	return v, ok
}

func (s *Memory) Put(id string, sess *agent.Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[id] = sess
}
