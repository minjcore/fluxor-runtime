package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPCheck_Success(t *testing.T) {
	// Create a test server that returns 200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := HTTPCheck(server.URL, 5*time.Second)
	if checker == nil {
		t.Fatal("HTTPCheck() should not return nil")
	}

	err := checker(context.Background())
	if err != nil {
		t.Errorf("HTTPCheck() returned error for successful request: %v", err)
	}
}

func TestHTTPCheck_Failure(t *testing.T) {
	// Create a test server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	checker := HTTPCheck(server.URL, 5*time.Second)
	if checker == nil {
		t.Fatal("HTTPCheck() should not return nil")
	}

	err := checker(context.Background())
	if err == nil {
		t.Error("HTTPCheck() should return error for 500 status")
	}

	// Verify it's a health.Error
	if _, ok := err.(*Error); !ok {
		t.Errorf("Expected health.Error, got %T", err)
	}
}

func TestHTTPCheck_Timeout(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := HTTPCheck(server.URL, 100*time.Millisecond)
	if checker == nil {
		t.Fatal("HTTPCheck() should not return nil")
	}

	err := checker(context.Background())
	if err == nil {
		t.Error("HTTPCheck() should return error on timeout")
	}
}

func TestHTTPCheck_InvalidURL(t *testing.T) {
	checker := HTTPCheck("http://invalid-url-that-does-not-exist.example.com", 1*time.Second)
	if checker == nil {
		t.Fatal("HTTPCheck() should not return nil")
	}

	err := checker(context.Background())
	if err == nil {
		t.Error("HTTPCheck() should return error for invalid URL")
	}
}

func TestHTTPCheck_DefaultTimeout(t *testing.T) {
	// Test that default timeout is used when 0 is provided
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := HTTPCheck(server.URL, 0)
	if checker == nil {
		t.Fatal("HTTPCheck() should not return nil")
	}

	err := checker(context.Background())
	if err != nil {
		t.Errorf("HTTPCheck() returned error with default timeout: %v", err)
	}
}

func TestHTTPCheckWithHeaders_Success(t *testing.T) {
	// Create a test server that checks headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify custom header is present
		if r.Header.Get("X-Custom-Header") != "custom-value" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	headers := map[string]string{
		"X-Custom-Header": "custom-value",
	}

	checker := HTTPCheckWithHeaders(server.URL, 5*time.Second, headers)
	if checker == nil {
		t.Fatal("HTTPCheckWithHeaders() should not return nil")
	}

	err := checker(context.Background())
	if err != nil {
		t.Errorf("HTTPCheckWithHeaders() returned error: %v", err)
	}
}

func TestHTTPCheckWithHeaders_MultipleHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify multiple headers
		if r.Header.Get("X-Header-1") != "value-1" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.Header.Get("X-Header-2") != "value-2" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	headers := map[string]string{
		"X-Header-1": "value-1",
		"X-Header-2": "value-2",
	}

	checker := HTTPCheckWithHeaders(server.URL, 5*time.Second, headers)
	if checker == nil {
		t.Fatal("HTTPCheckWithHeaders() should not return nil")
	}

	err := checker(context.Background())
	if err != nil {
		t.Errorf("HTTPCheckWithHeaders() returned error: %v", err)
	}
}

func TestHTTPCheckWithHeaders_DefaultTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	headers := map[string]string{
		"X-Custom-Header": "custom-value",
	}

	checker := HTTPCheckWithHeaders(server.URL, 0, headers)
	if checker == nil {
		t.Fatal("HTTPCheckWithHeaders() should not return nil")
	}

	err := checker(context.Background())
	if err != nil {
		t.Errorf("HTTPCheckWithHeaders() returned error with default timeout: %v", err)
	}
}

func TestHTTPCheckWithHeaders_Failure(t *testing.T) {
	// Create a test server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	headers := map[string]string{
		"X-Custom-Header": "custom-value",
	}

	checker := HTTPCheckWithHeaders(server.URL, 5*time.Second, headers)
	if checker == nil {
		t.Fatal("HTTPCheckWithHeaders() should not return nil")
	}

	err := checker(context.Background())
	if err == nil {
		t.Error("HTTPCheckWithHeaders() should return error for 500 status")
	}
}

func TestHTTPCheckWithHeaders_ContextCancellation(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	headers := map[string]string{
		"X-Custom-Header": "custom-value",
	}

	checker := HTTPCheckWithHeaders(server.URL, 5*time.Second, headers)
	if checker == nil {
		t.Fatal("HTTPCheckWithHeaders() should not return nil")
	}

	// Create a context that will be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := checker(ctx)
	if err == nil {
		t.Error("HTTPCheckWithHeaders() should return error when context is cancelled")
	}
}
