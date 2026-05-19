// Package web provides Gin HTTP server support for Fluxor.
// GinHTTPServer implements Server using gin.Engine and integrates with Fluxor (GoCMD, EventBus).

package web

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/gin-gonic/gin"
)

// GinMiddleware wraps a Fluxor Gin handler (runs outside Gin's native chain; composable with UseGin).
type GinMiddleware func(GinRequestHandler) GinRequestHandler

// GinHTTPServer implements Server using the Gin web framework.
// Use it when you prefer Gin's router and middleware ecosystem while keeping Fluxor context (EventBus, routing).
type GinHTTPServer struct {
	*core.BaseServer
	engine   *gin.Engine
	httpSrv  *http.Server
	addr     string
	router   *ginRouter

	ready     chan struct{}
	readyOnce sync.Once
}

// GinHTTPServerConfig configures the Gin HTTP server.
type GinHTTPServerConfig struct {
	Addr         string
	Mode         string // gin mode: "debug", "release", "test" (default "release")
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DefaultGinHTTPServerConfig returns default configuration.
func DefaultGinHTTPServerConfig(addr string) *GinHTTPServerConfig {
	return &GinHTTPServerConfig{
		Addr:         addr,
		Mode:         gin.ReleaseMode,
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}

// NewGinHTTPServer creates a new Gin HTTP server with Fluxor integration.
//
//	config := web.DefaultGinHTTPServerConfig(":8080")
//	server := web.NewGinHTTPServer(gocmd, config)
//	router := server.GinRouter()
//	router.GET("/health", func(ctx *web.GinRequestContext) error { return ctx.JSON(200, gin.H{"ok": true}) })
//	_ = server.Start()
func NewGinHTTPServer(gocmd core.GoCMD, config *GinHTTPServerConfig) *GinHTTPServer {
	if config == nil {
		config = DefaultGinHTTPServerConfig(":8080")
	}
	gin.SetMode(config.Mode)
	engine := gin.New()
	router := &ginRouter{engine: engine, gocmd: gocmd}
	s := &GinHTTPServer{
		BaseServer: core.NewBaseServer("gin-server", gocmd),
		engine:     engine,
		addr:       config.Addr,
		router:     router,
		ready:      make(chan struct{}),
		httpSrv: &http.Server{
			Addr:         config.Addr,
			Handler:      engine,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
		},
	}
	s.router.parent = s
	s.BaseServer.SetHooks(s.doStart, s.doStop)
	return s
}

func (s *GinHTTPServer) doStart() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("gin listen %q: %w", s.addr, err)
	}
	s.addr = ln.Addr().String()
	s.httpSrv.Addr = s.addr

	go func() {
		if err := s.httpSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.Logger().Error(fmt.Sprintf("gin http server failed: %v", err))
		}
	}()

	s.readyOnce.Do(func() { close(s.ready) })
	return nil
}

// Ready returns a channel that is closed after the TCP listener has accepted the address (use for readiness probes).
func (s *GinHTTPServer) Ready() <-chan struct{} {
	return s.ready
}

func (s *GinHTTPServer) doStop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.httpSrv.Shutdown(ctx)
}

// Router implements web.Server: the standard RequestHandler API is not wired to Gin.
// Use GinRouter() for Fluxor handlers, or Engine() for raw Gin. Registering routes on the
// returned value is intentionally unsupported (no-op) so the Server interface stays satisfied.
func (s *GinHTTPServer) Router() Router {
	return &ginRouterAdapter{}
}

// GinRouter returns the Gin router for registering Fluxor-aware handlers.
func (s *GinHTTPServer) GinRouter() *ginRouter {
	return s.router
}

// Engine returns the underlying *gin.Engine for advanced use (middleware, no-wrapper routes).
func (s *GinHTTPServer) Engine() *gin.Engine {
	return s.engine
}

// Addr returns the bound listen address (including a resolved :0 port) after Start succeeds.
func (s *GinHTTPServer) Addr() string {
	return s.addr
}

// ginRouter wraps gin.Engine and registers handlers that receive GinRequestContext with GoCMD/EventBus.
type ginRouter struct {
	engine *gin.Engine
	gocmd  core.GoCMD
	parent *GinHTTPServer

	mu          sync.RWMutex
	middlewares []GinMiddleware
}

// UseGin appends Fluxor-level middleware (runs inside wrap, around your GinRequestHandler).
// Order: first registered runs outermost (same as typical HTTP middleware chains).
// Middleware is bound when a route is registered (GET/POST/...): call UseGin before registering routes
// if new middleware should apply to those routes.
func (r *ginRouter) UseGin(mw ...GinMiddleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middlewares = append(r.middlewares, mw...)
}

// chain composes the middleware stack once (call at route registration, not per request).
func (r *ginRouter) chain(handler GinRequestHandler) GinRequestHandler {
	r.mu.RLock()
	mws := r.middlewares
	r.mu.RUnlock()
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i](handler)
	}
	return handler
}

// GET registers a GET handler with Fluxor context.
func (r *ginRouter) GET(relativePath string, handler GinRequestHandler) {
	final := r.chain(handler)
	r.engine.GET(relativePath, r.wrap(relativePath, final))
}

// POST registers a POST handler with Fluxor context.
func (r *ginRouter) POST(relativePath string, handler GinRequestHandler) {
	final := r.chain(handler)
	r.engine.POST(relativePath, r.wrap(relativePath, final))
}

// PUT registers a PUT handler with Fluxor context.
func (r *ginRouter) PUT(relativePath string, handler GinRequestHandler) {
	final := r.chain(handler)
	r.engine.PUT(relativePath, r.wrap(relativePath, final))
}

// DELETE registers a DELETE handler with Fluxor context.
func (r *ginRouter) DELETE(relativePath string, handler GinRequestHandler) {
	final := r.chain(handler)
	r.engine.DELETE(relativePath, r.wrap(relativePath, final))
}

// PATCH registers a PATCH handler with Fluxor context.
func (r *ginRouter) PATCH(relativePath string, handler GinRequestHandler) {
	final := r.chain(handler)
	r.engine.PATCH(relativePath, r.wrap(relativePath, final))
}

// Any registers a handler for any HTTP method with Fluxor context.
func (r *ginRouter) Any(relativePath string, handler GinRequestHandler) {
	final := r.chain(handler)
	r.engine.Any(relativePath, r.wrap(relativePath, final))
}

// Group returns a gin.RouterGroup; use GinRouter().Engine().Group() and wrap handlers manually if needed.
func (r *ginRouter) Group(relativePath string) *gin.RouterGroup {
	return r.engine.Group(relativePath)
}

// wrap builds a gin.HandlerFunc that creates GinRequestContext and calls the Fluxor handler (already chained).
func (r *ginRouter) wrap(relativePath string, handler GinRequestHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = core.GenerateRequestID()
		}
		c.Header("X-Request-ID", requestID)

		params := make(map[string]string)
		for _, p := range c.Params {
			params[p.Key] = p.Value
		}

		fluxorCtx := &GinRequestContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			GinCtx:             c,
			GoCMD:              r.gocmd,
			EventBus:           r.gocmd.EventBus(),
			Params:             params,
			requestID:          requestID,
		}
		if v, ok := c.Get(fluxorScopeKey); ok {
			if sc, ok := v.(core.Scope); ok {
				fluxorCtx.Scope = sc
			}
		}
		fluxorCtx.Set("real_ip", c.ClientIP())

		if fluxorCtx.Scope != nil {
			defer fluxorCtx.Scope.Stop()
		}

		err := handler(fluxorCtx)

		if err != nil {
			status, code, msg := ginHandlerHTTPError(err)
			c.AbortWithStatusJSON(status, APIError{Code: code, Message: msg})
		}
	}
}

func ginHandlerHTTPError(err error) (status int, code, msg string) {
	status = http.StatusInternalServerError
	code = "internal_error"
	msg = err.Error()

	var st *StatusError
	if errors.As(err, &st) && st != nil {
		status = st.StatusCode()
		if st.Code != "" {
			code = st.Code
		} else {
			code = "request_error"
		}
		msg = st.Msg
		return status, code, msg
	}

	if sc, ok := httpStatusFromChain(err); ok {
		status = sc
		code = "request_error"
		msg = err.Error()
	}
	return status, code, msg
}

// httpStatusFromChain finds the first error in the chain that exposes StatusCode() (value or pointer).
func httpStatusFromChain(err error) (status int, ok bool) {
	type statusCoder interface {
		StatusCode() int
	}
	for e := err; e != nil; e = errors.Unwrap(e) {
		if sc, ok := e.(statusCoder); ok {
			st := sc.StatusCode()
			if st >= 100 && st <= 599 {
				return st, true
			}
		}
	}
	return 0, false
}

// ginRouterAdapter implements web.Router with no-ops so Server.Router() type-checks.
// Use GinHTTPServer.GinRouter() to register actual handlers.
type ginRouterAdapter struct{}

func (ginRouterAdapter) GET(path string, handler RequestHandler)    {}
func (ginRouterAdapter) POST(path string, handler RequestHandler)   {}
func (ginRouterAdapter) PUT(path string, handler RequestHandler)    {}
func (ginRouterAdapter) DELETE(path string, handler RequestHandler)  {}
func (ginRouterAdapter) PATCH(path string, handler RequestHandler)  {}
func (ginRouterAdapter) Route(method, path string, handler RequestHandler) {}
func (ginRouterAdapter) Use(middleware Middleware)                  {}
