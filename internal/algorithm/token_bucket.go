package algorithm

import (
	"sync"
	"time"
)

// TokenBucket kept alongside SlidingWindow to highlight the tradeoff discussion in docs/tradeoffs.md.
type TokenBucket struct {
	mu			 sync.Mutex
	tokens		 float64
	capacity	 float64
	rate		 float64 // tokens per second
	lastFill 	 time.Time
}

func NewTokenBucket(capacity float64, ratePerSec float64) *TokenBucket {
	return &TokenBucket{
		tokens:		 capacity,
		capacity:	 capacity,
		rate:		 ratePerSec,
		lastFill: time.Now(),
	}
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastFill).Seconds()
	tb.tokens = minF(tb.capacity, tb.tokens+elapsed*tb.rate)
	tb.lastFill = now

	if tb.tokens < 1 {
		return false
	}
	tb.tokens--
	return true
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}