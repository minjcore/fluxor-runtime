package middleware

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
)

func TestRoutingMiddleware(t *testing.T) {
	gocmd := core.NewGoCMD(context.Background())
	defer gocmd.Close()

	server := web.NewFastHTTPServer(gocmd, web.DefaultFastHTTPServerConfig(":0"))
	router := server.FastRouter()

	// Create routing middleware
	middleware := DefaultRoutingMiddleware()
	router.UseFast(middleware.Handler())

	// Test route
	router.GETFast("/test", func(ctx *web.FastRequestContext) error {
		userID := ctx.UserID()
		floxid := ctx.FloxID()
		return ctx.JSON(200, map[string]interface{}{
			"user_id": userID,
			"floxid":  floxid,
		})
	})

	// Create request
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.Set("X-User-ID", "user-123")
	reqCtx.Request.Header.Set("X-Flox-ID", "stream-456")

	// Generate request ID (normally done in processRequest)
	requestID := core.GenerateRequestID()
	reqCtx.Request.Header.Set("X-Request-ID", requestID)

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	// Set requestID (normally done in processRequest)
	// We need to access the private field, so we'll test via public methods
	_ = fastCtx.RequestID() // This will return empty, but that's OK for this test

	// Execute
	router.ServeFastHTTP(fastCtx)

	// Verify response headers were set by middleware
	if userID := string(reqCtx.Response.Header.Peek("X-User-ID")); userID != "user-123" {
		t.Errorf("Expected X-User-ID header to be 'user-123', got '%s'", userID)
	}
	if floxid := string(reqCtx.Response.Header.Peek("X-Flox-ID")); floxid != "stream-456" {
		t.Errorf("Expected X-Flox-ID header to be 'stream-456', got '%s'", floxid)
	}
}

func TestFastRequestContext_UserID(t *testing.T) {
	gocmd := core.NewGoCMD(context.Background())
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	// Test Priority 1: JWT Claims (user_id claim) - should have highest priority
	fastCtx.Set("user", map[string]interface{}{
		"user_id": "user-from-jwt-user_id",
	})
	// Also set context and header to verify JWT takes priority
	fastCtx.Set("user_id", "user-from-context")
	reqCtx.Request.Header.Set("X-User-ID", "user-from-header")
	
	userID := fastCtx.UserID()
	if userID != "user-from-jwt-user_id" {
		t.Errorf("Expected UserID to be 'user-from-jwt-user_id' (JWT has highest priority), got '%s'", userID)
	}

	// Test Priority 1: JWT Claims (sub claim - JWT standard)
	fastCtx.Set("user", map[string]interface{}{
		"sub": "user-from-jwt-sub",
	})
	userID = fastCtx.UserID()
	if userID != "user-from-jwt-sub" {
		t.Errorf("Expected UserID to be 'user-from-jwt-sub' (from JWT sub claim), got '%s'", userID)
	}

	// Test Priority 2: Context (user_id key) - when JWT is not available
	fastCtx.Set("user", nil) // Clear JWT claims
	fastCtx.Set("user_id", "user-from-context")
	userID = fastCtx.UserID()
	if userID != "user-from-context" {
		t.Errorf("Expected UserID to be 'user-from-context' (from context when JWT not available), got '%s'", userID)
	}

	// Test Priority 3: HTTP Header - when JWT and Context are not available
	fastCtx.Set("user_id", "") // Clear context
	reqCtx.Request.Header.Set("X-User-ID", "user-from-header")
	userID = fastCtx.UserID()
	if userID != "user-from-header" {
		t.Errorf("Expected UserID to be 'user-from-header' (from header when JWT/Context not available), got '%s'", userID)
	}
}

func TestFastRequestContext_GetRoutingHeaders(t *testing.T) {
	gocmd := core.NewGoCMD(context.Background())
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.Set("X-User-ID", "user-123")
	reqCtx.Request.Header.Set("X-Flox-ID", "stream-456")
	reqCtx.Request.Header.Set("X-Session-ID", "session-789")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	headers := fastCtx.GetRoutingHeaders()

	// Verify all routing headers are extracted
	if headers["X-User-ID"] != "user-123" {
		t.Errorf("Expected X-User-ID to be 'user-123', got '%s'", headers["X-User-ID"])
	}
	if headers["X-Flox-ID"] != "stream-456" {
		t.Errorf("Expected X-Flox-ID to be 'stream-456', got '%s'", headers["X-Flox-ID"])
	}
	if headers["X-Session-ID"] != "session-789" {
		t.Errorf("Expected X-Session-ID to be 'session-789', got '%s'", headers["X-Session-ID"])
	}
}
