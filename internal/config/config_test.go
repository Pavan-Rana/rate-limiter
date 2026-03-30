package config

import (
	"os"
	"testing"
	"time"

	"github.com/Pavan-Rana/rate-limiter/internal/limiter"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear env to test fallbacks
	os.Unsetenv("REDIS_ADDR")
	os.Unsetenv("GRPC_ADDR")
	os.Unsetenv("HTTP_ADDR")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.RedisAddr != "localhost:6379" {
		t.Errorf("expected RedisAddr localhost:6379, got %s", cfg.RedisAddr)
	}

	if cfg.GRPCAddr != ":50051" {
		t.Errorf("expected GRPCAddr :50051, got %s", cfg.GRPCAddr)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("expected HTTPAddr :8080, got %s", cfg.HTTPAddr)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	os.Setenv("REDIS_ADDR", "redis:1234")
	os.Setenv("GRPC_ADDR", ":9999")
	os.Setenv("HTTP_ADDR", ":7777")

	defer os.Unsetenv("REDIS_ADDR")
	defer os.Unsetenv("GRPC_ADDR")
	defer os.Unsetenv("HTTP_ADDR")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.RedisAddr != "redis:1234" {
		t.Errorf("expected RedisAddr redis:1234, got %s", cfg.RedisAddr)
	}

	if cfg.GRPCAddr != ":9999" {
		t.Errorf("expected GRPCAddr :9999, got %s", cfg.GRPCAddr)
	}

	if cfg.HTTPAddr != ":7777" {
		t.Errorf("expected HTTPAddr :7777, got %s", cfg.HTTPAddr)
	}
}

func TestGetDefaultPolicy(t *testing.T) {
	cfg, _ := Load()

	p := cfg.GetDefault()

	if p.Limit != 100 {
		t.Errorf("expected limit 100, got %d", p.Limit)
	}

	if p.Window != time.Minute {
		t.Errorf("expected window 1 minute, got %v", p.Window)
	}
}

func TestPolicyFor_NotFound(t *testing.T) {
	cfg, _ := Load()

	_, ok := cfg.PolicyFor("nonexistent")

	if ok {
		t.Errorf("expected policy to not exist")
	}
}

func TestPolicyFor_Found(t *testing.T) {
	cfg, _ := Load()

	testPolicy := limiter.Policy{
		Limit:  50,
		Window: time.Second * 30,
	}

	cfg.policies["test-key"] = testPolicy

	p, ok := cfg.PolicyFor("test-key")
	if !ok {
		t.Fatalf("expected policy to exist")
	}

	if p.Limit != 50 {
		t.Errorf("expected limit 50, got %d", p.Limit)
	}

	if p.Window != 30*time.Second {
		t.Errorf("expected window 30s, got %v", p.Window)
	}
}
