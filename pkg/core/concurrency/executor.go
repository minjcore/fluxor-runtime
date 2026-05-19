package concurrency

import (
	"context"
	"time"
)

// ExecutorStats provides statistics about executor performance
type ExecutorStats struct {
	QueuedTasks      int64   // Current number of queued tasks
	ActiveWorkers    int     // Number of active worker goroutines
	CompletedTasks   int64   // Total completed tasks
	RejectedTasks    int64   // Total rejected tasks (backpressure)
	QueueCapacity    int     // Maximum queue capacity
	QueueUtilization float64 // Queue utilization percentage
}

// Executor abstracts goroutine pool management and task execution
// Hides channel operations and goroutine creation from application code
type Executor interface {
	// Submit queues a task for execution
	// Returns error if queue is full (backpressure) or executor is closed
	Submit(task Task) error

	// SubmitWithTimeout queues a task with a timeout
	// Returns error if task cannot be queued within timeout
	SubmitWithTimeout(task Task, timeout time.Duration) error

	// SubmitBatch queues multiple tasks for execution
	// Returns error if any task cannot be queued (partial submission may occur)
	// More efficient than multiple Submit() calls for high-throughput scenarios
	SubmitBatch(tasks []Task) error

	// Shutdown gracefully shuts down the executor
	// Waits for queued tasks to complete (up to ctx timeout)
	// Returns error if shutdown times out
	Shutdown(ctx context.Context) error

	// Stats returns current executor statistics
	Stats() ExecutorStats
}
