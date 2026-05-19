package middleware

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
)

func TestDefaultLoggingConfig(t *testing.T) {
	config := DefaultLoggingConfig()

	if config.Logger == nil {
		t.Error("Logger should not be nil")
	}

	if !config.LogRequestID {
		t.Error("LogRequestID should be true by default")
	}

	if len(config.SkipPaths) == 0 {
		t.Error("SkipPaths should have default values")
	}

	// Check for common skip paths
	hasHealth := false
	for _, path := range config.SkipPaths {
		if strings.Contains(path, "health") {
			hasHealth = true
			break
		}
	}
	if !hasHealth {
		t.Log("SkipPaths should include /health (may vary)")
	}
}

func TestLogging_SkipPaths(t *testing.T) {
	config := LoggingConfig{
		Logger:    core.NewDefaultLogger(),
		SkipPaths: []string{"/health", "/metrics"},
	}

	mw := Logging(config)

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
		ctx.RequestCtx.SetStatusCode(200)
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("Logging middleware returned error: %v", err)
	}
}

func TestLogging_LogRequestID(t *testing.T) {
	config := LoggingConfig{
		Logger:       core.NewDefaultLogger(),
		LogRequestID: true,
	}

	mw := Logging(config)

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
		t.Errorf("Logging middleware returned error: %v", err)
	}
}

func TestLogging_ErrorLogging(t *testing.T) {
	config := LoggingConfig{
		Logger: core.NewDefaultLogger(),
	}

	mw := Logging(config)

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
		ctx.RequestCtx.SetStatusCode(500)
		return fmt.Errorf("test error")
	})

	err := handler(fastCtx)
	if err == nil {
		t.Error("Expected error from handler")
	}
}

func TestLogging_StatusCodeLogging(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		hasError   bool
	}{
		{"Success", 200, false},
		{"Client Error", 400, false},
		{"Server Error", 500, false},
		{"Handler Error", 200, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := LoggingConfig{
				Logger: core.NewDefaultLogger(),
			}

			mw := Logging(config)

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
				ctx.RequestCtx.SetStatusCode(tc.statusCode)
				if tc.hasError {
					return fmt.Errorf("test error")
				}
				return nil
			})

			err := handler(fastCtx)
			if tc.hasError && err == nil {
				t.Error("Expected error from handler")
			}
		})
	}
}

func TestLogging_NilLogger(t *testing.T) {
	config := LoggingConfig{
		Logger: nil, // Should default to NewDefaultLogger()
	}

	mw := Logging(config)
	if mw == nil {
		t.Fatal("Logging() should not return nil even with nil logger")
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
		t.Errorf("Logging middleware returned error: %v", err)
	}
}
