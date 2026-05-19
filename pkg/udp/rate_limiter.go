package udp

import (
	"sync/atomic"
	"time"
)

// RateLimiter implements a fixed-window rate limiter for packets per second.
// When maxPPS is 0, allows unlimited rate (always returns true).
type RateLimiter struct {
	maxPPS       int64 // Max packets per second (0 = unlimited)
	count        int64 // Packets in current second (atomic)
	windowStart  int64 // Start of current window (unix second)
	droppedCount int64 // Dropped due to rate limit (atomic)
}

// NewRateLimiter creates a rate limiter for maxPPS packets per second.
// maxPPS 0 means unlimited (TryAcquire always returns true).
func NewRateLimiter(maxPPS int) *RateLimiter {
	if maxPPS <= 0 {
		return &RateLimiter{maxPPS: 0}
	}
	return &RateLimiter{
		maxPPS:      int64(maxPPS),
		windowStart: time.Now().Unix(),
	}
}

// TryAcquire attempts to allow one packet. Returns true if allowed, false if rate limited.
func (r *RateLimiter) TryAcquire() bool {
	if r.maxPPS <= 0 {
		return true
	}
	now := time.Now().Unix()
	windowStart := atomic.LoadInt64(&r.windowStart)
	if now > windowStart {
		if atomic.CompareAndSwapInt64(&r.windowStart, windowStart, now) {
			atomic.StoreInt64(&r.count, 0)
		}
	}
	for {
		cur := atomic.LoadInt64(&r.count)
		if cur >= r.maxPPS {
			atomic.AddInt64(&r.droppedCount, 1)
			return false
		}
		if atomic.CompareAndSwapInt64(&r.count, cur, cur+1) {
			return true
		}
	}
}

// DroppedCount returns total packets dropped due to rate limiting.
func (r *RateLimiter) DroppedCount() int64 {
	return atomic.LoadInt64(&r.droppedCount)
}
