package web

import (
	"net/http"
	"strings"

	"github.com/fluxorio/fluxor/pkg/lite/fx"
)

type HandlerFunc func(c *fx.Context) error

type Middleware func(next HandlerFunc) HandlerFunc

type route struct {
	method     string
	pattern    string
	handler    HandlerFunc
	middleware []Middleware
}

type Router struct {
	routes     []*route
	middleware []Middleware
	notFound   HandlerFunc
	onError    func(c *fx.Context, err error) error
}

func NewRouter() *Router {
	r := &Router{
		routes:     make([]*route, 0),
		middleware: make([]Middleware, 0),
	}
	r.notFound = func(c *fx.Context) error { return c.Error(http.StatusNotFound, "Not Found") }
	r.onError = func(c *fx.Context, err error) error {
		// Avoid leaking internal errors by default.
		c.Log().Error("handler error", "err", err)
		return c.Error(http.StatusInternalServerError, "Internal Server Error")
	}
	return r
}

func (r *Router) Use(middleware ...Middleware) {
	r.middleware = append(r.middleware, middleware...)
}

func (r *Router) SetNotFound(handler HandlerFunc) {
	r.notFound = handler
}

func (r *Router) SetErrorHandler(handler func(c *fx.Context, err error) error) {
	r.onError = handler
}

type Group struct {
	r          *Router
	prefix     string
	middleware []Middleware
}

func (r *Router) Group(prefix string) *Group {
	return &Group{
		r:      r,
		prefix: prefix,
	}
}

func (g *Group) Use(middleware ...Middleware) {
	g.middleware = append(g.middleware, middleware...)
}

func (r *Router) GET(path string, h HandlerFunc, middleware ...Middleware) {
	r.add(http.MethodGet, path, h, middleware...)
}

func (r *Router) POST(path string, h HandlerFunc, middleware ...Middleware) {
	r.add(http.MethodPost, path, h, middleware...)
}

func (g *Group) GET(path string, h HandlerFunc, middleware ...Middleware) {
	g.r.add(http.MethodGet, g.prefix+path, h, append(g.middleware, middleware...)...)
}

func (g *Group) POST(path string, h HandlerFunc, middleware ...Middleware) {
	g.r.add(http.MethodPost, g.prefix+path, h, append(g.middleware, middleware...)...)
}

func (r *Router) add(method, pattern string, h HandlerFunc, middleware ...Middleware) {
	r.routes = append(r.routes, &route{
		method:     method,
		pattern:    pattern,
		handler:    h,
		middleware: append([]Middleware(nil), middleware...),
	})
}

func (r *Router) Handle(c *fx.Context) error {
	path := c.R.URL.Path
	method := c.R.Method

	for _, rt := range r.routes {
		if rt.method != method {
			continue
		}
		params, ok := match(rt.pattern, path)
		if !ok {
			continue
		}

		// Reset and set params on the context
		clear(c.Params)
		for k, v := range params {
			c.Params[k] = v
		}

		// Build handler: route middleware then global middleware (global outermost).
		h := rt.handler
		for i := len(rt.middleware) - 1; i >= 0; i-- {
			h = rt.middleware[i](h)
		}
		for i := len(r.middleware) - 1; i >= 0; i-- {
			h = r.middleware[i](h)
		}

		if err := h(c); err != nil {
			return r.onError(c, err)
		}
		return nil
	}

	if err := r.notFound(c); err != nil {
		return r.onError(c, err)
	}
	return nil
}

func match(pattern, path string) (map[string]string, bool) {
	// Fast path
	if pattern == path {
		return map[string]string{}, true
	}

	trim := func(s string) string { return strings.Trim(s, "/") }
	pp := strings.Split(trim(pattern), "/")
	ap := strings.Split(trim(path), "/")

	if len(pp) != len(ap) {
		return nil, false
	}

	params := make(map[string]string)
	for i := range pp {
		if strings.HasPrefix(pp[i], ":") {
			k := strings.TrimPrefix(pp[i], ":")
			if k == "" {
				return nil, false
			}
			params[k] = ap[i]
			continue
		}
		if pp[i] != ap[i] {
			return nil, false
		}
	}
	return params, true
}
