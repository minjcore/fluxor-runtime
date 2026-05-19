package web

import (
	"fmt"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

// LoggingGinMiddleware records request duration with the given logger (reuse the same pattern for workers / EventBus by calling the inner logic with your own logger).
// Register with GinRouter().UseGin(LoggingGinMiddleware(logger)) before routes.
func LoggingGinMiddleware(logger core.Logger) GinMiddleware {
	if logger == nil {
		return func(next GinRequestHandler) GinRequestHandler { return next }
	}
	return func(next GinRequestHandler) GinRequestHandler {
		return func(ctx *GinRequestContext) error {
			start := time.Now()
			err := next(ctx)
			method, path := "", ""
			if ctx.GinCtx != nil && ctx.GinCtx.Request != nil {
				method = ctx.GinCtx.Request.Method
				path = ctx.GinCtx.FullPath()
			}
			logger.Debug(fmt.Sprintf("gin %s %s %s %v", method, path, ctx.RequestID(), time.Since(start)))
			return err
		}
	}
}
