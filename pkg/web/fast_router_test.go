package web

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/valyala/fasthttp"
)

func TestNewFastRouter(t *testing.T) {
	router := NewFastRouter()
	if router == nil {
		t.Fatal("NewFastRouter() should not return nil")
	}
}

func TestFastRouter_GETFast(t *testing.T) {
	router := NewFastRouter()
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	handlerCalled := false
	router.GETFast("/test", func(ctx *FastRequestContext) error {
		handlerCalled = true
		return ctx.JSON(200, map[string]string{"message": "test"})
	})

	// Create a test context
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/test")

	fastCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	router.ServeFastHTTP(fastCtx)

	if !handlerCalled {
		t.Error("Handler should have been called")
	}
	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status 200, got %d", reqCtx.Response.StatusCode())
	}
}

func TestFastRouter_POSTFast(t *testing.T) {
	router := NewFastRouter()
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	handlerCalled := false
	router.POSTFast("/test", func(ctx *FastRequestContext) error {
		handlerCalled = true
		return nil
	})

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("POST")
	reqCtx.Request.SetRequestURI("/test")

	fastCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	router.ServeFastHTTP(fastCtx)

	if !handlerCalled {
		t.Error("Handler should have been called")
	}
}

func TestFastRouter_PathParams(t *testing.T) {
	router := NewFastRouter()
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	router.GETFast("/users/:id", func(ctx *FastRequestContext) error {
		id := ctx.Params["id"]
		return ctx.JSON(200, map[string]string{"id": id})
	})

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/users/123")

	fastCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	router.ServeFastHTTP(fastCtx)

	if fastCtx.Params["id"] != "123" {
		t.Errorf("Expected param 'id' = '123', got '%s'", fastCtx.Params["id"])
	}
}

func TestFastRouter_MultipleParams(t *testing.T) {
	router := NewFastRouter()
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	router.GETFast("/users/:userId/posts/:postId", func(ctx *FastRequestContext) error {
		return ctx.JSON(200, map[string]string{
			"userId": ctx.Params["userId"],
			"postId": ctx.Params["postId"],
		})
	})

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/users/123/posts/456")

	fastCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	router.ServeFastHTTP(fastCtx)

	if fastCtx.Params["userId"] != "123" {
		t.Errorf("Expected userId = '123', got '%s'", fastCtx.Params["userId"])
	}
	if fastCtx.Params["postId"] != "456" {
		t.Errorf("Expected postId = '456', got '%s'", fastCtx.Params["postId"])
	}
}

func TestFastRouter_NotFound(t *testing.T) {
	router := NewFastRouter()
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	router.GETFast("/test", func(ctx *FastRequestContext) error {
		return nil
	})

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/notfound")

	fastCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	router.ServeFastHTTP(fastCtx)

	if reqCtx.Response.StatusCode() != 404 {
		t.Errorf("Expected status 404, got %d", reqCtx.Response.StatusCode())
	}
}

func TestFastRouter_DefaultHandler(t *testing.T) {
	router := NewFastRouter()
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	defaultCalled := false
	router.SetDefaultHandler(func(ctx *FastRequestContext) error {
		defaultCalled = true
		return ctx.JSON(200, map[string]string{"message": "default"})
	})

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/notfound")

	fastCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	router.ServeFastHTTP(fastCtx)

	if !defaultCalled {
		t.Error("Default handler should have been called")
	}
	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status 200, got %d", reqCtx.Response.StatusCode())
	}
}

func TestFastRouter_Middleware(t *testing.T) {
	router := NewFastRouter()
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	middlewareCalled := false
	middleware := func(next FastRequestHandler) FastRequestHandler {
		return func(ctx *FastRequestContext) error {
			middlewareCalled = true
			return next(ctx)
		}
	}

	router.UseFast(middleware)
	router.GETFast("/test", func(ctx *FastRequestContext) error {
		return ctx.JSON(200, map[string]string{"message": "test"})
	})

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/test")

	fastCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	router.ServeFastHTTP(fastCtx)

	if !middlewareCalled {
		t.Error("Middleware should have been called")
	}
}

func TestFastRouter_RouteSpecificMiddleware(t *testing.T) {
	router := NewFastRouter()
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	routeMwCalled := false
	globalMwCalled := false

	routeMw := func(next FastRequestHandler) FastRequestHandler {
		return func(ctx *FastRequestContext) error {
			routeMwCalled = true
			return next(ctx)
		}
	}

	globalMw := func(next FastRequestHandler) FastRequestHandler {
		return func(ctx *FastRequestContext) error {
			globalMwCalled = true
			return next(ctx)
		}
	}

	router.UseFast(globalMw)
	router.GETFastWith("/test", func(ctx *FastRequestContext) error {
		return ctx.JSON(200, map[string]string{"message": "test"})
	}, routeMw)

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/test")

	fastCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	router.ServeFastHTTP(fastCtx)

	if !routeMwCalled {
		t.Error("Route middleware should have been called")
	}
	if !globalMwCalled {
		t.Error("Global middleware should have been called")
	}
}

func TestFastRouter_HandlerError(t *testing.T) {
	router := NewFastRouter()
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	router.GETFast("/error", func(ctx *FastRequestContext) error {
		ctx.Error("test error", fasthttp.StatusInternalServerError)
		return nil
	})

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/error")

	fastCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	router.ServeFastHTTP(fastCtx)

	// Router should handle the error
	if reqCtx.Response.StatusCode() != 500 {
		t.Errorf("Expected status 500, got %d", reqCtx.Response.StatusCode())
	}
}

func TestFastRouter_MatchPath(t *testing.T) {
	router := NewFastRouter()

	tests := []struct {
		pattern string
		path    string
		match   bool
	}{
		{"/test", "/test", true},
		{"/test", "/test/", false},
		{"/test", "/other", false},
		{"/users/:id", "/users/123", true},
		{"/users/:id", "/users/123/posts", false},
		{"/users/:id/posts/:postId", "/users/123/posts/456", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+" vs "+tt.path, func(t *testing.T) {
			result := router.matchPath(tt.pattern, tt.path)
			if result != tt.match {
				t.Errorf("matchPath(%q, %q) = %v, want %v", tt.pattern, tt.path, result, tt.match)
			}
		})
	}
}

func TestFastRouter_ExtractParams(t *testing.T) {
	router := NewFastRouter()

	tests := []struct {
		pattern string
		path    string
		params  map[string]string
	}{
		{"/users/:id", "/users/123", map[string]string{"id": "123"}},
		{"/users/:userId/posts/:postId", "/users/123/posts/456", map[string]string{"userId": "123", "postId": "456"}},
		{"/test", "/test", map[string]string{}},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			params := make(map[string]string)
			router.extractParams(tt.pattern, tt.path, params)
			
			if len(params) != len(tt.params) {
				t.Errorf("Expected %d params, got %d", len(tt.params), len(params))
			}
			for k, v := range tt.params {
				if params[k] != v {
					t.Errorf("Param %s: expected '%s', got '%s'", k, v, params[k])
				}
			}
		})
	}
}

func TestFastRouter_ConcurrentAccess(t *testing.T) {
	router := NewFastRouter()
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Register routes concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			path := "/test" + string(rune('0'+id))
			router.GETFast(path, func(ctx *FastRequestContext) error {
				return ctx.JSON(200, map[string]string{"id": string(rune('0' + id))})
			})
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test that routes are accessible
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/test0")

	fastCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	router.ServeFastHTTP(fastCtx)

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status 200, got %d", reqCtx.Response.StatusCode())
	}
}

func TestFastRouter_AllHTTPMethods(t *testing.T) {
	router := NewFastRouter()
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	methods := []struct {
		name   string
		method string
		reg    func(string, FastRequestHandler)
	}{
		{"GET", "GET", router.GETFast},
		{"POST", "POST", router.POSTFast},
		{"PUT", "PUT", router.PUTFast},
		{"DELETE", "DELETE", router.DELETEFast},
		{"PATCH", "PATCH", router.PATCHFast},
	}

	for _, m := range methods {
		t.Run(m.name, func(t *testing.T) {
			handlerCalled := false
			m.reg("/test", func(ctx *FastRequestContext) error {
				handlerCalled = true
				return nil
			})

			reqCtx := &fasthttp.RequestCtx{}
			reqCtx.Request.Header.SetMethod(m.method)
			reqCtx.Request.SetRequestURI("/test")

			fastCtx := &FastRequestContext{
				BaseRequestContext: core.NewBaseRequestContext(),
				RequestCtx:         reqCtx,
				GoCMD:              gocmd,
				EventBus:           gocmd.EventBus(),
				Params:             make(map[string]string),
			}

			router.ServeFastHTTP(fastCtx)

			if !handlerCalled {
				t.Errorf("%s handler should have been called", m.name)
			}
		})
	}
}

func TestFastRouter_RouteFastWith(t *testing.T) {
	router := NewFastRouter()
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	middlewareCalled := false
	mw := func(next FastRequestHandler) FastRequestHandler {
		return func(ctx *FastRequestContext) error {
			middlewareCalled = true
			return next(ctx)
		}
	}

	router.RouteFastWith("GET", "/test", func(ctx *FastRequestContext) error {
		return ctx.JSON(200, map[string]string{"message": "test"})
	}, mw)

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/test")

	fastCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	router.ServeFastHTTP(fastCtx)

	if !middlewareCalled {
		t.Error("Middleware should have been called")
	}
}
