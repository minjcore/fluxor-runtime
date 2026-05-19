package middleware

import (
	"fmt"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
)

// RecoveryConfig configures panic recovery middleware
type RecoveryConfig struct {
	// Logger is the logger to use for panic logging (default: core.NewDefaultLogger())
	Logger core.Logger

	// StackTrace includes stack trace in error response (use with caution in production)
	StackTrace bool
}

// DefaultRecoveryConfig returns a default recovery configuration
func DefaultRecoveryConfig() RecoveryConfig {
	return RecoveryConfig{
		Logger:     core.NewDefaultLogger(),
		StackTrace: false,
	}
}

// Recovery middleware recovers from panics and returns 500 error
func Recovery(config RecoveryConfig) web.FastMiddleware {
	logger := config.Logger
	if logger == nil {
		logger = core.NewDefaultLogger()
	}

	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			defer func() {
				if r := recover(); r != nil {
					// Log panic with request context
					fields := make(map[string]interface{})
					fields["request_id"] = ctx.RequestID()
					fields["method"] = string(ctx.Method())
					fields["path"] = string(ctx.Path())
					fields["panic"] = r

					logger.WithFields(fields).Error(fmt.Sprintf("Panic recovered: %v", r))

					// Return 500 error
					ctx.RequestCtx.SetStatusCode(500)
					ctx.RequestCtx.SetContentType("application/json")

					errorMsg := "Internal Server Error"
					if config.StackTrace {
						errorMsg = fmt.Sprintf("Panic: %v", r)
					}

					// Error intentionally ignored - best effort response for panic recovery
					_, _ = ctx.RequestCtx.WriteString(fmt.Sprintf(`{"error":"internal_server_error","message":"%s","request_id":"%s"}`, errorMsg, ctx.RequestID()))
				}
			}()

			return next(ctx)
		}
	}
}
