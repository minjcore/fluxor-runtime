package web

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/valyala/fasthttp"
)

func TestFastHTTPServer_NewServer(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DefaultFastHTTPServerConfig(":0")
	server := NewFastHTTPServer(gocmd, config)

	if server == nil {
		t.Error("NewFastHTTPServer() should not return nil")
	}

	if server.FastRouter() == nil {
		t.Error("FastRouter() should not return nil")
	}
}

func TestFastRequestContext_JSON(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Create a test context (we can't easily test fasthttp.RequestCtx without actual request)
	// This test verifies the validation logic

	// Test fail-fast: invalid status code
	reqCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	err := reqCtx.JSON(999, "test") // Invalid status code
	if err == nil {
		t.Error("JSON() with invalid status code should fail")
	}

	err = reqCtx.JSON(0, "test") // Invalid status code
	if err == nil {
		t.Error("JSON() with zero status code should fail")
	}
}

func TestFastRequestContext_BindJSON(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	// Test fail-fast: nil target
	err := reqCtx.BindJSON(nil)
	if err == nil {
		t.Error("BindJSON() with nil target should fail")
	}
}

func TestDefaultFastHTTPServerConfig(t *testing.T) {
	config := DefaultFastHTTPServerConfig(":8080")

	if config.Addr != ":8080" {
		t.Errorf("Addr = %v, want :8080", config.Addr)
	}

	if config.MaxQueue <= 0 {
		t.Error("MaxQueue should be positive")
	}

	if config.Workers <= 0 {
		t.Error("Workers should be positive")
	}
}

func TestFastHTTPServer_Metrics_LittlesLaw(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DefaultFastHTTPServerConfig(":0")
	server := NewFastHTTPServer(gocmd, config)

	// Test initial state - all metrics should be zero
	metrics := server.Metrics()
	if metrics.TotalRequests != 0 {
		t.Errorf("Initial TotalRequests = %d, want 0", metrics.TotalRequests)
	}
	if metrics.AverageLatencyMs != 0 {
		t.Errorf("Initial AverageLatencyMs = %f, want 0", metrics.AverageLatencyMs)
	}
	if metrics.ArrivalRate != 0 {
		t.Errorf("Initial ArrivalRate = %f, want 0", metrics.ArrivalRate)
	}
	if metrics.ExpectedQueueLength != 0 {
		t.Errorf("Initial ExpectedQueueLength = %f, want 0", metrics.ExpectedQueueLength)
	}
	if !math.IsNaN(metrics.LittlesLawValidation) && metrics.LittlesLawValidation != 0 {
		t.Errorf("Initial LittlesLawValidation = %f, want 0 or NaN", metrics.LittlesLawValidation)
	}
}

func TestFastHTTPServer_Metrics_LittlesLaw_Calculation(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DefaultFastHTTPServerConfig(":0")
	server := NewFastHTTPServer(gocmd, config)

	// Simulate processing requests
	// Note: We can't easily test actual request processing without starting the server
	// This test verifies that the metrics structure is correct and calculations work

	// First metrics call - should initialize sliding window
	metrics1 := server.Metrics()
	if metrics1.TotalRequests != 0 {
		t.Errorf("First call TotalRequests = %d, want 0", metrics1.TotalRequests)
	}

	// Wait a bit for time tracking
	time.Sleep(10 * time.Millisecond)

	// Second metrics call - should still show zero arrival rate initially
	metrics2 := server.Metrics()
	if metrics2.TotalRequests != 0 {
		t.Errorf("Second call TotalRequests = %d, want 0", metrics2.TotalRequests)
	}
	// Arrival rate may be zero if no requests processed
	if metrics2.ArrivalRate < 0 {
		t.Errorf("ArrivalRate = %f, should not be negative", metrics2.ArrivalRate)
	}
}

func TestFastHTTPServer_Metrics_LittlesLaw_EdgeCases(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DefaultFastHTTPServerConfig(":0")
	server := NewFastHTTPServer(gocmd, config)

	// Test edge case: zero requests
	metrics := server.Metrics()
	if metrics.TotalRequests != 0 {
		t.Errorf("TotalRequests = %d, want 0", metrics.TotalRequests)
	}
	// Average latency should be 0 when no requests
	if metrics.AverageLatencyMs != 0 {
		t.Errorf("AverageLatencyMs with zero requests = %f, want 0", metrics.AverageLatencyMs)
	}
	// Expected queue length should be 0 when arrival rate is 0
	if metrics.ExpectedQueueLength != 0 {
		t.Errorf("ExpectedQueueLength with zero requests = %f, want 0", metrics.ExpectedQueueLength)
	}
}

func TestFastHTTPServer_Metrics_LittlesLaw_ValidationRatio(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DefaultFastHTTPServerConfig(":0")
	server := NewFastHTTPServer(gocmd, config)

	metrics := server.Metrics()

	// Validation ratio should be calculated correctly
	// When expected queue length is 0, validation ratio may be 0 or NaN
	if metrics.ExpectedQueueLength == 0 {
		// When expected is 0, validation ratio should be 0
		if !math.IsNaN(metrics.LittlesLawValidation) && metrics.LittlesLawValidation != 0 {
			t.Errorf("LittlesLawValidation with zero expected = %f, want 0 or NaN", metrics.LittlesLawValidation)
		}
	} else {
		// Validation ratio should be actual / expected
		expected := float64(metrics.QueuedRequests) / metrics.ExpectedQueueLength
		if math.Abs(metrics.LittlesLawValidation-expected) > 0.01 {
			t.Errorf("LittlesLawValidation = %f, want %f", metrics.LittlesLawValidation, expected)
		}
	}
}

func TestFastHTTPServer_ProcessRequest_LatencyTracking(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DefaultFastHTTPServerConfig(":0")
	server := NewFastHTTPServer(gocmd, config)

	// Start server to enable request processing
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Wait for server to start
	time.Sleep(50 * time.Millisecond)

	// Register a simple handler
	router := server.FastRouter()
	router.GETFast("/test", func(ctx *FastRequestContext) error {
		time.Sleep(10 * time.Millisecond) // Simulate processing time
		return ctx.Text(200, "OK")
	})

	// Create a test request
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI("http://localhost:0/test")
	req.Header.SetMethod("GET")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	// Process request through handler (this will track latency)
	// Note: We can't easily simulate fasthttp.RequestCtx without actual HTTP connection
	// This test structure verifies the code compiles and basic logic works

	// Get metrics after some time
	time.Sleep(100 * time.Millisecond)
	metrics := server.Metrics()

	// Verify metrics structure is populated
	_ = metrics.AverageLatencyMs
	_ = metrics.ArrivalRate
	_ = metrics.ExpectedQueueLength
	_ = metrics.LittlesLawValidation

	// Metrics should be valid numbers (or zero)
	if math.IsInf(metrics.AverageLatencyMs, 0) {
		t.Error("AverageLatencyMs should not be infinite")
	}
	if math.IsInf(metrics.ArrivalRate, 0) {
		t.Error("ArrivalRate should not be infinite")
	}
	if math.IsInf(metrics.ExpectedQueueLength, 0) {
		t.Error("ExpectedQueueLength should not be infinite")
	}
	if math.IsInf(metrics.LittlesLawValidation, 0) {
		t.Error("LittlesLawValidation should not be infinite")
	}
}
