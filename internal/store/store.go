// Package store abstracts the Redis backend behind an interface.
package store

import (
	"context"
	"github.com/Pavan-Rana/rate-limiter/internal/limiter"
)

// Store is the persistence interface for rate-limit state.
type Store interface {
	// AllowRequest atomically checks and records a request for the given key.
	// Returns true if the request is within policy limits.
	AllowRequest(ctx context.Context, key string, policy limiter.Policy) (bool, error)

	// Ping checks connectivity to the backing store.
	Ping(ctx context.Context) error
}