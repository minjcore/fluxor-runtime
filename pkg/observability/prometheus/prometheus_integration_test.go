package prometheus_test

import (
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/observability/prometheus"
)

func TestPrometheusMetrics(t *testing.T) {
	metrics := prometheus.GetMetrics()

	// Test HTTP request recording
	metrics.RecordHTTPRequest("GET", "/api/users", "200", 100*time.Millisecond, 100, 500)
	metrics.RecordHTTPRequest("POST", "/api/users", "201", 50*time.Millisecond, 200, 300)

	// Test EventBus message recording
	metrics.RecordEventBusMessage("user.created", "publish", 10*time.Millisecond)

	// Test database pool metrics
	metrics.UpdateDatabasePool(25, 5, 20, 0)

	// Test server metrics
	// queued, rejected, currentCCU, normalCCU, utilization, verticleCount
	metrics.UpdateServerMetrics(100, 0, 500, 670, 75.5, 5)

	// Test custom metrics
	counter := metrics.Counter("custom_events_total", "Total custom events", "type")
	counter.WithLabelValues("test").Inc()

	gauge := metrics.Gauge("custom_gauge", "Custom gauge", "label")
	gauge.WithLabelValues("test").Set(42.0)

	// If we get here without panic, metrics are working
}
