package diagnostic

import (
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
)

// Register registers diagnostic routes with the given router
// prefix is the route prefix (e.g., "" for root, "/admin" for /admin prefix)
//
// Routes registered:
// - GET {prefix}/api/diagnostic/deployment/:id - Get specific deployment diagnostic
// - GET {prefix}/api/diagnostic/system - Get system-wide diagnostic
// - GET {prefix}/api/diagnostic/deployments - Get all deployment diagnostics
func Register(router *web.FastRouter, gocmd core.GoCMD, prefix string) {
	handler := NewHandler(gocmd)

	// API endpoints
	router.GETFast(prefix+"/api/diagnostic/deployment/:id", handler.DeploymentHandler)
	router.GETFast(prefix+"/api/diagnostic/system", handler.SystemHandler)
	router.GETFast(prefix+"/api/diagnostic/deployments", handler.AllDeploymentsHandler)
}
