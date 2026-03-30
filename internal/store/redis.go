package store

import (
	"context"
	"time"

	"github.com/Pavan-Rana/rate-limiter/internal/limiter"
	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client  *redis.Client
	scripts *LuaScripts
}

func NewRedisStore(addr string) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	scripts, err := LoadScripts(ctx, client)
	if err != nil {
		return nil, err
	}

	return &RedisStore{client: client, scripts: scripts}, nil
}

// Single round-trip; no TOCTOU race possible across replicas.
func (r *RedisStore) AllowRequest(ctx context.Context, key string, policy limiter.Policy) (bool, error) {
	nowMs := time.Now().UnixMilli()
	windowMs := policy.Window.Milliseconds()

	result, err := r.client.EvalSha(ctx, r.scripts.SlidingWindowSHA, []string{key},
		nowMs, windowMs, policy.Limit,
	).Int()

	if err != nil {
		return false, err
	}

	return result == 1, nil
}

func (r *RedisStore) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
