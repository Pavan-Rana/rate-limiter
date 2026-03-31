//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// TestFailOpen verifies the service allows requests when Redis is unavailable.
// Fail-open behaviour is a deliberate design choice - documented in docs/tradeoffs.md.
//
// To run manually:
//  1. Stop Redis: docker stop <redis-container>
//  2. Run: go test -tags=integration -run TestFailOpen -v ./tests/integration/...
//  3. Expect: 200 OK with X-Rate-Limit-Status: fail-open
//  4. Restart Redis and verify normal operation resumes
func TestFailOpen(t *testing.T) {
	t.Skip("Run manually: stop Redis first, then unskip and run")

	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost,
		serviceAddr()+"/check", nil)
	req.Header.Set("X-API-Key", "fail-open-test")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Fail-open: expected 200, got %d", resp.StatusCode)
	}
}

// TestHealthAfterRedisRestart verifies the service recovers and resumes
// correct rate-limit decisions after Redis restarts.
//
// To run manually:
//  1. Restart Redis: docker restart <redis-container>
//  2. Run this test immediately after
func TestHealthAfterRedisRestart(t *testing.T) {
	t.Skip("Run manually after restarting Redis")

	client := &http.Client{Timeout: 10 * time.Second}

	// Poll until the service is healthy again (up to 30s)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost,
			serviceAddr()+"/check", nil)
		req.Header.Set("X-API-Key", "recovery-test")

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			t.Log("Service recovered successfully")
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Error("Service did not recover within 30s after Redis restart")
}

// TestRollingDeployment documents the manual procedure for verifying
// zero dropped requests during a Kubernetes rolling update.
//
// Procedure:
//  1. Start a background load test: k6 run tests/load/k6_ramp.js
//  2. In a second terminal: kubectl rollout restart deployment/rate-limiter
//  3. Observe k6 output — http_req_failed rate should remain < 0.1%
//  4. Verify p99 latency spike stays under 50ms during rollover
func TestRollingDeployment(t *testing.T) {
	t.Skip("Manual test - see procedure in comment above")
}
