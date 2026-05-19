package actor

import (
	"context"
	"fmt"

	"github.com/fluxorio/fluxor/pkg/core/concurrency"
	"github.com/fluxorio/fluxor/pkg/core/eventloop"
)

// Actor represents a logical actor: a stable key for routing plus an EventLoop for serialized message handling.
// Optional ReadPool/WritePool offload blocking I/O; callbacks are dispatched back to this actor's EventLoop.
type Actor struct {
	key       string
	eventLoop *eventloop.EventLoopGroup
	readPool  concurrency.WorkerPool
	writePool concurrency.WorkerPool
}

// Option configures an Actor.
type Option func(*Actor)

// WithReadPool sets the worker pool for blocking read I/O. Callbacks run on this actor's EventLoop.
func WithReadPool(pool concurrency.WorkerPool) Option {
	return func(a *Actor) { a.readPool = pool }
}

// WithWritePool sets the worker pool for blocking write I/O. Callbacks run on this actor's EventLoop.
func WithWritePool(pool concurrency.WorkerPool) Option {
	return func(a *Actor) { a.writePool = pool }
}

// NewActor creates an actor with the given routing key and event loop group.
func NewActor(key string, eventLoop *eventloop.EventLoopGroup, opts ...Option) *Actor {
	a := &Actor{
		key:       key,
		eventLoop: eventLoop,
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

// Key returns the actor's routing key (same key always routes to the same EventLoop).
func (a *Actor) Key() string { return a.key }

// EventLoop returns the EventLoopGroup this actor uses for dispatch.
func (a *Actor) EventLoop() *eventloop.EventLoopGroup { return a.eventLoop }

// Dispatch sends a handler to this actor's EventLoop (serialized with other work for this key).
// Uses the actor's key so all dispatches for this actor run on the same loop.
// The handler MUST NOT block; use SubmitRead/SubmitWrite for blocking work.
func (a *Actor) Dispatch(ctx context.Context, name string, handler func(ctx context.Context, event *eventloop.Event) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	event := &eventloop.Event{
		Key:     a.key,
		Address: name,
		Handler: handler,
	}
	return a.eventLoop.DispatchByKey(a.key, event)
}

// SubmitRead runs work on the actor's read worker pool; when done, callback is dispatched to this actor's EventLoop.
// The callback MUST NOT block; use another SubmitRead/SubmitWrite for further blocking work.
// Returns an error if ReadPool was not set (WithReadPool).
func (a *Actor) SubmitRead(name string, work func() (interface{}, error), callback func(interface{}, error)) error {
	if a.readPool == nil {
		return fmt.Errorf("actor: ReadPool not set")
	}
	task := &ioTaskWithResult{
		name:     name,
		work:     work,
		callback: callback,
		actor:    a,
	}
	return a.readPool.Submit(task)
}

// SubmitWrite runs work on the actor's write worker pool; when done, callback is dispatched to this actor's EventLoop.
// The callback MUST NOT block; use another SubmitRead/SubmitWrite for further blocking work.
// Returns an error if WritePool was not set (WithWritePool).
func (a *Actor) SubmitWrite(name string, work func() error, callback func(error)) error {
	if a.writePool == nil {
		return fmt.Errorf("actor: WritePool not set")
	}
	task := &ioTask{
		name:     name,
		work:     work,
		callback: callback,
		actor:    a,
	}
	return a.writePool.Submit(task)
}
