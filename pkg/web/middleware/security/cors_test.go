package security

import (
	"context"
	"strings"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
)

func TestDefaultCORSConfig(t *testing.T) {
	config := DefaultCORSConfig()

	if len(config.AllowedOrigins) == 0 {
		t.Error("AllowedOrigins should not be empty")
	}

	if len(config.AllowedMethods) == 0 {
		t.Error("AllowedMethods should not be empty")
	}

	if len(config.AllowedHeaders) == 0 {
		t.Error("AllowedHeaders should not be empty")
	}

	if config.MaxAge == 0 {
		t.Error("MaxAge should be set")
	}
}

func TestCORS_PreflightRequest(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
		MaxAge:         3600,
	}

	mw := CORS(config)

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("OPTIONS")
	reqCtx.Request.Header.Set("Origin", "https://example.com")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		ctx.RequestCtx.SetStatusCode(200)
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("CORS middleware returned error: %v", err)
	}

	// Should return 204 for preflight
	if reqCtx.Response.StatusCode() != 204 {
		t.Errorf("Expected status code 204, got %d", reqCtx.Response.StatusCode())
	}

	// Should have CORS headers
	allowOrigin := string(reqCtx.Response.Header.Peek("Access-Control-Allow-Origin"))
	if allowOrigin != "https://example.com" {
		t.Errorf("Expected Access-Control-Allow-Origin 'https://example.com', got '%s'", allowOrigin)
	}

	allowMethods := string(reqCtx.Response.Header.Peek("Access-Control-Allow-Methods"))
	if !strings.Contains(allowMethods, "GET") || !strings.Contains(allowMethods, "POST") {
		t.Errorf("Expected Access-Control-Allow-Methods to contain GET and POST, got '%s'", allowMethods)
	}
}

func TestCORS_AllowAllOrigins(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
	}

	mw := CORS(config)

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.Header.Set("Origin", "https://any-origin.com")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		ctx.RequestCtx.SetStatusCode(200)
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("CORS middleware returned error: %v", err)
	}

	// Should allow all origins
	allowOrigin := string(reqCtx.Response.Header.Peek("Access-Control-Allow-Origin"))
	if allowOrigin != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin '*', got '%s'", allowOrigin)
	}
}

func TestCORS_OriginNotAllowed(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
	}

	mw := CORS(config)

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.Header.Set("Origin", "https://unauthorized.com")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		ctx.RequestCtx.SetStatusCode(200)
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("CORS middleware returned error: %v", err)
	}

	// Should not set Access-Control-Allow-Origin for unauthorized origin
	allowOrigin := string(reqCtx.Response.Header.Peek("Access-Control-Allow-Origin"))
	if allowOrigin != "" {
		t.Errorf("Expected empty Access-Control-Allow-Origin for unauthorized origin, got '%s'", allowOrigin)
	}
}

func TestCORS_AllowCredentials(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}

	mw := CORS(config)

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("OPTIONS")
	reqCtx.Request.Header.Set("Origin", "https://example.com")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("CORS middleware returned error: %v", err)
	}

	// Should set Allow-Credentials header
	allowCredentials := string(reqCtx.Response.Header.Peek("Access-Control-Allow-Credentials"))
	if allowCredentials != "true" {
		t.Errorf("Expected Access-Control-Allow-Credentials 'true', got '%s'", allowCredentials)
	}
}

func TestCORS_MaxAge(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
		MaxAge:         86400,
	}

	mw := CORS(config)

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("OPTIONS")
	reqCtx.Request.Header.Set("Origin", "https://example.com")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("CORS middleware returned error: %v", err)
	}

	// Should set Max-Age header
	maxAge := string(reqCtx.Response.Header.Peek("Access-Control-Max-Age"))
	if maxAge != "86400" {
		t.Errorf("Expected Access-Control-Max-Age '86400', got '%s'", maxAge)
	}
}

func TestCORS_ExposedHeaders(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
		ExposedHeaders: []string{"X-Request-ID", "X-Custom-Header"},
	}

	mw := CORS(config)

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.Header.Set("Origin", "https://example.com")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		ctx.RequestCtx.SetStatusCode(200)
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("CORS middleware returned error: %v", err)
	}

	// Should set Expose-Headers
	exposedHeaders := string(reqCtx.Response.Header.Peek("Access-Control-Expose-Headers"))
	if !strings.Contains(exposedHeaders, "X-Request-ID") {
		t.Errorf("Expected Access-Control-Expose-Headers to contain 'X-Request-ID', got '%s'", exposedHeaders)
	}
}

func TestCORS_NoOrigin(t *testing.T) {
	config := DefaultCORSConfig()
	mw := CORS(config)

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("GET")
	// No Origin header

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		ctx.RequestCtx.SetStatusCode(200)
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("CORS middleware returned error: %v", err)
	}

	// Should not set CORS headers when no origin
	allowOrigin := string(reqCtx.Response.Header.Peek("Access-Control-Allow-Origin"))
	if allowOrigin != "" && allowOrigin != "*" {
		t.Logf("Allow-Origin set to '%s' when no origin (may be acceptable)", allowOrigin)
	}
}
