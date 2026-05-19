package security

import (
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/web"
)

// RateLimitConfig configures rate limiting
type RateLimitConfig struct {
	// RequestsPerMinute is the maximum number of requests per minute per client
	RequestsPerMinute int

	// RequestsPerSecond is the maximum number of requests per second per client (alternative to RequestsPerMinute)
	RequestsPerSecond int

	// KeyFunc extracts a key from the request to identify the client
	// Default: uses IP address
	KeyFunc func(ctx *web.FastRequestContext) string

	// OnLimitReached is called when rate limit is exceeded
	// If nil, returns 429 Too Many Requests
	OnLimitReached func(ctx *web.FastRequestContext) error
}

// DefaultRateLimitConfig returns a default rate limit configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerMinute: 100,
		KeyFunc: func(ctx *web.FastRequestContext) string {
			// Use IP address as key
			return ctx.RequestCtx.RemoteIP().String()
		},
	}
}

// rateLimiter implements token bucket rate limiting
type rateLimiter struct {
	mu            sync.RWMutex
	buckets       map[string]*tokenBucket
	cleanupTicker *time.Ticker
	cleanupDone   chan struct{}
}

type tokenBucket struct {
	tokens     int
	lastRefill time.Time
	capacity   int
	refillRate time.Duration // time between refills
	mu         sync.Mutex
}

// newRateLimiter creates a new rate limiter
func newRateLimiter(requestsPerMinute int) *rateLimiter {
	rl := &rateLimiter{
		buckets: make(map[string]*tokenBucket),
	}

	// Note: requestsPerMinute is used in allow() method for token bucket refill calculation
	// The limiter validates minimum 1 request per minute
	if requestsPerMinute < 1 {
		requestsPerMinute = 1
	}

	// Start cleanup goroutine
	rl.cleanupTicker = time.NewTicker(5 * time.Minute)
	rl.cleanupDone = make(chan struct{})
	go rl.cleanup()

	return rl
}

// cleanup removes old buckets periodically
func (rl *rateLimiter) cleanup() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.mu.Lock()
			now := time.Now()
			for key, bucket := range rl.buckets {
				bucket.mu.Lock()
				// Remove buckets that haven't been used in 10 minutes
				if now.Sub(bucket.lastRefill) > 10*time.Minute {
					delete(rl.buckets, key)
				}
				bucket.mu.Unlock()
			}
			rl.mu.Unlock()
		case <-rl.cleanupDone:
			return
		}
	}
}

// Stop stops the cleanup goroutine
// Exported for use by server shutdown
func (rl *rateLimiter) Stop() {
	rl.cleanupTicker.Stop()
	close(rl.cleanupDone)
}

// allow checks if a request is allowed
func (rl *rateLimiter) allow(key string, requestsPerMinute int) bool {
	rl.mu.RLock()
	bucket, exists := rl.buckets[key]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		// Double-check after acquiring write lock
		bucket, exists = rl.buckets[key]
		if !exists {
			refillRate := time.Minute / time.Duration(requestsPerMinute)
			if refillRate < time.Second {
				refillRate = time.Second
			}
			bucket = &tokenBucket{
				tokens:     requestsPerMinute,
				lastRefill: time.Now(),
				capacity:   requestsPerMinute,
				refillRate: refillRate,
			}
			rl.buckets[key] = bucket
		}
		rl.mu.Unlock()
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)
	if elapsed >= bucket.refillRate {
		tokensToAdd := int(elapsed / bucket.refillRate)
		bucket.tokens = bucket.capacity
		if tokensToAdd > 0 {
			bucket.tokens = bucket.capacity
		}
		bucket.lastRefill = now
	}

	// Check if tokens available
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

var (
	globalRateLimiters = make(map[int]*rateLimiter)
	rateLimitersMu     sync.Mutex
)

// getRateLimiter returns or creates a rate limiter for the given requests per minute
func getRateLimiter(requestsPerMinute int) *rateLimiter {
	rateLimitersMu.Lock()
	defer rateLimitersMu.Unlock()

	if limiter, exists := globalRateLimiters[requestsPerMinute]; exists {
		return limiter
	}

	limiter := newRateLimiter(requestsPerMinute)
	globalRateLimiters[requestsPerMinute] = limiter
	return limiter
}

// RateLimit middleware enforces rate limiting
func RateLimit(config RateLimitConfig) web.FastMiddleware {
	// Determine requests per minute
	requestsPerMinute := config.RequestsPerMinute
	if requestsPerMinute == 0 && config.RequestsPerSecond > 0 {
		requestsPerMinute = config.RequestsPerSecond * 60
	}
	if requestsPerMinute == 0 {
		requestsPerMinute = 100 // Default
	}

	// Get key function
	keyFunc := config.KeyFunc
	if keyFunc == nil {
		keyFunc = func(ctx *web.FastRequestContext) string {
			return ctx.RequestCtx.RemoteIP().String()
		}
	}

	// Get rate limiter
	limiter := getRateLimiter(requestsPerMinute)

	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			key := keyFunc(ctx)

			if !limiter.allow(key, requestsPerMinute) {
				// Rate limit exceeded
				if config.OnLimitReached != nil {
					return config.OnLimitReached(ctx)
				}

				// Default: return 429 Too Many Requests
				ctx.RequestCtx.SetStatusCode(429)
				ctx.RequestCtx.SetContentType("application/json")
				// Error intentionally ignored - best effort response for rate limiting
				_, _ = ctx.RequestCtx.WriteString(`{"error":"rate_limit_exceeded","message":"Too many requests"}`)
				return nil
			}

			return next(ctx)
		}
	}
}
