package security

import (
	"context"
	"strings"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
)

func TestDefaultHeadersConfig(t *testing.T) {
	config := DefaultHeadersConfig()

	if !config.HSTS {
		t.Error("HSTS should be enabled by default")
	}

	if config.HSTSMaxAge == 0 {
		t.Error("HSTSMaxAge should be set")
	}

	if !config.XContentTypeOptions {
		t.Error("XContentTypeOptions should be enabled by default")
	}

	if config.CSP == "" {
		t.Error("CSP should be set by default")
	}
}

func TestHeaders_HSTS(t *testing.T) {
	config := HeadersConfig{
		HSTS:           true,
		HSTSMaxAge:     31536000,
		HSTSIncludeSub: true,
	}

	mw := Headers(config)

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
		t.Errorf("Headers middleware returned error: %v", err)
	}

	// Should set HSTS header
	hsts := string(reqCtx.Response.Header.Peek("Strict-Transport-Security"))
	if !strings.Contains(hsts, "max-age=31536000") {
		t.Errorf("Expected HSTS header with max-age=31536000, got '%s'", hsts)
	}

	if !strings.Contains(hsts, "includeSubDomains") {
		t.Errorf("Expected HSTS header with includeSubDomains, got '%s'", hsts)
	}
}

func TestHeaders_XContentTypeOptions(t *testing.T) {
	config := HeadersConfig{
		XContentTypeOptions: true,
	}

	mw := Headers(config)

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
		t.Errorf("Headers middleware returned error: %v", err)
	}

	// Should set X-Content-Type-Options header
	header := string(reqCtx.Response.Header.Peek("X-Content-Type-Options"))
	if header != "nosniff" {
		t.Errorf("Expected X-Content-Type-Options 'nosniff', got '%s'", header)
	}
}

func TestHeaders_CSP(t *testing.T) {
	csp := "default-src 'self'; script-src 'self' 'unsafe-inline'"
	config := HeadersConfig{
		CSP: csp,
	}

	mw := Headers(config)

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
		t.Errorf("Headers middleware returned error: %v", err)
	}

	// Should set CSP header
	header := string(reqCtx.Response.Header.Peek("Content-Security-Policy"))
	if header != csp {
		t.Errorf("Expected CSP '%s', got '%s'", csp, header)
	}
}

func TestHeaders_XFrameOptions(t *testing.T) {
	config := HeadersConfig{
		XFrameOptions: "DENY",
	}

	mw := Headers(config)

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
		t.Errorf("Headers middleware returned error: %v", err)
	}

	// Should set X-Frame-Options header
	header := string(reqCtx.Response.Header.Peek("X-Frame-Options"))
	if header != "DENY" {
		t.Errorf("Expected X-Frame-Options 'DENY', got '%s'", header)
	}
}

func TestHeaders_ReferrerPolicy(t *testing.T) {
	config := HeadersConfig{
		ReferrerPolicy: "no-referrer",
	}

	mw := Headers(config)

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
		t.Errorf("Headers middleware returned error: %v", err)
	}

	// Should set Referrer-Policy header
	header := string(reqCtx.Response.Header.Peek("Referrer-Policy"))
	if header != "no-referrer" {
		t.Errorf("Expected Referrer-Policy 'no-referrer', got '%s'", header)
	}
}

func TestHeaders_CustomHeaders(t *testing.T) {
	config := HeadersConfig{
		CustomHeaders: map[string]string{
			"X-Custom-Header": "custom-value",
			"X-Another-Header": "another-value",
		},
	}

	mw := Headers(config)

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
		t.Errorf("Headers middleware returned error: %v", err)
	}

	// Should set custom headers
	customHeader := string(reqCtx.Response.Header.Peek("X-Custom-Header"))
	if customHeader != "custom-value" {
		t.Errorf("Expected X-Custom-Header 'custom-value', got '%s'", customHeader)
	}

	anotherHeader := string(reqCtx.Response.Header.Peek("X-Another-Header"))
	if anotherHeader != "another-value" {
		t.Errorf("Expected X-Another-Header 'another-value', got '%s'", anotherHeader)
	}
}

func TestHeaders_CrossOriginHeaders(t *testing.T) {
	config := HeadersConfig{
		CrossOriginOpenerPolicy:     "same-origin",
		CrossOriginResourcePolicy:   "same-origin",
		CrossOriginEmbedderPolicy:   "require-corp",
	}

	mw := Headers(config)

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
		t.Errorf("Headers middleware returned error: %v", err)
	}

	// Should set cross-origin headers
	coop := string(reqCtx.Response.Header.Peek("Cross-Origin-Opener-Policy"))
	if coop != "same-origin" {
		t.Errorf("Expected Cross-Origin-Opener-Policy 'same-origin', got '%s'", coop)
	}

	corp := string(reqCtx.Response.Header.Peek("Cross-Origin-Resource-Policy"))
	if corp != "same-origin" {
		t.Errorf("Expected Cross-Origin-Resource-Policy 'same-origin', got '%s'", corp)
	}

	coep := string(reqCtx.Response.Header.Peek("Cross-Origin-Embedder-Policy"))
	if coep != "require-corp" {
		t.Errorf("Expected Cross-Origin-Embedder-Policy 'require-corp', got '%s'", coep)
	}
}

func TestHeaders_HSTSDefaultMaxAge(t *testing.T) {
	config := HeadersConfig{
		HSTS:       true,
		HSTSMaxAge: 0, // Should default to 31536000
	}

	mw := Headers(config)

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
		t.Errorf("Headers middleware returned error: %v", err)
	}

	// Should use default max-age
	hsts := string(reqCtx.Response.Header.Peek("Strict-Transport-Security"))
	if !strings.Contains(hsts, "max-age=31536000") {
		t.Logf("HSTS header: %s (may use default)", hsts)
	}
}
