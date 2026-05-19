package udp

import (
	"github.com/fluxorio/fluxor/pkg/core"
)

// UDPVerticle is a verticle that runs a UDPServer
// It wraps UDPServer and implements Verticle interface
// This allows the server to be deployed as a verticle using DeployVerticle
// Mirrors pkg/tcp.TCPVerticle and pkg/web.FastHTTPVerticle for consistency
type UDPVerticle struct {
	*core.BaseVerticle // Embed base verticle for lifecycle management
	server             *UDPServer
	config             *UDPServerConfig
}

// NewUDPVerticle creates a new UDPVerticle
// The server will be created and started when the verticle is deployed
func NewUDPVerticle(config *UDPServerConfig) *UDPVerticle {
	return &UDPVerticle{
		BaseVerticle: core.NewBaseVerticle("udp-verticle"),
		config:       config,
	}
}

// doStart is called by BaseVerticle.Start() - implements hook method
func (v *UDPVerticle) doStart(ctx core.FluxorContext) error {
	gocmd := ctx.GoCMD()

	// Create UDPServer using gocmd from context
	if v.config == nil {
		v.config = DefaultUDPServerConfig(":9001")
	}
	v.server = NewUDPServer(gocmd, v.config)

	// Server is created but not started yet
	// Start will be called in AsyncStart
	return nil
}

// doStop is called by BaseVerticle.Stop() - implements hook method
func (v *UDPVerticle) doStop(ctx core.FluxorContext) error {
	// Synchronous stop - actual async stop happens in AsyncStop
	return nil
}

// AsyncStart implements AsyncVerticle.AsyncStart
// Starts the UDP server as part of verticle lifecycle
func (v *UDPVerticle) AsyncStart(ctx core.FluxorContext, resultHandler func(error)) {
	// UDPVerticle process: start UDP server as part of verticle lifecycle
	// Server is managed by gocmd from context, tied to context's lifecycle
	logger := core.NewDefaultLogger()
	logger.Info("UDPVerticle AsyncStart: starting UDP server process on:", v.config.Addr)

	serverLogger := v.server.Logger()
	serverLogger.Info("Starting UDP server process from verticle context on:", v.config.Addr)

	// server.Start() is blocking (calls ReadFrom loop)
	// Framework calls AsyncStart in goroutine, so server runs as verticle process
	// Server lifecycle is tied to verticle context - stops when context is cancelled

	// Notify framework that server startup is initiated
	resultHandler(nil)

	logger.Info("UDPVerticle AsyncStart: server startup initiated, starting server...")

	// Start server - blocks forever in framework's goroutine
	// Server runs as part of verticle process until context is cancelled
	if err := v.server.Start(); err != nil {
		serverLogger.Error("UDP server process failed:", err)
		logger.Error("UDPVerticle AsyncStart: server failed:", err)
	}
}

// AsyncStop implements AsyncVerticle.AsyncStop
// Stops the UDP server as part of verticle lifecycle
func (v *UDPVerticle) AsyncStop(ctx core.FluxorContext, resultHandler func(error)) {
	// UDPVerticle process: stop UDP server as part of verticle lifecycle
	logger := core.NewDefaultLogger()
	logger.Info("UDPVerticle AsyncStop: stopping UDP server process")

	if v.server != nil {
		if err := v.server.Stop(); err != nil {
			logger.Error("UDPVerticle AsyncStop: server stop error:", err)
			resultHandler(err)
			return
		}
		logger.Info("UDPVerticle AsyncStop: UDP server stopped successfully")
	}

	resultHandler(nil)
}

// Server returns the underlying UDPServer
// This allows access to handler and other server functionality
func (v *UDPVerticle) Server() *UDPServer {
	return v.server
}

// SetHandler sets the packet handler on the underlying server
// This is a convenience method to configure the server
func (v *UDPVerticle) SetHandler(handler PacketHandler) {
	if v.server != nil {
		v.server.SetHandler(handler)
	}
}

// Use adds middleware to the underlying server
// This is a convenience method to configure the server
func (v *UDPVerticle) Use(mw ...Middleware) {
	if v.server != nil {
		v.server.Use(mw...)
	}
}
