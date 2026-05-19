// Package limiter provides rate limiting to control the rate of operations.
//
// The limiter package implements token bucket rate limiting to control the rate
// at which operations can be executed. This helps prevent resource exhaustion
// and ensures fair usage of shared resources.
//
// The limiter package implements configurable rate limiting with support for:
//   - Token bucket algorithm
//   - Configurable rate and burst size
//   - Blocking and non-blocking modes
//   - Comprehensive lifecycle callbacks
//   - Detailed statistics and metrics
//
// Basic Usage:
//
//	import (
//	    "context"
//	    "github.com/fluxorio/fluxor/pkg/core/resilience/limiter"
//	)
//
//	// Create manager with default config (10 executions per second)
//	manager := limiter.NewManager()
//
//	// Execute function with rate limiting
//	err := manager.Execute(ctx, func(ctx context.Context) error {
//	    // Your operation that should be rate limited
//	    return someOperation(ctx)
//	})
//
// Advanced Usage with Custom Config:
//
//	config := limiter.Config{
//	    Rate:     100,              // 100 executions per interval
//	    Interval: time.Minute,      // 1 minute interval
//	    Burst:    150,              // Allow burst up to 150
//	    OnRateLimitExceeded: func(ctx context.Context, rate int, interval time.Duration) {
//	        log.Printf("Rate limit exceeded: %d per %v", rate, interval)
//	    },
//	    OnAllowed: func(ctx context.Context) {
//	        log.Printf("Execution allowed")
//	    },
//	    OnCompleted: func(ctx context.Context, err error) {
//	        log.Printf("Execution completed: %v", err)
//	    },
//	}
//
//	manager := limiter.NewManagerWithConfig(config)
//	err := manager.Execute(ctx, myFunction)
//
// Non-Blocking Mode:
//
//	// Check if execution would be allowed without blocking
//	if manager.Allow(ctx) {
//	    err := manager.Execute(ctx, myFunction)
//	    // ...
//	} else {
//	    // Rate limit exceeded - handle accordingly
//	}
//
// Blocking Mode:
//
//	// Wait until rate limit allows execution
//	if err := manager.Wait(ctx); err != nil {
//	    // Context cancelled or timeout
//	    return err
//	}
//	err := manager.Execute(ctx, myFunction)
//
// Token Bucket Algorithm:
//
// The limiter uses a token bucket algorithm:
//   - Tokens are added at a constant rate (Rate / Interval)
//   - Maximum tokens = Burst size
//   - Each execution consumes one token
//   - If no tokens available, execution is rate limited
//
// Thread Safety:
//
// The Manager is safe for concurrent use by multiple goroutines.
//
// Example: API Rate Limiting
//
//	manager := limiter.NewManagerWithConfig(limiter.Config{
//	    Rate:     60,              // 60 requests per minute
//	    Interval: time.Minute,
//	    Burst:    10,              // Allow burst of 10 requests
//	})
//
//	err := manager.Execute(ctx, func(ctx context.Context) error {
//	    // API call that should be rate limited
//	    return apiClient.Call(ctx, request)
//	})
package limiter
