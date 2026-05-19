package middleware

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
)

// TimeoutConfig configures request timeout middleware
type TimeoutConfig struct {
	// Timeout is the request timeout duration
	Timeout time.Duration

	// Logger is the logger to use for timeout logging (default: core.NewDefaultLogger())
	Logger core.Logger

	// Message is the error message when timeout occurs
	Message string

	// SkipPaths is a list of paths to skip timeout
	SkipPaths []string
}

// DefaultTimeoutConfig returns a default timeout configuration
func DefaultTimeoutConfig(timeout time.Duration) TimeoutConfig {
	return TimeoutConfig{
		Timeout:   timeout,
		Logger:    core.NewDefaultLogger(),
		Message:   "Request timeout",
		SkipPaths: []string{},
	}
}

// Timeout middleware enforces request timeouts
func Timeout(config TimeoutConfig) web.FastMiddleware {
	if config.Timeout <= 0 {
		panic("Timeout: timeout duration must be positive")
	}

	logger := config.Logger
	if logger == nil {
		logger = core.NewDefaultLogger()
	}

	message := config.Message
	if message == "" {
		message = "Request timeout"
	}

	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			// Check if path should be skipped
			path := string(ctx.Path())
			skip := false
			for _, skipPath := range config.SkipPaths {
				if path == skipPath || strings.HasPrefix(path, skipPath) {
					skip = true
					break
				}
			}

			if skip {
				return next(ctx)
			}

			// Create timeout context
			timeoutCtx, cancel := context.WithTimeout(ctx.Context(), config.Timeout)
			defer cancel()

			// Create new request context with timeout context
			// Note: We can't easily replace the context in FastRequestContext,
			// so we'll use a channel to signal completion
			done := make(chan error, 1)

			go func() {
				// Execute handler in goroutine
				// Note: This is a limitation - we can't easily pass the timeout context
				// to the handler. The handler should respect context cancellation.
				done <- next(ctx)
			}()

			select {
			case err := <-done:
				return err
			case <-timeoutCtx.Done():
				// Timeout occurred
				fields := make(map[string]interface{})
				fields["request_id"] = ctx.RequestID()
				fields["method"] = string(ctx.Method())
				fields["path"] = path
				fields["timeout"] = config.Timeout.String()

				logger.WithFields(fields).Info(fmt.Sprintf("Request timeout: %s %s", string(ctx.Method()), path))

				ctx.RequestCtx.SetStatusCode(504) // Gateway Timeout
				ctx.RequestCtx.SetContentType("application/json")
				if _, err := ctx.RequestCtx.WriteString(`{"error":"timeout","message":"` + message + `","request_id":"` + ctx.RequestID() + `"}`); err != nil {
					// Best-effort response write; ignore on error.
				}
				return nil
			}
		}
	}
}
