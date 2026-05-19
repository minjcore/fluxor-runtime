// Package sugar provides Express.js-like one-liner API for Fluxor
// Usage: sugar.Run("myapp", ":8080", func(r *sugar.Router) { ... })
package sugar

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/fx"
	"github.com/fluxorio/fluxor/pkg/web"
)

// JSON is alias for map - Express.js style
type JSON = map[string]interface{}

// ============================================================================
// Run - Express.js style one-liner app
// ============================================================================

// Config - minimal config
type Config struct {
	Name    string // Service name
	Addr    string // e.g., ":8080"
	LogFile string // e.g., "app.log" (empty = no file)
}

// Run - one-liner to create Express.js style app
// Usage: sugar.Run("myapp", ":8080", func(r *sugar.Router) { ... })
func Run(name, addr string, setup func(r *Router)) {
	RunWithConfig(Config{
		Name:    name,
		Addr:    addr,
		LogFile: name + ".log",
	}, setup)
}

// RunWithConfig - with custom config
func RunWithConfig(cfg Config, setup func(r *Router)) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup logger (console + optional file)
	consoleLog := core.NewHybridLogger(cfg.Name, false)
	var log *core.MultiLogger

	if cfg.LogFile != "" {
		fileLog, _ := core.NewFileAppendLogger(core.FileLoggerConfig{
			Service:  cfg.Name,
			FilePath: cfg.LogFile,
		})
		defer fileLog.Close()
		log = core.NewMultiLogger(consoleLog, fileLog)
	} else {
		log = core.NewMultiLogger(consoleLog)
	}

	// Create router setup invoker
	routerSetup := func(deps map[reflect.Type]interface{}) error {
		gocmd := deps[reflect.TypeOf((*core.GoCMD)(nil)).Elem()].(core.GoCMD)

		// Create router
		r := NewRouter(gocmd, cfg.Addr, log)

		// Call user's setup function
		setup(r)

		// Start server in background
		go r.Start()
		return nil
	}

	// Create and start app
	app, err := fx.New(ctx, fx.Invoke(fx.NewInvoker(routerSetup)))
	if err != nil {
		log.Error(fmt.Sprintf("Failed to create app: %v", err))
		os.Exit(1)
	}

	if err := app.Start(); err != nil {
		log.Error(fmt.Sprintf("Failed to start: %v", err))
		os.Exit(1)
	}

	log.Info(fmt.Sprintf("🍬 %s running on %s", cfg.Name, cfg.Addr))
	log.Info("   Press Ctrl+C to stop")

	// Wait for shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info("👋 Shutting down...")
	app.Stop()
}

// ============================================================================
// Router - Express.js style router wrapper
// ============================================================================

// Router wraps FastRouter with Express.js style API
type Router struct {
	server *web.FastHTTPServer
	router *web.FastRouter
	log    *core.MultiLogger
}

// NewRouter creates Express-style router
func NewRouter(gocmd core.GoCMD, addr string, log *core.MultiLogger) *Router {
	cfg := web.DefaultFastHTTPServerConfig(addr)
	server := web.NewFastHTTPServer(gocmd, cfg)
	return &Router{
		server: server,
		router: server.FastRouter(),
		log:    log,
	}
}

// Handler type - Express.js style
type Handler func(c *web.FastRequestContext) error

// Start the server
func (r *Router) Start() error {
	return r.server.Start()
}

// morgan middleware - auto logging like Express morgan
func (r *Router) morgan(h Handler) func(*web.FastRequestContext) error {
	return func(c *web.FastRequestContext) error {
		start := time.Now()
		err := h(c)
		r.log.Info(fmt.Sprintf("%s %s %d %v",
			string(c.RequestCtx.Method()),
			string(c.RequestCtx.Path()),
			c.RequestCtx.Response.StatusCode(),
			time.Since(start).Round(time.Microsecond),
		))
		return err
	}
}

// GET route with auto logging
func (r *Router) GET(path string, h Handler) {
	r.router.GETFast(path, r.morgan(h))
}

// POST route with auto logging
func (r *Router) POST(path string, h Handler) {
	r.router.POSTFast(path, r.morgan(h))
}

// PUT route with auto logging
func (r *Router) PUT(path string, h Handler) {
	r.router.PUTFast(path, r.morgan(h))
}

// DELETE route with auto logging
func (r *Router) DELETE(path string, h Handler) {
	r.router.DELETEFast(path, r.morgan(h))
}

// PATCH route with auto logging
func (r *Router) PATCH(path string, h Handler) {
	r.router.PATCHFast(path, r.morgan(h))
}

// Log returns the logger for custom logging
func (r *Router) Log() *core.MultiLogger {
	return r.log
}
