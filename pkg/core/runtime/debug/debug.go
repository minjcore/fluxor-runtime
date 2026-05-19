package debug

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// Manager provides debug information collection and management.
type Manager interface {
	// Enable enables debug mode.
	Enable() error

	// Disable disables debug mode.
	Disable() error

	// IsEnabled returns whether debug mode is enabled.
	IsEnabled() bool

	// StackTrace returns the current stack trace.
	StackTrace() []byte

	// GoroutineDump returns a dump of all goroutines.
	GoroutineDump() []byte

	// MemoryStats returns current memory statistics.
	MemoryStats() MemoryStats

	// GCStats returns garbage collection statistics.
	GCStats() GCStats

	// Collect collects all debug information.
	Collect(ctx context.Context) (*Info, error)

	// RegisterCollector registers a custom debug data collector.
	RegisterCollector(name string, collector Collector) error

	// UnregisterCollector unregisters a collector.
	UnregisterCollector(name string)

	// Stats returns statistics about debug operations.
	Stats() Stats

	// History returns the collection history if enabled.
	History() []*Info
}

// Collector is a function that collects custom debug data.
type Collector func(ctx context.Context) (interface{}, error)

// Config configures debug behavior.
type Config struct {
	// Enabled determines if debug mode is enabled by default.
	Enabled bool

	// OnCollect is called when debug information is collected.
	OnCollect func(info *Info)

	// OnCollectAsync is called asynchronously when debug information is collected.
	OnCollectAsync func(info *Info)

	// CollectTimeout is the maximum time to wait for collection to complete.
	// Zero means no timeout.
	CollectTimeout time.Duration

	// MaxGoroutines is the maximum number of goroutines to include in dumps.
	// Zero means no limit.
	MaxGoroutines int

	// IncludeMemoryStats determines if memory stats should be included.
	IncludeMemoryStats bool

	// IncludeGCStats determines if GC stats should be included.
	IncludeGCStats bool

	// IncludeStackTrace determines if stack traces should be included.
	IncludeStackTrace bool

	// IncludeGoroutineDump determines if goroutine dumps should be included.
	IncludeGoroutineDump bool

	// ParallelCollect controls whether collectors are run in parallel.
	// Defaults to false (sequential).
	ParallelCollect bool

	// HistorySize is the maximum number of collection history entries to keep.
	// Zero means no history is kept.
	HistorySize int

	// StackTraceDepth limits the depth of stack traces. Zero means no limit.
	StackTraceDepth int
}

// DefaultConfig returns the default debug configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:               false,
		CollectTimeout:        5 * time.Second,
		IncludeMemoryStats:    true,
		IncludeGCStats:        true,
		IncludeStackTrace:     true,
		IncludeGoroutineDump:  true,
		MaxGoroutines:         0, // no limit
		ParallelCollect:       false,
		HistorySize:           0, // no history by default
		StackTraceDepth:       0, // no limit
	}
}

// MemoryStats contains memory statistics.
type MemoryStats struct {
	// Alloc is bytes of allocated heap objects.
	Alloc uint64

	// TotalAlloc is cumulative bytes allocated for heap objects.
	TotalAlloc uint64

	// Sys is the total bytes of memory obtained from the OS.
	Sys uint64

	// Lookups is the number of pointer lookups performed by the runtime.
	Lookups uint64

	// Mallocs is the cumulative count of heap objects allocated.
	Mallocs uint64

	// Frees is the cumulative count of heap objects freed.
	Frees uint64

	// HeapAlloc is bytes of allocated heap objects.
	HeapAlloc uint64

	// HeapSys is bytes of heap memory obtained from the OS.
	HeapSys uint64

	// HeapIdle is bytes in idle (unused) spans.
	HeapIdle uint64

	// HeapInuse is bytes in in-use spans.
	HeapInuse uint64

	// HeapReleased is bytes of physical memory returned to the OS.
	HeapReleased uint64

	// HeapObjects is the number of allocated heap objects.
	HeapObjects uint64

	// StackInuse is bytes used for stack spans.
	StackInuse uint64

	// StackSys is bytes of stack memory obtained from the OS.
	StackSys uint64

	// MSpanInuse is bytes used for mspan structures.
	MSpanInuse uint64

	// MSpanSys is bytes of memory obtained from the OS for mspan structures.
	MSpanSys uint64

	// MCacheInuse is bytes used for mcache structures.
	MCacheInuse uint64

	// MCacheSys is bytes of memory obtained from the OS for mcache structures.
	MCacheSys uint64

	// BuckHashSys is bytes of memory in profiling bucket hash tables.
	BuckHashSys uint64

	// GCSys is bytes of memory in garbage collection system metadata.
	GCSys uint64

	// OtherSys is bytes of memory in miscellaneous off-heap runtime allocations.
	OtherSys uint64

	// NextGC is the target heap size of the next GC cycle.
	NextGC uint64

	// LastGC is the time the last garbage collection finished.
	LastGC time.Time

	// PauseTotalNs is the cumulative nanoseconds in GC stop-the-world pauses.
	PauseTotalNs uint64

	// NumGC is the number of completed GC cycles.
	NumGC uint32

	// NumForcedGC is the number of GC cycles that were forced by the application.
	NumForcedGC uint32

	// GCCPUFraction is the fraction of CPU time used by GC.
	GCCPUFraction float64
}

// GCStats contains garbage collection statistics.
type GCStats struct {
	// LastGC is when the last garbage collection finished.
	LastGC time.Time

	// NumGC is the number of completed GC cycles.
	NumGC uint64

	// PauseTotal is the cumulative time spent in GC pauses.
	PauseTotal time.Duration

	// Pause is the most recent GC pause durations.
	Pause []time.Duration

	// PauseQuantiles contains quantiles of recent GC pause durations.
	PauseQuantiles []time.Duration
}

// Info contains collected debug information.
type Info struct {
	// Timestamp is when the debug information was collected.
	Timestamp time.Time

	// Duration is how long the collection took.
	Duration time.Duration

	// Enabled indicates if debug mode was enabled.
	Enabled bool

	// StackTrace is the current stack trace, if included.
	StackTrace []byte

	// GoroutineDump is the goroutine dump, if included.
	GoroutineDump []byte

	// MemoryStats contains memory statistics, if included.
	MemoryStats *MemoryStats

	// GCStats contains GC statistics, if included.
	GCStats *GCStats

	// CustomData contains data from custom collectors.
	CustomData map[string]interface{}

	// GoVersion is the Go version.
	GoVersion string

	// NumGoroutines is the number of goroutines.
	NumGoroutines int

	// NumCPU is the number of logical CPUs.
	NumCPU int

	// CollectorErrors contains errors from individual collectors.
	CollectorErrors map[string]error
}

// Stats contains statistics about debug operations.
type Stats struct {
	// TotalCollections is the total number of debug information collections.
	TotalCollections int64

	// TotalErrors is the total number of collection errors.
	TotalErrors int64

	// LastCollectionTime is when the last collection completed.
	LastCollectionTime time.Time

	// LastCollectionDuration is how long the last collection took.
	LastCollectionDuration time.Duration

	// IsEnabled indicates if debug mode is currently enabled.
	IsEnabled bool

	// TotalCollectors is the number of registered custom collectors.
	TotalCollectors int64

	// AverageCollectionDuration is the average time taken for collections.
	AverageCollectionDuration time.Duration
}

// debugManager implements the Manager interface.
type debugManager struct {
	config     Config
	enabled    int32 // atomic
	mu         sync.RWMutex
	collectors map[string]Collector
	stats      Stats
	history    []*Info // collection history
}

// NewManager creates a new debug manager with the given configuration.
func NewManager(config Config) Manager {
	if config.CollectTimeout == 0 {
		config.CollectTimeout = 5 * time.Second
	}
	dm := &debugManager{
		config:     config,
		collectors: make(map[string]Collector),
	}
	if config.Enabled {
		atomic.StoreInt32(&dm.enabled, 1)
		dm.stats.IsEnabled = true
	}
	return dm
}

// Enable enables debug mode.
func (dm *debugManager) Enable() error {
	atomic.StoreInt32(&dm.enabled, 1)
	dm.mu.Lock()
	dm.stats.IsEnabled = true
	dm.mu.Unlock()
	return nil
}

// Disable disables debug mode.
func (dm *debugManager) Disable() error {
	atomic.StoreInt32(&dm.enabled, 0)
	dm.mu.Lock()
	dm.stats.IsEnabled = false
	dm.mu.Unlock()
	return nil
}

// IsEnabled returns whether debug mode is enabled.
func (dm *debugManager) IsEnabled() bool {
	return atomic.LoadInt32(&dm.enabled) == 1
}

// StackTrace returns the current stack trace.
func (dm *debugManager) StackTrace() []byte {
	return debug.Stack()
}

// GoroutineDump returns a dump of all goroutines.
// If MaxGoroutines is set in config, it will limit the output.
func (dm *debugManager) GoroutineDump() []byte {
	buf := make([]byte, 1024*1024) // 1MB buffer
	n := runtime.Stack(buf, true)
	dump := buf[:n]

	// Apply goroutine limit if configured
	if dm.config.MaxGoroutines > 0 {
		dump = dm.filterGoroutines(dump, dm.config.MaxGoroutines)
	}

	return dump
}

// filterGoroutines filters the goroutine dump to include only the first N goroutines.
func (dm *debugManager) filterGoroutines(dump []byte, max int) []byte {
	dumpStr := string(dump)
	lines := make([]string, 0)
	currentGoroutine := make([]string, 0)
	goroutineCount := 0

	for _, line := range splitLines(dumpStr) {
		if isGoroutineHeader(line) {
			if len(currentGoroutine) > 0 {
				lines = append(lines, currentGoroutine...)
				goroutineCount++
				if goroutineCount >= max {
					break
				}
			}
			currentGoroutine = []string{line}
		} else if len(currentGoroutine) > 0 {
			currentGoroutine = append(currentGoroutine, line)
		}
	}

	// Add the last goroutine if we haven't reached the limit
	if len(currentGoroutine) > 0 && goroutineCount < max {
		lines = append(lines, currentGoroutine...)
	}

	if len(lines) == 0 {
		return dump
	}

	result := ""
	for _, line := range lines {
		result += line + "\n"
	}
	return []byte(result)
}

// splitLines splits a string into lines, preserving empty lines.
func splitLines(s string) []string {
	lines := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// isGoroutineHeader checks if a line is a goroutine header.
// Goroutine headers start with "goroutine" followed by a number.
func isGoroutineHeader(line string) bool {
	if len(line) < 9 {
		return false
	}
	return line[:9] == "goroutine "
}

// MemoryStats returns current memory statistics.
func (dm *debugManager) MemoryStats() MemoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	var lastGC time.Time
	if m.LastGC > 0 {
		lastGC = time.Unix(0, int64(m.LastGC))
	}

	return MemoryStats{
		Alloc:         m.Alloc,
		TotalAlloc:    m.TotalAlloc,
		Sys:           m.Sys,
		Lookups:       m.Lookups,
		Mallocs:       m.Mallocs,
		Frees:         m.Frees,
		HeapAlloc:     m.HeapAlloc,
		HeapSys:       m.HeapSys,
		HeapIdle:      m.HeapIdle,
		HeapInuse:     m.HeapInuse,
		HeapReleased:  m.HeapReleased,
		HeapObjects:   m.HeapObjects,
		StackInuse:    m.StackInuse,
		StackSys:      m.StackSys,
		MSpanInuse:    m.MSpanInuse,
		MSpanSys:      m.MSpanSys,
		MCacheInuse:   m.MCacheInuse,
		MCacheSys:     m.MCacheSys,
		BuckHashSys:   m.BuckHashSys,
		GCSys:         m.GCSys,
		OtherSys:      m.OtherSys,
		NextGC:        m.NextGC,
		LastGC:        lastGC,
		PauseTotalNs:  m.PauseTotalNs,
		NumGC:         m.NumGC,
		NumForcedGC:   m.NumForcedGC,
		GCCPUFraction: m.GCCPUFraction,
	}
}

// GCStats returns garbage collection statistics.
func (dm *debugManager) GCStats() GCStats {
	var stats debug.GCStats
	debug.ReadGCStats(&stats)

	return GCStats{
		LastGC:        stats.LastGC,
		NumGC:         uint64(stats.NumGC),
		PauseTotal:    stats.PauseTotal,
		Pause:         stats.Pause,
		PauseQuantiles: stats.PauseQuantiles,
	}
}

// Collect collects all debug information.
func (dm *debugManager) Collect(ctx context.Context) (*Info, error) {
	if ctx == nil {
		return nil, NewError(ErrCodeNilContext, "context cannot be nil")
	}

	start := time.Now()

	// Use context timeout or default timeout
	timeout := dm.config.CollectTimeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
		if timeout <= 0 {
			timeout = dm.config.CollectTimeout
		}
	}

	collectCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	info := &Info{
		Timestamp:       time.Now(),
		Enabled:         dm.IsEnabled(),
		CustomData:      make(map[string]interface{}),
		CollectorErrors: make(map[string]error),
		GoVersion:       runtime.Version(),
		NumGoroutines:   runtime.NumGoroutine(),
		NumCPU:          runtime.NumCPU(),
	}

	// Collect stack trace if enabled
	if dm.config.IncludeStackTrace {
		info.StackTrace = dm.StackTrace()
	}

	// Collect goroutine dump if enabled
	if dm.config.IncludeGoroutineDump {
		info.GoroutineDump = dm.GoroutineDump()
	}

	// Collect memory stats if enabled
	if dm.config.IncludeMemoryStats {
		stats := dm.MemoryStats()
		info.MemoryStats = &stats
	}

	// Collect GC stats if enabled
	if dm.config.IncludeGCStats {
		gcStats := dm.GCStats()
		info.GCStats = &gcStats
	}

	// Collect custom data
	dm.mu.RLock()
	collectors := make(map[string]Collector)
	for name, collector := range dm.collectors {
		collectors[name] = collector
	}
	dm.mu.RUnlock()

	// Run collectors in parallel or sequentially based on config
	if dm.config.ParallelCollect {
		dm.collectParallel(collectCtx, collectors, info)
	} else {
		dm.collectSequential(collectCtx, collectors, info)
	}

	// Check if context was cancelled (timeout)
	select {
	case <-collectCtx.Done():
		// If we have collector errors due to timeout, return the first one
		if len(info.CollectorErrors) > 0 {
			for name, err := range info.CollectorErrors {
				if err == context.DeadlineExceeded || err == context.Canceled {
					return nil, NewError(ErrCodeCollectTimeout, fmt.Sprintf("collection timeout while collecting %s: %v", name, err))
				}
			}
		}
		return nil, NewError(ErrCodeCollectTimeout, fmt.Sprintf("collection timeout: %v", collectCtx.Err()))
	default:
	}

	// Calculate duration
	info.Duration = time.Since(start)

	// Update statistics
	atomic.AddInt64(&dm.stats.TotalCollections, 1)
	dm.mu.Lock()
	dm.stats.LastCollectionTime = time.Now()
	dm.stats.LastCollectionDuration = info.Duration

	// Calculate average duration
	totalCollections := atomic.LoadInt64(&dm.stats.TotalCollections)
	if totalCollections > 0 {
		// Simple moving average approximation
		avg := dm.stats.AverageCollectionDuration
		if avg == 0 {
			avg = info.Duration
		} else {
			// Exponential moving average with alpha = 0.1
			avg = time.Duration(float64(avg)*0.9 + float64(info.Duration)*0.1)
		}
		dm.stats.AverageCollectionDuration = avg
	}

	// Store in history if configured
	if dm.config.HistorySize > 0 {
		dm.history = append(dm.history, info)
		if len(dm.history) > dm.config.HistorySize {
			dm.history = dm.history[1:]
		}
	}
	dm.mu.Unlock()

	// Call callbacks
	if dm.config.OnCollect != nil {
		dm.config.OnCollect(info)
	}

	if dm.config.OnCollectAsync != nil {
		go dm.config.OnCollectAsync(info)
	}

	return info, nil
}

// collectSequential collects data from all collectors sequentially.
func (dm *debugManager) collectSequential(ctx context.Context, collectors map[string]Collector, info *Info) {
	for name, collector := range collectors {
		// Check if context is already done before calling collector
		select {
		case <-ctx.Done():
			info.CollectorErrors[name] = ctx.Err()
			return // Return early on timeout
		default:
		}

		// Call collector in a goroutine to allow timeout checking
		type result struct {
			data interface{}
			err  error
		}
		resultChan := make(chan result, 1)
		go func(n string, c Collector) {
			data, err := c(ctx)
			select {
			case resultChan <- result{data: data, err: err}:
				// Successfully sent
			case <-ctx.Done():
				// Context cancelled, don't block on send
				return
			}
		}(name, collector)

		select {
		case <-ctx.Done():
			info.CollectorErrors[name] = ctx.Err()
			return
		case res := <-resultChan:
			if res.err != nil {
				atomic.AddInt64(&dm.stats.TotalErrors, 1)
				info.CollectorErrors[name] = res.err
				// Continue collecting other data even if one collector fails
				info.CustomData[name] = map[string]interface{}{
					"error": res.err.Error(),
				}
			} else {
				info.CustomData[name] = res.data
			}
		}
	}
}

// collectParallel collects data from all collectors in parallel.
func (dm *debugManager) collectParallel(ctx context.Context, collectors map[string]Collector, info *Info) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, collector := range collectors {
		wg.Add(1)
		go func(n string, c Collector) {
			defer wg.Done()

			// Check if context is already done
			select {
			case <-ctx.Done():
				mu.Lock()
				info.CollectorErrors[n] = ctx.Err()
				mu.Unlock()
				return
			default:
			}

			// Call collector
			type result struct {
				data interface{}
				err  error
			}
			resultChan := make(chan result, 1)
			go func() {
				data, err := c(ctx)
				resultChan <- result{data: data, err: err}
			}()

			select {
			case <-ctx.Done():
				mu.Lock()
				info.CollectorErrors[n] = ctx.Err()
				mu.Unlock()
				return
			case res := <-resultChan:
				mu.Lock()
				if res.err != nil {
					atomic.AddInt64(&dm.stats.TotalErrors, 1)
					info.CollectorErrors[n] = res.err
					info.CustomData[n] = map[string]interface{}{
						"error": res.err.Error(),
					}
				} else {
					info.CustomData[n] = res.data
				}
				mu.Unlock()
			}
		}(name, collector)
	}

	wg.Wait()
}

// RegisterCollector registers a custom debug data collector.
func (dm *debugManager) RegisterCollector(name string, collector Collector) error {
	if name == "" {
		return NewError(ErrCodeInvalidCollector, "collector name cannot be empty")
	}
	if collector == nil {
		return NewError(ErrCodeInvalidCollector, "collector cannot be nil")
	}

	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, exists := dm.collectors[name]; exists {
		return NewError(ErrCodeCollectorExists, fmt.Sprintf("collector %s already registered", name))
	}

	dm.collectors[name] = collector
	atomic.AddInt64(&dm.stats.TotalCollectors, 1)

	return nil
}

// UnregisterCollector unregisters a collector.
func (dm *debugManager) UnregisterCollector(name string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, exists := dm.collectors[name]; exists {
		delete(dm.collectors, name)
		atomic.AddInt64(&dm.stats.TotalCollectors, -1)
	}
}

// Stats returns statistics about debug operations.
func (dm *debugManager) Stats() Stats {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	stats := dm.stats
	stats.IsEnabled = dm.IsEnabled()
	stats.TotalCollectors = atomic.LoadInt64(&dm.stats.TotalCollectors)

	return stats
}

// History returns the collection history if enabled.
// Returns nil if history is not enabled or empty.
func (dm *debugManager) History() []*Info {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if len(dm.history) == 0 {
		return nil
	}

	// Return a copy to prevent external modification
	history := make([]*Info, len(dm.history))
	copy(history, dm.history)
	return history
}
