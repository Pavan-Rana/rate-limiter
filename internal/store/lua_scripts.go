package store

import (
	"context"
	"embed"

	"github.com/redis/go-redis/v9"
)

var slidingWindowScript string

var tokenBucketScript string

// LuaScripts holds SHA digests for pre-loaded Lua scripts.
// Using EVALSHA (vs EVAL) avoids re-parsing the script on every request.
type LuaScripts struct {
	SlidingWindowSHA string
	TokenBucketSHA   string
}

// LoadScripts pre-loads Lua scripts into Redis at startup.
// Subsequent calls use EVALSHA with the digest - faster and avoids redundant parsing.
func LoadScripts(ctx context.Context, client *redis.Client) (*LuaScripts, error) {
	swSHA, err := client.ScriptLoad(ctx, slidingWindowScript).Result()
	if err != nil {
		return nil, err
	}

	tbSHA, err := client.ScriptLoad(ctx, tokenBucketScript).Result()
	if err != nil {
		return nil, err
	}

	return &LuaScripts{
		SlidingWindowSHA: swSHA,
		TokenBucketSHA:   tbSHA,
	}, nil
}