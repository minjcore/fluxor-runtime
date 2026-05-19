package proxy

import (
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector collects and calculates proxy metrics
type MetricsCollector struct {
	mu sync.RWMutex

	// Response time tracking
	responseTimes []time.Duration
	maxSamples    int

	// Request rate tracking
	requestCounts []time.Time
	rateWindow    time.Duration

	// Per-backend metrics
	backendMetrics map[string]*BackendMetrics
}

// BackendMetrics tracks metrics for a specific backend
type BackendMetrics struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	TotalLatency       time.Duration
	RequestCount       int
	LastRequestTime    time.Time
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(maxSamples int, rateWindow time.Duration) *MetricsCollector {
	return &MetricsCollector{
		responseTimes:  make([]time.Duration, 0, maxSamples),
		maxSamples:     maxSamples,
		requestCounts:  make([]time.Time, 0),
		rateWindow:     rateWindow,
		backendMetrics: make(map[string]*BackendMetrics),
	}
}

// RecordResponseTime records a response time
func (mc *MetricsCollector) RecordResponseTime(duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.responseTimes = append(mc.responseTimes, duration)
	if len(mc.responseTimes) > mc.maxSamples {
		mc.responseTimes = mc.responseTimes[1:]
	}
}

// RecordRequest records a request
func (mc *MetricsCollector) RecordRequest(backendURL string, success bool, latency time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	now := time.Now()
	mc.requestCounts = append(mc.requestCounts, now)

	// Clean old request counts
	cutoff := now.Add(-mc.rateWindow)
	validCounts := make([]time.Time, 0, len(mc.requestCounts))
	for _, t := range mc.requestCounts {
		if t.After(cutoff) {
			validCounts = append(validCounts, t)
		}
	}
	mc.requestCounts = validCounts

	// Update backend metrics
	bm, exists := mc.backendMetrics[backendURL]
	if !exists {
		bm = &BackendMetrics{}
		mc.backendMetrics[backendURL] = bm
	}

	atomic.AddInt64(&bm.TotalRequests, 1)
	if success {
		atomic.AddInt64(&bm.SuccessfulRequests, 1)
	} else {
		atomic.AddInt64(&bm.FailedRequests, 1)
	}
	bm.TotalLatency += latency
	bm.RequestCount++
	bm.LastRequestTime = now
}

// AverageResponseTime calculates the average response time
func (mc *MetricsCollector) AverageResponseTime() float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if len(mc.responseTimes) == 0 {
		return 0
	}

	var total time.Duration
	for _, rt := range mc.responseTimes {
		total += rt
	}

	return float64(total.Milliseconds()) / float64(len(mc.responseTimes))
}

// RequestsPerSecond calculates requests per second
func (mc *MetricsCollector) RequestsPerSecond() float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if len(mc.requestCounts) == 0 {
		return 0
	}

	// Count requests in the rate window
	now := time.Now()
	cutoff := now.Add(-mc.rateWindow)
	count := 0
	for _, t := range mc.requestCounts {
		if t.After(cutoff) {
			count++
		}
	}

	// Calculate RPS
	windowSeconds := mc.rateWindow.Seconds()
	if windowSeconds == 0 {
		return 0
	}

	return float64(count) / windowSeconds
}

// GetBackendMetrics returns metrics for a specific backend
func (mc *MetricsCollector) GetBackendMetrics(backendURL string) *BackendMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return mc.backendMetrics[backendURL]
}

// Reset resets all metrics
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.responseTimes = make([]time.Duration, 0, mc.maxSamples)
	mc.requestCounts = make([]time.Time, 0)
	mc.backendMetrics = make(map[string]*BackendMetrics)
}
