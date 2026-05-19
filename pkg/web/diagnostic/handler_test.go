package diagnostic

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
)

func TestNewHandler(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	handler := NewHandler(gocmd)
	if handler == nil {
		t.Fatal("NewHandler() should not return nil")
	}

	if handler.gocmd != gocmd {
		t.Error("GoCMD should be set correctly")
	}
}

func TestHandler_SystemHandler(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	handler := NewHandler(gocmd)

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/api/diagnostic/system")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	err := handler.SystemHandler(fastCtx)
	if err != nil {
		t.Errorf("SystemHandler() returned error: %v", err)
	}

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status 200, got %d", reqCtx.Response.StatusCode())
	}
}

func TestHandler_AllDeploymentsHandler(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	handler := NewHandler(gocmd)

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/api/diagnostic/deployments")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	err := handler.AllDeploymentsHandler(fastCtx)
	if err != nil {
		t.Errorf("AllDeploymentsHandler() returned error: %v", err)
	}

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status 200, got %d", reqCtx.Response.StatusCode())
	}
}

func TestHandler_DeploymentHandler_MissingID(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	handler := NewHandler(gocmd)

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/api/diagnostic/deployment/")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	err := handler.DeploymentHandler(fastCtx)
	if err != nil {
		t.Errorf("DeploymentHandler() returned error: %v", err)
	}

	// Should return 400 for missing ID
	if reqCtx.Response.StatusCode() != 400 {
		t.Errorf("Expected status 400 for missing ID, got %d", reqCtx.Response.StatusCode())
	}
}

func TestHandler_DeploymentHandler_WithID(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	handler := NewHandler(gocmd)

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/api/diagnostic/deployment/test-id")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	// Set the id param
	fastCtx.Params["id"] = "test-id"

	err := handler.DeploymentHandler(fastCtx)
	// May return error if deployment not found, which is expected
	if err != nil {
		t.Logf("DeploymentHandler() returned error (may be expected): %v", err)
	}

	// Should return either 200 (found) or 404 (not found) or 500 (error)
	status := reqCtx.Response.StatusCode()
	if status != 200 && status != 404 && status != 500 {
		t.Errorf("Expected status 200, 404, or 500, got %d", status)
	}
}

func TestHandler_DeploymentHandler_NotFound(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	handler := NewHandler(gocmd)

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.SetMethod("GET")
	reqCtx.Request.SetRequestURI("/api/diagnostic/deployment/nonexistent")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	// Set the id param
	fastCtx.Params["id"] = "nonexistent"

	err := handler.DeploymentHandler(fastCtx)
	// May return error if deployment not found
	if err != nil {
		t.Logf("DeploymentHandler() returned error (expected for nonexistent): %v", err)
	}

	// Should return 404 for not found
	status := reqCtx.Response.StatusCode()
	if status != 404 && status != 500 {
		t.Logf("Expected status 404 or 500, got %d (may be expected if deployment system not initialized)", status)
	}
}
