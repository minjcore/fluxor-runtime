// Package bulkhead provides concurrency limiting to prevent resource exhaustion.
//
// The bulkhead pattern isolates resources so that failures in one area don't
// cascade and bring down the entire system. It limits the number of concurrent
// executions to protect critical resources.
//
// The bulkhead package implements configurable concurrency limiting with support for:
//   - Maximum concurrent execution limits
//   - Optional queuing with configurable queue size
//   - Queue timeout support
//   - Comprehensive statistics and metrics
//   - Execution lifecycle callbacks
//
// Basic Usage:
//
//	import (
//	    "context"
//	    "github.com/fluxorio/fluxor/pkg/core/resilience/bulkhead"
//	)
//
//	// Create manager with default config (max 10 concurrent executions)
//	manager := bulkhead.NewManager()
//
//	// Execute function within bulkhead limits
//	err := manager.Execute(ctx, func(ctx context.Context) error {
//	    // Your operation that should be limited
//	    return someResourceIntensiveOperation(ctx)
//	})
//
// Advanced Usage with Custom Config:
//
//	config := bulkhead.Config{
//	    MaxConcurrency: 5,        // Maximum 5 concurrent executions
//	    MaxQueueSize:   10,       // Queue up to 10 waiting executions
//	    QueueTimeout:   5*time.Second, // Timeout for queue wait
//	    OnRejected: func(ctx context.Context, reason string) {
//	        log.Printf("Execution rejected: %s", reason)
//	    },
//	    OnExecuting: func(ctx context.Context) {
//	        log.Printf("Execution started")
//	    },
//	    OnCompleted: func(ctx context.Context, err error) {
//	        log.Printf("Execution completed: %v", err)
//	    },
//	}
//
//	manager := bulkhead.NewManagerWithConfig(config)
//	err := manager.Execute(ctx, myFunction)
//
// Queuing Behavior:
//
// When MaxQueueSize is 0 (default):
//   - Executions are rejected immediately if bulkhead is full
//   - Returns ErrCodeBulkheadFull error
//
// When MaxQueueSize > 0:
//   - Executions wait in queue if bulkhead is full
//   - QueueTimeout controls maximum wait time in queue
//   - If QueueTimeout is 0, executions wait indefinitely (until context cancelled)
//   - If queue is also full, execution is rejected with ErrCodeBulkheadFull
//
// Context Support:
//
// The bulkhead package fully supports context cancellation and timeouts:
//
//   - If context is cancelled before acquiring slot, ErrCodeContextCanceled is returned
//   - If context times out while waiting in queue, ErrCodeContextTimeout is returned
//   - The function execution respects context cancellation
//
// Thread Safety:
//
// The Manager is safe for concurrent use by multiple goroutines.
//
// Example: Limiting Database Connections
//
//	bulkhead := bulkhead.NewManagerWithConfig(bulkhead.Config{
//	    MaxConcurrency: 10, // Limit to 10 concurrent DB operations
//	    MaxQueueSize:   50, // Queue up to 50 waiting operations
//	    QueueTimeout:   30*time.Second, // Timeout after 30s in queue
//	})
//
//	err := bulkhead.Execute(ctx, func(ctx context.Context) error {
//	    // Database operation
//	    return db.Query(ctx, query)
//	})
package bulkhead
