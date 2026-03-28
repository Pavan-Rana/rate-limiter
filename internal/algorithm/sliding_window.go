package algorithm

import (
	"sync"
	"time"
)

type SlidingWindow struct {
	mu	   	 sync.Mutex
	requests []time.Time
	limit	 int
	window 	 time.Duration
}

func NewSlidingWindoe(limit int, window time.Duration) *SlidingWindow {
	return &SlidingWindow{limit: limit, window: window}
}

func (sw: *SlidingWindow) Allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.window)

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