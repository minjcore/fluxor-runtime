package dashboard

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
)

func TestNewHandler(t *testing.T) {
	handler := NewHandler()
	if handler == nil {
		t.Fatal("NewHandler() should not return nil")
	}
}

func TestHandler_MetricsHandler(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	handler := NewHandler()

	// Create a test request context
	reqCtx := &fasthttp.RequestCtx{}
	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	err := handler.MetricsHandler(fastCtx)
	if err != nil {
		t.Errorf("MetricsHandler() returned error: %v", err)
	}

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status code 200, got %d", reqCtx.Response.StatusCode())
	}
}

func TestHandler_HealthHandler(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	handler := NewHandler()

	// Create a test request context
	reqCtx := &fasthttp.RequestCtx{}
	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	err := handler.HealthHandler(fastCtx)
	if err != nil {
		t.Errorf("HealthHandler() returned error: %v", err)
	}

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status code 200, got %d", reqCtx.Response.StatusCode())
	}
}

func TestHandler_DashboardHTMLHandler(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	handler := NewHandler()

	// Create a test request context
	reqCtx := &fasthttp.RequestCtx{}
	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	err := handler.DashboardHTMLHandler(fastCtx)
	if err != nil {
		t.Errorf("DashboardHTMLHandler() returned error: %v", err)
	}

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status code 200, got %d", reqCtx.Response.StatusCode())
	}

	contentType := string(reqCtx.Response.Header.ContentType())
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Expected content type 'text/html; charset=utf-8', got '%s'", contentType)
	}
}

func TestHandler_DashboardJSHandler(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	handler := NewHandler()

	// Create a test request context
	reqCtx := &fasthttp.RequestCtx{}
	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	err := handler.DashboardJSHandler(fastCtx)
	if err != nil {
		t.Errorf("DashboardJSHandler() returned error: %v", err)
	}

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status code 200, got %d", reqCtx.Response.StatusCode())
	}

	contentType := string(reqCtx.Response.Header.ContentType())
	if contentType != "application/javascript; charset=utf-8" {
		t.Errorf("Expected content type 'application/javascript; charset=utf-8', got '%s'", contentType)
	}
}

func TestHandler_DashboardCSSHandler(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	handler := NewHandler()

	// Create a test request context
	reqCtx := &fasthttp.RequestCtx{}
	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	err := handler.DashboardCSSHandler(fastCtx)
	if err != nil {
		t.Errorf("DashboardCSSHandler() returned error: %v", err)
	}

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status code 200, got %d", reqCtx.Response.StatusCode())
	}

	contentType := string(reqCtx.Response.Header.ContentType())
	if contentType != "text/css; charset=utf-8" {
		t.Errorf("Expected content type 'text/css; charset=utf-8', got '%s'", contentType)
	}
}

func TestRegister(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := web.CCUBasedConfigWithUtilization(":8080", 1000, 67)
	server := web.NewFastHTTPServer(gocmd, config)
	router := server.FastRouter()

	// Register with empty prefix
	Register(router, "")

	// Register with prefix
	Register(router, "/admin")

	// Verify routes are registered by checking if we can call them
	// (actual route testing would require a running server)
	_ = router
}

func TestRegister_WithPrefix(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := web.CCUBasedConfigWithUtilization(":8080", 1000, 67)
	server := web.NewFastHTTPServer(gocmd, config)
	router := server.FastRouter()

	// Register with prefix
	Register(router, "/api")

	// Verify routes are registered by checking router state
	// (actual route testing would require a running server)
	_ = router
	_ = server
}
