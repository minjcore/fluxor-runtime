// Package timeout provides timeout functionality for operations and functions.
//
// The timeout package implements configurable timeout enforcement with support for:
//   - Function execution timeouts
//   - Context-based timeout and cancellation
//   - Timeout callbacks for observability
//   - Comprehensive statistics and metrics
//
// Basic Usage:
//
//	import (
//	    "context"
//	    "time"
//	    "github.com/fluxorio/fluxor/pkg/core/resilience/timeout"
//	)
//
//	manager := timeout.NewManager()
//
//	// Execute with default timeout (uses context timeout if available)
//	err := manager.Execute(ctx, func(ctx context.Context) error {
//	    // Your operation that may take too long
//	    return someSlowOperation(ctx)
//	})
//
//	// Execute with specific timeout
//	err := manager.ExecuteWithTimeout(ctx, myFunction, 5*time.Second)
//
// Advanced Usage with Custom Config:
//
//	config := timeout.Config{
//	    Timeout: 10*time.Second,
//	    OnTimeout: func(ctx context.Context, timeout time.Duration) {
//	        log.Printf("Operation timed out after %v", timeout)
//	    },
//	    OnTimeoutAsync: func(ctx context.Context, timeout time.Duration) {
//	        // Send metrics asynchronously
//	        metrics.RecordTimeout(timeout)
//	    },
//	}
//
//	err := manager.ExecuteWithConfig(ctx, myFunction, config)
//
// Context Support:
//
// The timeout package fully supports context cancellation and timeouts:
//
//   - If Config.Timeout is set, it creates a timeout context with that duration
//   - If Config.Timeout is zero, it uses the context's deadline if available
//   - The function execution respects context cancellation
//   - If the context is cancelled before the timeout, cancellation is returned
//
// Timeout Behavior:
//
// When a timeout occurs:
//   - The function execution continues in the background but is no longer waited for
//   - An error is immediately returned with ErrCodeTimeoutExceeded
//   - Timeout callbacks are invoked (synchronously and/or asynchronously)
//   - The context passed to the function will be cancelled when the timeout expires
//
// Thread Safety:
//
// The Manager is safe for concurrent use by multiple goroutines.
package timeout
