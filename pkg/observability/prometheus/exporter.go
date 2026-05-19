package prometheus

import (
	"net/http"

	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

// RegisterMetricsEndpoint registers the /metrics endpoint on the FastRouter
// This allows Prometheus to scrape metrics from the application
// router should be obtained via server.FastRouter()
func RegisterMetricsEndpoint(router interface {
	GETFast(path string, handler web.FastRequestHandler)
}, path string) {
	metricsHandler := promhttp.HandlerFor(DefaultRegistry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})

	// Convert standard http.Handler to FastRequestHandler
	router.GETFast(path, func(ctx *web.FastRequestContext) error {
		// Adapt standard http.Handler to fasthttp
		adaptor := fasthttpadaptor.NewFastHTTPHandler(metricsHandler)
		adaptor(ctx.RequestCtx)
		return nil
	})
}

// FastHTTPHandler returns a FastRequestHandler for the metrics endpoint
func FastHTTPHandler() web.FastRequestHandler {
	metricsHandler := promhttp.HandlerFor(DefaultRegistry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
	adaptor := fasthttpadaptor.NewFastHTTPHandler(metricsHandler)

	return func(ctx *web.FastRequestContext) error {
		adaptor(ctx.RequestCtx)
		return nil
	}
}

// Handler returns an HTTP handler for the metrics endpoint (for standard http)
func Handler() http.Handler {
	return promhttp.HandlerFor(DefaultRegistry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// HandlerFor returns an HTTP handler for a custom registry
func HandlerFor(registry *prometheus.Registry) http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}
