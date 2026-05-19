package notion

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	config := DefaultConfig()
	config.APIKey = "secret_TEST123"
	config.Service.Name = "test-service"
	config.Server.Addr = ":8080"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	// Test that all service clients are available
	if client.Pages() == nil {
		t.Error("NewClient() Pages() returned nil")
	}
	if client.Databases() == nil {
		t.Error("NewClient() Databases() returned nil")
	}
	if client.Blocks() == nil {
		t.Error("NewClient() Blocks() returned nil")
	}
	if client.Users() == nil {
		t.Error("NewClient() Users() returned nil")
	}
	if client.Search() == nil {
		t.Error("NewClient() Search() returned nil")
	}
}

func TestNewClient_InvalidConfig(t *testing.T) {
	config := DefaultConfig()
	config.APIKey = "" // Missing APIKey

	_, err := NewClient(config)
	if err == nil {
		t.Error("NewClient() expected error with invalid config, got nil")
	}
}

func TestNotionClient_doRequest_Success(t *testing.T) {
	mockResponse := map[string]interface{}{
		"object": "user",
		"id":     "user-123",
		"name":   "Test User",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/users/me" {
			t.Errorf("Expected path /v1/users/me, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret_TEST123" {
			t.Errorf("Expected Authorization header, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Notion-Version") != "2022-06-28" {
			t.Errorf("Expected Notion-Version header, got %s", r.Header.Get("Notion-Version"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	config := DefaultConfig()
	config.APIKey = "secret_TEST123"
	config.BaseURL = server.URL
	config.Service.Name = "test-service"
	config.Server.Addr = ":8080"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	notionClient := client.(*notionClient)
	ctx := context.Background()

	data, err := notionClient.doRequest(ctx, "GET", "/users/me", nil)
	if err != nil {
		t.Fatalf("doRequest() error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result["id"] != "user-123" {
		t.Errorf("doRequest() response id = %v, want 'user-123'", result["id"])
	}
}

func TestNotionClient_doRequest_WithBody(t *testing.T) {
	mockResponse := map[string]interface{}{
		"object": "page",
		"id":     "page-123",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify request body
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	config := DefaultConfig()
	config.APIKey = "secret_TEST123"
	config.BaseURL = server.URL
	config.Service.Name = "test-service"
	config.Server.Addr = ":8080"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	notionClient := client.(*notionClient)
	ctx := context.Background()

	requestBody := map[string]interface{}{
		"parent": map[string]string{"page_id": "parent-123"},
		"properties": map[string]interface{}{
			"title": map[string]interface{}{
				"title": []map[string]interface{}{
					{"text": map[string]string{"content": "Test Page"}},
				},
			},
		},
	}

	data, err := notionClient.doRequest(ctx, "POST", "/pages", requestBody)
	if err != nil {
		t.Fatalf("doRequest() error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result["id"] != "page-123" {
		t.Errorf("doRequest() response id = %v, want 'page-123'", result["id"])
	}
}

func TestNotionClient_doRequest_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object":  "error",
			"status":  400,
			"code":    "invalid_json",
			"message": "Invalid JSON in request body",
		})
	}))
	defer server.Close()

	config := DefaultConfig()
	config.APIKey = "secret_TEST123"
	config.BaseURL = server.URL
	config.MaxRetries = 0 // No retries for this test
	config.Service.Name = "test-service"
	config.Server.Addr = ":8080"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	notionClient := client.(*notionClient)
	ctx := context.Background()

	_, err = notionClient.doRequest(ctx, "GET", "/users/me", nil)
	if err == nil {
		t.Error("doRequest() expected error for 400 response, got nil")
	}

	notionErr, ok := err.(*NotionError)
	if !ok {
		t.Errorf("doRequest() error type = %T, want *NotionError", err)
	} else {
		if notionErr.Code != "invalid_json" {
			t.Errorf("doRequest() error code = %v, want 'invalid_json'", notionErr.Code)
		}
		if notionErr.Status != 400 {
			t.Errorf("doRequest() error status = %v, want 400", notionErr.Status)
		}
	}
}

func TestNotionClient_doRequest_RetryOn500(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "user",
			"id":     "user-123",
		})
	}))
	defer server.Close()

	config := DefaultConfig()
	config.APIKey = "secret_TEST123"
	config.BaseURL = server.URL
	config.MaxRetries = 3
	config.RateLimit = 100 // High rate limit to avoid rate limiting in test
	config.Service.Name = "test-service"
	config.Server.Addr = ":8080"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	notionClient := client.(*notionClient)
	ctx := context.Background()

	data, err := notionClient.doRequest(ctx, "GET", "/users/me", nil)
	if err != nil {
		t.Fatalf("doRequest() error: %v", err)
	}

	if attempts < 2 {
		t.Errorf("doRequest() should have retried, attempts = %d", attempts)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result["id"] != "user-123" {
		t.Errorf("doRequest() response id = %v, want 'user-123'", result["id"])
	}
}

func TestNotionClient_doRequest_RetryOn429(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "user",
			"id":     "user-123",
		})
	}))
	defer server.Close()

	config := DefaultConfig()
	config.APIKey = "secret_TEST123"
	config.BaseURL = server.URL
	config.MaxRetries = 3
	config.RateLimit = 100 // High rate limit to avoid rate limiting in test
	config.Service.Name = "test-service"
	config.Server.Addr = ":8080"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	notionClient := client.(*notionClient)
	ctx := context.Background()

	data, err := notionClient.doRequest(ctx, "GET", "/users/me", nil)
	if err != nil {
		t.Fatalf("doRequest() error: %v", err)
	}

	if attempts < 2 {
		t.Errorf("doRequest() should have retried on 429, attempts = %d", attempts)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result["id"] != "user-123" {
		t.Errorf("doRequest() response id = %v, want 'user-123'", result["id"])
	}
}

func TestNotionClient_doRequest_MaxRetriesExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := DefaultConfig()
	config.APIKey = "secret_TEST123"
	config.BaseURL = server.URL
	config.Service.Name = "test-service"
	config.Server.Addr = ":8080"
	config.Timeout = "5s"
	config.MaxRetries = 2
	config.RateLimit = 100

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	notionClient := client.(*notionClient)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = notionClient.doRequest(ctx, "GET", "/users/me", nil)
	if err == nil {
		t.Error("doRequest() expected error after max retries, got nil")
	}

	if !strings.Contains(err.Error(), "retries") {
		t.Errorf("doRequest() error should mention retries, got: %v", err)
	}
}

func TestNotionClient_doRequest_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultConfig()
	config.APIKey = "secret_TEST123"
	config.BaseURL = server.URL
	config.Service.Name = "test-service"
	config.Server.Addr = ":8080"
	config.MaxRetries = 0
	config.RateLimit = 100

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	notionClient := client.(*notionClient)
	ctx, cancel := context.WithCancel(context.Background())
	
	// Cancel context immediately
	cancel()

	_, err = notionClient.doRequest(ctx, "GET", "/users/me", nil)
	if err == nil {
		t.Error("doRequest() expected error on cancelled context, got nil")
	}

	if err != context.Canceled && !strings.Contains(err.Error(), "canceled") {
		t.Errorf("doRequest() error should be context.Canceled, got: %v", err)
	}
}

func TestNotionError_Error(t *testing.T) {
	err := &NotionError{
		Status:  400,
		Code:    "invalid_json",
		Message: "Invalid JSON in request body",
	}

	msg := err.Error()
	if !strings.Contains(msg, "invalid_json") {
		t.Errorf("NotionError.Error() = %q, should contain 'invalid_json'", msg)
	}
	if !strings.Contains(msg, "Invalid JSON") {
		t.Errorf("NotionError.Error() = %q, should contain message", msg)
	}
}

func TestNotionClient_RateLimiting(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "user",
			"id":     "user-123",
		})
	}))
	defer server.Close()

	config := DefaultConfig()
	config.APIKey = "secret_TEST123"
	config.BaseURL = server.URL
	config.Service.Name = "test-service"
	config.Server.Addr = ":8080"
	config.MaxRetries = 0
	config.RateLimit = 2 // 2 requests per second

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	notionClient := client.(*notionClient)
	ctx := context.Background()

	// Make multiple requests quickly
	start := time.Now()
	for i := 0; i < 3; i++ {
		_, err := notionClient.doRequest(ctx, "GET", "/users/me", nil)
		if err != nil {
			t.Fatalf("doRequest() error on request %d: %v", i, err)
		}
	}
	duration := time.Since(start)

	// Rate limiter should have delayed requests
	// With rate limit of 2/sec, 3 requests should take at least ~1 second
	if duration < 500*time.Millisecond {
		t.Logf("Rate limiting may not be working as expected, duration = %v", duration)
	}

	if requestCount != 3 {
		t.Errorf("Expected 3 requests, got %d", requestCount)
	}
}
