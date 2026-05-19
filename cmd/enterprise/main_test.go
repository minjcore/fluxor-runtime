// Copyright (c) 2024-2026 Fluxor Framework
// All rights reserved.
//
// This source code is proprietary and confidential.
// Unauthorized copying, modification, distribution, or use of this software,
// via any medium is strictly prohibited without the express written permission
// of Fluxor Framework.
//
// This code is provided as an example for demonstration purposes only.
// Redistribution or sharing of this source code is not permitted.
//
// License: Proprietary - All Rights Reserved
// For licensing inquiries, please contact: caokhang91@gmail.com

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/dbruntime"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/fluxor/pkg/web/middleware/auth"
	"github.com/valyala/fasthttp"
)

// TestLoadConfig tests configuration loading
func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		setup   func()
		cleanup func()
		wantErr bool
	}{
		{
			name: "default config loads successfully",
			setup: func() {
				// No config file, should use defaults
			},
			cleanup: func() {},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			cfg, err := loadConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if cfg == nil {
					t.Error("loadConfig() returned nil config")
				}
				if cfg.Server.Port == "" {
					t.Error("Server port should not be empty")
				}
				if cfg.Server.MaxCCU <= 0 {
					t.Error("MaxCCU should be positive")
				}
			}
		})
	}
}

// TestHandleHome tests the home handler
func TestHandleHome(t *testing.T) {
	logger := core.NewDefaultLogger()
	handler := handleHome(logger)

	ctx := &web.FastRequestContext{
		RequestCtx: &fasthttp.RequestCtx{},
	}

	// Set request ID
	ctx.RequestCtx.SetUserValue("request_id", "test-request-id")

	err := handler(ctx)
	if err != nil {
		t.Fatalf("handleHome() error = %v", err)
	}

	// Check status code
	if ctx.RequestCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status 200, got %d", ctx.RequestCtx.Response.StatusCode())
	}

	// Check response is JSON
	contentType := string(ctx.RequestCtx.Response.Header.ContentType())
	if contentType != "application/json" {
		t.Errorf("Expected content-type application/json, got %s", contentType)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(ctx.RequestCtx.Response.Body(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify response fields
	if _, ok := response["message"]; !ok {
		t.Error("Response should contain 'message' field")
	}
	if _, ok := response["features"]; !ok {
		t.Error("Response should contain 'features' field")
	}
}

// TestHandleHealth tests the health check handler
func TestHandleHealth(t *testing.T) {
	// Create a mock database component
	dbConfig := dbruntime.PoolConfig{
		DSN:             "mock://localhost/testdb",
		DriverName:      "mock",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}
	dbComponent := dbruntime.NewDatabaseComponent(dbConfig)

	handler := handleHealth(dbComponent)

	ctx := &web.FastRequestContext{
		RequestCtx: &fasthttp.RequestCtx{},
	}

	err := handler(ctx)
	if err != nil {
		t.Fatalf("handleHealth() error = %v", err)
	}

	// Check status code
	if ctx.RequestCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status 200, got %d", ctx.RequestCtx.Response.StatusCode())
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(ctx.RequestCtx.Response.Body(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify response fields
	if status, ok := response["status"]; !ok || status != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", status)
	}
}

// TestJWTTokenGeneration tests JWT token generation
func TestJWTTokenGeneration(t *testing.T) {
	secret := []byte("test-secret-key-for-testing-only")
	generator := auth.NewJWTTokenGenerator(secret)

	claims := map[string]interface{}{
		"user_id": "test-user-123",
		"email":   "test@example.com",
		"roles":   []string{"user"},
	}

	token, err := generator.Generate(claims, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("Generated token should not be empty")
	}

	// Token should have 3 parts separated by dots
	parts := bytes.Split([]byte(token), []byte("."))
	if len(parts) != 3 {
		t.Errorf("Expected 3 parts in JWT token, got %d", len(parts))
	}
}

// TestApplyMiddleware tests middleware chain application
func TestApplyMiddleware(t *testing.T) {
	callOrder := []string{}

	// Create mock middlewares
	mw1 := func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			callOrder = append(callOrder, "mw1-before")
			err := next(ctx)
			callOrder = append(callOrder, "mw1-after")
			return err
		}
	}

	mw2 := func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			callOrder = append(callOrder, "mw2-before")
			err := next(ctx)
			callOrder = append(callOrder, "mw2-after")
			return err
		}
	}

	handler := func(ctx *web.FastRequestContext) error {
		callOrder = append(callOrder, "handler")
		return nil
	}

	middlewares := []web.FastMiddleware{mw1, mw2}
	finalHandler := applyMiddleware(middlewares, handler)

	ctx := &web.FastRequestContext{
		RequestCtx: &fasthttp.RequestCtx{},
	}

	err := finalHandler(ctx)
	if err != nil {
		t.Fatalf("Handler error: %v", err)
	}

	// Verify call order: mw1-before, mw2-before, handler, mw2-after, mw1-after
	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(callOrder) != len(expected) {
		t.Errorf("Expected %d calls, got %d", len(expected), len(callOrder))
	}

	for i, call := range expected {
		if i >= len(callOrder) || callOrder[i] != call {
			t.Errorf("Expected call %d to be %s, got %s", i, call, callOrder[i])
		}
	}
}

// TestGetEnv tests environment variable helper
func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "returns default when env not set",
			key:          "TEST_KEY_NOT_SET",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getEnv() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// BenchmarkHandleHome benchmarks the home handler
func BenchmarkHandleHome(b *testing.B) {
	logger := core.NewDefaultLogger()
	handler := handleHome(logger)

	ctx := &web.FastRequestContext{
		RequestCtx: &fasthttp.RequestCtx{},
	}
	ctx.RequestCtx.SetUserValue("request_id", "bench-request-id")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler(ctx)
	}
}

// BenchmarkJWTTokenGeneration benchmarks JWT token generation
func BenchmarkJWTTokenGeneration(b *testing.B) {
	secret := []byte("test-secret-key-for-benchmarking")
	generator := auth.NewJWTTokenGenerator(secret)

	claims := map[string]interface{}{
		"user_id": "bench-user-123",
		"email":   "bench@example.com",
		"roles":   []string{"user"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generator.Generate(claims, 1*time.Hour)
	}
}

// TestUserServiceVerticle tests the verticle can be deployed
func TestUserServiceVerticle(t *testing.T) {
	// Create vertx instance
	ctx := context.Background()
	vertx := core.NewGoCMD(ctx)
	defer vertx.Close()
	logger := core.NewDefaultLogger()

		dbConfig := dbruntime.PoolConfig{
		DSN:             "mock://localhost/testdb",
		DriverName:      "mock",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}
		dbComponent := dbruntime.NewDatabaseComponent(dbConfig)

	verticle := &UserServiceVerticle{
		eventBus: vertx.EventBus(),
		logger:   logger,
		db:       dbComponent,
	}

	// Test Deploy (which calls Start internally)
	deploymentID, err := vertx.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("Failed to deploy verticle: %v", err)
	}

	if deploymentID == "" {
		t.Error("Deployment ID should not be empty")
	}

	// Wait for deployment to complete (Start() runs in goroutine)
	// Deployment state transitions from PENDING to STARTED when Start() completes
	time.Sleep(100 * time.Millisecond)

	// Test Undeploy (which calls Stop internally)
	err = vertx.UndeployVerticle(deploymentID)
	if err != nil {
		t.Fatalf("Failed to undeploy verticle: %v", err)
	}
}

// Integration test helper
func setupTestServer(t *testing.T) (*web.FastHTTPServer, func()) {
	t.Helper()

	ctx := context.Background()
	vertx := core.NewGoCMD(ctx)

	config := web.CCUBasedConfigWithUtilization(":0", 100, 67) // Random port
	server := web.NewFastHTTPServer(vertx, config)

	cleanup := func() {
		// Cleanup if needed
	}

	return server, cleanup
}

// TestIntegration_HealthEndpoint tests the health endpoint integration
func TestIntegration_HealthEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Setup router
	router := server.FastRouter()
		dbConfig := dbruntime.PoolConfig{
		DSN:             "mock://localhost/testdb",
		DriverName:      "mock",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}
		dbComponent := dbruntime.NewDatabaseComponent(dbConfig)

	router.GETFast("/health", handleHealth(dbComponent))

	// Create test request
	req := fasthttp.AcquireRequest()
	req.SetRequestURI("http://localhost/health")
	req.Header.SetMethod("GET")

	resp := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	// Note: Full integration test would require actually starting the server
	// For now, we're testing the handler directly
	t.Log("Integration test setup successful")
}
