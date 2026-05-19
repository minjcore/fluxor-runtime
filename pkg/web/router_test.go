package web

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"testing"
)

func TestNewRouter(t *testing.T) {
	router := NewRouter()
	if router == nil {
		t.Fatal("NewRouter() should not return nil")
	}
}

func TestRouter_GET(t *testing.T) {
	router := NewRouter().(*router)
	
	router.GET("/test", func(ctx *RequestContext) error {
		return ctx.Text(200, "GET handler")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "GET handler" {
		t.Errorf("Expected body 'GET handler', got '%s'", w.Body.String())
	}
}

func TestRouter_POST(t *testing.T) {
	router := NewRouter().(*router)
	
	router.POST("/test", func(ctx *RequestContext) error {
		return ctx.Text(200, "POST handler")
	})

	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRouter_PUT(t *testing.T) {
	router := NewRouter().(*router)
	
	router.PUT("/test", func(ctx *RequestContext) error {
		return ctx.Text(200, "PUT handler")
	})

	req := httptest.NewRequest("PUT", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRouter_DELETE(t *testing.T) {
	router := NewRouter().(*router)
	
	router.DELETE("/test", func(ctx *RequestContext) error {
		return ctx.Text(200, "DELETE handler")
	})

	req := httptest.NewRequest("DELETE", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRouter_PATCH(t *testing.T) {
	router := NewRouter().(*router)
	
	router.PATCH("/test", func(ctx *RequestContext) error {
		return ctx.Text(200, "PATCH handler")
	})

	req := httptest.NewRequest("PATCH", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRouter_Route(t *testing.T) {
	router := NewRouter().(*router)
	
	router.Route("OPTIONS", "/test", func(ctx *RequestContext) error {
		return ctx.Text(200, "OPTIONS handler")
	})

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRouter_PathParams(t *testing.T) {
	router := NewRouter().(*router)
	
	router.GET("/users/:id", func(ctx *RequestContext) error {
		id := ctx.Params["id"]
		return ctx.Text(200, "User ID: "+id)
	})

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "User ID: 123" {
		t.Errorf("Expected 'User ID: 123', got '%s'", w.Body.String())
	}
}

func TestRouter_MultipleParams(t *testing.T) {
	router := NewRouter().(*router)
	
	router.GET("/users/:userId/posts/:postId", func(ctx *RequestContext) error {
		userId := ctx.Params["userId"]
		postId := ctx.Params["postId"]
		return ctx.Text(200, userId+"/"+postId)
	})

	req := httptest.NewRequest("GET", "/users/123/posts/456", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "123/456" {
		t.Errorf("Expected '123/456', got '%s'", w.Body.String())
	}
}

func TestRouter_NotFound(t *testing.T) {
	router := NewRouter().(*router)
	
	router.GET("/test", func(ctx *RequestContext) error {
		return ctx.Text(200, "test")
	})

	req := httptest.NewRequest("GET", "/notfound", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestRouter_MethodNotAllowed(t *testing.T) {
	router := NewRouter().(*router)
	
	router.GET("/test", func(ctx *RequestContext) error {
		return ctx.Text(200, "test")
	})

	// POST to a GET-only route should return 404 (not found, not method not allowed)
	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestRouter_Middleware(t *testing.T) {
	router := NewRouter().(*router)
	
	middlewareCalled := false
	middleware := func(next RequestHandler) RequestHandler {
		return func(ctx *RequestContext) error {
			middlewareCalled = true
			return next(ctx)
		}
	}

	router.Use(middleware)
	router.GET("/test", func(ctx *RequestContext) error {
		return ctx.Text(200, "test")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !middlewareCalled {
		t.Error("Middleware should have been called")
	}
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRouter_MultipleMiddleware(t *testing.T) {
	router := NewRouter().(*router)
	
	callOrder := []string{}
	
	mw1 := func(next RequestHandler) RequestHandler {
		return func(ctx *RequestContext) error {
			callOrder = append(callOrder, "mw1")
			return next(ctx)
		}
	}
	
	mw2 := func(next RequestHandler) RequestHandler {
		return func(ctx *RequestContext) error {
			callOrder = append(callOrder, "mw2")
			return next(ctx)
		}
	}

	router.Use(mw1)
	router.Use(mw2)
	router.GET("/test", func(ctx *RequestContext) error {
		callOrder = append(callOrder, "handler")
		return ctx.Text(200, "test")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Middleware should be called in order: mw1, mw2, handler
	expectedOrder := []string{"mw1", "mw2", "handler"}
	if len(callOrder) != len(expectedOrder) {
		t.Errorf("Expected %d calls, got %d", len(expectedOrder), len(callOrder))
	}
	for i, expected := range expectedOrder {
		if i < len(callOrder) && callOrder[i] != expected {
			t.Errorf("Call %d: expected '%s', got '%s'", i, expected, callOrder[i])
		}
	}
}

func TestRouter_HandlerError(t *testing.T) {
	router := NewRouter().(*router)
	
	router.GET("/test", func(ctx *RequestContext) error {
		return ctx.Text(500, "error")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Handler returns error, but router should still process it
	// The error is returned but router doesn't handle it specially
	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestRouter_JSONResponse(t *testing.T) {
	router := NewRouter().(*router)
	
	router.GET("/test", func(ctx *RequestContext) error {
		return ctx.JSON(200, map[string]string{"message": "test"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}
}

func TestRouter_MatchPath(t *testing.T) {
	router := NewRouter().(*router)
	
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
		{"/users/:id/posts/:postId", "/users/123/posts", false},
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

func TestRouter_ExtractParams(t *testing.T) {
	router := NewRouter().(*router)
	
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
			params := router.extractParams(tt.pattern, tt.path)
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

func TestRouter_ConcurrentAccess(t *testing.T) {
	router := NewRouter().(*router)
	
	// Register routes concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			path := "/test" + fmt.Sprintf("%d", id)
			router.GET(path, func(ctx *RequestContext) error {
				return ctx.Text(200, "test")
			})
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Give a moment for all routes to be registered
	// Test that routes are accessible
	req := httptest.NewRequest("GET", "/test0", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}


func TestRouter_ServeHTTP_ErrorHandling(t *testing.T) {
	router := NewRouter().(*router)
	
	router.GET("/error", func(ctx *RequestContext) error {
		// Return an error that will be handled by router
		return ctx.Text(500, "internal error")
	})

	req := httptest.NewRequest("GET", "/error", nil)
	w := httptest.NewRecorder()
	
	// Router should handle the error gracefully
	router.ServeHTTP(w, req)
	
	// The handler sets status 500, so that's what we expect
	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestRouter_RouteOrdering(t *testing.T) {
	router := NewRouter().(*router)
	
	// Register multiple routes
	router.GET("/users/:id", func(ctx *RequestContext) error {
		return ctx.Text(200, "user")
	})
	
	router.GET("/users/me", func(ctx *RequestContext) error {
		return ctx.Text(200, "me")
	})

	// First match wins (exact match should come before param match in real router)
	// But our simple router matches in registration order
	req := httptest.NewRequest("GET", "/users/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should match first route (param route)
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	// The param route will match and extract "me" as the id
	if w.Body.String() != "user" {
		t.Errorf("Expected 'user', got '%s'", w.Body.String())
	}
}

func TestRouter_EmptyPath(t *testing.T) {
	router := NewRouter().(*router)
	
	router.GET("/", func(ctx *RequestContext) error {
		return ctx.Text(200, "root")
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "root" {
		t.Errorf("Expected 'root', got '%s'", w.Body.String())
	}
}

func TestRouter_PathWithQueryString(t *testing.T) {
	router := NewRouter().(*router)
	
	router.GET("/test", func(ctx *RequestContext) error {
		return ctx.Text(200, "test")
	})

	req := httptest.NewRequest("GET", "/test?foo=bar", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Query string should not affect path matching
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRouter_RequestBody(t *testing.T) {
	router := NewRouter().(*router)
	
	router.POST("/test", func(ctx *RequestContext) error {
		body := make([]byte, 100)
		n, _ := ctx.Request.Body.Read(body)
		return ctx.Text(200, string(body[:n]))
	})

	body := bytes.NewBufferString("test body")
	req := httptest.NewRequest("POST", "/test", body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
