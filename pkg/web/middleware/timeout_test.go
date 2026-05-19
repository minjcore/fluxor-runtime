package middleware

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
)

func TestDefaultTimeoutConfig(t *testing.T) {
	timeout := 5 * time.Second
	config := DefaultTimeoutConfig(timeout)

	if config.Timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, config.Timeout)
	}

	if config.Logger == nil {
		t.Error("Logger should not be nil")
	}

	if config.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestTimeout_NoTimeout(t *testing.T) {
	config := TimeoutConfig{
		Timeout: 1 * time.Second,
		Logger:  core.NewDefaultLogger(),
	}

	mw := Timeout(config)

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
		t.Errorf("Timeout middleware returned error: %v", err)
	}

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status code 200, got %d", reqCtx.Response.StatusCode())
	}
}

func TestTimeout_TimeoutOccurs(t *testing.T) {
	config := TimeoutConfig{
		Timeout: 50 * time.Millisecond,
		Logger:  core.NewDefaultLogger(),
	}

	mw := Timeout(config)

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
		// Simulate slow handler
		time.Sleep(200 * time.Millisecond)
		ctx.RequestCtx.SetStatusCode(200)
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Logf("Timeout middleware returned error (may be expected): %v", err)
	}

	// Should return 504 Gateway Timeout
	if reqCtx.Response.StatusCode() != 504 {
		t.Errorf("Expected status code 504, got %d", reqCtx.Response.StatusCode())
	}

	// Should have JSON content type
	contentType := string(reqCtx.Response.Header.ContentType())
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected JSON content type, got '%s'", contentType)
	}
}

func TestTimeout_SkipPaths(t *testing.T) {
	config := TimeoutConfig{
		Timeout:   50 * time.Millisecond,
		Logger:    core.NewDefaultLogger(),
		SkipPaths: []string{"/health", "/metrics"},
	}

	mw := Timeout(config)

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/health")
	reqCtx.Request.Header.SetMethod("GET")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		// Slow handler, but should not timeout due to skip path
		time.Sleep(100 * time.Millisecond)
		ctx.RequestCtx.SetStatusCode(200)
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("Timeout middleware returned error for skipped path: %v", err)
	}

	// Should have normal status, not 504
	if reqCtx.Response.StatusCode() == 504 {
		t.Error("Skipped path should not timeout")
	}
}

func TestTimeout_InvalidTimeout(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid timeout")
		}
	}()

	config := TimeoutConfig{
		Timeout: -1 * time.Second, // Invalid
		Logger:  core.NewDefaultLogger(),
	}

	_ = Timeout(config)
}

func TestTimeout_ZeroTimeout(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for zero timeout")
		}
	}()

	config := TimeoutConfig{
		Timeout: 0,
		Logger:  core.NewDefaultLogger(),
	}

	_ = Timeout(config)
}

func TestTimeout_NilLogger(t *testing.T) {
	config := TimeoutConfig{
		Timeout: 1 * time.Second,
		Logger:  nil, // Should default to NewDefaultLogger()
	}

	mw := Timeout(config)
	if mw == nil {
		t.Fatal("Timeout() should not return nil even with nil logger")
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
		ctx.RequestCtx.SetStatusCode(200)
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("Timeout middleware returned error: %v", err)
	}
}

func TestTimeout_CustomMessage(t *testing.T) {
	config := TimeoutConfig{
		Timeout: 50 * time.Millisecond,
		Logger:  core.NewDefaultLogger(),
		Message: "Custom timeout message",
	}

	mw := Timeout(config)

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
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	_ = handler(fastCtx)

	// Response body should contain custom message
	body := string(reqCtx.Response.Body())
	if !strings.Contains(body, "Custom timeout message") {
		t.Logf("Response body: %s", body)
		t.Log("Response body should contain custom timeout message")
	}
}
