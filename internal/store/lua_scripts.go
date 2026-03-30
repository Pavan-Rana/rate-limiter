package store

import (
	"context"
	_ "embed"
	"os"

	"github.com/redis/go-redis/v9"
)

// LuaScripts holds SHA digests for pre-loaded Lua scripts.
// Using EVALSHA (vs EVAL) avoids re-parsing the script on every request.
type LuaScripts struct {
	SlidingWindowSHA string
	TokenBucketSHA   string
}

// LoadScripts pre-loads Lua scripts into Redis at startup.
// Subsequent calls use EVALSHA with the digest - faster and avoids redundant parsing.
func LoadScripts(ctx context.Context, client *redis.Client) (*LuaScripts, error) {
	sw, err := os.ReadFile("../../scripts/lua/sliding_window.lua")
	if err != nil {
		return nil, err
	}
	tb, err := os.ReadFile("../../scripts/lua/token_bucket.lua")
	if err != nil {
		return nil, err
	}

	swSHA, err := client.ScriptLoad(ctx, string(sw)).Result()
	if err != nil {
		return nil, err
	}

	tbSHA, err := client.ScriptLoad(ctx, string(tb)).Result()
	if err != nil {
		return nil, err
	}

	return &LuaScripts{
		SlidingWindowSHA: swSHA,
		TokenBucketSHA:   tbSHA,
	}, nil
}
