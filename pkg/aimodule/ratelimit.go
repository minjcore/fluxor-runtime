package aimodule

import (
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	requestsPerMinute int
	requestsPerDay    int
	tokensPerMinute   int
	tokensPerDay      int
	lastMinuteReset   time.Time
	lastDayReset      time.Time
	minuteTokens      int
	dayTokens         int
	mu                sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute, requestsPerDay int) *RateLimiter {
	now := time.Now()
	return &RateLimiter{
		requestsPerMinute: requestsPerMinute,
		requestsPerDay:    requestsPerDay,
		tokensPerMinute:   requestsPerMinute,
		tokensPerDay:      requestsPerDay,
		lastMinuteReset:   now,
		lastDayReset:      now,
		minuteTokens:      requestsPerMinute,
		dayTokens:         requestsPerDay,
	}
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Reset minute tokens if needed
	if now.Sub(rl.lastMinuteReset) >= time.Minute {
		rl.minuteTokens = rl.requestsPerMinute
		rl.lastMinuteReset = now
	}

	// Reset day tokens if needed
	if now.Sub(rl.lastDayReset) >= 24*time.Hour {
		rl.dayTokens = rl.requestsPerDay
		rl.lastDayReset = now
	}

	// Check if we have tokens available
	if rl.minuteTokens <= 0 || rl.dayTokens <= 0 {
		return false
	}

	// Consume tokens
	rl.minuteTokens--
	rl.dayTokens--

	return true
}

// Wait blocks until a request is allowed
func (rl *RateLimiter) Wait() {
	for !rl.Allow() {
		time.Sleep(100 * time.Millisecond)
	}
}

// RemainingTokens returns the number of remaining tokens
func (rl *RateLimiter) RemainingTokens() (minute, day int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Reset minute tokens if needed
	if now.Sub(rl.lastMinuteReset) >= time.Minute {
		rl.minuteTokens = rl.requestsPerMinute
		rl.lastMinuteReset = now
	}

	// Reset day tokens if needed
	if now.Sub(rl.lastDayReset) >= 24*time.Hour {
		rl.dayTokens = rl.requestsPerDay
		rl.lastDayReset = now
	}

	return rl.minuteTokens, rl.dayTokens
}
