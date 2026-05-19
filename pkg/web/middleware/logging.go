package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
)

// LoggingConfig configures request logging middleware
type LoggingConfig struct {
	// Logger is the logger to use (default: core.NewDefaultLogger())
	Logger core.Logger

	// LogRequestID includes request ID in logs
	LogRequestID bool

	// LogRequestBody logs request body (use with caution for sensitive data)
	LogRequestBody bool

	// LogResponseBody logs response body (use with caution for sensitive data)
	LogResponseBody bool

	// SkipPaths is a list of paths to skip logging
	SkipPaths []string
}

// DefaultLoggingConfig returns a default logging configuration
func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		Logger:       core.NewDefaultLogger(),
		LogRequestID: true,
		SkipPaths:    []string{"/health", "/ready", "/metrics"},
	}
}

// Logging middleware logs HTTP requests and responses
func Logging(config LoggingConfig) web.FastMiddleware {
	logger := config.Logger
	if logger == nil {
		logger = core.NewDefaultLogger()
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

			start := time.Now()
			method := string(ctx.Method())

			// Log request
			if !skip {
				fields := make(map[string]interface{})
				if config.LogRequestID {
					fields["request_id"] = ctx.RequestID()
				}
				fields["method"] = method
				fields["path"] = path
				fields["remote_addr"] = ctx.RequestCtx.RemoteIP().String()

				logger.WithFields(fields).Info(fmt.Sprintf("Request: %s %s", method, path))
			}

			// Execute handler
			err := next(ctx)

			// Calculate duration
			duration := time.Since(start)
			statusCode := ctx.RequestCtx.Response.StatusCode()

			// Log response
			if !skip {
				fields := make(map[string]interface{})
				if config.LogRequestID {
					fields["request_id"] = ctx.RequestID()
				}
				fields["method"] = method
				fields["path"] = path
				fields["status"] = statusCode
				fields["duration_ms"] = duration.Milliseconds()
				fields["duration"] = duration.String()

				if err != nil {
					fields["error"] = err.Error()
					logger.WithFields(fields).Error(fmt.Sprintf("Request failed: %s %s - %d - %v", method, path, statusCode, err))
				} else if statusCode >= 500 {
					logger.WithFields(fields).Error(fmt.Sprintf("Request error: %s %s - %d", method, path, statusCode))
				} else if statusCode >= 400 {
					logger.WithFields(fields).Info(fmt.Sprintf("Request warning: %s %s - %d", method, path, statusCode))
				} else {
					logger.WithFields(fields).Info(fmt.Sprintf("Request completed: %s %s - %d", method, path, statusCode))
				}
			}

			return err
		}
	}
}
