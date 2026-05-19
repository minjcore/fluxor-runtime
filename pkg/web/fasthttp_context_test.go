package web

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/valyala/fasthttp"
)

func TestFastRequestContext_RequestID(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Create a mock RequestCtx
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.Set("X-Request-ID", "test-request-id")

	fastCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
		requestID:          "test-request-id",
	}

	id := fastCtx.RequestID()
	if id != "test-request-id" {
		t.Errorf("RequestID() = %v, want test-request-id", id)
	}
}

func TestFastRequestContext_Context(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	fastCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         &fasthttp.RequestCtx{},
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
		requestID:          "test-request-id",
	}

	goCtx := fastCtx.Context()
	if goCtx == nil {
		t.Error("Context() should not return nil")
	}

	// Check that request ID is in context
	requestID := core.GetRequestID(goCtx)
	if requestID != "test-request-id" {
		t.Errorf("GetRequestID() from context = %v, want test-request-id", requestID)
	}
}

func TestFastRequestContext_SetGet(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	fastCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         &fasthttp.RequestCtx{},
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	fastCtx.Set("key1", "value1")
	fastCtx.Set("key2", 42)

	val1 := fastCtx.Get("key1")
	if val1 != "value1" {
		t.Errorf("Get(key1) = %v, want value1", val1)
	}

	val2 := fastCtx.Get("key2")
	if val2 != 42 {
		t.Errorf("Get(key2) = %v, want 42", val2)
	}

	val3 := fastCtx.Get("nonexistent")
	if val3 != nil {
		t.Errorf("Get(nonexistent) = %v, want nil", val3)
	}
}
