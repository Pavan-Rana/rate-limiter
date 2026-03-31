package integration_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Pavan-Rana/rate-limiter/internal/limiter"
	"github.com/Pavan-Rana/rate-limiter/internal/store"
)

type testConfig struct {
	policy limiter.Policy
}

func (c *testConfig) PolicyFor(apiKey string) (limiter.Policy, bool) {
	return c.policy, true
}

func (c *testConfig) GetDefault() limiter.Policy {
	return c.policy
}

func TestMultiReplica_GlobalLimitEnforced(t *testing.T) {
	ctx := context.Background()
	redisAddr := "localhost:6379"
	apiKey := "test-key"

	policy := limiter.Policy{
		Limit:  50,
		Window: time.Second,
	}

	cfg := &testConfig{policy: policy}
	const replicas = 4
	const totalRequests = 200
	limiters := make([]*limiter.Limiter, replicas)
	for i := 0; i < replicas; i++ {
		store, err := store.NewRedisStore(redisAddr)
		if err != nil {
			t.Fatalf("failed to create redis store: %v", err)
		}
		limiters[i] = limiter.New(store, cfg)
	}

	var allowed int64
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()
			<-start
			l := limiters[i%replicas]

			ok, err := l.AllowRequest(ctx, apiKey)
			if err != nil {
				t.Logf("store error: %v", err)
			}
			if ok {
				atomic.AddInt64(&allowed, 1)
			}
		}(i)
	}

	close(start)
	wg.Wait()

	if allowed > int64(policy.Limit) {
		t.Fatalf("global limit violated: allowed=%d > limit=%d", allowed, policy.Limit)
	}
	t.Logf("allowed=%d (limit=%d)", allowed, policy.Limit)
}
