--[[
  Sliding Window Rate Limiter Redis Script

  This script runs atomically inside the Redis Lua interpreter (single-threaded).
  No external locking is needed. No TOCTOU race is possible across service replicas,
  all replicas share this Redis instance as the single source of truth for decisions.

  KEYS[1]  — rate-limit key (e.g. "rl:my-api-key")
  ARGV[1]  — current timestamp in milliseconds (Unix ms)
  ARGV[2]  — window size in milliseconds
  ARGV[3]  — request limit (max allowed per window)

  Returns:  1 if the request is allowed, 0 if rejected.

  Algorithm:
    1. Remove all entries with score < (now - window_ms).
       These timestamps are outside the sliding window — expired.
    2. Count remaining entries. These are requests within the active window.
    3. If count < limit: record this request (ZADD with score = now) and allow.
    4. If count >= limit: reject without recording.
    5. Reset the key TTL to window_ms so idle keys are cleaned up automatically.
       Without this, sorted sets for inactive keys persist in Redis indefinitely.
]]

local key =         KEYS[1]
local now =         tonumber(ARGV[1])
local window_ms =   tonumber(ARGV[2])
local limit =       tonumber(ARGV[3])
local cutoff =      now - window_ms

-- Score range: -inf to cutoff (exclusive of the active window)
redis.call('ZREMRANGEBYSCORE', key, '-inf', cutoff)

local count = redis.call('ZCARD', key)

if count < limit then
    -- Member must be unique within the sorted set.
    -- Appending a microsecond-precision suffix handles bursts at identical millisecond timestamps.
    local member = tostring(now) .. '-' .. tostring(redis.call('INCR', key .. ':seq'))
    redis.call('ZADD', key, now, member)
    redis.call('PEXPIRE', key, window_ms)
    redis.call('PEXPIRE', key .. ':seq', window_ms)

    return 1
else
    return 0
end