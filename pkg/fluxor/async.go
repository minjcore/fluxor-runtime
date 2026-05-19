package entrypoint

import (
	"context"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

// Async provides async/await-style patterns combining Vert.x and Node.js paradigms
// This package provides type-safe, context-aware async operations

// FutureT is a type-safe Future using Go generics
// Combines Vert.x Future patterns with Node.js Promise ergonomics
// This is a struct, not an interface, because Go doesn't allow type parameters on interface methods
type FutureT[T any] struct {
	future Future
}

// PromiseT is a type-safe Promise using Go generics
type PromiseT[T any] struct {
	FutureT[T]
}

// NewFutureT creates a new type-safe Future
func NewFutureT[T any]() *FutureT[T] {
	return &FutureT[T]{
		future: NewFuture(),
	}
}

// NewPromiseT creates a new type-safe Promise
func NewPromiseT[T any]() *PromiseT[T] {
	return &PromiseT[T]{
		FutureT: FutureT[T]{
			future: NewPromise(),
		},
	}
}

// Await waits for the future to complete and returns the typed result
// Provides async/await-style syntax: result, err := future.Await(ctx)
func (f *FutureT[T]) Await(ctx context.Context) (T, error) {
	var zero T
	result, err := f.future.Await(ctx)
	if err != nil {
		return zero, err
	}

	// Type assertion
	typed, ok := result.(T)
	if !ok {
		return zero, &Error{Message: "type assertion failed"}
	}
	return typed, nil
}

// Then chains a success handler (Node.js Promise.then() style)
// Returns a new Future with the transformed type
// Accepts both FutureT and PromiseT (PromiseT embeds FutureT)
func Then[T any, R any](f interface {
	Await(context.Context) (T, error)
}, fn func(T) (R, error)) *FutureT[R] {
	mapped := NewFutureT[R]()

	// Get the underlying Future from FutureT or PromiseT
	var future Future
	switch v := f.(type) {
	case *FutureT[T]:
		future = v.future
	case *PromiseT[T]:
		future = v.future
	default:
		mapped.future.Fail(&Error{Message: "invalid future type"})
		return mapped
	}

	future.OnSuccess(func(result interface{}) {
		typed, ok := result.(T)
		if !ok {
			mapped.future.Fail(&Error{Message: "type assertion failed"})
			return
		}

		newResult, err := fn(typed)
		if err != nil {
			mapped.future.Fail(err)
		} else {
			mapped.future.Complete(newResult)
		}
	})

	future.OnFailure(func(err error) {
		mapped.future.Fail(err)
	})

	return mapped
}

// Catch chains an error handler (Node.js Promise.catch() style)
// Returns a new Future that recovers from errors
// Accepts both FutureT and PromiseT
func Catch[T any](f interface {
	Await(context.Context) (T, error)
}, fn func(error) (T, error)) *FutureT[T] {
	mapped := NewFutureT[T]()

	// Get the underlying Future
	var future Future
	switch v := f.(type) {
	case *FutureT[T]:
		future = v.future
	case *PromiseT[T]:
		future = v.future
	default:
		mapped.future.Fail(&Error{Message: "invalid future type"})
		return mapped
	}

	future.OnSuccess(func(result interface{}) {
		typed, ok := result.(T)
		if !ok {
			mapped.future.Fail(&Error{Message: "type assertion failed"})
			return
		}
		mapped.future.Complete(typed)
	})

	future.OnFailure(func(err error) {
		newResult, handlerErr := fn(err)
		if handlerErr != nil {
			mapped.future.Fail(handlerErr)
		} else {
			mapped.future.Complete(newResult)
		}
	})

	return mapped
}

// Map transforms the result synchronously
// Accepts both FutureT and PromiseT
func Map[T any, R any](f interface {
	Await(context.Context) (T, error)
}, fn func(T) R) *FutureT[R] {
	mapped := NewFutureT[R]()

	// Get the underlying Future
	var future Future
	switch v := f.(type) {
	case *FutureT[T]:
		future = v.future
	case *PromiseT[T]:
		future = v.future
	default:
		mapped.future.Fail(&Error{Message: "invalid future type"})
		return mapped
	}

	future.OnSuccess(func(result interface{}) {
		typed, ok := result.(T)
		if !ok {
			mapped.future.Fail(&Error{Message: "type assertion failed"})
			return
		}
		mapped.future.Complete(fn(typed))
	})

	future.OnFailure(func(err error) {
		mapped.future.Fail(err)
	})

	return mapped
}

// OnSuccess registers a callback (Vert.x style)
func (f *FutureT[T]) OnSuccess(handler func(T)) *FutureT[T] {
	f.future.OnSuccess(func(result interface{}) {
		typed, ok := result.(T)
		if ok {
			handler(typed)
		}
	})
	return f
}

// OnFailure registers an error callback (Vert.x style)
func (f *FutureT[T]) OnFailure(handler func(error)) *FutureT[T] {
	f.future.OnFailure(handler)
	return f
}

// Complete completes the promise with a typed value
func (p *PromiseT[T]) Complete(value T) {
	p.future.Complete(value)
}

// Fail fails the promise with an error
func (p *PromiseT[T]) Fail(err error) {
	p.future.Fail(err)
}

// RequestAsync sends a request and returns a type-safe Future
// Combines Vert.x request-reply with async/await patterns
func RequestAsync[T any](eb core.EventBus, ctx context.Context, address string, data interface{}, timeout time.Duration) *FutureT[T] {
	promise := NewPromiseT[T]()

	// Execute request in goroutine
	go func() {
		msg, err := eb.Request(address, data, timeout)
		if err != nil {
			promise.Fail(err)
			return
		}

		// Decode message body to type T
		var result T
		if msg.Body() != nil {
			// Try to decode JSON if body is []byte
			if bodyBytes, ok := msg.Body().([]byte); ok {
				if err := core.JSONDecode(bodyBytes, &result); err != nil {
					// If decode fails, try direct type assertion
					if typed, ok := msg.Body().(T); ok {
						result = typed
					} else {
						promise.Fail(&Error{Message: "failed to decode message body"})
						return
					}
				}
			} else if typed, ok := msg.Body().(T); ok {
				result = typed
			} else {
				promise.Fail(&Error{Message: "message body type mismatch"})
				return
			}
		}

		promise.Complete(result)
	}()

	return &promise.FutureT
}

// SendAsync sends a message and returns a Future that completes when sent
func SendAsync(eb core.EventBus, ctx context.Context, address string, data interface{}) *FutureT[bool] {
	promise := NewPromiseT[bool]()

	go func() {
		if err := eb.Send(address, data); err != nil {
			promise.Fail(err)
			return
		}
		promise.Complete(true)
	}()

	return &promise.FutureT
}

// PublishAsync publishes a message and returns a Future that completes when published
func PublishAsync(eb core.EventBus, ctx context.Context, address string, data interface{}) *FutureT[bool] {
	promise := NewPromiseT[bool]()

	go func() {
		if err := eb.Publish(address, data); err != nil {
			promise.Fail(err)
			return
		}
		promise.Complete(true)
	}()

	return &promise.FutureT
}

// All waits for all futures to complete (Promise.all() style)
// Accepts both FutureT and PromiseT
func All[T any](ctx context.Context, futures ...interface {
	Await(context.Context) (T, error)
}) *FutureT[[]T] {
	promise := NewPromiseT[[]T]()

	go func() {
		results := make([]T, 0, len(futures))
		for _, f := range futures {
			result, err := f.Await(ctx)
			if err != nil {
				promise.Fail(err)
				return
			}
			results = append(results, result)
		}
		promise.Complete(results)
	}()

	return &promise.FutureT
}

// Race returns the first future that completes (Promise.race() style)
// Accepts both FutureT and PromiseT
func Race[T any](ctx context.Context, futures ...interface {
	Await(context.Context) (T, error)
}) *FutureT[T] {
	promise := NewPromiseT[T]()

	go func() {
		resultChan := make(chan T, 1)
		errChan := make(chan error, 1)

		for _, f := range futures {
			go func(future interface {
				Await(context.Context) (T, error)
			}) {
				result, err := future.Await(ctx)
				if err != nil {
					select {
					case errChan <- err:
					default:
					}
				} else {
					select {
					case resultChan <- result:
					default:
					}
				}
			}(f)
		}

		select {
		case result := <-resultChan:
			promise.Complete(result)
		case err := <-errChan:
			promise.Fail(err)
		case <-ctx.Done():
			promise.Fail(ctx.Err())
		}
	}()

	return &promise.FutureT
}
