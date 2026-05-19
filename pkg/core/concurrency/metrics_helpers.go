package concurrency

import "context"

// NewExecutorWithMetrics creates a new Executor and automatically registers it
// with the dashboard metrics registry for metrics collection.
//
// This is a convenience function that combines executor creation with metrics registration.
// The executor will appear in dashboard metrics with the provided ID.
//
// Example:
//
//	executor := concurrency.NewExecutorWithMetrics(
//	    ctx.GoCMD().Context(),
//	    "queue-processor",  // ID for metrics
//	    concurrency.DefaultExecutorConfig(),
//	)
//	// Executor is automatically registered and will appear in dashboard metrics
func NewExecutorWithMetrics(ctx context.Context, id string, config ExecutorConfig) Executor {
	executor := NewExecutor(ctx, config)
	if id != "" && executor != nil {
		RegisterExecutor(id, executor)
	}
	return executor
}

// NewWorkerPoolWithMetrics creates a new WorkerPool and automatically registers it
// with the dashboard metrics registry for metrics collection.
//
// This is a convenience function that combines worker pool creation with metrics registration.
// The worker pool will appear in dashboard metrics with the provided ID.
//
// Example:
//
//	pool := concurrency.NewWorkerPoolWithMetrics(
//	    ctx.GoCMD().Context(),
//	    "image-processor",  // ID for metrics
//	    concurrency.DefaultWorkerPoolConfig(),
//	)
//	if err := pool.Start(); err != nil {
//	    // handle error
//	}
//	// Worker pool is automatically registered and will appear in dashboard metrics
func NewWorkerPoolWithMetrics(ctx context.Context, id string, config WorkerPoolConfig) WorkerPool {
	pool := NewWorkerPool(ctx, config)
	if id != "" && pool != nil {
		RegisterWorkerPool(id, pool)
	}
	return pool
}
