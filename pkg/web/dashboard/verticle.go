package dashboard

import (
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/fluxor/pkg/web/admin"
)

// DashboardVerticleConfig configures the DashboardVerticle
type DashboardVerticleConfig struct {
	// Address is the HTTP server address (e.g., ":8080")
	// If empty, will use "http_addr" from context config or default to ":8080"
	Address string

	// Prefix is the route prefix for dashboard routes (e.g., "" for root, "/admin" for /admin prefix)
	// If empty, routes are registered at root
	Prefix string
}

// DefaultDashboardVerticleConfig returns default dashboard verticle configuration
func DefaultDashboardVerticleConfig() DashboardVerticleConfig {
	return DashboardVerticleConfig{
		Address: ":8080",
		Prefix:  "",
	}
}

// DashboardVerticle is a reusable verticle that provides dashboard functionality
// It can be deployed into any application to add dashboard/metrics endpoints
type DashboardVerticle struct {
	*core.BaseVerticle
	server *web.FastHTTPServer
	config DashboardVerticleConfig
}

// NewDashboardVerticle creates a new DashboardVerticle with default configuration
func NewDashboardVerticle() *DashboardVerticle {
	return NewDashboardVerticleWithConfig(DefaultDashboardVerticleConfig())
}

// NewDashboardVerticleWithConfig creates a new DashboardVerticle with custom configuration
func NewDashboardVerticleWithConfig(config DashboardVerticleConfig) *DashboardVerticle {
	return &DashboardVerticle{
		BaseVerticle: core.NewBaseVerticle("dashboard"),
		config:       config,
	}
}

// Start initializes the HTTP server and registers dashboard routes
func (v *DashboardVerticle) Start(ctx core.FluxorContext) error {
	// Start base verticle first
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}

	logger := core.NewDefaultLogger()
	logger.Info("DashboardVerticle starting...")

	// Get HTTP address from config or context
	addr := v.config.Address
	if addr == "" {
		if val, ok := ctx.Config()["http_addr"].(string); ok && val != "" {
			addr = val
		} else {
			addr = ":8080" // Default
		}
	}

	// Create FastHTTPServer
	serverConfig := web.DefaultFastHTTPServerConfig(addr)
	v.server = web.NewFastHTTPServer(ctx.GoCMD(), serverConfig)

	// Get router
	router := v.server.FastRouter()

	// Register dashboard routes
	Register(router, v.config.Prefix)
	// Register admin UI shell (P1: layout, sidebar, placeholder pages)
	admin.Register(router, v.config.Prefix)

	// Start server in goroutine
	go func() {
		logger.Info("Dashboard HTTP server starting on " + addr)
		if err := v.server.Start(); err != nil {
			logger.Error("Dashboard server error: " + err.Error())
		}
	}()

	logger.Info("DashboardVerticle started successfully")
	return nil
}

// Stop stops the HTTP server
func (v *DashboardVerticle) Stop(ctx core.FluxorContext) error {
	logger := core.NewDefaultLogger()
	logger.Info("DashboardVerticle stopping...")

	if v.server != nil {
		if err := v.server.Stop(); err != nil {
			logger.Error("Error stopping dashboard server: " + err.Error())
			return err
		}
	}

	return v.BaseVerticle.Stop(ctx)
}
