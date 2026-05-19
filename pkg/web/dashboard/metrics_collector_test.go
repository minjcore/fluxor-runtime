package dashboard

import (
	"context"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
)

func TestGetMetricsCollector(t *testing.T) {
	collector1 := GetMetricsCollector()
	if collector1 == nil {
		t.Fatal("GetMetricsCollector() should not return nil")
	}

	// Should return the same instance (singleton)
	collector2 := GetMetricsCollector()
	if collector1 != collector2 {
		t.Error("GetMetricsCollector() should return the same instance")
	}
}

func TestMetricsCollector_RegisterHTTPServer(t *testing.T) {
	collector := GetMetricsCollector()

	// Create a mock HTTP server
	mockServer := &mockHTTPServer{
		metrics: HTTPServerMetricsData{
			QueuedRequests:   10,
			RejectedRequests: 2,
			TotalRequests:    100,
			SuccessfulRequests: 95,
			ErrorRequests:    5,
			QueueCapacity:    1000,
			QueueUtilization: 1.0,
			Workers:          100,
			CurrentCCU:       50,
			CCUUtilization:   5.0,
			BytesSent:        10000,
			BytesReceived:    5000,
		},
	}

	collector.RegisterHTTPServer("test-server", mockServer)

	// Verify server is registered
	metrics := collector.CollectAllMetrics()
	found := false
	for _, server := range metrics.HTTPServers {
		if server.Name == "test-server" {
			found = true
			if server.QueuedRequests != 10 {
				t.Errorf("Expected QueuedRequests 10, got %d", server.QueuedRequests)
			}
			break
		}
	}

	if !found {
		t.Error("HTTP server should be registered")
	}
}

func TestMetricsCollector_UnregisterHTTPServer(t *testing.T) {
	collector := GetMetricsCollector()

	mockServer := &mockHTTPServer{
		metrics: HTTPServerMetricsData{},
	}

	collector.RegisterHTTPServer("test-server", mockServer)
	collector.UnregisterHTTPServer("test-server")

	// Verify server is unregistered
	metrics := collector.CollectAllMetrics()
	for _, server := range metrics.HTTPServers {
		if server.Name == "test-server" {
			t.Error("HTTP server should be unregistered")
		}
	}
}

func TestMetricsCollector_CollectAllMetrics(t *testing.T) {
	collector := GetMetricsCollector()

	metrics := collector.CollectAllMetrics()
	if metrics == nil {
		t.Fatal("CollectAllMetrics() should not return nil")
	}

	// Should have timestamp
	if metrics.Timestamp.IsZero() {
		t.Error("Metrics should have timestamp")
	}

	// Should have runtime metrics
	if metrics.Runtime == nil {
		t.Error("Metrics should have runtime data")
	} else {
		if metrics.Runtime.NumCPU == 0 {
			t.Error("Runtime should have NumCPU")
		}
		if metrics.Runtime.Goroutines == 0 {
			t.Log("Goroutines may be 0 in test environment")
		}
	}
}

func TestMetricsCollector_StartProfiling(t *testing.T) {
	collector := GetMetricsCollector()

	// Test with valid context
	ctx := context.Background()
	collector.StartProfiling(ctx)

	// Test with nil context (should use context.Background())
	collector.StartProfiling(nil)

	// Test with invalid context type
	collector.StartProfiling("invalid")

	// Should not panic
	_ = collector
}

func TestMetricsCollector_CollectAllMetrics_WithHTTPServer(t *testing.T) {
	collector := GetMetricsCollector()

	mockServer := &mockHTTPServer{
		metrics: HTTPServerMetricsData{
			QueuedRequests:     5,
			RejectedRequests:   1,
			TotalRequests:      50,
			SuccessfulRequests: 48,
			ErrorRequests:      2,
			QueueCapacity:      500,
			QueueUtilization:   1.0,
			Workers:            50,
			CurrentCCU:         25,
			CCUUtilization:    5.0,
			BytesSent:          5000,
			BytesReceived:      2500,
		},
	}

	collector.RegisterHTTPServer("test", mockServer)

	metrics := collector.CollectAllMetrics()

	if len(metrics.HTTPServers) == 0 {
		t.Error("Should have HTTP server metrics")
	}

	server := metrics.HTTPServers[0]
	if server.Name != "test" {
		t.Errorf("Expected server name 'test', got '%s'", server.Name)
	}

	if server.QueuedRequests != 5 {
		t.Errorf("Expected QueuedRequests 5, got %d", server.QueuedRequests)
	}

	if server.QueueUtilization != 1.0 {
		t.Errorf("Expected QueueUtilization 1.0, got %.2f", server.QueueUtilization)
	}
}

func TestMetricsCollector_AllocationRate(t *testing.T) {
	collector := GetMetricsCollector()

	// First collection
	metrics1 := collector.CollectAllMetrics()
	_ = metrics1

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Second collection
	metrics2 := collector.CollectAllMetrics()

	if metrics2.Runtime == nil {
		t.Fatal("Runtime metrics should not be nil")
	}

	// Allocation rate should be calculated
	// (may be 0 if no allocations occurred)
	_ = metrics2.Runtime.AllocRate
}

func TestSanitizeFloat64(t *testing.T) {
	testCases := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"normal value", 1.5, 1.5},
		{"zero", 0.0, 0.0},
		{"NaN", 0.0 / 0.0, 0.0}, // NaN
		{"positive infinity", 1.0 / 0.0, 0.0}, // +Inf
		{"negative infinity", -1.0 / 0.0, 0.0}, // -Inf
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeFloat64(tc.input)
			if tc.name == "normal value" || tc.name == "zero" {
				if result != tc.expected {
					t.Errorf("sanitizeFloat64(%f) = %f, want %f", tc.input, result, tc.expected)
				}
			} else {
				// For NaN/Inf, result should be 0.0
				if result != 0.0 {
					t.Errorf("sanitizeFloat64(%f) = %f, want 0.0", tc.input, result)
				}
			}
		})
	}
}

func TestConvertGoroutineStates(t *testing.T) {
	// This is a helper function, test it indirectly through metrics collection
	collector := GetMetricsCollector()
	metrics := collector.CollectAllMetrics()

	if metrics.Profiling != nil {
		// Verify goroutine states are converted to strings
		states := metrics.Profiling.Goroutines.ByState
		if states != nil {
			// All keys should be strings
			for key := range states {
				if key == "" {
					t.Error("Goroutine state key should not be empty")
				}
			}
		}
	}
}

func TestConvertWorkTypes(t *testing.T) {
	// This is a helper function, test it indirectly through metrics collection
	collector := GetMetricsCollector()
	metrics := collector.CollectAllMetrics()

	if metrics.Profiling != nil {
		// Verify work types are converted to strings
		workTypes := metrics.Profiling.Goroutines.ByWorkType
		if workTypes != nil {
			// All keys should be strings
			for key := range workTypes {
				if key == "" {
					t.Error("Work type key should not be empty")
				}
			}
		}
	}
}

// mockHTTPServer is a test helper that implements HTTPServerMetricsProvider
type mockHTTPServer struct {
	metrics HTTPServerMetricsData
}

func (m *mockHTTPServer) Metrics() HTTPServerMetricsData {
	return m.metrics
}

func TestMetricsCollector_MultipleHTTPServers(t *testing.T) {
	collector := GetMetricsCollector()

	server1 := &mockHTTPServer{
		metrics: HTTPServerMetricsData{
			TotalRequests: 100,
			Workers:       50,
		},
	}

	server2 := &mockHTTPServer{
		metrics: HTTPServerMetricsData{
			TotalRequests: 200,
			Workers:       100,
		},
	}

	collector.RegisterHTTPServer("server1", server1)
	collector.RegisterHTTPServer("server2", server2)

	metrics := collector.CollectAllMetrics()

	if len(metrics.HTTPServers) < 2 {
		t.Errorf("Expected at least 2 HTTP servers, got %d", len(metrics.HTTPServers))
	}

	// Clean up
	collector.UnregisterHTTPServer("server1")
	collector.UnregisterHTTPServer("server2")
}

func TestMetricsCollector_ConcurrentAccess(t *testing.T) {
	collector := GetMetricsCollector()

	// Test concurrent registration and collection
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			server := &mockHTTPServer{
				metrics: HTTPServerMetricsData{
					TotalRequests: int64(id * 10),
				},
			}
			collector.RegisterHTTPServer("server", server)
			_ = collector.CollectAllMetrics()
			collector.UnregisterHTTPServer("server")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic
	_ = collector
}
