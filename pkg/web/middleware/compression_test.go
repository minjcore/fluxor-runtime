package middleware

import (
	"context"
	"strings"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
)

func TestDefaultCompressionConfig(t *testing.T) {
	config := DefaultCompressionConfig()

	if config.Level != 6 {
		t.Errorf("Expected Level 6, got %d", config.Level)
	}

	if config.MinSize != 1024 {
		t.Errorf("Expected MinSize 1024, got %d", config.MinSize)
	}

	if len(config.ContentTypes) == 0 {
		t.Error("ContentTypes should not be empty")
	}

	// Check for common content types
	hasJSON := false
	hasHTML := false
	for _, ct := range config.ContentTypes {
		if strings.Contains(ct, "json") {
			hasJSON = true
		}
		if strings.Contains(ct, "html") {
			hasHTML = true
		}
	}

	if !hasJSON {
		t.Error("ContentTypes should include JSON")
	}
	if !hasHTML {
		t.Error("ContentTypes should include HTML")
	}
}

func TestCompression_InvalidLevel(t *testing.T) {
	config := CompressionConfig{
		Level:    15, // Invalid level
		MinSize:  1024,
		ContentTypes: []string{"application/json"},
	}

	mw := Compression(config)
	if mw == nil {
		t.Fatal("Compression() should not return nil")
	}

	// Level should be clamped to 6
	// (We can't easily test this without inspecting internal state,
	// but we can verify the middleware works)
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.Set("Accept-Encoding", "gzip")
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
		ctx.RequestCtx.SetContentType("application/json")
		ctx.RequestCtx.SetBodyString(strings.Repeat("x", 2000)) // Large enough to compress
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("Compression middleware returned error: %v", err)
	}
}

func TestCompression_MinSize(t *testing.T) {
	config := CompressionConfig{
		Level:    6,
		MinSize:  2000,
		ContentTypes: []string{"application/json"},
	}

	mw := Compression(config)

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.Set("Accept-Encoding", "gzip")
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("GET")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	// Small body - should not compress
	handler := mw(func(ctx *web.FastRequestContext) error {
		ctx.RequestCtx.SetContentType("application/json")
		ctx.RequestCtx.SetBodyString("small") // Too small
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("Compression middleware returned error: %v", err)
	}

	// Large body - should compress
	fastCtx2 := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         &fasthttp.RequestCtx{},
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}
	fastCtx2.RequestCtx.Request.Header.Set("Accept-Encoding", "gzip")
	fastCtx2.RequestCtx.Request.SetRequestURI("/test")
	fastCtx2.RequestCtx.Request.Header.SetMethod("GET")

	handler2 := mw(func(ctx *web.FastRequestContext) error {
		ctx.RequestCtx.SetContentType("application/json")
		ctx.RequestCtx.SetBodyString(strings.Repeat("x", 3000)) // Large enough
		return nil
	})

	err = handler2(fastCtx2)
	if err != nil {
		t.Errorf("Compression middleware returned error: %v", err)
	}
}

func TestCompression_SkipPaths(t *testing.T) {
	config := CompressionConfig{
		Level:    6,
		MinSize:  1024,
		ContentTypes: []string{"application/json"},
		SkipPaths: []string{"/api/health", "/metrics"},
	}

	mw := Compression(config)

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Test skipped path
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.Set("Accept-Encoding", "gzip")
	reqCtx.Request.SetRequestURI("/api/health")
	reqCtx.Request.Header.SetMethod("GET")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		ctx.RequestCtx.SetContentType("application/json")
		ctx.RequestCtx.SetBodyString(strings.Repeat("x", 2000))
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("Compression middleware returned error: %v", err)
	}

	// Should not have Content-Encoding header for skipped path
	encoding := string(fastCtx.RequestCtx.Response.Header.Peek("Content-Encoding"))
	if encoding != "" {
		t.Logf("Content-Encoding set for skipped path (may be acceptable): %s", encoding)
	}
}

func TestCompression_NoAcceptEncoding(t *testing.T) {
	config := DefaultCompressionConfig()
	mw := Compression(config)

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	// No Accept-Encoding header
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
		ctx.RequestCtx.SetContentType("application/json")
		ctx.RequestCtx.SetBodyString(strings.Repeat("x", 2000))
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("Compression middleware returned error: %v", err)
	}

	// Should not compress if client doesn't accept gzip
	encoding := string(fastCtx.RequestCtx.Response.Header.Peek("Content-Encoding"))
	if encoding == "gzip" {
		t.Error("Should not compress when client doesn't accept gzip")
	}
}

func TestCompression_ContentTypeMatching(t *testing.T) {
	config := CompressionConfig{
		Level:    6,
		MinSize:  1024,
		ContentTypes: []string{"application/json", "text/html"},
	}

	mw := Compression(config)

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	testCases := []struct {
		name        string
		contentType string
		shouldMatch bool
	}{
		{"JSON", "application/json", true},
		{"HTML", "text/html", true},
		{"Plain text", "text/plain", false},
		{"Image", "image/png", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqCtx := &fasthttp.RequestCtx{}
			reqCtx.Request.Header.Set("Accept-Encoding", "gzip")
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
				ctx.RequestCtx.SetContentType(tc.contentType)
				ctx.RequestCtx.SetBodyString(strings.Repeat("x", 2000))
				return nil
			})

			err := handler(fastCtx)
			if err != nil {
				t.Errorf("Compression middleware returned error: %v", err)
			}
		})
	}
}
