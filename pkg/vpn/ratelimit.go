package vpn

import (
	"sync"
	"time"
)

// RateLimiter limits the rate of operations
type RateLimiter struct {
	mu           sync.RWMutex
	requests     map[string][]time.Time
	maxRequests  int
	windowSize   time.Duration
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxRequests int, windowSize time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests:    make(map[string][]time.Time),
		maxRequests: maxRequests,
		windowSize:  windowSize,
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine
	rl.cleanupTicker = time.NewTicker(windowSize)
	go rl.cleanup()

	return rl
}

// Allow checks if a request from the given key should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.windowSize)

	// Clean up old entries for this key
	requests := rl.requests[key]
	validRequests := make([]time.Time, 0, len(requests))
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Check if limit exceeded
	if len(validRequests) >= rl.maxRequests {
		rl.requests[key] = validRequests
		return false
	}

	// Add new request
	validRequests = append(validRequests, now)
	rl.requests[key] = validRequests
	return true
}

// Reset resets the rate limiter for a specific key
func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.requests, key)
}

// cleanup periodically removes old entries
func (rl *RateLimiter) cleanup() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.mu.Lock()
			now := time.Now()
			cutoff := now.Add(-rl.windowSize * 2) // Keep some buffer

			for key, requests := range rl.requests {
				validRequests := make([]time.Time, 0, len(requests))
				for _, reqTime := range requests {
					if reqTime.After(cutoff) {
						validRequests = append(validRequests, reqTime)
					}
				}

				if len(validRequests) == 0 {
					delete(rl.requests, key)
				} else {
					rl.requests[key] = validRequests
				}
			}
			rl.mu.Unlock()

		case <-rl.stopCleanup:
			return
		}
	}
}

// Stop stops the rate limiter
func (rl *RateLimiter) Stop() {
	if rl.cleanupTicker != nil {
		rl.cleanupTicker.Stop()
	}
	close(rl.stopCleanup)
}

// ConnectionRateLimiter limits connection attempts per IP
type ConnectionRateLimiter struct {
	*RateLimiter
}

// NewConnectionRateLimiter creates a new connection rate limiter
func NewConnectionRateLimiter(maxConnections int, windowSize time.Duration) *ConnectionRateLimiter {
	return &ConnectionRateLimiter{
		RateLimiter: NewRateLimiter(maxConnections, windowSize),
	}
}
