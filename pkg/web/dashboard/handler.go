package dashboard

import (
	"embed"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/concurrency"
	"github.com/fluxorio/fluxor/pkg/web"
)

//go:embed dashboard.html dashboard.js dashboard.css
var dashboardFiles embed.FS

// Handler provides admin dashboard HTTP handlers
type Handler struct{}

// NewHandler creates a new admin dashboard handler
func NewHandler() *Handler {
	return &Handler{}
}

// MetricsHandler handles GET /api/dashboard/metrics
func (h *Handler) MetricsHandler(ctx *web.FastRequestContext) error {
	collector := GetMetricsCollector()
	metrics := collector.CollectAllMetrics()
	return ctx.JSON(200, metrics)
}

// HealthHandler handles GET /api/dashboard/health
func (h *Handler) HealthHandler(ctx *web.FastRequestContext) error {
	metrics := concurrency.GetDashboardMetrics()

	// Simple health check: system is healthy if there are no critical errors
	// This is a basic implementation - can be enhanced
	status := "UP"
	hasErrors := false

	for _, exec := range metrics.Executors {
		if exec.QueueUtilization >= 100.0 {
			hasErrors = true
			break
		}
	}

	for _, pool := range metrics.WorkerPools {
		if pool.QueueUtilization >= 100.0 {
			hasErrors = true
			break
		}
	}

	if hasErrors {
		status = "DEGRADED"
	}

	return ctx.JSON(200, map[string]interface{}{
		"status":    status,
		"timestamp": time.Now(),
		"metrics":   metrics,
	})
}

// DashboardHTMLHandler handles GET /dashboard - serves the HTML dashboard (fallback)
func (h *Handler) DashboardHTMLHandler(ctx *web.FastRequestContext) error {
	data, err := dashboardFiles.ReadFile("dashboard.html")
	if err != nil {
		ctx.RequestCtx.SetStatusCode(500)
		ctx.RequestCtx.SetBodyString("Internal Server Error: Failed to read dashboard.html")
		return err
	}

	ctx.RequestCtx.SetContentType("text/html; charset=utf-8")
	ctx.RequestCtx.SetBody(data)
	return nil
}

// DashboardJSHandler handles GET /dashboard.js - serves the JavaScript file
func (h *Handler) DashboardJSHandler(ctx *web.FastRequestContext) error {
	data, err := dashboardFiles.ReadFile("dashboard.js")
	if err != nil {
		ctx.RequestCtx.SetStatusCode(500)
		ctx.RequestCtx.SetBodyString("Internal Server Error: Failed to read dashboard.js")
		return err
	}

	ctx.RequestCtx.SetContentType("application/javascript; charset=utf-8")
	ctx.RequestCtx.SetBody(data)
	return nil
}

// DashboardCSSHandler handles GET /dashboard.css - serves the CSS file
func (h *Handler) DashboardCSSHandler(ctx *web.FastRequestContext) error {
	data, err := dashboardFiles.ReadFile("dashboard.css")
	if err != nil {
		ctx.RequestCtx.SetStatusCode(500)
		ctx.RequestCtx.SetBodyString("Internal Server Error: Failed to read dashboard.css")
		return err
	}

	ctx.RequestCtx.SetContentType("text/css; charset=utf-8")
	ctx.RequestCtx.SetBody(data)
	return nil
}

// Register registers admin dashboard routes with the given router
// prefix is the route prefix (e.g., "" for root, "/admin" for /admin prefix)
//
// For React dashboard:
// - API endpoints are registered at /api/dashboard/metrics and /api/dashboard/health
// - React app should be served separately (via Vite dev server or build files)
// - React app should be configured to proxy API calls to this server
//
// For simple HTML dashboard (fallback):
// - Static files are served at /dashboard, /dashboard.js, /dashboard.css
func Register(router *web.FastRouter, prefix string) {
	handler := NewHandler()

	// API endpoints - these are used by the React dashboard
	router.GETFast(prefix+"/api/dashboard/metrics", handler.MetricsHandler)
	router.GETFast(prefix+"/api/dashboard/health", handler.HealthHandler)

	// Simple HTML dashboard (fallback)
	router.GETFast(prefix+"/dashboard", handler.DashboardHTMLHandler)
	router.GETFast(prefix+"/dashboard.js", handler.DashboardJSHandler)
	router.GETFast(prefix+"/dashboard.css", handler.DashboardCSSHandler)
}
