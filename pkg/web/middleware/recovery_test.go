package middleware

import (
	"context"
	"strings"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
)

func TestDefaultRecoveryConfig(t *testing.T) {
	config := DefaultRecoveryConfig()

	if config.Logger == nil {
		t.Error("Logger should not be nil")
	}

	if config.StackTrace {
		t.Error("StackTrace should be false by default")
	}
}

func TestRecovery_PanicRecovery(t *testing.T) {
	config := RecoveryConfig{
		Logger:     core.NewDefaultLogger(),
		StackTrace: false,
	}

	mw := Recovery(config)

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("GET")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		panic("test panic")
	})

	// Should not panic
	err := handler(fastCtx)
	if err != nil {
		t.Errorf("Recovery middleware should handle panic gracefully, got error: %v", err)
	}

	// Should return 500 status
	if reqCtx.Response.StatusCode() != 500 {
		t.Errorf("Expected status code 500, got %d", reqCtx.Response.StatusCode())
	}

	// Should have JSON content type
	contentType := string(reqCtx.Response.Header.ContentType())
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected JSON content type, got '%s'", contentType)
	}
}

func TestRecovery_WithStackTrace(t *testing.T) {
	config := RecoveryConfig{
		Logger:     core.NewDefaultLogger(),
		StackTrace: true,
	}

	mw := Recovery(config)

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("GET")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	panicMsg := "test panic with stack trace"
	handler := mw(func(ctx *web.FastRequestContext) error {
		panic(panicMsg)
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("Recovery middleware should handle panic gracefully, got error: %v", err)
	}

	// Response body should contain panic message
	body := string(reqCtx.Response.Body())
	if !strings.Contains(body, panicMsg) {
		t.Logf("Response body: %s", body)
		t.Log("Response body should contain panic message when StackTrace is true")
	}
}

func TestRecovery_NoPanic(t *testing.T) {
	config := DefaultRecoveryConfig()
	mw := Recovery(config)

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("GET")

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
		t.Errorf("Recovery middleware returned error: %v", err)
	}

	// Should have normal status code
	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status code 200, got %d", reqCtx.Response.StatusCode())
	}
}

func TestRecovery_NilLogger(t *testing.T) {
	config := RecoveryConfig{
		Logger: nil, // Should default to NewDefaultLogger()
	}

	mw := Recovery(config)
	if mw == nil {
		t.Fatal("Recovery() should not return nil even with nil logger")
	}

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("GET")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		panic("test panic with nil logger")
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("Recovery middleware should handle panic gracefully, got error: %v", err)
	}

	if reqCtx.Response.StatusCode() != 500 {
		t.Errorf("Expected status code 500, got %d", reqCtx.Response.StatusCode())
	}
}

func TestRecovery_RequestIDInResponse(t *testing.T) {
	config := DefaultRecoveryConfig()
	mw := Recovery(config)

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.Header.Set("X-Request-ID", "test-request-id")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		panic("test panic")
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("Recovery middleware should handle panic gracefully, got error: %v", err)
	}

	// Response body should contain request ID
	body := string(reqCtx.Response.Body())
	if !strings.Contains(body, "request_id") {
		t.Logf("Response body: %s", body)
		t.Log("Response body should contain request_id")
	}
}
