package web

import (
	"strings"
	"sync"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/valyala/fasthttp"
)

// FastRouter implements Router for fasthttp
type FastRouter struct {
	routes         []*fastRoute
	middleware     []FastMiddleware
	defaultHandler FastRequestHandler // Default handler for unmatched routes (like nginx default_server)
	mu             sync.RWMutex
}

type fastRoute struct {
	method  string
	path    string
	handler FastRequestHandler
	// middleware is applied only for this route (in addition to any global middleware).
	middleware []FastMiddleware
}

// FastRequestHandler handles fasthttp requests
type FastRequestHandler func(ctx *FastRequestContext) error

// FastMiddleware is middleware for fasthttp
type FastMiddleware func(handler FastRequestHandler) FastRequestHandler

// NewFastRouter creates a new fasthttp router
func NewFastRouter() *FastRouter {
	return &FastRouter{
		routes:     make([]*fastRoute, 0),
		middleware: make([]FastMiddleware, 0),
	}
}

// ServeFastHTTP implements fasthttp request handler
func (r *FastRouter) ServeFastHTTP(ctx *FastRequestContext) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	method := string(ctx.Method())
	path := string(ctx.Path())

	for _, route := range r.routes {
		matched := route.method == method && r.matchPath(route.path, path)
		if matched {
			// Extract params
			r.extractParams(route.path, path, ctx.Params)

			// Apply middleware chain (route-specific then global).
			// We apply route middleware first so global middleware remains outermost.
			handler := route.handler
			for i := len(route.middleware) - 1; i >= 0; i-- {
				handler = route.middleware[i](handler)
			}
			for i := len(r.middleware) - 1; i >= 0; i-- {
				handler = r.middleware[i](handler)
			}

			// Execute handler
			if err := handler(ctx); err != nil {
				ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
			}
			return
		}
	}

	// Not found - use default handler if set, otherwise return 404
	if r.defaultHandler != nil {
		// Apply global middleware to default handler
		handler := r.defaultHandler
		for i := len(r.middleware) - 1; i >= 0; i-- {
			handler = r.middleware[i](handler)
		}
		if err := handler(ctx); err != nil {
			ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		}
		return
	}

	// No default handler set, return 404
	ctx.Error("Not Found", fasthttp.StatusNotFound)
}

func (r *FastRouter) GETFast(path string, handler FastRequestHandler) {
	r.RouteFast("GET", path, handler)
}

func (r *FastRouter) POSTFast(path string, handler FastRequestHandler) {
	r.RouteFast("POST", path, handler)
}

func (r *FastRouter) PUTFast(path string, handler FastRequestHandler) {
	r.RouteFast("PUT", path, handler)
}

func (r *FastRouter) DELETEFast(path string, handler FastRequestHandler) {
	r.RouteFast("DELETE", path, handler)
}

func (r *FastRouter) PATCHFast(path string, handler FastRequestHandler) {
	r.RouteFast("PATCH", path, handler)
}

// GETFastWith registers a GET route with per-route middleware.
func (r *FastRouter) GETFastWith(path string, handler FastRequestHandler, middleware ...FastMiddleware) {
	r.RouteFastWith("GET", path, handler, middleware...)
}

// POSTFastWith registers a POST route with per-route middleware.
func (r *FastRouter) POSTFastWith(path string, handler FastRequestHandler, middleware ...FastMiddleware) {
	r.RouteFastWith("POST", path, handler, middleware...)
}

// PUTFastWith registers a PUT route with per-route middleware.
func (r *FastRouter) PUTFastWith(path string, handler FastRequestHandler, middleware ...FastMiddleware) {
	r.RouteFastWith("PUT", path, handler, middleware...)
}

// DELETEFastWith registers a DELETE route with per-route middleware.
func (r *FastRouter) DELETEFastWith(path string, handler FastRequestHandler, middleware ...FastMiddleware) {
	r.RouteFastWith("DELETE", path, handler, middleware...)
}

// PATCHFastWith registers a PATCH route with per-route middleware.
func (r *FastRouter) PATCHFastWith(path string, handler FastRequestHandler, middleware ...FastMiddleware) {
	r.RouteFastWith("PATCH", path, handler, middleware...)
}

// Implement Router interface for compatibility (not used with fasthttp)
func (r *FastRouter) GET(path string, handler RequestHandler) {
	// Not implemented for standard http - use GETFast instead
}

func (r *FastRouter) POST(path string, handler RequestHandler) {
	// Not implemented for standard http - use POSTFast instead
}

func (r *FastRouter) PUT(path string, handler RequestHandler) {
	// Not implemented for standard http
}

func (r *FastRouter) DELETE(path string, handler RequestHandler) {
	// Not implemented for standard http
}

func (r *FastRouter) PATCH(path string, handler RequestHandler) {
	// Not implemented for standard http
}

// RouteFast registers a fast handler
func (r *FastRouter) RouteFast(method, path string, handler FastRequestHandler) {
	r.RouteFastWith(method, path, handler)
}

// RouteFastWith registers a fast handler with per-route middleware.
func (r *FastRouter) RouteFastWith(method, path string, handler FastRequestHandler, middleware ...FastMiddleware) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.routes = append(r.routes, &fastRoute{
		method:     method,
		path:       path,
		handler:    handler,
		middleware: append([]FastMiddleware(nil), middleware...),
	})
}

func (r *FastRouter) Route(method, path string, handler RequestHandler) {
	// Convert to FastRequestHandler
	r.RouteFast(method, path, func(ctx *FastRequestContext) error {
		reqCtx := &RequestContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			Request:            nil,
			Response:           nil,
			GoCMD:              ctx.GoCMD,
			EventBus:           ctx.EventBus,
			Params:             ctx.Params,
		}
		return handler(reqCtx)
	})
}

func (r *FastRouter) Use(middleware Middleware) {
	// Convert middleware to FastMiddleware
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middleware = append(r.middleware, func(next FastRequestHandler) FastRequestHandler {
		return func(ctx *FastRequestContext) error {
			reqCtx := &RequestContext{
				BaseRequestContext: core.NewBaseRequestContext(),
				Request:            nil,
				Response:           nil,
				GoCMD:              ctx.GoCMD,
				EventBus:           ctx.EventBus,
				Params:             ctx.Params,
			}
			wrapped := middleware(func(reqCtx *RequestContext) error {
				return next(ctx)
			})
			return wrapped(reqCtx)
		}
	})
}

// UseFast registers global fasthttp middleware.
func (r *FastRouter) UseFast(middleware ...FastMiddleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middleware = append(r.middleware, middleware...)
}

// SetDefaultHandler sets a default handler for unmatched routes (like nginx default_server)
// This handler will be called when no route matches the request
func (r *FastRouter) SetDefaultHandler(handler FastRequestHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultHandler = handler
}

func (r *FastRouter) matchPath(pattern, path string) bool {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	for i, part := range patternParts {
		if strings.HasPrefix(part, "*") {
			return true // wildcard matches rest of path (including zero segments)
		}
		if strings.HasPrefix(part, ":") {
			if i >= len(pathParts) {
				return false
			}
			continue
		}
		if i >= len(pathParts) || part != pathParts[i] {
			return false
		}
	}

	return len(patternParts) == len(pathParts)
}

func (r *FastRouter) extractParams(pattern, path string, params map[string]string) {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	for i, part := range patternParts {
		if strings.HasPrefix(part, "*") {
			paramName := strings.TrimPrefix(part, "*")
			if i < len(pathParts) {
				params[paramName] = strings.Join(pathParts[i:], "/")
			}
			return
		}
		if strings.HasPrefix(part, ":") {
			paramName := strings.TrimPrefix(part, ":")
			if i < len(pathParts) {
				params[paramName] = pathParts[i]
			}
		}
	}
}
