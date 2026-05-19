package tcp

import (
	"github.com/fluxorio/fluxor/pkg/core"
)

// TCPVerticle is a verticle that runs a TCPServer
// It wraps TCPServer and implements Verticle interface
// This allows the server to be deployed as a verticle using DeployVerticle
// Mirrors pkg/web.FastHTTPVerticle for consistency
type TCPVerticle struct {
	*core.BaseVerticle // Embed base verticle for lifecycle management
	server             *TCPServer
	config             *TCPServerConfig
}

// NewTCPVerticle creates a new TCPVerticle
// The server will be created and started when the verticle is deployed
func NewTCPVerticle(config *TCPServerConfig) *TCPVerticle {
	return &TCPVerticle{
		BaseVerticle: core.NewBaseVerticle("tcp-verticle"),
		config:       config,
	}
}

// doStart is called by BaseVerticle.Start() - implements hook method
func (v *TCPVerticle) doStart(ctx core.FluxorContext) error {
	gocmd := ctx.GoCMD()

	// Create TCPServer using gocmd from context
	if v.config == nil {
		v.config = DefaultTCPServerConfig(":9000")
	}
	v.server = NewTCPServer(gocmd, v.config)

	// Server is created but not started yet
	// Start will be called in AsyncStart
	return nil
}

// doStop is called by BaseVerticle.Stop() - implements hook method
func (v *TCPVerticle) doStop(ctx core.FluxorContext) error {
	// Synchronous stop - actual async stop happens in AsyncStop
	return nil
}

// AsyncStart implements AsyncVerticle.AsyncStart
// Starts the TCP server as part of verticle lifecycle
func (v *TCPVerticle) AsyncStart(ctx core.FluxorContext, resultHandler func(error)) {
	// TCPVerticle process: start TCP server as part of verticle lifecycle
	// Server is managed by gocmd from context, tied to context's lifecycle
	logger := core.NewDefaultLogger()
	logger.Info("TCPVerticle AsyncStart: starting TCP server process on:", v.config.Addr)

	serverLogger := v.server.Logger()
	serverLogger.Info("Starting TCP server process from verticle context on:", v.config.Addr)

	// server.Start() is blocking (calls Accept loop)
	// Framework calls AsyncStart in goroutine, so server runs as verticle process
	// Server lifecycle is tied to verticle context - stops when context is cancelled

	// Notify framework that server startup is initiated
	resultHandler(nil)

	logger.Info("TCPVerticle AsyncStart: server startup initiated, starting server...")

	// Start server - blocks forever in framework's goroutine
	// Server runs as part of verticle process until context is cancelled
	if err := v.server.Start(); err != nil {
		serverLogger.Error("TCP server process failed:", err)
		logger.Error("TCPVerticle AsyncStart: server failed:", err)
	}
}

// AsyncStop implements AsyncVerticle.AsyncStop
// Stops the TCP server as part of verticle lifecycle
func (v *TCPVerticle) AsyncStop(ctx core.FluxorContext, resultHandler func(error)) {
	// TCPVerticle process: stop TCP server as part of verticle lifecycle
	logger := core.NewDefaultLogger()
	logger.Info("TCPVerticle AsyncStop: stopping TCP server process")

	if v.server != nil {
		if err := v.server.Stop(); err != nil {
			logger.Error("TCPVerticle AsyncStop: server stop error:", err)
			resultHandler(err)
			return
		}
		logger.Info("TCPVerticle AsyncStop: TCP server stopped successfully")
	}

	resultHandler(nil)
}

// Server returns the underlying TCPServer
// This allows access to handler and other server functionality
func (v *TCPVerticle) Server() *TCPServer {
	return v.server
}

// SetHandler sets the connection handler on the underlying server
// This is a convenience method to configure the server
func (v *TCPVerticle) SetHandler(handler ConnectionHandler) {
	if v.server != nil {
		v.server.SetHandler(handler)
	}
}

// Use adds middleware to the underlying server
// This is a convenience method to configure the server
func (v *TCPVerticle) Use(mw ...Middleware) {
	if v.server != nil {
		v.server.Use(mw...)
	}
}
