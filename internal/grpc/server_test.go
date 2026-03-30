package grpc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Pavan-Rana/rate-limiter/internal/grpc"
	pb "github.com/Pavan-Rana/rate-limiter/proto"
)

// mockLimiter implements limiter.Limiter interface for testing
type mockLimiter struct {
	allow bool
	err   error
}

func (m *mockLimiter) AllowRequest(ctx context.Context, apiKey string) (bool, error) {
	return m.allow, m.err
}

func TestServer_AllowRequest(t *testing.T) {
	tests := []struct {
		name       string
		allow      bool
		err        error
		wantAllow  bool
		wantReason string
	}{
		{
			name:       "allowed",
			allow:      true,
			err:        nil,
			wantAllow:  true,
			wantReason: "allowed",
		},
		{
			name:       "rate limit exceeded",
			allow:      false,
			err:        nil,
			wantAllow:  false,
			wantReason: "rate limit exceeded",
		},
		{
			name:       "fail-open on error",
			allow:      false, // limiter return ignored on error
			err:        errors.New("redis down"),
			wantAllow:  true,
			wantReason: "fail-open: redis down",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockLimiter{
				allow: tt.allow,
				err:   tt.err,
			}
			srv := grpc.New(mock)

			req := &pb.AllowRequestMessage{ApiKey: "test-key"}
			resp, err := srv.AllowRequest(context.Background(), req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp.Allowed != tt.wantAllow {
				t.Errorf("Allowed = %v, want %v", resp.Allowed, tt.wantAllow)
			}
			if resp.Reason != tt.wantReason {
				t.Errorf("Reason = %q, want %q", resp.Reason, tt.wantReason)
			}
		})
	}
}
