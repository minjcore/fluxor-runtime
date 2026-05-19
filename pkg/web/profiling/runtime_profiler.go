package profiling

import (
	"context"
	"sync"
	"time"
)

// RuntimeProfiler performs periodic runtime profiling
type RuntimeProfiler struct {
	interval   time.Duration
	classifier *WorkClassifier
	profiler   *GoroutineProfiler
	mu         sync.RWMutex
	running    bool
	stopChan   chan struct{}
}

// NewRuntimeProfiler creates a new runtime profiler
func NewRuntimeProfiler(interval time.Duration, classifier *WorkClassifier, profiler *GoroutineProfiler) *RuntimeProfiler {
	return &RuntimeProfiler{
		interval:   interval,
		classifier: classifier,
		profiler:   profiler,
		stopChan:   make(chan struct{}),
	}
}

// Start starts the profiler
func (rp *RuntimeProfiler) Start(ctx context.Context) {
	rp.mu.Lock()
	if rp.running {
		rp.mu.Unlock()
		return
	}
	rp.running = true
	rp.mu.Unlock()

	// Use context.Background() if ctx is nil
	if ctx == nil {
		ctx = context.Background()
	}

	go rp.run(ctx)
}

// Stop stops the profiler
func (rp *RuntimeProfiler) Stop() {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	
	if !rp.running {
		return
	}
	
	rp.running = false
	close(rp.stopChan)
}

// IsRunning returns whether the profiler is running
func (rp *RuntimeProfiler) IsRunning() bool {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	return rp.running
}

// run runs the profiling loop
func (rp *RuntimeProfiler) run(ctx context.Context) {
	ticker := time.NewTicker(rp.interval)
	defer ticker.Stop()

	// Initial profile
	rp.Profile()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rp.stopChan:
			return
		case <-ticker.C:
			rp.Profile()
		}
	}
}

// Profile performs a single profiling run
func (rp *RuntimeProfiler) Profile() error {
	// Capture all goroutine stacks
	goroutines := CaptureAllGoroutines()

	// Classify and update profiles
	for goroutineID, stackTrace := range goroutines {
		// Classify work type
		workType := rp.classifier.Classify(goroutineID, stackTrace)
		
		// Update profiler
		rp.profiler.Update(goroutineID, stackTrace, workType)
	}

	return nil
}

// GetProfilingData returns current profiling data
func (rp *RuntimeProfiler) GetProfilingData() *ProfilingData {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	// Get work type stats
	workTypeStats := rp.classifier.GetWorkTypeStats()

	// Get goroutine stats
	goroutineStats := rp.profiler.GetStats()

	// Calculate active workers by type
	ioBoundActive := workTypeStats[WorkTypeIOBound]
	cpuBoundActive := workTypeStats[WorkTypeCPUBound]
	mixedActive := workTypeStats[WorkTypeMixed]

	// Calculate utilization percentages, avoiding division by zero
	var ioUtilization, cpuUtilization, mixedUtilization float64
	if goroutineStats.Total > 0 {
		ioUtilization = float64(ioBoundActive) / float64(goroutineStats.Total) * 100.0
		cpuUtilization = float64(cpuBoundActive) / float64(goroutineStats.Total) * 100.0
		mixedUtilization = float64(mixedActive) / float64(goroutineStats.Total) * 100.0
	}

	return &ProfilingData{
		WorkClassification: WorkClassification{
			IOBound: WorkTypeStats{
				ActiveWorkers: ioBoundActive,
				QueuedTasks:    0, // Will be set by caller
				Utilization:    ioUtilization,
			},
			CPUBound: WorkTypeStats{
				ActiveWorkers: cpuBoundActive,
				QueuedTasks:    0, // Will be set by caller
				Utilization:    cpuUtilization,
			},
			Mixed: WorkTypeStats{
				ActiveWorkers: mixedActive,
				QueuedTasks:    0, // Will be set by caller
				Utilization:    mixedUtilization,
				Warning:        "",
			},
		},
		Goroutines: goroutineStats,
	}
}

// ProfilingData represents profiling data
type ProfilingData struct {
	WorkClassification WorkClassification `json:"workClassification"`
	Goroutines         GoroutineStats     `json:"goroutines"`
}

// WorkClassification represents work type classification
type WorkClassification struct {
	IOBound  WorkTypeStats `json:"ioBound"`
	CPUBound WorkTypeStats `json:"cpuBound"`
	Mixed    WorkTypeStats `json:"mixed"`
}

// WorkTypeStats represents statistics for a work type
type WorkTypeStats struct {
	ActiveWorkers int     `json:"activeWorkers"`
	QueuedTasks   int64   `json:"queuedTasks"`
	Utilization   float64 `json:"utilization"`
	Warning       string  `json:"warning,omitempty"`
}
