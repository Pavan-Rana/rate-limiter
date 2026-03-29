// These have no Redis dependency and exist to make the logic unit-testable in isolation.
// Production traffic goes through the Lua scripts in scripts/lua/ which run atomically on Redis.
package algorithm

import (
	"sync"
	"time"
)

// SlidingWindow is an in-memory sliding window counter.
// Not safe for distributed use — use the Redis Lua implementation for multi-replica deployments.
type SlidingWindow struct {
	mu	   	 sync.Mutex
	requests []time.Time
	limit	 int
	window 	 time.Duration
}

func NewSlidingWindow(limit int, window time.Duration) *SlidingWindow {
	return &SlidingWindow{limit: limit, window: window}
}

func (sw *SlidingWindow) Allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.window)

	// Evict expired entries
	valid := sw.requests[:0]
	for _, t := range sw.requests {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	sw.requests = valid

	if len(sw.requests) >= sw.limit {
		return false
	}

	sw.requests = append(sw.requests, now)
	return true
}

func (sw *SlidingWindow) Count() int {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return len(sw.requests)
}