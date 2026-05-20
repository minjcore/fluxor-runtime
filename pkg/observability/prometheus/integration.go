package prometheus

import (
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/web"
)

var lastRejectedRequests int64

// FastHTTPMetricsMiddleware creates middleware that records HTTP metrics
func FastHTTPMetricsMiddleware() web.FastMiddleware {
	metrics := GetMetrics()
	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			start := time.Now()
			method := string(ctx.Method())
			path := string(ctx.Path())

			// Get request size
			requestSize := int64(len(ctx.RequestCtx.PostBody()))

			// Execute handler
			err := next(ctx)

			// Calculate duration
			duration := time.Since(start)

			// Get response status and size
			status := ctx.RequestCtx.Response.StatusCode()
			statusStr := statusCodeString(status)
			responseSize := int64(ctx.RequestCtx.Response.Header.ContentLength())
			if responseSize < 0 {
				responseSize = 0
			}

			// Record metrics
			metrics.RecordHTTPRequest(method, path, statusStr, duration, requestSize, responseSize)

			return err
		}
	}
}

// UpdateServerMetrics updates server metrics from FastHTTPServer
func UpdateServerMetrics(server *web.FastHTTPServer) {
	metrics := GetMetrics()
	serverMetrics := server.Metrics()

	current := serverMetrics.RejectedRequests
	prev := atomic.SwapInt64(&lastRejectedRequests, current)
	delta := current - prev
	if delta < 0 {
		delta = 0
	}

	metrics.UpdateServerMetrics(
		serverMetrics.QueuedRequests,
		delta,
		serverMetrics.CurrentCCU,
		serverMetrics.NormalCCU,
		serverMetrics.CCUUtilization,
		server.GoCMD().DeploymentCount(),
	)
}

// statusCodeString converts status code to string
func statusCodeString(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}
