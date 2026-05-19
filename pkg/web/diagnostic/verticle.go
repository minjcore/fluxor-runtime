package diagnostic

import (
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
)

// DiagnosticVerticleConfig configures the DiagnosticVerticle
type DiagnosticVerticleConfig struct {
	// Address is the HTTP server address (e.g., ":8080")
	// If empty, will use "http_addr" from context config or default to ":8080"
	Address string

	// Prefix is the route prefix for diagnostic routes (e.g., "" for root, "/admin" for /admin prefix)
	// If empty, routes are registered at root
	Prefix string
}

// DefaultDiagnosticVerticleConfig returns default diagnostic verticle configuration
func DefaultDiagnosticVerticleConfig() DiagnosticVerticleConfig {
	return DiagnosticVerticleConfig{
		Address: ":8080",
		Prefix:  "",
	}
}

// DiagnosticVerticle is a reusable verticle that provides diagnostic functionality
// It can be deployed into any application to add diagnostic endpoints
type DiagnosticVerticle struct {
	*core.BaseVerticle
	server *web.FastHTTPServer
	config DiagnosticVerticleConfig
}

// NewDiagnosticVerticle creates a new DiagnosticVerticle with default configuration
func NewDiagnosticVerticle() *DiagnosticVerticle {
	return NewDiagnosticVerticleWithConfig(DefaultDiagnosticVerticleConfig())
}

// NewDiagnosticVerticleWithConfig creates a new DiagnosticVerticle with custom configuration
func NewDiagnosticVerticleWithConfig(config DiagnosticVerticleConfig) *DiagnosticVerticle {
	return &DiagnosticVerticle{
		BaseVerticle: core.NewBaseVerticle("diagnostic"),
		config:       config,
	}
}

// Start initializes the HTTP server and registers diagnostic routes
func (v *DiagnosticVerticle) Start(ctx core.FluxorContext) error {
	// Get server address from config or context
	addr := v.config.Address
	if addr == "" {
		if addrVal, ok := ctx.Config()["http_addr"].(string); ok && addrVal != "" {
			addr = addrVal
		} else {
			addr = ":8080"
		}
	}

	// Create FastHTTP server
	serverCfg := web.DefaultFastHTTPServerConfig(addr)
	server := web.NewFastHTTPServer(ctx.GoCMD(), serverCfg)
	v.server = server

	// Get router and register diagnostic routes
	router := server.FastRouter()
	Register(router, ctx.GoCMD(), v.config.Prefix)

	// Start server in goroutine (non-blocking)
	go func() {
		if err := server.Start(); err != nil {
			logger := core.NewDefaultLogger()
			logger.Error("Diagnostic HTTP server failed to start: " + err.Error())
		}
	}()

	return nil
}

// Stop stops the HTTP server
func (v *DiagnosticVerticle) Stop(ctx core.FluxorContext) error {
	if v.server != nil {
		return v.server.Stop()
	}
	return nil
}
