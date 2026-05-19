package web

import (
	"github.com/fluxorio/fluxor/pkg/core"
)

// FastHTTPVerticle is a verticle that runs a FastHTTPServer
// It wraps FastHTTPServer and implements Verticle interface
// This allows the server to be deployed as a verticle using DeployVerticle
type FastHTTPVerticle struct {
	*core.BaseVerticle // Embed base verticle for lifecycle management
	server             *FastHTTPServer
	config             *FastHTTPServerConfig
}

// NewFastHTTPVerticle creates a new FastHTTPVerticle
// The server will be created and started when the verticle is deployed
func NewFastHTTPVerticle(config *FastHTTPServerConfig) *FastHTTPVerticle {
	return &FastHTTPVerticle{
		BaseVerticle: core.NewBaseVerticle("fasthttp-verticle"),
		config:       config,
	}
}

// doStart is called by BaseVerticle.Start() - implements hook method
func (v *FastHTTPVerticle) doStart(ctx core.FluxorContext) error {
	gocmd := ctx.GoCMD()

	// Create FastHTTPServer using gocmd from context
	if v.config == nil {
		v.config = DefaultFastHTTPServerConfig(":8080")
	}
	v.server = NewFastHTTPServer(gocmd, v.config)

	// Server is created but not started yet
	// Start will be called in AsyncStart
	return nil
}

// doStop is called by BaseVerticle.Stop() - implements hook method
func (v *FastHTTPVerticle) doStop(ctx core.FluxorContext) error {
	// Synchronous stop - actual async stop happens in AsyncStop
	return nil
}

// AsyncStart implements AsyncVerticle.AsyncStart
// Starts the HTTP server as part of verticle lifecycle
func (v *FastHTTPVerticle) AsyncStart(ctx core.FluxorContext, resultHandler func(error)) {
	// FastHTTPVerticle process: start HTTP server as part of verticle lifecycle
	// Server is managed by gocmd from context, tied to context's lifecycle
	logger := core.NewDefaultLogger()
	logger.Info("FastHTTPVerticle AsyncStart: starting HTTP server process on:", v.config.Addr)

	serverLogger := v.server.Logger()
	serverLogger.Info("Starting HTTP server process from verticle context on:", v.config.Addr)

	// server.Start() is blocking (calls ListenAndServe)
	// Framework calls AsyncStart in goroutine, so server runs as verticle process
	// Server lifecycle is tied to verticle context - stops when context is cancelled

	// Notify framework that server startup is initiated
	resultHandler(nil)

	logger.Info("FastHTTPVerticle AsyncStart: server startup initiated, starting server...")

	// Start server - blocks forever in framework's goroutine
	// Server runs as part of verticle process until context is cancelled
	if err := v.server.Start(); err != nil {
		serverLogger.Error("HTTP server process failed:", err)
		logger.Error("FastHTTPVerticle AsyncStart: server failed:", err)
	}
}

// AsyncStop implements AsyncVerticle.AsyncStop
// Stops the HTTP server as part of verticle lifecycle
func (v *FastHTTPVerticle) AsyncStop(ctx core.FluxorContext, resultHandler func(error)) {
	// FastHTTPVerticle process: stop HTTP server as part of verticle lifecycle
	logger := core.NewDefaultLogger()
	logger.Info("FastHTTPVerticle AsyncStop: stopping HTTP server process")

	if v.server != nil {
		if err := v.server.Stop(); err != nil {
			logger.Error("FastHTTPVerticle AsyncStop: server stop error:", err)
			resultHandler(err)
			return
		}
		logger.Info("FastHTTPVerticle AsyncStop: HTTP server stopped successfully")
	}

	resultHandler(nil)
}

// Server returns the underlying FastHTTPServer
// This allows access to router and other server functionality
func (v *FastHTTPVerticle) Server() *FastHTTPServer {
	return v.server
}

// FastRouter returns the fast router for route registration
// This is a convenience method to access the server's router
func (v *FastHTTPVerticle) FastRouter() *FastRouter {
	if v.server == nil {
		return nil
	}
	return v.server.FastRouter()
}
