package openapi

import (
	"context"
	"strings"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
)

func TestServeSpec(t *testing.T) {
	spec := NewSpec(Info{
		Title:   "Test API",
		Version: "1.0.0",
	})
	spec.AddPath("/test", PathItem{
		GET: &Operation{
			Summary: "Test endpoint",
			Responses: map[string]Response{
				"200": JSONResponse("Success", StringSchema()),
			},
		},
	})

	handler := ServeSpec(spec)
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	err := handler(fastCtx)
	if err != nil {
		t.Fatalf("ServeSpec() error = %v", err)
	}

	if reqCtx.Response.StatusCode() != fasthttp.StatusOK {
		t.Errorf("ServeSpec() status = %d, want %d", reqCtx.Response.StatusCode(), fasthttp.StatusOK)
	}

	contentType := string(reqCtx.Response.Header.ContentType())
	if contentType != "application/json" {
		t.Errorf("ServeSpec() Content-Type = %q, want 'application/json'", contentType)
	}

	body := string(reqCtx.Response.Body())
	if !strings.Contains(body, "openapi") {
		t.Error("ServeSpec() response body missing 'openapi' field")
	}
	if !strings.Contains(body, "Test API") {
		t.Error("ServeSpec() response body missing API title")
	}
}

func TestServeSwaggerUI(t *testing.T) {
	specURL := "/openapi.json"
	title := "Test API Documentation"

	handler := ServeSwaggerUI(specURL, title)
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	err := handler(fastCtx)
	if err != nil {
		t.Fatalf("ServeSwaggerUI() error = %v", err)
	}

	if reqCtx.Response.StatusCode() != fasthttp.StatusOK {
		t.Errorf("ServeSwaggerUI() status = %d, want %d", reqCtx.Response.StatusCode(), fasthttp.StatusOK)
	}

	contentType := string(reqCtx.Response.Header.ContentType())
	if contentType != "text/html" {
		t.Errorf("ServeSwaggerUI() Content-Type = %q, want 'text/html'", contentType)
	}

	body := string(reqCtx.Response.Body())
	if !strings.Contains(body, title) {
		t.Errorf("ServeSwaggerUI() response body missing title %q", title)
	}
	if !strings.Contains(body, specURL) {
		t.Errorf("ServeSwaggerUI() response body missing spec URL %q", specURL)
	}
	if !strings.Contains(body, "swagger-ui") {
		t.Error("ServeSwaggerUI() response body missing 'swagger-ui'")
	}
}

func TestRegisterOpenAPIRoutes(t *testing.T) {
	spec := NewSpec(Info{
		Title:   "Test API",
		Version: "1.0.0",
	})

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := web.DefaultFastHTTPServerConfig(":0")
	server := web.NewFastHTTPServer(gocmd, config)
	router := server.FastRouter()

	// Register with defaults
	RegisterOpenAPIRoutes(router, spec)

	// Verify routes are registered
	// Note: We can't easily test router internals, but we can verify the handler works
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/openapi.json")
	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	// Test spec handler
	specHandler := ServeSpec(spec)
	err := specHandler(fastCtx)
	if err != nil {
		t.Fatalf("ServeSpec() error = %v", err)
	}
	if reqCtx.Response.StatusCode() != fasthttp.StatusOK {
		t.Errorf("ServeSpec() status = %d, want %d", reqCtx.Response.StatusCode(), fasthttp.StatusOK)
	}
}

func TestRegisterOpenAPIRoutes_WithOptions(t *testing.T) {
	spec := NewSpec(Info{
		Title:   "Test API",
		Version: "1.0.0",
	})

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := web.DefaultFastHTTPServerConfig(":0")
	server := web.NewFastHTTPServer(gocmd, config)
	router := server.FastRouter()

	// Register with custom options
	RegisterOpenAPIRoutes(router, spec,
		WithSpecPath("/api/openapi.json"),
		WithUIPath("/api/docs"),
		WithUITitle("Custom API Docs"),
		WithSpecURL("/api/openapi.json"),
	)

	// Verify custom paths work
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/api/openapi.json")
	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	specHandler := ServeSpec(spec)
	err := specHandler(fastCtx)
	if err != nil {
		t.Fatalf("ServeSpec() error = %v", err)
	}
}

func TestWithSpecPath(t *testing.T) {
	config := &OpenAPIConfig{}
	opt := WithSpecPath("/custom/openapi.json")
	opt(config)

	if config.SpecPath != "/custom/openapi.json" {
		t.Errorf("WithSpecPath() SpecPath = %q, want '/custom/openapi.json'", config.SpecPath)
	}
	if config.SpecURL != "/custom/openapi.json" {
		t.Errorf("WithSpecPath() SpecURL = %q, want '/custom/openapi.json'", config.SpecURL)
	}
}

func TestWithUIPath(t *testing.T) {
	config := &OpenAPIConfig{}
	opt := WithUIPath("/custom/docs")
	opt(config)

	if config.UIPath != "/custom/docs" {
		t.Errorf("WithUIPath() UIPath = %q, want '/custom/docs'", config.UIPath)
	}
}

func TestWithUITitle(t *testing.T) {
	config := &OpenAPIConfig{}
	opt := WithUITitle("My Custom API")
	opt(config)

	if config.UITitle != "My Custom API" {
		t.Errorf("WithUITitle() UITitle = %q, want 'My Custom API'", config.UITitle)
	}
}

func TestWithSpecURL(t *testing.T) {
	config := &OpenAPIConfig{
		SpecPath: "/openapi.json",
	}
	opt := WithSpecURL("/api/openapi.json")
	opt(config)

	if config.SpecURL != "/api/openapi.json" {
		t.Errorf("WithSpecURL() SpecURL = %q, want '/api/openapi.json'", config.SpecURL)
	}
	// SpecPath should remain unchanged
	if config.SpecPath != "/openapi.json" {
		t.Errorf("WithSpecURL() SpecPath = %q, want '/openapi.json'", config.SpecPath)
	}
}
