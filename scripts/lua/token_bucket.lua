--[[
  Token Bucket Rate Limiter Redis Script

  Kept alongside sliding_window.lua to highlights the tradeoffs in docs/tradeoffs.md.

  KEYS[1]  — rate-limit key
  ARGV[1]  — current timestamp in seconds (float precision)
  ARGV[2]  — bucket capacity (max tokens)
  ARGV[3]  — refill rate (tokens per second)

  Returns:  1 if allowed, 0 if rejected.
]]

local key =         KEYS[1]
local now =         tonumber(ARGV[1])
local capacity =    tonumber(ARGV[2])
local rate =        tonumber(ARGV[3])

local data =        redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens =      tonumber(data[1]) or capacity
local last_refill = tonumber(data[2]) or now

-- Refill tokens proportional to elapsed time since last request
local elapsed =     math.max(0, now - last_refill)
local new_tokens =  math.floor(capacity, tokens + elapsed * rate)

if new_tokens < 1 then
    return 0
end

-- Consume one token and persist updated state
redis.call('HMSET', key, 'tokens', new_tokens - 1, 'last_refill', now)
redis.call('EXPIRE', key, math.ceil(capacity / rate) + 1)

return 1