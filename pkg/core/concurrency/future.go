package concurrency

import (
	"context"
	"sync"
	"sync/atomic"
)

// Future represents an asynchronous computation result
// Provides type-safe async/await pattern for blocking work
type Future[T any] interface {
	// Await waits for the future to complete and returns the result
	// Blocks until the future completes or context is cancelled
	Await(ctx context.Context) (T, error)

	// OnSuccess registers a success handler
	// Handler is called when the future completes successfully
	OnSuccess(handler func(T)) Future[T]

	// OnFailure registers a failure handler
	// Handler is called when the future fails
	OnFailure(handler func(error)) Future[T]
}

// Promise is a writable Future
// Used to create and complete futures from blocking work
type Promise[T any] interface {
	Future[T]

	// Complete completes the promise with a result
	Complete(result T)

	// Fail fails the promise with an error
	Fail(err error)
}

// future implements Future[T]
type future[T any] struct {
	mu          sync.RWMutex
	resultChan  chan futureResult[T]
	result      *futureResult[T]
	completed   int32 // Atomic flag
	successHandlers []func(T)
	failureHandlers []func(error)
	handlersMu  sync.Mutex
}

// futureResult holds the result or error
type futureResult[T any] struct {
	value T
	err   error
}

// NewFuture creates a new Future
func NewFuture[T any]() Future[T] {
	return &future[T]{
		resultChan:      make(chan futureResult[T], 1),
		successHandlers: make([]func(T), 0),
		failureHandlers: make([]func(error), 0),
	}
}

// NewPromise creates a new Promise
func NewPromise[T any]() Promise[T] {
	return &future[T]{
		resultChan:      make(chan futureResult[T], 1),
		successHandlers: make([]func(T), 0),
		failureHandlers: make([]func(error), 0),
	}
}

// Await waits for the future to complete and returns the result
func (f *future[T]) Await(ctx context.Context) (T, error) {
	var zero T

	// Check if already completed (fast path)
	if atomic.LoadInt32(&f.completed) == 1 {
		f.mu.RLock()
		result := f.result
		f.mu.RUnlock()
		if result != nil {
			if result.err != nil {
				return zero, result.err
			}
			return result.value, nil
		}
	}

	// Wait for completion or context cancellation
	select {
	case result := <-f.resultChan:
		// Store result for future reads
		f.mu.Lock()
		f.result = &result
		atomic.StoreInt32(&f.completed, 1)
		f.mu.Unlock()

		if result.err != nil {
			return zero, result.err
		}
		return result.value, nil
	case <-ctx.Done():
		return zero, ctx.Err()
	}
}

// OnSuccess registers a success handler
func (f *future[T]) OnSuccess(handler func(T)) Future[T] {
	if handler == nil {
		return f
	}

	f.handlersMu.Lock()
	f.successHandlers = append(f.successHandlers, handler)
	handlers := make([]func(T), len(f.successHandlers))
	copy(handlers, f.successHandlers)
	f.handlersMu.Unlock()

	// If already completed, call handler immediately
	if atomic.LoadInt32(&f.completed) == 1 {
		f.mu.RLock()
		result := f.result
		f.mu.RUnlock()
		if result != nil && result.err == nil {
			handler(result.value)
		}
	}

	return f
}

// OnFailure registers a failure handler
func (f *future[T]) OnFailure(handler func(error)) Future[T] {
	if handler == nil {
		return f
	}

	f.handlersMu.Lock()
	f.failureHandlers = append(f.failureHandlers, handler)
	handlers := make([]func(error), len(f.failureHandlers))
	copy(handlers, f.failureHandlers)
	f.handlersMu.Unlock()

	// If already completed, call handler immediately
	if atomic.LoadInt32(&f.completed) == 1 {
		f.mu.RLock()
		result := f.result
		f.mu.RUnlock()
		if result != nil && result.err != nil {
			handler(result.err)
		}
	}

	return f
}

// Complete completes the promise with a result
func (f *future[T]) Complete(result T) {
	if !atomic.CompareAndSwapInt32(&f.completed, 0, 1) {
		return // Already completed
	}

	f.mu.Lock()
	f.result = &futureResult[T]{value: result}
	f.mu.Unlock()

	// Send result to channel (non-blocking)
	select {
	case f.resultChan <- futureResult[T]{value: result}:
	default:
		// Channel already has result (shouldn't happen, but safe)
	}

	// Call success handlers
	f.handlersMu.Lock()
	handlers := make([]func(T), len(f.successHandlers))
	copy(handlers, f.successHandlers)
	f.handlersMu.Unlock()

	for _, handler := range handlers {
		handler(result)
	}
}

// Fail fails the promise with an error
func (f *future[T]) Fail(err error) {
	if err == nil {
		return // Cannot fail with nil error
	}

	if !atomic.CompareAndSwapInt32(&f.completed, 0, 1) {
		return // Already completed
	}

	f.mu.Lock()
	f.result = &futureResult[T]{err: err}
	f.mu.Unlock()

	// Send error to channel (non-blocking)
	select {
	case f.resultChan <- futureResult[T]{err: err}:
	default:
		// Channel already has result (shouldn't happen, but safe)
	}

	// Call failure handlers
	f.handlersMu.Lock()
	handlers := make([]func(error), len(f.failureHandlers))
	copy(handlers, f.failureHandlers)
	f.handlersMu.Unlock()

	for _, handler := range handlers {
		handler(err)
	}
}
