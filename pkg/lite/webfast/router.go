package webfast

import (
	"bytes"
	"sync"
	"unsafe"

	"github.com/fluxorio/fluxor/pkg/lite/core"
	"github.com/fluxorio/fluxor/pkg/lite/fx"
	"github.com/valyala/fasthttp"
)

type HandlerFunc func(c *fx.FastContext) error
type Middleware func(next HandlerFunc) HandlerFunc

const (
	methodGET uint8 = iota + 1
	methodPOST
)

type route struct {
	method     uint8
	pattern    string
	staticPath []byte // only for routes without params
	segments   []segment
	handler    HandlerFunc
	middleware []Middleware
	hasParams  bool
}

type segment struct {
	static []byte
	param  string // if non-empty, this segment captures into param
}

type Router struct {
	getRoutes  []*route
	postRoutes []*route
	middleware []Middleware
	notFound   HandlerFunc
	onError    func(c *fx.FastContext, err error) error

	coreCtx *core.FluxorContext

	paramPool sync.Pool // stores []fx.Param
}

func NewRouter() *Router {
	r := &Router{
		getRoutes:  make([]*route, 0, 32),
		postRoutes: make([]*route, 0, 32),
		middleware: make([]Middleware, 0),
	}
	r.notFound = func(c *fx.FastContext) error {
		return c.Error(fasthttp.StatusNotFound, "Not Found")
	}
	r.onError = func(c *fx.FastContext, err error) error {
		c.Log().Error("handler error", "err", err)
		return c.Error(fasthttp.StatusInternalServerError, "Internal Server Error")
	}
	return r
}

// Bind attaches the runtime context (required before serving requests).
func (r *Router) Bind(coreCtx *core.FluxorContext) {
	r.coreCtx = coreCtx
}

func (r *Router) Use(mw ...Middleware) { r.middleware = append(r.middleware, mw...) }

func (r *Router) GET(path string, h HandlerFunc, mw ...Middleware) {
	r.add(methodGET, path, h, mw...)
}
func (r *Router) POST(path string, h HandlerFunc, mw ...Middleware) {
	r.add(methodPOST, path, h, mw...)
}

func (r *Router) add(method uint8, pattern string, h HandlerFunc, mw ...Middleware) {
	segs, hasParams := compilePattern(pattern)
	rt := &route{
		method:     method,
		pattern:    pattern,
		staticPath: nil,
		segments:   segs,
		handler:    h,
		middleware: append([]Middleware(nil), mw...),
		hasParams:  hasParams,
	}
	if !hasParams {
		rt.staticPath = []byte(pattern)
	}

	switch method {
	case methodGET:
		r.getRoutes = append(r.getRoutes, rt)
	case methodPOST:
		r.postRoutes = append(r.postRoutes, rt)
	}
}

func (r *Router) Handler() fasthttp.RequestHandler {
	return func(rc *fasthttp.RequestCtx) {
		// Params map is allocated only when we match a param route.
		c := fx.NewFastContext(rc, nil, r.coreCtx)

		method := rc.Method()
		path := rc.Path()

		var routes []*route
		// Fast method dispatch (only GET/POST supported in litefast).
		if len(method) == 3 && method[0] == 'G' { // GET
			routes = r.getRoutes
		} else if len(method) == 4 && method[0] == 'P' && method[1] == 'O' { // POST
			routes = r.postRoutes
		} else {
			_ = r.notFound(c)
			return
		}

		for _, rt := range routes {
			if ok := matchAndFill(rt, path, c, &r.paramPool); !ok {
				continue
			}

			h := rt.handler
			for i := len(rt.middleware) - 1; i >= 0; i-- {
				h = rt.middleware[i](h)
			}
			for i := len(r.middleware) - 1; i >= 0; i-- {
				h = r.middleware[i](h)
			}

			if err := h(c); err != nil {
				_ = r.onError(c, err)
			}

			// Return params slice to pool (if used)
			if c.Params != nil {
				c.Params = c.Params[:0]
				r.paramPool.Put(c.Params)
			}
			return
		}

		_ = r.notFound(c)
	}
}

func compilePattern(pattern string) ([]segment, bool) {
	// Split on '/', ignore leading empty segment.
	var out []segment
	hasParams := false

	p := []byte(pattern)
	i := 0
	for i < len(p) && p[i] == '/' {
		i++
	}
	start := i
	for i <= len(p) {
		if i == len(p) || p[i] == '/' {
			if i > start {
				part := p[start:i]
				if len(part) > 0 && part[0] == ':' && len(part) > 1 {
					hasParams = true
					out = append(out, segment{param: string(part[1:])})
				} else {
					out = append(out, segment{static: append([]byte(nil), part...)})
				}
			} else if i == start {
				// double slash or trailing slash -> treat as empty segment (unsupported)
				out = append(out, segment{static: []byte{}})
			}
			i++
			start = i
			continue
		}
		i++
	}
	return out, hasParams
}

func matchAndFill(rt *route, path []byte, c *fx.FastContext, pool *sync.Pool) bool {
	// Fast exact match for patterns without params.
	if !rt.hasParams && bytes.Equal(path, rt.staticPath) {
		return true
	}

	// Split path into segments without allocations.
	// Reject paths with empty segments for simplicity.
	if rt.hasParams {
		if s, ok := pool.Get().([]fx.Param); ok {
			c.Params = s[:0]
		} else {
			c.Params = make([]fx.Param, 0, 4)
		}
	} else {
		c.Params = nil
	}

	i := 0
	for i < len(path) && path[i] == '/' {
		i++
	}
	segIdx := 0
	start := i
	for i <= len(path) {
		if segIdx >= len(rt.segments) {
			return false
		}
		if i == len(path) || path[i] == '/' {
			if i == start {
				return false
			}
			part := path[start:i]
			seg := rt.segments[segIdx]
			if seg.param != "" {
				// Zero-copy view into request memory for perf. Do not store beyond request lifetime.
				c.Params = append(c.Params, fx.Param{Key: seg.param, Value: b2s(part)})
			} else {
				if !bytes.Equal(seg.static, part) {
					return false
				}
			}
			segIdx++
			i++
			start = i
			continue
		}
		i++
	}

	return segIdx == len(rt.segments)
}

func b2s(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	// #nosec G103 -- performance-critical, zero-copy conversion.
	// The resulting string only lives within the request handling path and must
	// not be stored beyond the request lifetime.
	return unsafe.String(unsafe.SliceData(b), len(b))
}
