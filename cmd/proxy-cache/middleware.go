package main

import (
	"log"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	tokens     int
	maxTokens  int
	refillRate int
	mu         sync.Mutex
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter
// requestsPerSecond: maximum requests per second
func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	rl := &RateLimiter{
		tokens:     requestsPerSecond,
		maxTokens:  requestsPerSecond,
		refillRate: requestsPerSecond,
		lastRefill: time.Now(),
	}
	return rl
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()

	// Refill tokens based on elapsed time
	tokensToAdd := int(elapsed * float64(rl.refillRate))
	rl.tokens = min(rl.maxTokens, rl.tokens+tokensToAdd)
	rl.lastRefill = now

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	return false
}

// RateLimitMiddleware wraps an HTTP handler with rate limiting
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				w.Header().Set("Retry-After", "1")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequestLogger logs HTTP requests
type RequestLogger struct {
	mu       sync.Mutex
	requests int64
	bytes    int64
}

// LogRequest logs a request
func (rl *RequestLogger) LogRequest(path string, statusCode int, bytes int64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.requests++
	rl.bytes += bytes
}

// LoggingMiddleware wraps an HTTP handler with logging
func LoggingMiddleware(logger *RequestLogger, verbose bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status and size
			wrappedWriter := &responseWriter{ResponseWriter: w}

			next.ServeHTTP(wrappedWriter, r)

			duration := time.Since(start)
			logger.LogRequest(r.URL.Path, wrappedWriter.statusCode, int64(wrappedWriter.size))

			if verbose {
				log.Printf("[%s] %s %s %d %dms %d bytes",
					time.Now().Format("15:04:05"),
					r.Method,
					r.URL.Path,
					wrappedWriter.statusCode,
					duration.Milliseconds(),
					wrappedWriter.size,
				)
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status and size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	w.size += n
	return n, err
}

// CORSMiddleware adds CORS headers
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			if len(allowedOrigins) > 0 && allowedOrigins[0] == "*" {
				allowed = true
			} else {
				for _, ao := range allowedOrigins {
					if ao == origin {
						allowed = true
						break
					}
				}
			}

			if allowed || len(allowedOrigins) == 0 {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
