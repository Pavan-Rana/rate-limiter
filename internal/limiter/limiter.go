// Package limiter is the core domain. AllowRequest is the single decision point.
package limiter

import (
	"context"
	"fmt"
)

// Store is the interface the limiter depends on.
// Defined here to avoid an import cycle with the store package.
type Store interface {
	AllowRequest(ctx context.Context, key string, policy Policy) (bool, error)
	Ping(ctx context.Context) error
}

type Config interface {
	PolicyFor(apiKey string) (Policy, bool)
	GetDefault() Policy
}

type Limiter struct {
	store 	Store
	config 	Config
}

func New(s Store, cfg Config) *Limiter {
	return &Limiter{store: s, config: cfg}
}

// AllowRequest is the single entry point for all rate-limit decisions.
// It delegates the atomic check to the Redis store (Lua script) so decisions
// are serialised on the Redis server - safe across multiple stateless replicas.
func (l *Limiter) AllowRequest(ctx context.Context, apiKey string) (bool, error) {
	policy, ok := l.config.PolicyFor(apiKey)
	if !ok {
		policy = l.config.GetDefault()
	}

	allowed, err := l.store.AllowRequest(ctx, apiKey, policy)
	if err != nil {
		// Fail-open: on Redis failure, allow the request.
		// Tradeoff documented in docs/tradeoffs.md.
		return true, fmt.Errorf("Store error: %w", err)
	}

	return allowed, nil
}