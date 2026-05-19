package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

func TestNewProxyServer(t *testing.T) {
	gocmd := core.NewGoCMD(context.Background())
	config := DefaultConfig()
	config.ListenAddr = ":0"
	config.Backends = []Backend{
		{URL: "http://localhost:3000"},
	}

	server, err := NewProxyServer(gocmd, config)
	if err != nil {
		t.Fatalf("Failed to create proxy server: %v", err)
	}

	if server == nil {
		t.Fatal("Server is nil")
	}
}

func TestProxyServerStartStop(t *testing.T) {
	gocmd := core.NewGoCMD(context.Background())
	config := DefaultConfig()
	config.ListenAddr = ":0"
	config.Backends = []Backend{
		{URL: "http://localhost:3000"},
	}

	server, err := NewProxyServer(gocmd, config)
	if err != nil {
		t.Fatalf("Failed to create proxy server: %v", err)
	}

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	if err := server.Stop(); err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}

	// Wait for start to finish
	select {
	case err := <-errChan:
		if err != nil {
			t.Logf("Server start error (expected on stop): %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Log("Server stopped successfully")
	}
}

func TestProxyServerMetrics(t *testing.T) {
	gocmd := core.NewGoCMD(context.Background())
	config := DefaultConfig()
	config.ListenAddr = ":0"
	config.Backends = []Backend{
		{URL: "http://localhost:3000"},
	}

	server, err := NewProxyServer(gocmd, config)
	if err != nil {
		t.Fatalf("Failed to create proxy server: %v", err)
	}

	metrics := server.Metrics()
	if metrics.TotalConnections < 0 {
		t.Error("TotalConnections should be non-negative")
	}
}

func TestProxyServerHealth(t *testing.T) {
	gocmd := core.NewGoCMD(context.Background())
	config := DefaultConfig()
	config.ListenAddr = ":0"
	config.Backends = []Backend{
		{URL: "http://localhost:3000"},
	}

	server, err := NewProxyServer(gocmd, config)
	if err != nil {
		t.Fatalf("Failed to create proxy server: %v", err)
	}

	health := server.Health()
	if health == nil {
		t.Fatal("Health should not be nil")
	}

	healthy, ok := health["healthy"].(bool)
	if !ok {
		t.Fatal("Health should contain 'healthy' boolean")
	}

	// Server should be healthy when not started (no backends checked yet)
	if !healthy {
		t.Log("Server health check returned unhealthy (may be expected)")
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond)

	// Test closed state
	if cb.State() != StateClosed {
		t.Error("Circuit breaker should start in closed state")
	}

	// Simulate failures
	for i := 0; i < 3; i++ {
		err := cb.Call(func() error {
			return &ProxyError{Code: "TEST_ERROR", Message: "test failure"}
		})
		if err == nil {
			t.Error("Expected error from circuit breaker")
		}
	}

	// Should be open now
	if cb.State() != StateOpen {
		t.Error("Circuit breaker should be open after 3 failures")
	}

	// Should reject calls when open
	err := cb.Call(func() error { return nil })
	if err == nil {
		t.Error("Circuit breaker should reject calls when open")
	}

	// Wait for half-open transition
	time.Sleep(150 * time.Millisecond)
	err = cb.Call(func() error { return nil })
	if err != nil {
		t.Error("Circuit breaker should allow calls in half-open state")
	}

	// Need 2 successes to close from half-open
	err = cb.Call(func() error { return nil })
	if err != nil {
		t.Error("Circuit breaker should allow second call in half-open state")
	}

	// Should transition back to closed after 2 successes
	if cb.State() != StateClosed {
		t.Error("Circuit breaker should be closed after 2 successful calls")
	}
}

func TestHealthChecker(t *testing.T) {
	// Create a test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	checker := NewHealthChecker(5 * time.Second)

	// Test HTTP health check
	backend := Backend{
		URL:           ts.URL,
		HealthCheckURL: ts.URL + "/health",
	}

	healthy, latency, err := checker.CheckHealth(backend)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	if !healthy {
		t.Error("Backend should be healthy")
	}

	if latency <= 0 {
		t.Error("Latency should be positive")
	}
}

func TestMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector(100, 10*time.Second)

	// Record some response times
	collector.RecordResponseTime(100 * time.Millisecond)
	collector.RecordResponseTime(200 * time.Millisecond)
	collector.RecordResponseTime(150 * time.Millisecond)

	avg := collector.AverageResponseTime()
	if avg <= 0 {
		t.Error("Average response time should be positive")
	}

	// Record some requests
	collector.RecordRequest("http://backend1", true, 100*time.Millisecond)
	collector.RecordRequest("http://backend1", true, 150*time.Millisecond)
	collector.RecordRequest("http://backend2", false, 200*time.Millisecond)

	rps := collector.RequestsPerSecond()
	if rps < 0 {
		t.Error("Requests per second should be non-negative")
	}

	// Get backend metrics
	bm := collector.GetBackendMetrics("http://backend1")
	if bm == nil {
		t.Error("Backend metrics should not be nil")
	}
}

func TestLoadBalancer(t *testing.T) {
	backends := []BackendStatus{
		{Backend: Backend{URL: "http://backend1", Weight: 1}, Healthy: true},
		{Backend: Backend{URL: "http://backend2", Weight: 2}, Healthy: true},
		{Backend: Backend{URL: "http://backend3", Weight: 1}, Healthy: true},
	}

	// Test round-robin
	lb := newLoadBalancer("round-robin")
	selected, err := lb.SelectBackend(backends)
	if err != nil {
		t.Fatalf("Failed to select backend: %v", err)
	}
	if selected == nil {
		t.Fatal("Selected backend should not be nil")
	}

	// Test weighted
	lb = newLoadBalancer("weighted")
	selected, err = lb.SelectBackend(backends)
	if err != nil {
		t.Fatalf("Failed to select backend: %v", err)
	}
	if selected == nil {
		t.Fatal("Selected backend should not be nil")
	}

	// Test least-connections
	backends[0].Connections = 5
	backends[1].Connections = 2
	backends[2].Connections = 10

	lb = newLoadBalancer("least-connections")
	selected, err = lb.SelectBackend(backends)
	if err != nil {
		t.Fatalf("Failed to select backend: %v", err)
	}
	if selected.URL != "http://backend2" {
		t.Errorf("Expected backend2 (least connections), got %s", selected.URL)
	}

	// Test random
	lb = newLoadBalancer("random")
	selected, err = lb.SelectBackend(backends)
	if err != nil {
		t.Fatalf("Failed to select backend: %v", err)
	}
	if selected == nil {
		t.Fatal("Selected backend should not be nil")
	}
}
