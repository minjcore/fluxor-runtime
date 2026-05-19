// Package breaker provides circuit breaker functionality to prevent cascading failures.
//
// The circuit breaker pattern prevents cascading failures by "opening" the circuit
// when failures exceed a threshold, rejecting requests immediately instead of
// allowing them to propagate failures. After a timeout, it transitions to "half-open"
// state to test if the service has recovered.
//
// The breaker package implements configurable circuit breaker with support for:
//   - Three states: Closed, Open, HalfOpen
//   - Configurable failure and success thresholds
//   - Automatic state transitions
//   - Comprehensive lifecycle callbacks
//   - Detailed statistics and metrics
//
// Basic Usage:
//
//	import (
//	    "context"
//	    "github.com/fluxorio/fluxor/pkg/core/resilience/breaker"
//	)
//
//	// Create manager with default config
//	manager := breaker.NewManager()
//
//	// Execute function with circuit breaker protection
//	err := manager.Execute(ctx, func(ctx context.Context) error {
//	    // Your operation that should be protected
//	    return someOperation(ctx)
//	})
//
// Advanced Usage with Custom Config:
//
//	config := breaker.Config{
//	    FailureThreshold:    5,   // Open after 5 consecutive failures
//	    SuccessThreshold:    2,   // Close after 2 consecutive successes in HalfOpen
//	    Timeout:             30*time.Second, // Wait 30s before HalfOpen
//	    HalfOpenMaxRequests: 3,   // Allow up to 3 requests in HalfOpen
//	    OnOpened: func(ctx context.Context) {
//	        log.Printf("Circuit breaker opened")
//	    },
//	    OnClosed: func(ctx context.Context) {
//	        log.Printf("Circuit breaker closed")
//	    },
//	}
//
//	manager := breaker.NewManagerWithConfig(config)
//	err := manager.Execute(ctx, myFunction)
//
// Circuit Breaker States:
//
// 1. CLOSED (Normal Operation):
//    - All requests pass through
//    - Failures are tracked
//    - Opens when failure threshold is exceeded
//
// 2. OPEN (Failing Fast):
//    - All requests are rejected immediately
//    - Returns ErrCodeCircuitOpen error
//    - Transitions to HalfOpen after timeout
//
// 3. HALF_OPEN (Testing Recovery):
//    - Allows limited requests (HalfOpenMaxRequests)
//    - Closes if success threshold is met
//    - Opens again if any failure occurs
//
// State Transitions:
//
// CLOSED → OPEN: When consecutive failures >= FailureThreshold
// OPEN → HALF_OPEN: After Timeout duration
// HALF_OPEN → CLOSED: When consecutive successes >= SuccessThreshold
// HALF_OPEN → OPEN: On any failure
//
// Manual Control:
//
//	// Check if execution is allowed (non-blocking)
//	if manager.Allow(ctx) {
//	    err := manager.Execute(ctx, myFunction)
//	    if err != nil {
//	        manager.Failure() // Manually record failure
//	    } else {
//	        manager.Success() // Manually record success
//	    }
//	}
//
//	// Reset circuit breaker to Closed state
//	manager.Reset()
//
// Thread Safety:
//
// The Manager is safe for concurrent use by multiple goroutines.
//
// Example: Database Connection Protection
//
//	manager := breaker.NewManagerWithConfig(breaker.Config{
//	    FailureThreshold: 5,
//	    Timeout:         30*time.Second,
//	})
//
//	err := manager.Execute(ctx, func(ctx context.Context) error {
//	    // Database operation
//	    return db.Query(ctx, query)
//	})
//
//	if err != nil {
//	    // Circuit might be open - handle accordingly
//	    if breakerErr, ok := err.(*breaker.Error); ok {
//	        if breakerErr.Code == breaker.ErrCodeCircuitOpen {
//	            // Use fallback or return cached data
//	            return getCachedData()
//	        }
//	    }
//	    return err
//	}
package breaker
