package limiter_test

import (
	"context"
	"testing"
	"time"

	"github.com/Pavan-Rana/rate-limiter/internal/limiter"
)

type fakeStore struct {
	allowResult bool
	callCount   int
}

func (f *fakeStore) AllowRequest(_ context.Context, _ string, _ limiter.Policy) (bool, error) {
	f.callCount++
	return f.allowResult, nil
}

func (f *fakeStore) Ping(_ context.Context) error { return nil }

type fakeConfig struct {}

func(f *fakeConfig) PolicyFor(_ string) (limiter.Policy, bool) { return limiter.Policy{}, false }
func(f *fakeConfig) GetDefault() limiter.Policy { return limiter.Policy{Limit: 10, Window: time.Minute} }

func TestAllowRequest_Allowed(t *testing.T) {
	store := &fakeStore{allowResult: true}
	lim := limiter.New(store, &fakeConfig{})

	allowed, err := lim.AllowRequest(context.Background(), "test-key")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !allowed {
		t.Fatalf("Expected request to be allowed")
	}
}

func TestAllowRequest_Rejected(t *testing.T) {
	store := &fakeStore{allowResult: false}
	lim := limiter.New(store, &fakeConfig{})

	allowed, err := lim.AllowRequest(context.Background(), "test-key")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if allowed {
		t.Fatalf("Expected request to be rejected")
	}
}