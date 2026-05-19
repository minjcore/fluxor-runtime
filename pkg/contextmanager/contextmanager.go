package contextmanager

import (
	"context"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// WithScope runs fn with a child context that is cancelled when fn returns.
// Use for scoped work (request handling, goroutine) so cancellation and cleanup
// happen when the scope exits. Parent must be non-nil (fail-fast).
//
// Example:
//
//	err := contextmanager.WithScope(ctx, func(scoped context.Context) error {
//	    return doWork(scoped)
//	})
func WithScope(parent context.Context, fn func(ctx context.Context) error) error {
	failfast.NotNil(parent, "parent context")
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	return fn(ctx)
}

// WithCancel returns a copy of parent with a new Done channel and a cancel function.
// Parent must be non-nil (fail-fast). Caller must call cancel when done (e.g. defer cancel()).
func WithCancel(parent context.Context) (context.Context, context.CancelFunc) {
	failfast.NotNil(parent, "parent context")
	return context.WithCancel(parent)
}

// WithTimeout returns a copy of parent with a deadline of now+duration.
// Parent must be non-nil; duration must be positive (fail-fast).
func WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	failfast.NotNil(parent, "parent context")
	failfast.If(timeout > 0, "timeout must be positive, got %v", timeout)
	return context.WithTimeout(parent, timeout)
}

// WithDeadline returns a copy of parent with deadline d.
// Parent must be non-nil (fail-fast).
func WithDeadline(parent context.Context, d time.Time) (context.Context, context.CancelFunc) {
	failfast.NotNil(parent, "parent context")
	return context.WithDeadline(parent, d)
}

// Run runs fn with ctx and returns fn's error. Use when fn is the only work in a scope
// and you want to propagate context cancellation. Does not create a new context.
func Run(ctx context.Context, fn func(context.Context) error) error {
	failfast.NotNil(ctx, "context")
	failfast.NotNil(fn, "fn")
	return fn(ctx)
}
