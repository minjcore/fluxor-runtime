package concurrency

import (
	"context"
)

// WorkerPoolStats provides statistics about worker pool performance
type WorkerPoolStats struct {
	QueuedTasks      int64   // Current number of queued tasks
	ActiveWorkers    int     // Number of active worker goroutines
	CompletedTasks   int64   // Total completed tasks
	RejectedTasks    int64   // Total rejected tasks (backpressure)
	QueueCapacity    int     // Maximum queue capacity
	QueueUtilization float64 // Queue utilization percentage
}

// WorkerPool abstracts worker goroutine management
// Hides go func() calls and goroutine lifecycle from application code
type WorkerPool interface {
	// Start starts the worker pool
	// Initializes worker goroutines and begins processing tasks
	Start() error

	// Stop gracefully stops the worker pool
	// Waits for in-flight tasks to complete (up to ctx timeout)
	// Returns error if stop times out
	Stop(ctx context.Context) error

	// Submit submits a task to the worker pool
	// Returns error if pool is closed or queue is full
	Submit(task Task) error

	// SubmitBatch submits multiple tasks to the worker pool
	// Returns error if any task cannot be queued (partial submission may occur)
	// More efficient than multiple Submit() calls for high-throughput scenarios
	SubmitBatch(tasks []Task) error

	// Workers returns the number of worker goroutines
	Workers() int

	// IsRunning returns true if the worker pool is running
	IsRunning() bool

	// Stats returns current worker pool statistics
	Stats() WorkerPoolStats
}
