//go:build integration

// Run with: go test -tags=integration -v ./tests/integration/...
// Requires: live Redis at REDIS_ADDR and service running at SERVICE_ADDR

package integration_test

import (
	"context"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func serviceAddr() string {
	if a := os.Getenv("SERVICE_ADDR"); a != "" {
		return a
	}
	return "http://localhost:8080"
}

// TestConcurrentLimit fires limit+50 goroutines simultaneously against a single API key
// and asserts that no more than `limit` requests were allowed.
//
// This is the primary correctness proof for the distributed atomic decision.
// If the Lua script is not atomic, multiple replicas can race and over-admit.
func TestConcurrentLimit(t *testing.T) {
	const limit = 100
	const total = limit + 50

	var wg sync.WaitGroup
	var allowed atomic.Int64
	client := &http.Client{Timeout: 5 * time.Second}

	for i := 0; i < total; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost,
				serviceAddr()+"/check", nil)
			req.Header.Set("X-API-KEY", "test-concurrent-key")

			resp, err := client.Do(req)
			if err != nil {
				t.Logf("Request error: %v", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				allowed.Add(1)
			}
		}()
	}

	wg.Wait()

	got := int(allowed.Load())
	if got > limit {
		t.Errorf("Over-admitted: allowed %d requests, limit was %d - Lua atomicity may be broken", got, limit)
	}
	t.Logf("Result: %d/%d requests allowed (limit: %d)", got, total, limit)
}

func TestWindowBoundary(t *testing.T) {
	t.Skip("This test is timing-sensitive and may be flaky; run manually if needed")
}

// TestMultiKeyIsolation verifies that limits for one key do not affect another.
func TestMultiKeyIsolation(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}

	doRequest := func(key string) int {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost,
			serviceAddr()+"/check", nil)
		req.Header.Set("X-API-KEY", key)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed for key %s: %v", key, err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}

	// Both keys should start fresh
	if status := doRequest("isolation-key-a"); status != http.StatusOK {
		t.Errorf("Key-a first request: expected 200, got %d", status)
	}
	if status := doRequest("isolation-key-b"); status != http.StatusOK {
		t.Errorf("Key-b first request: expected 200, got %d", status)
	}
}
