package prometheus

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// DefaultRegistry is the default Prometheus registry
	DefaultRegistry = prometheus.NewRegistry()

	// DefaultRegisterer is the default Prometheus registerer
	DefaultRegisterer = prometheus.WrapRegistererWith(prometheus.Labels{"service": "fluxor"}, DefaultRegistry)

	// Metrics collection
	metricsOnce sync.Once
	metrics     *Metrics
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// HTTP request metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestSize     *prometheus.HistogramVec
	HTTPResponseSize    *prometheus.HistogramVec

	// EventBus metrics
	EventBusMessagesTotal   *prometheus.CounterVec
	EventBusMessageDuration *prometheus.HistogramVec

	// Database pool metrics
	DatabaseConnectionsOpen  prometheus.Gauge
	DatabaseConnectionsIdle  prometheus.Gauge
	DatabaseConnectionsInUse prometheus.Gauge
	DatabaseConnectionsWait  prometheus.Counter
	DatabaseQueryDuration    *prometheus.HistogramVec

	// Server metrics
	ServerQueuedRequests          prometheus.Gauge
	ServerRejectedRequests        prometheus.Counter
	ServerCurrentCCU              prometheus.Gauge
	ServerNormalCCU               prometheus.Gauge
	ServerCCUUtilization          prometheus.Gauge
	ServerBackpressureQueueLength prometheus.Gauge
	ServerVerticleCount           prometheus.Gauge

	// Verticle metrics
	VerticleCount prometheus.Gauge

	// Custom metrics registry
	CustomCounters   map[string]*prometheus.CounterVec
	CustomGauges     map[string]*prometheus.GaugeVec
	CustomHistograms map[string]*prometheus.HistogramVec
	customMu         sync.RWMutex
}

// GetMetrics returns the global metrics instance
func GetMetrics() *Metrics {
	metricsOnce.Do(func() {
		metrics = NewMetrics(DefaultRegisterer)
	})
	return metrics
}

// NewMetrics creates a new metrics collection
func NewMetrics(registerer prometheus.Registerer) *Metrics {
	if registerer == nil {
		registerer = DefaultRegisterer
	}

	m := &Metrics{
		// HTTP request metrics
		HTTPRequestsTotal: promauto.With(registerer).NewCounterVec(
			prometheus.CounterOpts{
				Name: "fluxor_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: promauto.With(registerer).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "fluxor_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestSize: promauto.With(registerer).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "fluxor_http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 7), // 100B to 100MB
			},
			[]string{"method", "path"},
		),
		HTTPResponseSize: promauto.With(registerer).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "fluxor_http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 7), // 100B to 100MB
			},
			[]string{"method", "path", "status"},
		),

		// EventBus metrics
		EventBusMessagesTotal: promauto.With(registerer).NewCounterVec(
			prometheus.CounterOpts{
				Name: "fluxor_eventbus_messages_total",
				Help: "Total number of EventBus messages",
			},
			[]string{"address", "type"}, // type: publish, send, request
		),
		EventBusMessageDuration: promauto.With(registerer).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "fluxor_eventbus_message_duration_seconds",
				Help:    "EventBus message processing duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"address", "type"},
		),

		// Database pool metrics
		DatabaseConnectionsOpen: promauto.With(registerer).NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxor_database_connections_open",
				Help: "Number of open database connections",
			},
		),
		DatabaseConnectionsIdle: promauto.With(registerer).NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxor_database_connections_idle",
				Help: "Number of idle database connections",
			},
		),
		DatabaseConnectionsInUse: promauto.With(registerer).NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxor_database_connections_in_use",
				Help: "Number of database connections in use",
			},
		),
		DatabaseConnectionsWait: promauto.With(registerer).NewCounter(
			prometheus.CounterOpts{
				Name: "fluxor_database_connections_wait_total",
				Help: "Total number of database connection wait events",
			},
		),
		DatabaseQueryDuration: promauto.With(registerer).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "fluxor_database_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"operation"}, // operation: query, exec, begin
		),

		// Server metrics
		ServerQueuedRequests: promauto.With(registerer).NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxor_server_queued_requests",
				Help: "Number of queued HTTP requests",
			},
		),
		ServerRejectedRequests: promauto.With(registerer).NewCounter(
			prometheus.CounterOpts{
				Name: "fluxor_server_rejected_requests_total",
				Help: "Total number of rejected HTTP requests (503)",
			},
		),
		ServerCurrentCCU: promauto.With(registerer).NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxor_server_current_ccu",
				Help: "Current concurrent users (CCU)",
			},
		),
		ServerNormalCCU: promauto.With(registerer).NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxor_server_normal_ccu",
				Help: "Normal capacity CCU (target utilization)",
			},
		),
		ServerCCUUtilization: promauto.With(registerer).NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxor_server_ccu_utilization",
				Help: "CCU utilization percentage (0-100)",
			},
		),
		ServerBackpressureQueueLength: promauto.With(registerer).NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxor_backpressure_queue_length",
				Help: "Current backpressure queue length",
			},
		),
		ServerVerticleCount: promauto.With(registerer).NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxor_server_verticle_count",
				Help: "Number of deployed verticles",
			},
		),

		// Verticle metrics
		VerticleCount: promauto.With(registerer).NewGauge(
			prometheus.GaugeOpts{
				Name: "fluxor_verticle_count",
				Help: "Number of deployed verticles",
			},
		),

		// Custom metrics
		CustomCounters:   make(map[string]*prometheus.CounterVec),
		CustomGauges:     make(map[string]*prometheus.GaugeVec),
		CustomHistograms: make(map[string]*prometheus.HistogramVec),
	}

	return m
}

// RecordHTTPRequest records an HTTP request metric
func (m *Metrics) RecordHTTPRequest(method, path, status string, duration time.Duration, requestSize, responseSize int64) {
	m.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path, status).Observe(duration.Seconds())
	m.HTTPRequestSize.WithLabelValues(method, path).Observe(float64(requestSize))
	m.HTTPResponseSize.WithLabelValues(method, path, status).Observe(float64(responseSize))
}

// RecordEventBusMessage records an EventBus message metric
func (m *Metrics) RecordEventBusMessage(address, msgType string, duration time.Duration) {
	m.EventBusMessagesTotal.WithLabelValues(address, msgType).Inc()
	m.EventBusMessageDuration.WithLabelValues(address, msgType).Observe(duration.Seconds())
}

// UpdateDatabasePool updates database pool metrics
func (m *Metrics) UpdateDatabasePool(open, idle, inUse int, waitCount int64) {
	m.DatabaseConnectionsOpen.Set(float64(open))
	m.DatabaseConnectionsIdle.Set(float64(idle))
	m.DatabaseConnectionsInUse.Set(float64(inUse))
	if waitCount > 0 {
		m.DatabaseConnectionsWait.Add(float64(waitCount))
	}
}

// RecordDatabaseQuery records a database query metric
func (m *Metrics) RecordDatabaseQuery(operation string, duration time.Duration) {
	m.DatabaseQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// UpdateServerMetrics updates server metrics
func (m *Metrics) UpdateServerMetrics(queued int64, rejected int64, currentCCU int, normalCCU int, utilization float64, verticleCount int) {
	m.ServerQueuedRequests.Set(float64(queued))
	m.ServerBackpressureQueueLength.Set(float64(queued)) // Alias for backpressure
	if rejected > 0 {
		m.ServerRejectedRequests.Add(float64(rejected))
	}
	m.ServerCurrentCCU.Set(float64(currentCCU))
	m.ServerNormalCCU.Set(float64(normalCCU))
	m.ServerCCUUtilization.Set(utilization)
	m.ServerVerticleCount.Set(float64(verticleCount))
	m.VerticleCount.Set(float64(verticleCount))
}

// UpdateVerticleCount updates the verticle count metric
func (m *Metrics) UpdateVerticleCount(count int) {
	m.VerticleCount.Set(float64(count))
}

// Counter creates or returns a custom counter metric
func (m *Metrics) Counter(name, help string, labels ...string) *prometheus.CounterVec {
	m.customMu.RLock()
	if counter, exists := m.CustomCounters[name]; exists {
		m.customMu.RUnlock()
		return counter
	}
	m.customMu.RUnlock()

	m.customMu.Lock()
	defer m.customMu.Unlock()

	// Double-check after acquiring write lock
	if counter, exists := m.CustomCounters[name]; exists {
		return counter
	}

	counter := promauto.With(DefaultRegisterer).NewCounterVec(
		prometheus.CounterOpts{
			Name: name,
			Help: help,
		},
		labels,
	)
	m.CustomCounters[name] = counter
	return counter
}

// Gauge creates or returns a custom gauge metric
func (m *Metrics) Gauge(name, help string, labels ...string) *prometheus.GaugeVec {
	m.customMu.RLock()
	if gauge, exists := m.CustomGauges[name]; exists {
		m.customMu.RUnlock()
		return gauge
	}
	m.customMu.RUnlock()

	m.customMu.Lock()
	defer m.customMu.Unlock()

	// Double-check after acquiring write lock
	if gauge, exists := m.CustomGauges[name]; exists {
		return gauge
	}

	gauge := promauto.With(DefaultRegisterer).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: name,
			Help: help,
		},
		labels,
	)
	m.CustomGauges[name] = gauge
	return gauge
}

// Histogram creates or returns a custom histogram metric
func (m *Metrics) Histogram(name, help string, buckets []float64, labels ...string) *prometheus.HistogramVec {
	m.customMu.RLock()
	if histogram, exists := m.CustomHistograms[name]; exists {
		m.customMu.RUnlock()
		return histogram
	}
	m.customMu.RUnlock()

	m.customMu.Lock()
	defer m.customMu.Unlock()

	// Double-check after acquiring write lock
	if histogram, exists := m.CustomHistograms[name]; exists {
		return histogram
	}

	opts := prometheus.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: buckets,
	}
	if buckets == nil {
		opts.Buckets = prometheus.DefBuckets
	}

	histogram := promauto.With(DefaultRegisterer).NewHistogramVec(opts, labels)
	m.CustomHistograms[name] = histogram
	return histogram
}

// Convenience functions for global metrics

// Counter returns a custom counter metric (creates if doesn't exist)
func Counter(name, help string, labels ...string) *prometheus.CounterVec {
	return GetMetrics().Counter(name, help, labels...)
}

// Gauge returns a custom gauge metric (creates if doesn't exist)
func Gauge(name, help string, labels ...string) *prometheus.GaugeVec {
	return GetMetrics().Gauge(name, help, labels...)
}

// Histogram returns a custom histogram metric (creates if doesn't exist)
func Histogram(name, help string, buckets []float64, labels ...string) *prometheus.HistogramVec {
	return GetMetrics().Histogram(name, help, buckets, labels...)
}
