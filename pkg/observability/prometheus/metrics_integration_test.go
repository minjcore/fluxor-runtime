package prometheus_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/observability/prometheus"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

// TestMetricsEndpoint_Integration tests the /metrics endpoint with real data
func TestMetricsEndpoint_Integration(t *testing.T) {
	// Create GoCMD and server
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Deploy a test verticle
	testVerticle := &TestVerticle{}
	_, err := gocmd.DeployVerticle(testVerticle)
	if err != nil {
		t.Fatalf("Failed to deploy verticle: %v", err)
	}

	// Create server with backpressure
	config := web.CCUBasedConfigWithUtilization(":0", 1000, 67) // Random port
	server := web.NewFastHTTPServer(gocmd, config)

	// Get router and register metrics endpoint
	router := server.FastRouter()
	prometheus.RegisterMetricsEndpoint(router, "/metrics")

	// Add a test route with metrics middleware
	metricsMiddleware := prometheus.FastHTTPMetricsMiddleware()
	router.GETFast("/test", metricsMiddleware(func(ctx *web.FastRequestContext) error {
		return ctx.JSON(200, map[string]string{"status": "ok"})
	}))

	// Setup in-memory listener for testing
	ln := fasthttputil.NewInmemoryListener()
	handler := func(rc *fasthttp.RequestCtx) {
		reqCtx := &web.FastRequestContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			RequestCtx:         rc,
			GoCMD:              gocmd,
			EventBus:           gocmd.EventBus(),
			Params:             make(map[string]string),
		}
		router.ServeFastHTTP(reqCtx)
	}

	// Start server in background
	go func() {
		srv := &fasthttp.Server{Handler: handler}
		srv.Serve(ln)
	}()
	defer ln.Close()

	// Create HTTP client that uses in-memory listener
	httpClient := &http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				return ln.Dial()
			},
		},
	}

	baseURL := "http://test"

	// Test 1: Make some requests to generate metrics
	t.Run("GenerateMetrics", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			resp, err := httpClient.Get(baseURL + "/test")
			if err != nil {
				t.Fatalf("Failed to make test request: %v", err)
			}
			resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		}
	})

	// Update server metrics manually
	prometheus.UpdateServerMetrics(server)

	// Update verticle count
	metrics := prometheus.GetMetrics()
	metrics.UpdateVerticleCount(gocmd.DeploymentCount())

	// Test 2: Scrape /metrics endpoint
	t.Run("ScrapeMetrics", func(t *testing.T) {
		resp, err := httpClient.Get(baseURL + "/metrics")
		if err != nil {
			t.Fatalf("Failed to scrape metrics: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		metricsOutput := string(body)

		// Test 3: Verify required metrics are present
		requiredMetrics := []string{
			"fluxor_http_requests_total",
			"fluxor_http_request_duration_seconds",
			"fluxor_server_current_ccu",
			"fluxor_server_normal_ccu",
			"fluxor_server_ccu_utilization",
			"fluxor_backpressure_queue_length",
			"fluxor_verticle_count",
		}

		for _, metric := range requiredMetrics {
			if !strings.Contains(metricsOutput, metric) {
				t.Errorf("Metrics output missing required metric: %s", metric)
			}
		}

		// Test 4: Verify metric values make sense
		t.Run("VerifyMetricValues", func(t *testing.T) {
			// Check that we recorded HTTP requests (with service label)
			if !strings.Contains(metricsOutput, `fluxor_http_requests_total{method="GET",path="/test",service="fluxor",status="2xx"} 5`) {
				t.Logf("Metrics output:\n%s", metricsOutput)
				t.Error("Expected to find 5 GET requests to /test with 2xx status")
			}

			// Check verticle count (with service label)
			if !strings.Contains(metricsOutput, `fluxor_verticle_count{service="fluxor"} 1`) {
				t.Error("Expected verticle_count to be 1")
			}

			// Check CCU metrics exist (values may vary)
			if !strings.Contains(metricsOutput, "fluxor_server_current_ccu") {
				t.Error("Expected to find current_ccu metric")
			}

			if !strings.Contains(metricsOutput, "fluxor_server_normal_ccu") && !strings.Contains(metricsOutput, "670") {
				t.Logf("Looking for normal_ccu=670 in:\n%s", metricsOutput)
				t.Error("Expected normal_ccu to be 670 (67% of 1000)")
			}
		})

		// Test 5: Verify Prometheus format
		t.Run("VerifyPrometheusFormat", func(t *testing.T) {
			// Check for HELP and TYPE comments
			if !strings.Contains(metricsOutput, "# HELP") {
				t.Error("Expected HELP comments in Prometheus format")
			}
			if !strings.Contains(metricsOutput, "# TYPE") {
				t.Error("Expected TYPE comments in Prometheus format")
			}

			// Check content type
			contentType := resp.Header.Get("Content-Type")
			if !strings.Contains(contentType, "text/plain") && !strings.Contains(contentType, "application/openmetrics-text") {
				t.Errorf("Expected Prometheus content type, got: %s", contentType)
			}
		})
	})

	// Test 6: Multiple scrapes should work
	t.Run("MultipleScrapes", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			resp, err := httpClient.Get(baseURL + "/metrics")
			if err != nil {
				t.Fatalf("Scrape %d failed: %v", i, err)
			}
			resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Errorf("Scrape %d: expected status 200, got %d", i, resp.StatusCode)
			}
		}
	})
}

// TestVerticle is a simple test verticle
type TestVerticle struct{}

func (v *TestVerticle) Start(ctx core.FluxorContext) error {
	return nil
}

func (v *TestVerticle) Stop(ctx core.FluxorContext) error {
	return nil
}

// TestMetricsCollection tests metric collection without server
func TestMetricsCollection(t *testing.T) {
	metrics := prometheus.GetMetrics()

	t.Run("HTTPMetrics", func(t *testing.T) {
		// Record some HTTP metrics
		metrics.RecordHTTPRequest("GET", "/api/test", "2xx", 10*time.Millisecond, 100, 500)
		metrics.RecordHTTPRequest("POST", "/api/test", "2xx", 20*time.Millisecond, 200, 600)
		metrics.RecordHTTPRequest("GET", "/api/test", "4xx", 5*time.Millisecond, 50, 100)

		// Metrics should be recorded (can't easily test counter values without exposing internals)
		// But we can verify no panics occurred
	})

	t.Run("ServerMetrics", func(t *testing.T) {
		// Update server metrics
		metrics.UpdateServerMetrics(
			10,    // queued
			2,     // rejected
			150,   // current CCU
			670,   // normal CCU
			22.39, // utilization
			5,     // verticle count
		)

		// Should not panic
	})

	t.Run("VerticleMetrics", func(t *testing.T) {
		// Update verticle count
		metrics.UpdateVerticleCount(5)

		// Should not panic
	})

	t.Run("DatabaseMetrics", func(t *testing.T) {
		// Update database metrics
		metrics.UpdateDatabasePool(10, 5, 5, 2)
		metrics.RecordDatabaseQuery("SELECT", 15*time.Millisecond)

		// Should not panic
	})

	t.Run("EventBusMetrics", func(t *testing.T) {
		// Record EventBus metrics
		metrics.RecordEventBusMessage("user.events", "publish", 5*time.Millisecond)
		metrics.RecordEventBusMessage("order.events", "send", 8*time.Millisecond)

		// Should not panic
	})
}

// BenchmarkMetricsEndpoint benchmarks the /metrics endpoint
func BenchmarkMetricsEndpoint(b *testing.B) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := web.CCUBasedConfigWithUtilization(":0", 1000, 67)
	server := web.NewFastHTTPServer(gocmd, config)

	router := server.FastRouter()
	prometheus.RegisterMetricsEndpoint(router, "/metrics")

	// Setup in-memory listener
	ln := fasthttputil.NewInmemoryListener()
	handler := func(rc *fasthttp.RequestCtx) {
		reqCtx := &web.FastRequestContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			RequestCtx:         rc,
			GoCMD:              gocmd,
			EventBus:           gocmd.EventBus(),
			Params:             make(map[string]string),
		}
		router.ServeFastHTTP(reqCtx)
	}

	go func() {
		srv := &fasthttp.Server{Handler: handler}
		srv.Serve(ln)
	}()
	defer ln.Close()

	httpClient := &http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				return ln.Dial()
			},
		},
	}

	baseURL := "http://test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := httpClient.Get(baseURL + "/metrics")
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
