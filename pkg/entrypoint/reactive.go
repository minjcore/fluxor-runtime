package entrypoint

import (
	"context"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

// Reactive provides reactive programming utilities
// Inspired by Vert.x reactive patterns
type Reactive interface {
	// Future represents an asynchronous result
	Future() Future

	// Promise represents a writable future
	Promise() Promise

	// Compose composes multiple futures
	Compose(futures ...Future) Future
}

// Future represents an asynchronous computation
type Future interface {
	// Complete completes the future with a result
	Complete(result interface{})

	// Fail fails the future with an error
	Fail(err error)

	// Result returns the result channel
	Result() <-chan FutureResult

	// OnSuccess registers a success handler
	OnSuccess(handler func(interface{})) Future

	// OnFailure registers a failure handler
	OnFailure(handler func(error)) Future

	// Map transforms the result
	Map(fn func(interface{}) interface{}) Future

	// Await waits for the future to complete and returns the result
	// This provides async/await-style syntax for Go
	// Blocks until the future completes or context is cancelled
	Await(ctx context.Context) (interface{}, error)

	// Then chains a success handler (Node.js Promise style)
	// Returns a new Future that completes with the result of the handler
	Then(fn func(interface{}) (interface{}, error)) Future

	// Catch chains an error handler (Node.js Promise style)
	// Returns a new Future that completes with the result of the error handler
	Catch(fn func(error) (interface{}, error)) Future
}

// Promise is a writable Future
type Promise interface {
	Future

	// TryComplete attempts to complete the promise
	TryComplete(result interface{}) bool

	// TryFail attempts to fail the promise
	TryFail(err error) bool
}

// FutureResult represents the result of a future
type FutureResult struct {
	Value interface{}
	Error error
}

// Error represents a reactive error
type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

// future implements Future
type future struct {
	resultChan      chan FutureResult
	once            sync.Once
	mu              sync.RWMutex
	completed       bool
	result          FutureResult
	successHandlers []func(interface{})
	failureHandlers []func(error)
}

// NewFuture creates a new future
func NewFuture() Future {
	return &future{
		resultChan:      make(chan FutureResult, 1),
		successHandlers: make([]func(interface{}), 0),
		failureHandlers: make([]func(error), 0),
	}
}

func (f *future) Complete(result interface{}) {
	f.once.Do(func() {
		f.mu.Lock()
		f.completed = true
		f.result = FutureResult{Value: result}
		f.mu.Unlock()

		select {
		case f.resultChan <- f.result:
		default:
		}

		// Call success handlers
		for _, handler := range f.successHandlers {
			handler(result)
		}
	})
}

func (f *future) Fail(err error) {
	f.once.Do(func() {
		f.mu.Lock()
		f.completed = true
		f.result = FutureResult{Error: err}
		f.mu.Unlock()

		select {
		case f.resultChan <- f.result:
		default:
		}

		// Call failure handlers
		for _, handler := range f.failureHandlers {
			handler(err)
		}
	})
}

func (f *future) Result() <-chan FutureResult {
	return f.resultChan
}

func (f *future) OnSuccess(handler func(interface{})) Future {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.completed && f.result.Error == nil {
		handler(f.result.Value)
	} else {
		f.successHandlers = append(f.successHandlers, handler)
	}

	return f
}

func (f *future) OnFailure(handler func(error)) Future {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.completed && f.result.Error != nil {
		handler(f.result.Error)
	} else {
		f.failureHandlers = append(f.failureHandlers, handler)
	}

	return f
}

func (f *future) Map(fn func(interface{}) interface{}) Future {
	mapped := NewFuture()

	f.OnSuccess(func(result interface{}) {
		mapped.Complete(fn(result))
	})

	f.OnFailure(func(err error) {
		mapped.Fail(err)
	})

	return mapped
}

// Await waits for the future to complete and returns the result
// Provides async/await-style syntax: result, err := future.Await(ctx)
func (f *future) Await(ctx context.Context) (interface{}, error) {
	// Check if already completed
	f.mu.RLock()
	if f.completed {
		result := f.result
		f.mu.RUnlock()
		if result.Error != nil {
			return nil, result.Error
		}
		return result.Value, nil
	}
	f.mu.RUnlock()

	// Wait for completion or context cancellation
	select {
	case result := <-f.resultChan:
		if result.Error != nil {
			return nil, result.Error
		}
		return result.Value, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Then chains a success handler (Node.js Promise.then() style)
// Returns a new Future that completes with the result of the handler
func (f *future) Then(fn func(interface{}) (interface{}, error)) Future {
	mapped := NewFuture()

	f.OnSuccess(func(result interface{}) {
		newResult, err := fn(result)
		if err != nil {
			mapped.Fail(err)
		} else {
			mapped.Complete(newResult)
		}
	})

	f.OnFailure(func(err error) {
		mapped.Fail(err)
	})

	return mapped
}

// Catch chains an error handler (Node.js Promise.catch() style)
// Returns a new Future that completes with the result of the error handler
func (f *future) Catch(fn func(error) (interface{}, error)) Future {
	mapped := NewFuture()

	f.OnSuccess(func(result interface{}) {
		mapped.Complete(result)
	})

	f.OnFailure(func(err error) {
		newResult, handlerErr := fn(err)
		if handlerErr != nil {
			mapped.Fail(handlerErr)
		} else {
			mapped.Complete(newResult)
		}
	})

	return mapped
}

// promise implements Promise
type promise struct {
	Future
}

// NewPromise creates a new promise
func NewPromise() Promise {
	return &promise{
		Future: NewFuture(),
	}
}

func (p *promise) TryComplete(result interface{}) bool {
	p.Complete(result)
	return true
}

func (p *promise) TryFail(err error) bool {
	p.Fail(err)
	return true
}

// ReactiveVerticle is a verticle that uses reactive patterns
type ReactiveVerticle struct {
	core.Verticle
	gocmd core.GoCMD
}

// NewReactiveVerticle creates a new reactive verticle
func NewReactiveVerticle(gocmd core.GoCMD) *ReactiveVerticle {
	return &ReactiveVerticle{
		gocmd: gocmd,
	}
}

// ExecuteReactive executes a task reactively using the event bus
func (rv *ReactiveVerticle) ExecuteReactive(ctx context.Context, address string, data interface{}) Future {
	promise := NewPromise()

	// Send request via event bus
	msg, err := rv.gocmd.EventBus().Request(address, data, 5*time.Second)
	if err != nil {
		promise.Fail(err)
		return promise
	}

	// Handle reply asynchronously
	go func() {
		// In a real implementation, we'd wait for the reply message
		// For now, we'll complete with the message body
		if msg.Body() != nil {
			promise.Complete(msg.Body())
		} else {
			promise.Fail(&Error{Message: "no reply received"})
		}
	}()

	return promise
}
