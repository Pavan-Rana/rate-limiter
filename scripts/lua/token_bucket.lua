local key = KEYS[1]
local now = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local rate = tonumber(ARGV[3])

local data = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(data[1]) or capacity
local last_refill = tonumber(data[2]) or now

local elapsed = math.max(0, now - last_refill)
local new_tokens = math.floor(capacity, tokens + elapsed * rate)

if new_tokens < 1 then
    return 0
end

redis.call('HMSET', key, 'tokens', new_tokens - 1, 'last_refill', now)
redis.call('EXPIRE', key, math.ceil(capacity / rate) + 1)

return 1