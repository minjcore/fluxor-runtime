package health

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
)

func TestNewAggregator(t *testing.T) {
	aggregator := NewAggregator(nil)
	if aggregator == nil {
		t.Fatal("NewAggregator() should not return nil")
	}

	// Test with custom registry
	registry := NewRegistry()
	aggregator2 := NewAggregator(registry)
	if aggregator2 == nil {
		t.Fatal("NewAggregator() should not return nil with custom registry")
	}
}

func TestHandler(t *testing.T) {
	handler := Handler()
	if handler == nil {
		t.Fatal("Handler() should not return nil")
	}

	// Test that handler can be called
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/health")
	reqCtx.Request.Header.SetMethod("GET")

	fastCtx := &web.FastRequestContext{
		RequestCtx: reqCtx,
		Params:     make(map[string]string),
	}

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("Handler() returned error: %v", err)
	}

	// Should return 200 or 503 depending on health status
	statusCode := reqCtx.Response.StatusCode()
	if statusCode != 200 && statusCode != 503 {
		t.Errorf("Expected status code 200 or 503, got %d", statusCode)
	}
}

func TestReadyHandler(t *testing.T) {
	handler := ReadyHandler()
	if handler == nil {
		t.Fatal("ReadyHandler() should not return nil")
	}

	// Test that handler can be called
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/ready")
	reqCtx.Request.Header.SetMethod("GET")

	fastCtx := &web.FastRequestContext{
		RequestCtx: reqCtx,
		Params:     make(map[string]string),
	}

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("ReadyHandler() returned error: %v", err)
	}

	// Should return 200 or 503 depending on health status
	statusCode := reqCtx.Response.StatusCode()
	if statusCode != 200 && statusCode != 503 {
		t.Errorf("Expected status code 200 or 503, got %d", statusCode)
	}
}

func TestAggregator_HandleHealth(t *testing.T) {
	registry := NewRegistry()
	aggregator := NewAggregator(registry)

	// Register a passing health check
	registry.Register("test", func(ctx context.Context) error {
		return nil
	})

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/health")
	reqCtx.Request.Header.SetMethod("GET")

	fastCtx := &web.FastRequestContext{
		RequestCtx: reqCtx,
		Params:     make(map[string]string),
	}

	err := aggregator.HandleHealth(fastCtx)
	if err != nil {
		t.Errorf("HandleHealth() returned error: %v", err)
	}

	// Should return 200 when all checks pass
	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status code 200, got %d", reqCtx.Response.StatusCode())
	}

	// Should have JSON content type
	contentType := string(reqCtx.Response.Header.ContentType())
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected JSON content type, got '%s'", contentType)
	}

	// Parse response body
	var response HealthResponse
	body := reqCtx.Response.Body()
	if err := json.Unmarshal(body, &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Status != string(StatusUp) {
		t.Errorf("Expected status 'UP', got '%s'", response.Status)
	}
}

func TestAggregator_HandleHealth_WithFailingCheck(t *testing.T) {
	registry := NewRegistry()
	aggregator := NewAggregator(registry)

	// Register a failing health check
	registry.Register("failing", func(ctx context.Context) error {
		return &Error{Message: "test failure"}
	})

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/health")
	reqCtx.Request.Header.SetMethod("GET")

	fastCtx := &web.FastRequestContext{
		RequestCtx: reqCtx,
		Params:     make(map[string]string),
	}

	err := aggregator.HandleHealth(fastCtx)
	if err != nil {
		t.Errorf("HandleHealth() returned error: %v", err)
	}

	// Should return 503 when any check fails
	if reqCtx.Response.StatusCode() != 503 {
		t.Errorf("Expected status code 503, got %d", reqCtx.Response.StatusCode())
	}

	// Parse response body
	var response HealthResponse
	body := reqCtx.Response.Body()
	if err := json.Unmarshal(body, &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response.Status != string(StatusDown) {
		t.Errorf("Expected status 'DOWN', got '%s'", response.Status)
	}
}

func TestAggregator_HandleReady(t *testing.T) {
	registry := NewRegistry()
	aggregator := NewAggregator(registry)

	// Register a passing health check
	registry.Register("test", func(ctx context.Context) error {
		return nil
	})

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/ready")
	reqCtx.Request.Header.SetMethod("GET")

	fastCtx := &web.FastRequestContext{
		RequestCtx: reqCtx,
		Params:     make(map[string]string),
	}

	err := aggregator.HandleReady(fastCtx)
	if err != nil {
		t.Errorf("HandleReady() returned error: %v", err)
	}

	// Should return 200 when all checks pass
	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status code 200, got %d", reqCtx.Response.StatusCode())
	}
}

func TestAggregator_GetHealthStatus(t *testing.T) {
	registry := NewRegistry()
	aggregator := NewAggregator(registry)

	// Register a passing health check
	registry.Register("test", func(ctx context.Context) error {
		return nil
	})

	status, checks := aggregator.GetHealthStatus(context.Background())
	if status != StatusUp {
		t.Errorf("Expected status UP, got %s", status)
	}

	if len(checks) != 1 {
		t.Errorf("Expected 1 check, got %d", len(checks))
	}

	if checks["test"].Status != StatusUp {
		t.Errorf("Expected check status UP, got %s", checks["test"].Status)
	}
}

func TestFormatHealthResponse(t *testing.T) {
	checks := map[string]CheckResult{
		"test": {
			Status: StatusUp,
		},
	}

	data, err := FormatHealthResponse(StatusUp, checks, "test-request-id")
	if err != nil {
		t.Errorf("FormatHealthResponse() returned error: %v", err)
	}

	if len(data) == 0 {
		t.Error("FormatHealthResponse() should return non-empty data")
	}

	// Verify it's valid JSON
	var response HealthResponse
	if err := json.Unmarshal(data, &response); err != nil {
		t.Errorf("FormatHealthResponse() returned invalid JSON: %v", err)
	}

	if response.Status != string(StatusUp) {
		t.Errorf("Expected status 'UP', got '%s'", response.Status)
	}

	if response.RequestID != "test-request-id" {
		t.Errorf("Expected request ID 'test-request-id', got '%s'", response.RequestID)
	}
}

func TestFormatHealthResponseString(t *testing.T) {
	checks := map[string]CheckResult{
		"test": {
			Status: StatusUp,
		},
	}

	str, err := FormatHealthResponseString(StatusUp, checks, "test-request-id")
	if err != nil {
		t.Errorf("FormatHealthResponseString() returned error: %v", err)
	}

	if len(str) == 0 {
		t.Error("FormatHealthResponseString() should return non-empty string")
	}

	// Verify it's valid JSON
	var response HealthResponse
	if err := json.Unmarshal([]byte(str), &response); err != nil {
		t.Errorf("FormatHealthResponseString() returned invalid JSON: %v", err)
	}
}
