package algorithm_test

import (
	"sync"
	"time"
	"testing"

	"github.com/Pavan-Rana/rate-limiter/internal/algorithm"
)

func TestSlidingWindow_BasicLimit(t *testing.T) {
	sw := algorithm.NewSlidingWindow(5, time.Second)

	for i := 0; i < 5; i++ {
		if !sw.Allow() {
			t.Fatalf("Expected request %d to be allowed", i+1)
		}
	}
	if sw.Allow() {
		t.Fatal("Expected 6th request to be rejected")
	}
}

func TestSlidingWindow_WindowExpiry(t *testing.T) {
	sw := algorithm.NewSlidingWindow(3, 100*time.Millisecond)

	for i := 0; i < 3; i++ {
		sw.Allow()
	}
	if sw.Allow() {
		t.Fatal("Expected 4th request to be rejected")
	}

	time.Sleep(110 * time.Millisecond)

	if !sw.Allow(){
		t.Fatal("Expected request to be allowed after window expiry")
	}
}

func TestSlidingWindow_ConcurrentAccess(t *testing.T) {
	const limit = 100
	sw := algorithm.NewSlidingWindow(limit, time.Second)

	var (
		wg 		sync.WaitGroup
		mu 		sync.Mutex
		allowed int
	)

	for i := 0; i < limit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if sw.Allow() {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if allowed > limit {
		t.Fatalf("Allowed %d requests, limit is %d", allowed, limit)
	}
}