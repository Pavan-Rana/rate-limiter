# Design Tradeoffs

This document records the engineering decisions made during design and implementation

---

## Algorithm: fixed window vs token bucket vs sliding window

**Chosen: sliding window counter**

### Fixed window (rejected)

Fixed windows divide time into discrete buckets (e.g. 12:00-12:01, 12:01-12:02).
The limit applies per bucket. Flaw: a client can fire `limit` requests at 12:00:59
and another `limit` at 12:01:00 - the burst of `2xlimit` within two seconds, defeating
the intent of the quota. Know as the "boundary burst" problem.

### Token bucket (rejected as primary)

Tokens refill at a constant rate. A client can consume tokens as fast as they accumulate
or save them up for a burst up to `capacity`. Advantages: smooth burst handling, simple
mental model for "sustained rate".

Disadvantages for this use case:
- Storing `tokens` and `last_refill` as floats in Redis and performing arithmetic in Lua
introduces floating-point imprecision that is harder to audit for correctness.
- A quota of "100 requests/minute" is harder to explain with token bucked than with
sliding window, which maps directly to "count requests in the last 60 seconds".
- The token_bucket.lua script is kept in `internal/store/scripts/lua`

### Sliding Window counter (chosen)

Stores each request timestamp as a member in a Redis sorted set, scored by timestamp.
On each decision:
1. Remove members with score < `new_window` (expired requests).
2. Count remaining members (requests within the active window)
3. Allow if count < limit; reject otherwise

Correctness is easy to reason about: "at most N timestamps exist in the set at any time"
Memory is bounded to `limit` entries per key (rejected requests are never added).

**Tradeoff:** sorted sets use more memory per entry (~50-80 bytes) than a simple counter.
For high-limit keys (e.g. 10,000 req/min), this is measurable. If memory is a concern,
switch to a token_bucket or approximate counter approach.

---

## Atomicity: application-level locking vs WATCH/MULTI/EXEC vs Redis Lua

**Chosen: Redis Lua scripts (EVALSHA)**

### Application-level locking (rejected)

A read-modify-write in Go application code:
```
count = GET key
if count < limit:
    SET key (count + 1)
    return allow
```
This is a TOCTOU race across replicas. Two replicas can both read `count - 90`
(limit: 100), both decide to allow and both write `100` - admitting 101 requests.
Incorrect by design.

### WATCH/MULTI/EXEC - optimistic transactions (rejected)

Redis's optimistic locking aborts and retries the transaction if a watched key
changes between WATCH and EXEC. Under high concurrency - many replicas targetting
the same popular API key - the retry rate can grow unboundedly, causing p99 latency
spikes. Each retry is a full round-trip

### Redis Lua scripts (chosen)

Redis executes Lua scripts atomically in its single-threaded interpreter.
The script runs to completion before any other command is processed.
This serialises all decisions at the Redis server, not application layer.

Benfits:
- Single round-trip (EVALSHA sends only the 40-char SHA digest, not the script text)
- No retries, no contention-dependent latency growth
- Correctness arguement is simple: Redis processes one Lua call at a time

The correctness prodd is in `tests/integration/concurrency_test.go`:
N goroutines firing simultaneously against a single key, asserting exactly
`limit` requests are admitted.

---

## Failure behaviour: fail-open vs fail-closed

**Chosen: fail-open**

When Redis is unreachable `AllowRequest` returns `(true, error)` - the request
is allowed through and the error is recorded in `ratelimiter_redis_error_total`.

**Rationale:** This service enforces API quotas, not security boundaries.
The cost of over-admitting requests during a Redis outage (a few extra requests
reach the upstream API) is lower than the cost of dropping all traffic (total
service outage from the perspective of the callers).

**When to choose fail-closed:** If the rate-limiter enforces a billing limit,
prevents credential stuffing or protects a resource with strict capacity limits,
fail-closed is correct. Change 'limiter.go' line:
```go
return true, fmt.Errorf("store error (failing open): %w", err)
// → 
return false, fmt.Errorf("store error (failing closed): %w", err)
```

**Observability:** Alert on `ratelimiter_redis_errors_total > 0` to detect and
measure fail-open windows. The Grafana dashboard includes this panel.

---

## Stateless Replicas

Multiple service replicas are safe because no rate-limit state lives in the Go process.
All state is in Redis. A request landing on any replica gets the same correct decision,
because the decision is made atomically by the Lua script on Redis.

**What this requires:**
- Redis must be reachable from all replicas (handled by the Redis Service in k8s)
- Redis must not be sharded such that the same key could land on different shards
    without coordination. For very high scale, Redis Cluster with `{key}` hashtags
    can ensure per-key routing to a single shard.

**HPA implication:** Because replicas are stateless, the HPA can scale them freely
without worrying about state migration. See `deploy/k8s/hpa.yml`.

---
## Memory Growth in Redis

Each admitted request adds one member to the sorted set for the API key.
Rejected requests are not recorded (the Lua script returns 0 before ZADD).

Memory per key ≈ `limit x 70 bytes` (approximate sorted set entry size).

At 100 req/min limit: ~7 KB per key.
At 10,000 req/min limit: ~700 KB per key.
At 10,000 active keys x 100 req/min limit: ~70 MB total - acceptable.

The `ZREMRANGEBYSCORE` call at the top of every Lua invocation keeps each set
bounded to the active window. The `PEXPIRE` call ensures idle keys (no requests
for a full window duration) are evicted automatically by Redis.

Without `PEXPIRE`, sorted sets for inactive keys would persist indefinitely.