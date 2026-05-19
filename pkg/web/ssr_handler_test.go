package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/valyala/fasthttp"
)

func TestNewSSRHandler(t *testing.T) {
	handler := NewSSRHandler("http://localhost:3001")
	if handler == nil {
		t.Fatal("NewSSRHandler() should not return nil")
	}

	if handler.ssrServerURL != "http://localhost:3001" {
		t.Errorf("Expected ssrServerURL 'http://localhost:3001', got '%s'", handler.ssrServerURL)
	}

	if handler.timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", handler.timeout)
	}

	if handler.httpClient == nil {
		t.Error("HTTP client should not be nil")
	}
}

func TestNewSSRHandler_EmptyURL(t *testing.T) {
	handler := NewSSRHandler("")
	if handler == nil {
		t.Fatal("NewSSRHandler() should not return nil")
	}

	// Should default to localhost:3001
	if handler.ssrServerURL != "http://localhost:3001" {
		t.Errorf("Expected default ssrServerURL 'http://localhost:3001', got '%s'", handler.ssrServerURL)
	}
}

func TestSSRHandler_SetTimeout(t *testing.T) {
	handler := NewSSRHandler("http://localhost:3001")
	
	newTimeout := 5 * time.Second
	handler.SetTimeout(newTimeout)

	if handler.timeout != newTimeout {
		t.Errorf("Expected timeout %v, got %v", newTimeout, handler.timeout)
	}

	if handler.httpClient.Timeout != newTimeout {
		t.Errorf("Expected HTTP client timeout %v, got %v", newTimeout, handler.httpClient.Timeout)
	}
}

func TestSSRHandler_ProxyHandler(t *testing.T) {
	handler := NewSSRHandler("http://localhost:3001")
	proxyHandler := handler.ProxyHandler()

	if proxyHandler == nil {
		t.Error("ProxyHandler() should not return nil")
	}
}

func TestSSRHandler_HandleWithContext(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	handler := NewSSRHandler("http://localhost:3001")
	fluxorCtx := newTestFluxorContext(gocmd)

	handlerFunc := handler.HandleWithContext(fluxorCtx)
	if handlerFunc == nil {
		t.Error("HandleWithContext() should not return nil")
	}
}

func TestSSRHandler_Handle_RequestCreation(t *testing.T) {
	// Create a mock SSR server
	ssrServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("SSR Response"))
	}))
	defer ssrServer.Close()

	handler := NewSSRHandler(ssrServer.URL)
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Create FastHTTP request context
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/test")

	fastCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	err := handler.Handle(fastCtx)
	if err != nil {
		t.Errorf("Handle() returned error: %v", err)
	}

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status 200, got %d", reqCtx.Response.StatusCode())
	}

	body := string(reqCtx.Response.Body())
	if body != "SSR Response" {
		t.Errorf("Expected body 'SSR Response', got '%s'", body)
	}
}

func TestSSRHandler_Handle_HeadersPreserved(t *testing.T) {
	// Create a mock SSR server that checks headers
	ssrServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that custom header is preserved
		if r.Header.Get("X-Custom-Header") != "test-value" {
			w.WriteHeader(400)
			return
		}
		w.WriteHeader(200)
	}))
	defer ssrServer.Close()

	handler := NewSSRHandler(ssrServer.URL)
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.Set("X-Custom-Header", "test-value")

	fastCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	err := handler.Handle(fastCtx)
	if err != nil {
		t.Errorf("Handle() returned error: %v", err)
	}

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status 200, got %d", reqCtx.Response.StatusCode())
	}
}

func TestSSRHandler_Handle_SSRServerError(t *testing.T) {
	// Handler with invalid URL (server doesn't exist)
	handler := NewSSRHandler("http://localhost:99999")
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/test")

	fastCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	// Set a short timeout for faster test
	handler.SetTimeout(100 * time.Millisecond)

	err := handler.Handle(fastCtx)
	if err == nil {
		t.Error("Handle() should return error when SSR server is unavailable")
	}

	// Should return 502 Bad Gateway
	if reqCtx.Response.StatusCode() != 502 {
		t.Errorf("Expected status 502, got %d", reqCtx.Response.StatusCode())
	}
}

func TestSSRHandler_Handle_ResponseHeaders(t *testing.T) {
	// Create a mock SSR server that sets custom headers
	ssrServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-SSR-Header", "ssr-value")
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		w.Write([]byte("SSR Response"))
	}))
	defer ssrServer.Close()

	handler := NewSSRHandler(ssrServer.URL)
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/test")

	fastCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	err := handler.Handle(fastCtx)
	if err != nil {
		t.Errorf("Handle() returned error: %v", err)
	}

	// Check that headers are copied
	if string(reqCtx.Response.Header.Peek("X-SSR-Header")) != "ssr-value" {
		t.Error("Response header should be copied from SSR server")
	}
}

func TestSSRHandler_Handle_POSTRequest(t *testing.T) {
	// Create a mock SSR server that accepts POST
	ssrServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(405)
			return
		}
		body := make([]byte, 100)
		n, _ := r.Body.Read(body)
		w.WriteHeader(200)
		w.Write([]byte("Received: " + string(body[:n])))
	}))
	defer ssrServer.Close()

	handler := NewSSRHandler(ssrServer.URL)
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("POST")
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.SetBodyString("test body")

	fastCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	err := handler.Handle(fastCtx)
	if err != nil {
		t.Errorf("Handle() returned error: %v", err)
	}

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status 200, got %d", reqCtx.Response.StatusCode())
	}
}
