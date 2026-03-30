package store

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Pavan-Rana/rate-limiter/internal/limiter"
)

func newTestStore(t *testing.T) (*RedisStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(mr.Addr())
	require.NoError(t, err)
	return store, mr
}

func policy(limit int, window time.Duration) limiter.Policy {
	return limiter.Policy{Limit: limit, Window: window}
}

func TestNewRedisStore_UnreachableServer(t *testing.T) {
	store, err := NewRedisStore("127.0.0.1:1")
	assert.Error(t, err)
	assert.Nil(t, store)
}

func TestPing_AfterServerDown(t *testing.T) {
	store, mr := newTestStore(t)
	mr.Close()
	assert.Error(t, store.Ping(context.Background()))
}

func TestAllowRequest_WithinAndBeyondLimit(t *testing.T) {
	store, _ := newTestStore(t)
	p := policy(3, time.Minute)
	ctx := context.Background()

	for i := 0; i < p.Limit; i++ {
		allowed, err := store.AllowRequest(ctx, "client:limit", p)
		require.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed", i+1)
	}

	allowed, err := store.AllowRequest(ctx, "client:limit", p)
	require.NoError(t, err)
	assert.False(t, allowed, "request beyond limit should be denied")
}

func TestAllowRequest_WindowExpiry_ResetsCount(t *testing.T) {
	store, mr := newTestStore(t)
	window := 100 * time.Millisecond
	p := policy(1, window)
	ctx := context.Background()

	_, err := store.AllowRequest(ctx, "client:expiry", p)
	require.NoError(t, err)

	mr.FastForward(window + 10*time.Millisecond)

	allowed, err := store.AllowRequest(ctx, "client:expiry", p)
	require.NoError(t, err)
	assert.True(t, allowed, "should be allowed after window expiry")
}

func TestAllowRequest_DifferentKeysAreIsolated(t *testing.T) {
	store, _ := newTestStore(t)
	p := policy(1, time.Minute)
	ctx := context.Background()

	_, err := store.AllowRequest(ctx, "client:A", p)
	require.NoError(t, err)

	allowed, err := store.AllowRequest(ctx, "client:B", p)
	require.NoError(t, err)
	assert.True(t, allowed, "different keys should not share rate-limit state")
}

func TestAllowRequest_CancelledContext(t *testing.T) {
	store, _ := newTestStore(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.AllowRequest(ctx, "client:ctx", policy(10, time.Minute))
	assert.Error(t, err)
}
