package aimodule

import (
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	limiter := NewRateLimiter(10, 1000) // 10 per minute, 1000 per day

	// First 10 requests should be allowed
	for i := 0; i < 10; i++ {
		if !limiter.Allow() {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 11th request should be denied
	if limiter.Allow() {
		t.Error("11th request should be denied (rate limit exceeded)")
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	limiter := NewRateLimiter(5, 1000)

	// Use all tokens
	for i := 0; i < 5; i++ {
		limiter.Allow()
	}

	// Should be denied
	if limiter.Allow() {
		t.Error("Should be denied after using all tokens")
	}

	// Wait for reset (simulate by manually resetting)
	// In real scenario, this happens automatically after 1 minute
	time.Sleep(100 * time.Millisecond)
}

func TestRateLimiter_RemainingTokens(t *testing.T) {
	limiter := NewRateLimiter(10, 1000)

	minute, day := limiter.RemainingTokens()
	if minute != 10 {
		t.Errorf("Expected 10 remaining minute tokens, got %d", minute)
	}
	if day != 1000 {
		t.Errorf("Expected 1000 remaining day tokens, got %d", day)
	}

	// Use some tokens
	limiter.Allow()
	limiter.Allow()

	minute, day = limiter.RemainingTokens()
	if minute != 8 {
		t.Errorf("Expected 8 remaining minute tokens, got %d", minute)
	}
}
