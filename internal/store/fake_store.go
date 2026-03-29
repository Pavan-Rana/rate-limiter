package store

import (
	"context"
	"sync"
	"time"

	"github.com/Pavan-Rana/rate-limiter/internal/limiter"
)

// FakeStore is an in-memory store for unit tests.
// Mirrors the sliding window logic of the Lua script without requiring Redis.
type FakeStore struct {
	mu    		sync.Mutex
	requests 	map[string][]time.Time
}

func NewFakeStore() *FakeStore {
	return &FakeStore{requests: make(map[string][]time.Time)}
}

func ( f *FakeStore) AllowRequest(_ context.Context, key string, policy limiter.Policy) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-policy.Window)

	valid := f.requests[key][:0]
	for _, t := range f.requests[key] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	f.requests[key] = valid

	if len(f.requests[key]) >= policy.Limit {
		return false, nil
	}

	f.requests[key] = append(f.requests[key], now)
	return true, nil
}


func (f *FakeStore) Ping(_ context.Context) error {return nil}