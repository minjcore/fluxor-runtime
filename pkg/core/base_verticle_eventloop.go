package core

import (
	"context"

	"github.com/fluxorio/fluxor/pkg/core/concurrency"
)

// EventLoop returns the event loop executor for this verticle
// Each verticle has its own event loop for sequential event processing
func (bv *BaseVerticle) EventLoop() concurrency.Executor {
	bv.mu.RLock()
	defer bv.mu.RUnlock()
	return bv.eventLoop
}

// RunOnEventLoop executes a task on this verticle's event loop
// Tasks are processed sequentially (single worker) - ensures race-free operation
// All tasks submitted via this method execute in the same goroutine, preventing
// concurrent access to verticle state
//
// IMPORTANT: Event loop tasks must complete in < 20µs
// Use SubmitBlocking() for any work that might take longer (CPU-bound, blocking I/O)
func (bv *BaseVerticle) RunOnEventLoop(task concurrency.Task) error {
	// Fail-fast: task cannot be nil
	if task == nil {
		return &EventBusError{Code: "INVALID_TASK", Message: "task cannot be nil"}
	}
	bv.mu.RLock()
	eventLoop := bv.eventLoop
	bv.mu.RUnlock()

	if eventLoop == nil {
		return &EventBusError{Code: "NOT_STARTED", Message: "verticle not started - event loop not available"}
	}

	return eventLoop.Submit(task)
}

// SubmitBlocking submits blocking work to the worker pool and returns a Future
// This is the recommended way to execute CPU-bound or blocking operations
// without blocking the event loop
//
// Pattern:
//   - EventLoop: IO + dispatch only (< 20µs per task)
//   - WorkerPool: Blocking work (CPU-bound, blocking I/O)
//   - Future: Async result handling
//
// Example:
//
//	future := v.SubmitBlocking(func() (string, error) {
//	    // CPU-intensive work or blocking I/O
//	    result := heavyComputation()
//	    return result, nil
//	})
//
//	// Option 1: Await result
//	result, err := future.Await(ctx)
//
//	// Option 2: Use callbacks
//	future.OnSuccess(func(result string) {
//	    // Handle success
//	}).OnFailure(func(err error) {
//	    // Handle error
//	})
// SubmitBlockingFunc is a helper function to submit blocking work
// This is a standalone function because Go doesn't support generic methods
func SubmitBlockingFunc[T any](bv *BaseVerticle, fn func() (T, error)) concurrency.Future[T] {
	// Fail-fast: function cannot be nil
	if fn == nil {
		promise := concurrency.NewPromise[T]()
		promise.Fail(&EventBusError{Code: "INVALID_FUNCTION", Message: "function cannot be nil"})
		return promise
	}

	bv.mu.RLock()
	workerPool := bv.workerPool
	bv.mu.RUnlock()

	if workerPool == nil {
		promise := concurrency.NewPromise[T]()
		promise.Fail(&EventBusError{Code: "NOT_STARTED", Message: "verticle not started - worker pool not available"})
		return promise
	}

	// Create promise for async result
	promise := concurrency.NewPromise[T]()

	// Create task that executes the function and completes the promise
	task := concurrency.TaskFunc(func(ctx context.Context) error {
		result, err := fn()
		if err != nil {
			promise.Fail(err)
			return err
		}
		promise.Complete(result)
		return nil
	})

	// Submit to worker pool (non-blocking)
	if err := workerPool.Submit(task); err != nil {
		promise.Fail(err)
	}

	return promise
}
