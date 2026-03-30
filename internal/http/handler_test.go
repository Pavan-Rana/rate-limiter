package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	handler "github.com/Pavan-Rana/rate-limiter/internal/http"
)

// mockLimiter implements the Limiter interface for testing
type mockLimiter struct {
	allow bool
	err   error
}

func (m *mockLimiter) AllowRequest(ctx context.Context, apiKey string) (bool, error) {
	return m.allow, m.err
}

func TestCheckHandler(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		apiKey      string
		allow       bool
		wantStatus  int
		wantAllowed bool
	}{
		{
			name:        "allowed",
			method:      stdhttp.MethodPost,
			apiKey:      "abc123",
			allow:       true,
			wantStatus:  stdhttp.StatusOK,
			wantAllowed: true,
		},
		{
			name:        "rate limit exceeded",
			method:      stdhttp.MethodPost,
			apiKey:      "abc123",
			allow:       false,
			wantStatus:  stdhttp.StatusTooManyRequests,
			wantAllowed: false,
		},
		{
			name:       "missing API key",
			method:     stdhttp.MethodPost,
			apiKey:     "",
			wantStatus: stdhttp.StatusBadRequest,
		},
		{
			name:       "wrong method",
			method:     stdhttp.MethodGet,
			apiKey:     "abc123",
			wantStatus: stdhttp.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockLimiter{allow: tt.allow}
			router := handler.NewRouter(mock)

			req := httptest.NewRequest(tt.method, "/check", bytes.NewReader([]byte{}))
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status code = %v, want %v", rr.Code, tt.wantStatus)
			}

			// Only decode JSON for POST requests with API key
			if tt.method == stdhttp.MethodPost && tt.apiKey != "" {
				var body map[string]any
				if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if body["allowed"] != tt.wantAllowed {
					t.Errorf("allowed = %v, want %v", body["allowed"], tt.wantAllowed)
				}
				if body["api_key"] != tt.apiKey {
					t.Errorf("api_key = %v, want %v", body["api_key"], tt.apiKey)
				}
			}
		})
	}
}

func TestMetricsEndpoint(t *testing.T) {
	mock := &mockLimiter{allow: true}
	router := handler.NewRouter(mock)

	req := httptest.NewRequest(stdhttp.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != stdhttp.StatusOK {
		t.Errorf("/metrics status code = %v, want %v", rr.Code, stdhttp.StatusOK)
	}
	if rr.Body.Len() == 0 {
		t.Error("/metrics returned empty body")
	}
}
