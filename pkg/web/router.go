package web

import (
	"net/http"
	"strings"
	"sync"

	"github.com/fluxorio/fluxor/pkg/core"
)

// router implements Router
type router struct {
	routes     []*route
	middleware []Middleware
	mu         sync.RWMutex
}

type route struct {
	method  string
	path    string
	handler RequestHandler
}

// NewRouter creates a new router
func NewRouter() Router {
	return &router{
		routes:     make([]*route, 0),
		middleware: make([]Middleware, 0),
	}
}

func (r *router) GET(path string, handler RequestHandler) {
	r.Route(http.MethodGet, path, handler)
}

func (r *router) POST(path string, handler RequestHandler) {
	r.Route(http.MethodPost, path, handler)
}

func (r *router) PUT(path string, handler RequestHandler) {
	r.Route(http.MethodPut, path, handler)
}

func (r *router) DELETE(path string, handler RequestHandler) {
	r.Route(http.MethodDelete, path, handler)
}

func (r *router) PATCH(path string, handler RequestHandler) {
	r.Route(http.MethodPatch, path, handler)
}

func (r *router) Route(method, path string, handler RequestHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Apply middleware
	for i := len(r.middleware) - 1; i >= 0; i-- {
		handler = r.middleware[i](handler)
	}

	r.routes = append(r.routes, &route{
		method:  method,
		path:    path,
		handler: handler,
	})
}

func (r *router) Use(middleware Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middleware = append(r.middleware, middleware)
}

// ServeHTTP implements http.Handler
func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, route := range r.routes {
		if route.method == req.Method && r.matchPath(route.path, req.URL.Path) {
			ctx := &RequestContext{
				BaseRequestContext: core.NewBaseRequestContext(),
				Context:            req.Context(),
				Request:            req,
				Response:           w,
				Params:             r.extractParams(route.path, req.URL.Path),
			}

			if err := route.handler(ctx); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}

	http.NotFound(w, req)
}

func (r *router) matchPath(pattern, path string) bool {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(patternParts) != len(pathParts) {
		return false
	}

	for i, part := range patternParts {
		if strings.HasPrefix(part, ":") {
			continue // Parameter
		}
		if part != pathParts[i] {
			return false
		}
	}

	return true
}

func (r *router) extractParams(pattern, path string) map[string]string {
	params := make(map[string]string)
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	for i, part := range patternParts {
		if strings.HasPrefix(part, ":") {
			paramName := strings.TrimPrefix(part, ":")
			if i < len(pathParts) {
				params[paramName] = pathParts[i]
			}
		}
	}

	return params
}
