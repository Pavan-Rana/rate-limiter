package config

import (
	"os"
	"time"

	"github.com/Pavan-Rana/rate-limiter/internal/limiter"
)

type Config struct {
	RedisAddr     string
	GRPCAddr      string
	HTTPAddr      string
	defaultPolicy limiter.Policy
	policies      map[string]limiter.Policy // per-API-key overrides
}

func Load() (*Config, error) {
	return &Config{
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
		GRPCAddr:  getEnv("GRPC_ADDR", ":50051"),
		HTTPAddr:  getEnv("HTTP_ADDR", ":8080"),
		defaultPolicy: limiter.Policy{
			Limit:  100,
			Window: time.Minute,
		},
		policies: make(map[string]limiter.Policy),
	}, nil
}

func (c *Config) PolicyFor(apiKey string) (limiter.Policy, bool) {
	p, ok := c.policies[apiKey]
	return p, ok
}

func (c *Config) GetDefault() limiter.Policy {
	return c.defaultPolicy
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
