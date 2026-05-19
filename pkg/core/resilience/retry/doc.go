// Package retry provides configurable retry logic for handling transient failures.
//
// The retry package implements a flexible retry mechanism with support for:
//   - Multiple backoff strategies (fixed, exponential, linear)
//   - Configurable retry predicates (retry on specific errors only)
//   - Context support for cancellation and timeouts
//   - Comprehensive statistics and metrics
//   - Retry callbacks for observability
//
// Basic Usage:
//
//	import (
//	    "context"
//	    "github.com/fluxorio/fluxor/pkg/core/resilience/retry"
//	)
//
//	manager := retry.NewManager()
//
//	err := manager.Execute(ctx, func(ctx context.Context) error {
//	    // Your operation that may fail
//	    return someOperation(ctx)
//	})
//
//	if err != nil {
//	    // Handle error (may include retry exhaustion)
//	}
//
// Advanced Usage with Custom Config:
//
//	config := retry.Config{
//	    MaxRetries: 5,
//	    Backoff: retry.NewExponentialBackoff(
//	        100*time.Millisecond, // Initial delay
//	        30*time.Second,       // Max delay
//	        2.0,                  // Multiplier
//		true,                   // Enable jitter
//	    ),
//	    Predicate: retry.RetryOnNetworkError, // Only retry on network errors
//	    Timeout: 10*time.Second,              // Overall timeout
//	    OnRetry: func(attempt int, err error) {
//	        log.Printf("Retry attempt %d: %v", attempt+1, err)
//	    },
//	}
//
//	err := manager.ExecuteWithConfig(ctx, myFunction, config)
//
// Backoff Strategies:
//
// Fixed Backoff:
//
//	backoff := retry.NewFixedBackoff(100 * time.Millisecond)
//
// Exponential Backoff (recommended for most cases):
//
//	backoff := retry.NewExponentialBackoff(
//	    100*time.Millisecond, // Initial delay
//	    30*time.Second,       // Max delay
//	    2.0,                  // Multiplier (doubles each retry)
//	    true,                 // Enable jitter to prevent thundering herd
//	)
//
// Linear Backoff:
//
//	backoff := retry.NewLinearBackoff(
//	    100*time.Millisecond, // Initial delay
//	    30*time.Second,       // Max delay
//	    100*time.Millisecond, // Increment per retry
//	)
//
// Retry Predicates:
//
// Predefined predicates are available for common scenarios:
//
//   - AlwaysRetry: Retry on all errors
//   - NeverRetry: Never retry
//   - RetryOnTimeout: Retry only on timeout errors
//   - RetryOnTemporary: Retry on temporary errors
//   - RetryOnNetworkError: Retry on network-related errors
//   - RetryOnErrorType: Retry on specific error types
//   - RetryOnErrorMessage: Retry based on error message content
//
// Combining predicates:
//
//	predicate := retry.RetryOnAny(
//	    retry.RetryOnTimeout,
//	    retry.RetryOnNetworkError,
//	)
//
// Context Support:
//
// The retry package fully supports context cancellation and timeouts:
//
//   - If the context is cancelled, retries stop immediately
//   - Config.Timeout provides an overall timeout for all retry attempts
//   - Both initial execution and backoff delays respect context cancellation
//
// Thread Safety:
//
// The Manager is safe for concurrent use by multiple goroutines.
package retry
