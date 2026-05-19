package kernel

import (
	"fmt"
	"log"
	"time"
)

// Handler is transport-agnostic application logic (HTTP, EventBus, CLI, workers).
type Handler func(ctx AppContext) error

// Middleware composes handlers (Nest/Spring-style), independent of Gin.
type Middleware func(Handler) Handler

// Chain builds one Middleware from outer → inner (first mw runs first on the request path).
func Chain(mw ...Middleware) Middleware {
	return func(final Handler) Handler {
		for i := len(mw) - 1; i >= 0; i-- {
			final = mw[i](final)
		}
		return final
	}
}

// Logging logs latency; pass nil logger to use the standard log package.
func Logging(logger func(format string, args ...interface{})) Middleware {
	if logger == nil {
		logger = log.Printf
	}
	return func(next Handler) Handler {
		return func(ctx AppContext) error {
			start := time.Now()
			err := next(ctx)
			logger("kernel handler %s took %v err=%v", ctx.RequestID(), time.Since(start), err)
			return err
		}
	}
}

// Recovery catches panics and returns them as errors.
func Recovery() Middleware {
	return func(next Handler) Handler {
		return func(ctx AppContext) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic: %v", r)
				}
			}()
			return next(ctx)
		}
	}
}
