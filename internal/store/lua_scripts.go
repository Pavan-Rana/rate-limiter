package store

import (
	"context"
	_ "embed"

	"github.com/redis/go-redis/v9"
)

// LuaScripts holds SHA digests for pre-loaded Lua scripts.
// Using EVALSHA (vs EVAL) avoids re-parsing the script on every request.
type LuaScripts struct {
	SlidingWindowSHA string
	TokenBucketSHA   string
}

//go:embed scripts/lua/sliding_window.lua
var slidingWindowLua []byte

//go:embed scripts/lua/token_bucket.lua
var tokenBucketLua []byte

// LoadScripts pre-loads Lua scripts into Redis at startup.
// Subsequent calls use EVALSHA with the digest - faster and avoids redundant parsing.
func LoadScripts(ctx context.Context, client *redis.Client) (*LuaScripts, error) {

	swSHA, err := client.ScriptLoad(ctx, string(slidingWindowLua)).Result()
	if err != nil {
		return nil, err
	}

	tbSHA, err := client.ScriptLoad(ctx, string(tokenBucketLua)).Result()
	if err != nil {
		return nil, err
	}

	return &LuaScripts{
		SlidingWindowSHA: swSHA,
		TokenBucketSHA:   tbSHA,
	}, nil
}
