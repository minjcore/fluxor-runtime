package dashboard

import (
	"context"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/concurrency"
	"github.com/fluxorio/fluxor/pkg/web/profiling"
)

// HTTPServerMetricsProvider is an interface for HTTP servers that provide metrics
type HTTPServerMetricsProvider interface {
	Metrics() HTTPServerMetricsData
}

// HTTPServerMetricsData contains the metrics data from an HTTP server
type HTTPServerMetricsData struct {
	QueuedRequests        int64   `json:"queuedRequests"`
	RejectedRequests      int64   `json:"rejectedRequests"`
	TotalRequests         int64   `json:"totalRequests"`
	SuccessfulRequests    int64   `json:"successfulRequests"`
	ErrorRequests         int64   `json:"errorRequests"`
	QueueCapacity         int     `json:"queueCapacity"`
	QueueUtilization      float64 `json:"queueUtilization"`
	Workers               int     `json:"workers"`
	CurrentCCU            int     `json:"currentCCU"`
	CCUUtilization        float64 `json:"ccuUtilization"`
	BytesSent             int64   `json:"bytesSent"`
	BytesReceived         int64   `json:"bytesReceived"`
	// Little's Law metrics
	AverageLatencyMs      float64 `json:"averageLatencyMs"`
	ArrivalRate           float64 `json:"arrivalRate"`
	ExpectedQueueLength   float64 `json:"expectedQueueLength"`
	LittlesLawValidation  float64 `json:"littlesLawValidation"`
}

// MetricsCollector collects metrics from various sources
type MetricsCollector struct {
	mu              sync.RWMutex
	httpServers     map[string]HTTPServerMetricsProvider
	profiler        *profiling.RuntimeProfiler
	classifier      *profiling.WorkClassifier
	goroutineProf   *profiling.GoroutineProfiler
	bottleneckDet   *profiling.BottleneckDetector
	lastGCStats     runtime.MemStats
	lastAllocCount  uint64
	lastGCCount      uint32
	lastGCTime       time.Time
}

var (
	globalCollector *MetricsCollector
	collectorOnce  sync.Once
)

// GetMetricsCollector returns the global metrics collector
func GetMetricsCollector() *MetricsCollector {
	collectorOnce.Do(func() {
		globalCollector = &MetricsCollector{
			httpServers: make(map[string]HTTPServerMetricsProvider),
		}
		// Initialize profiling components
		globalCollector.classifier = profiling.NewWorkClassifier()
		globalCollector.goroutineProf = profiling.NewGoroutineProfiler(globalCollector.classifier)
		globalCollector.bottleneckDet = profiling.NewBottleneckDetector()
		globalCollector.profiler = profiling.NewRuntimeProfiler(5*time.Second, globalCollector.classifier, globalCollector.goroutineProf)
	})
	return globalCollector
}

// RegisterHTTPServer registers an HTTP server for metrics collection
func (mc *MetricsCollector) RegisterHTTPServer(name string, server HTTPServerMetricsProvider) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.httpServers[name] = server
}

// UnregisterHTTPServer unregisters an HTTP server
func (mc *MetricsCollector) UnregisterHTTPServer(name string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	delete(mc.httpServers, name)
}

// StartProfiling starts the profiling system
func (mc *MetricsCollector) StartProfiling(ctx interface{}) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if mc.profiler != nil && !mc.profiler.IsRunning() {
		// Use context.Background() if ctx is nil or invalid
		var profilerCtx context.Context
		if ctx != nil {
			if c, ok := ctx.(context.Context); ok && c != nil {
				profilerCtx = c
			} else {
				profilerCtx = context.Background()
			}
		} else {
			profilerCtx = context.Background()
		}
		mc.profiler.Start(profilerCtx)
	}
}

// CollectAllMetrics collects metrics from all sources
func (mc *MetricsCollector) CollectAllMetrics() *AllMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metrics := &AllMetrics{
		Timestamp: time.Now(),
	}

	// Collect concurrency metrics (existing)
	concurrencyMetrics := concurrency.GetDashboardMetrics()
	metrics.Executors = concurrencyMetrics.Executors
	metrics.WorkerPools = concurrencyMetrics.WorkerPools

	// Collect HTTP server metrics
	metrics.HTTPServers = make([]HTTPServerMetrics, 0, len(mc.httpServers))
	for name, server := range mc.httpServers {
		serverMetricsData := server.Metrics()
		metrics.HTTPServers = append(metrics.HTTPServers, HTTPServerMetrics{
			Name:                 name,
			QueuedRequests:       serverMetricsData.QueuedRequests,
			RejectedRequests:     serverMetricsData.RejectedRequests,
			TotalRequests:        serverMetricsData.TotalRequests,
			SuccessfulRequests:   serverMetricsData.SuccessfulRequests,
			ErrorRequests:        serverMetricsData.ErrorRequests,
			QueueCapacity:        serverMetricsData.QueueCapacity,
			QueueUtilization:     serverMetricsData.QueueUtilization,
			Workers:              serverMetricsData.Workers,
			CurrentCCU:           serverMetricsData.CurrentCCU,
			CCUUtilization:       serverMetricsData.CCUUtilization,
			BytesSent:            serverMetricsData.BytesSent,
			BytesReceived:        serverMetricsData.BytesReceived,
			AverageLatencyMs:     serverMetricsData.AverageLatencyMs,
			ArrivalRate:          serverMetricsData.ArrivalRate,
			ExpectedQueueLength:  serverMetricsData.ExpectedQueueLength,
			LittlesLawValidation: serverMetricsData.LittlesLawValidation,
		})
	}

	// Collect profiling metrics
	if mc.profiler != nil {
		profilingData := mc.profiler.GetProfilingData()
		goroutineStats := mc.goroutineProf.GetStats()

		metrics.Profiling = &ProfilingMetrics{
			WorkClassification: profilingData.WorkClassification,
			Goroutines: GoroutineMetrics{
				Total:      goroutineStats.Total,
				ByState:    convertGoroutineStates(goroutineStats.ByState),
				ByWorkType: convertWorkTypes(goroutineStats.ByWorkType),
				MixedWork:  goroutineStats.MixedWork,
			},
		}

		// Detect bottlenecks
		if len(metrics.HTTPServers) > 0 {
			// Use first server for bottleneck detection
			firstServer := metrics.HTTPServers[0]
			profilingMetrics := &profiling.ServerMetricsForProfiling{
				QueuedRequests:     firstServer.QueuedRequests,
				RejectedRequests:   firstServer.RejectedRequests,
				QueueCapacity:      firstServer.QueueCapacity,
				Workers:            firstServer.Workers,
				QueueUtilization:   firstServer.QueueUtilization,
				CurrentCCU:         firstServer.CurrentCCU,
				CCUUtilization:     firstServer.CCUUtilization,
				IOBoundWorkersActive:  int(profilingData.WorkClassification.IOBound.ActiveWorkers),
				CPUBoundWorkersActive: int(profilingData.WorkClassification.CPUBound.ActiveWorkers),
				MixedWorkersActive:    int(profilingData.WorkClassification.Mixed.ActiveWorkers),
			}
			metrics.Profiling.Bottlenecks = mc.bottleneckDet.Detect(profilingMetrics, &goroutineStats)
		}
	}

	// Collect runtime metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	maxProcs := runtime.GOMAXPROCS(0)
	actualOSThreads := getOSThreadCount() // Get actual OS thread count from system
	metrics.Runtime = &RuntimeMetrics{
		Goroutines:      runtime.NumGoroutine(),
		NumCPU:          runtime.NumCPU(),
		GOMAXPROCS:      maxProcs,
		OSThreads:       actualOSThreads, // Actual OS threads from system (more accurate than GOMAXPROCS)
		Alloc:           memStats.Alloc,
		TotalAlloc:      memStats.TotalAlloc,
		Sys:             memStats.Sys,
		Mallocs:         memStats.Mallocs,
		Frees:           memStats.Frees,
		NumGC:           memStats.NumGC,
		PauseTotalNs:    memStats.PauseTotalNs,
		LastGC:          time.Unix(0, int64(memStats.LastGC)),
	}

	// Calculate allocation rate
	if !mc.lastGCTime.IsZero() {
		deltaTime := time.Since(mc.lastGCTime).Seconds()
		if deltaTime > 0 {
			deltaAllocs := memStats.Mallocs - mc.lastAllocCount
			if deltaAllocs >= 0 {
				metrics.Runtime.AllocRate = float64(deltaAllocs) / deltaTime
			} else {
				metrics.Runtime.AllocRate = 0.0
			}
			deltaGC := int64(memStats.NumGC) - int64(mc.lastGCCount)
			if deltaGC >= 0 {
				metrics.Runtime.GCRate = float64(deltaGC) / deltaTime
			} else {
				metrics.Runtime.GCRate = 0.0
			}
		} else {
			// If deltaTime is 0 or negative, set rates to 0
			metrics.Runtime.AllocRate = 0.0
			metrics.Runtime.GCRate = 0.0
		}
	} else {
		// First run, set rates to 0
		metrics.Runtime.AllocRate = 0.0
		metrics.Runtime.GCRate = 0.0
	}
	mc.lastAllocCount = memStats.Mallocs
	mc.lastGCCount = memStats.NumGC
	mc.lastGCTime = time.Now()

	// Sanitize all float64 values to prevent NaN/Inf in JSON
	sanitizeMetrics(metrics)

	return metrics
}

// sanitizeFloat64 replaces NaN and Inf with 0.0
func sanitizeFloat64(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0.0
	}
	return v
}

// sanitizeMetrics sanitizes all float64 values in metrics to prevent JSON encoding errors
func sanitizeMetrics(metrics *AllMetrics) {
	// Sanitize HTTP server metrics
	for i := range metrics.HTTPServers {
		metrics.HTTPServers[i].QueueUtilization = sanitizeFloat64(metrics.HTTPServers[i].QueueUtilization)
		metrics.HTTPServers[i].CCUUtilization = sanitizeFloat64(metrics.HTTPServers[i].CCUUtilization)
		// Sanitize Little's Law metrics
		metrics.HTTPServers[i].AverageLatencyMs = sanitizeFloat64(metrics.HTTPServers[i].AverageLatencyMs)
		metrics.HTTPServers[i].ArrivalRate = sanitizeFloat64(metrics.HTTPServers[i].ArrivalRate)
		metrics.HTTPServers[i].ExpectedQueueLength = sanitizeFloat64(metrics.HTTPServers[i].ExpectedQueueLength)
		metrics.HTTPServers[i].LittlesLawValidation = sanitizeFloat64(metrics.HTTPServers[i].LittlesLawValidation)
	}

	// Sanitize profiling metrics
	if metrics.Profiling != nil {
		metrics.Profiling.WorkClassification.IOBound.Utilization = sanitizeFloat64(metrics.Profiling.WorkClassification.IOBound.Utilization)
		metrics.Profiling.WorkClassification.CPUBound.Utilization = sanitizeFloat64(metrics.Profiling.WorkClassification.CPUBound.Utilization)
		metrics.Profiling.WorkClassification.Mixed.Utilization = sanitizeFloat64(metrics.Profiling.WorkClassification.Mixed.Utilization)
	}

	// Sanitize runtime metrics
	if metrics.Runtime != nil {
		metrics.Runtime.AllocRate = sanitizeFloat64(metrics.Runtime.AllocRate)
		metrics.Runtime.GCRate = sanitizeFloat64(metrics.Runtime.GCRate)
	}
}

// AllMetrics contains all collected metrics
type AllMetrics struct {
	Timestamp   time.Time                    `json:"timestamp"`
	Executors   []concurrency.ExecutorMetrics   `json:"executors"`
	WorkerPools []concurrency.WorkerPoolMetrics `json:"workerPools"`
	HTTPServers []HTTPServerMetrics          `json:"httpServers"`
	Profiling   *ProfilingMetrics            `json:"profiling,omitempty"`
	Runtime     *RuntimeMetrics              `json:"runtime,omitempty"`
}

// HTTPServerMetrics contains HTTP server metrics
type HTTPServerMetrics struct {
	Name                 string  `json:"name"`
	QueuedRequests       int64   `json:"queuedRequests"`
	RejectedRequests     int64   `json:"rejectedRequests"`
	TotalRequests        int64   `json:"totalRequests"`
	SuccessfulRequests   int64   `json:"successfulRequests"`
	ErrorRequests        int64   `json:"errorRequests"`
	QueueCapacity        int     `json:"queueCapacity"`
	QueueUtilization     float64 `json:"queueUtilization"`
	Workers              int     `json:"workers"`
	CurrentCCU           int     `json:"currentCCU"`
	CCUUtilization       float64 `json:"ccuUtilization"`
	BytesSent            int64   `json:"bytesSent"`
	BytesReceived        int64   `json:"bytesReceived"`
	// Little's Law metrics
	AverageLatencyMs     float64 `json:"averageLatencyMs"`
	ArrivalRate          float64 `json:"arrivalRate"`
	ExpectedQueueLength  float64 `json:"expectedQueueLength"`
	LittlesLawValidation float64 `json:"littlesLawValidation"`
}

// ProfilingMetrics contains profiling metrics
type ProfilingMetrics struct {
	WorkClassification profiling.WorkClassification `json:"workClassification"`
	Goroutines         GoroutineMetrics              `json:"goroutines"`
	Bottlenecks        []profiling.Bottleneck       `json:"bottlenecks"`
}

// GoroutineMetrics contains goroutine statistics
type GoroutineMetrics struct {
	Total      int            `json:"total"`
	ByState    map[string]int `json:"byState"`
	ByWorkType map[string]int `json:"byWorkType"`
	MixedWork  int            `json:"mixedWork"`
}

// RuntimeMetrics contains Go runtime metrics
type RuntimeMetrics struct {
	Goroutines   int       `json:"goroutines"`
	NumCPU       int       `json:"numCPU"`
	GOMAXPROCS   int       `json:"goMaxProcs"`
	OSThreads    int       `json:"osThreads"`   // Actual number of OS threads (may be > GOMAXPROCS due to blocking I/O, CGO, runtime threads)
	Alloc        uint64    `json:"alloc"`        // bytes
	TotalAlloc   uint64    `json:"totalAlloc"`  // bytes
	Sys          uint64    `json:"sys"`         // bytes
	Mallocs      uint64    `json:"mallocs"`     // total allocations
	Frees        uint64    `json:"frees"`      // total frees
	NumGC        uint32    `json:"numGC"`       // GC cycles
	PauseTotalNs uint64    `json:"pauseTotalNs"` // total GC pause time
	LastGC       time.Time `json:"lastGC"`
	AllocRate    float64   `json:"allocRate"`   // allocations per second
	GCRate       float64   `json:"gcRate"`      // GC cycles per second
}

// Helper functions
func convertGoroutineStates(states map[profiling.GoroutineState]int) map[string]int {
	result := make(map[string]int)
	for state, count := range states {
		result[string(state)] = count
	}
	return result
}

func convertWorkTypes(workTypes map[profiling.WorkType]int) map[string]int {
	result := make(map[string]int)
	for workType, count := range workTypes {
		result[string(workType)] = count
	}
	return result
}
