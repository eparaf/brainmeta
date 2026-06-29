//go:build redis

// Build with: go get github.com/redis/go-redis/v9 && go build -tags redis ./...
// Activates a Redis-backed session Store so multiple API/webhook instances share
// conversation state (horizontal scale). Registered via init() so the default
// (untagged) build needs no Redis dependency.
package session

import (
	"context"
	"encoding/json"
	"time"

	"disci/brain/internal/agent"
	"github.com/redis/go-redis/v9"
)

func init() {
	RedisFactory = func(url string) (Store, error) {
		opt, err := redis.ParseURL(url)
		if err != nil {
			return nil, err
		}
		return &redisStore{c: redis.NewClient(opt), ttl: 24 * time.Hour}, nil
	}
}

type redisStore struct {
	c   *redis.Client
	ttl time.Duration
}

func (s *redisStore) key(id string) string { return "sess:" + id }

func (s *redisStore) Get(id string) (*agent.Session, bool) {
	b, err := s.c.Get(context.Background(), s.key(id)).Bytes()
	if err != nil {
		return nil, false
	}
	var sess agent.Session
	if json.Unmarshal(b, &sess) != nil {
		return nil, false
	}
	return &sess, true
}

func (s *redisStore) Put(id string, sess *agent.Session) {
	if b, err := json.Marshal(sess); err == nil {
		s.c.Set(context.Background(), s.key(id), b, s.ttl)
	}
}
