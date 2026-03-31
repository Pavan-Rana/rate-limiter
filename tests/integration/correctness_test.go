//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestExactLimitBoundary(t *testing.T) {
	const limit = 100
	client := &http.Client{Timeout: 5 * time.Second}
	apiKey := "test-boundary-sequential"

	for i := 1; i <= limit; i++ {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost,
			serviceAddr()+"/check", nil)
		req.Header.Set("X-API-Key", apiKey)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("request %d/%d: expected 200, got %d", i, limit, resp.StatusCode)
		}
	}

	// (limit+1)th should be rejected
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost,
		serviceAddr()+"/check", nil)
	req.Header.Set("X-API-Key", apiKey)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("over-limit request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("over-limit request: expected 429, got %d", resp.StatusCode)
	}
}

func TestBurstExhaustion(t *testing.T) {
	const limit = 100
	const burst = 200
	client := &http.Client{Timeout: 2 * time.Second}
	apiKey := "test-burst-key"

	allowed := 0
	for i := 0; i < burst; i++ {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost,
			serviceAddr()+"/check", nil)
		req.Header.Set("X-API-Key", apiKey)

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			allowed++
		}
	}

	if allowed > limit {
		t.Errorf("burst admitted %d requests, limit is %d", allowed, limit)
	}
	t.Logf("burst result: %d/%d allowed (limit: %d)", allowed, burst, limit)
}

func TestMissingAPIKey(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost,
		serviceAddr()+"/check", nil)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing key, got %d", resp.StatusCode)
	}
}
