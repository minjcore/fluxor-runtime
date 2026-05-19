// Package hedge provides resilience through parallel request execution.
//
// The hedge pattern sends multiple requests to redundant services in parallel
// and uses the first successful response. This reduces latency by not waiting
// for the slowest service, while maintaining redundancy for fault tolerance.
//
// The hedge package implements configurable parallel execution with support for:
//   - Configurable concurrency limits
//   - Automatic cancellation of remaining requests on first success
//   - Comprehensive lifecycle callbacks
//   - Detailed statistics and metrics
//
// Basic Usage:
//
//	import (
//	    "context"
//	    "github.com/fluxorio/fluxor/pkg/core/resilience/hedge"
//	)
//
//	// Create manager with default config
//	manager := hedge.NewManager()
//
//	// Execute multiple functions in parallel
//	err := manager.Execute(ctx, []hedge.Executable{
//	    func(ctx context.Context) error {
//	        return service1.Call(ctx)
//	    },
//	    func(ctx context.Context) error {
//	        return service2.Call(ctx)
//	    },
//	    func(ctx context.Context) error {
//	        return service3.Call(ctx)
//	    },
//	})
//
// Advanced Usage with Custom Config:
//
//	config := hedge.Config{
//	    MaxConcurrency: 5, // Execute up to 5 hedges in parallel
//	    CancelRemainingOnSuccess: true, // Cancel remaining on first success
//	    OnHedgeSucceeded: func(ctx context.Context, index int, result error) {
//	        log.Printf("Hedge %d succeeded", index)
//	    },
//	    OnAllHedgesFailed: func(ctx context.Context, errors []error) {
//	        log.Printf("All hedges failed: %v", errors)
//	    },
//	}
//
//	manager := hedge.NewManager()
//	err := manager.ExecuteWithConfig(ctx, functions, config)
//
// Execution Flow:
//
// 1. Multiple functions are executed in parallel (up to MaxConcurrency)
// 2. First successful result is returned immediately
// 3. If CancelRemainingOnSuccess is true, remaining requests are cancelled
// 4. If all requests fail, ErrCodeAllHedgesFailed is returned with all errors
//
// Concurrency Control:
//
// MaxConcurrency limits the number of parallel hedge executions. If you have
// 10 functions but MaxConcurrency is 3, only 3 will execute at a time.
//
// Cancellation:
//
// When CancelRemainingOnSuccess is true (default), remaining hedge requests
// are cancelled via context cancellation when the first hedge succeeds. This
// reduces resource usage and prevents unnecessary work.
//
// Thread Safety:
//
// The Manager is safe for concurrent use by multiple goroutines.
//
// Example: Multi-Region Service Calls
//
//	manager := hedge.NewManager()
//
//	err := manager.Execute(ctx, []hedge.Executable{
//	    // Region 1
//	    func(ctx context.Context) error {
//	        return region1Service.GetData(ctx)
//	    },
//	    // Region 2
//	    func(ctx context.Context) error {
//	        return region2Service.GetData(ctx)
//	    },
//	    // Region 3
//	    func(ctx context.Context) error {
//	        return region3Service.GetData(ctx)
//	    },
//	})
//	// Returns as soon as any region responds successfully
package hedge
