package actor

import (
	"context"

	"github.com/fluxorio/fluxor/pkg/core/eventloop"
)

// ioTask implements concurrency.Task for write I/O: runs work on a WorkerPool, then dispatches callback to actor's EventLoop.
type ioTask struct {
	name     string
	work     func() error
	callback func(error)
	actor    *Actor
}

func (t *ioTask) Name() string { return t.name }

func (t *ioTask) Execute(ctx context.Context) error {
	err := t.work()
	if t.callback != nil {
		errCopy := err
		_ = t.actor.Dispatch(ctx, "io-callback-"+t.name, func(ctx context.Context, event *eventloop.Event) error {
			t.callback(errCopy)
			return nil
		})
	}
	return nil
}

// ioTaskWithResult implements concurrency.Task for read I/O: runs work on a WorkerPool, then dispatches callback to actor's EventLoop.
type ioTaskWithResult struct {
	name     string
	work     func() (interface{}, error)
	callback func(interface{}, error)
	actor    *Actor
}

func (t *ioTaskWithResult) Name() string { return t.name }

func (t *ioTaskWithResult) Execute(ctx context.Context) error {
	result, err := t.work()
	if t.callback != nil {
		resultCopy := result
		errCopy := err
		_ = t.actor.Dispatch(ctx, "io-callback-"+t.name, func(ctx context.Context, event *eventloop.Event) error {
			t.callback(resultCopy, errCopy)
			return nil
		})
	}
	return nil
}
