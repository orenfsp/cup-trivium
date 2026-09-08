package httpapi

import (
	"sync"
	"testing"
	"time"
)

func TestRateLimiterAllow(t *testing.T) {
	rl := newRateLimiter()
	for i := 0; i < 5; i++ {
		if !rl.allow("k", 5, time.Minute) {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}
	if rl.allow("k", 5, time.Minute) {
		t.Error("6th attempt within window should be blocked")
	}
	// Другой ключ не зависит от первого.
	if !rl.allow("other", 5, time.Minute) {
		t.Error("independent key should be allowed")
	}
}

func TestRateLimiterReset(t *testing.T) {
	rl := newRateLimiter()
	for i := 0; i < 3; i++ {
		rl.allow("k", 3, time.Minute)
	}
	if rl.allow("k", 3, time.Minute) {
		t.Error("should be blocked before reset")
	}
	rl.reset("k")
	if !rl.allow("k", 3, time.Minute) {
		t.Error("should be allowed after reset")
	}
}

func TestRateLimiterWindowExpiry(t *testing.T) {
	rl := newRateLimiter()
	const window = 30 * time.Millisecond
	if !rl.allow("k", 1, window) {
		t.Fatal("first attempt should be allowed")
	}
	if rl.allow("k", 1, window) {
		t.Fatal("second attempt within window should be blocked")
	}
	time.Sleep(2 * window)
	if !rl.allow("k", 1, window) {
		t.Error("attempt after window expiry should be allowed")
	}
}

// Истёкшие ключи не должны копиться в памяти.
func TestRateLimiterNoKeyLeak(t *testing.T) {
	rl := newRateLimiter()
	const window = 20 * time.Millisecond
	rl.allow("once", 2, window)
	time.Sleep(2 * window)
	rl.allow("once", 2, window) // затирает истёкший хит
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if n := len(rl.hits["once"]); n != 1 {
		t.Errorf("expected 1 fresh hit after expiry, got %d", n)
	}
}

func TestRateLimiterConcurrent(t *testing.T) {
	rl := newRateLimiter()
	const goroutines, perG = 8, 50
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perG; j++ {
				rl.allow("shared", 10, time.Minute)
			}
		}()
	}
	wg.Wait()
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if n := len(rl.hits["shared"]); n != 10 {
		t.Errorf("expected capped 10 hits, got %d", n)
	}
}
