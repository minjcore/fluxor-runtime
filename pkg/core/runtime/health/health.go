package health

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// Manager provides health checking for runtime components.
type Manager interface {
	// Check performs a health check and returns the result.
	Check(ctx context.Context) (*Result, error)

	// RegisterChecker registers a custom health checker.
	RegisterChecker(name string, checker Checker) error

	// RegisterCheckerWithTimeout registers a custom health checker with a specific timeout.
	RegisterCheckerWithTimeout(name string, checker Checker, timeout time.Duration) error

	// UnregisterChecker unregisters a health checker.
	UnregisterChecker(name string)

	// RegisterThreshold registers a threshold-based health checker for runtime metrics.
	RegisterThreshold(name string, threshold Threshold) error

	// UnregisterThreshold unregisters a threshold-based health checker.
	UnregisterThreshold(name string)

	// ListCheckers returns the names of all registered checkers.
	ListCheckers() []string

	// ListThresholds returns the names of all registered thresholds.
	ListThresholds() []string

	// HasChecker checks if a checker with the given name exists.
	HasChecker(name string) bool

	// HasThreshold checks if a threshold with the given name exists.
	HasThreshold(name string) bool

	// IsHealthy returns true if all health checks pass.
	IsHealthy(ctx context.Context) (bool, error)

	// Stats returns statistics about health checks.
	Stats() Stats
}

// Checker is a function that performs a health check.
type Checker func(ctx context.Context) error

// CheckerInfo contains metadata about a registered checker.
type CheckerInfo struct {
	// Name is the unique identifier for the checker.
	Name string

	// Checker is the function to execute.
	Checker Checker

	// Timeout is the timeout for this specific checker.
	// Zero means use the default timeout.
	Timeout time.Duration

	// RegisteredAt is when the checker was registered.
	RegisteredAt time.Time
}

// Threshold defines thresholds for runtime metrics.
type Threshold struct {
	// MaxGoroutines is the maximum number of goroutines. Zero means no limit.
	MaxGoroutines int

	// MaxMemoryAlloc is the maximum memory allocation in bytes. Zero means no limit.
	MaxMemoryAlloc uint64

	// MaxMemorySys is the maximum system memory in bytes. Zero means no limit.
	MaxMemorySys uint64

	// MaxGCPause is the maximum GC pause duration. Zero means no limit.
	MaxGCPause time.Duration

	// MinNumGC is the minimum number of GC cycles. Used to detect if GC is stuck.
	// Zero means no check.
	MinNumGC uint32
}

// Config configures health check behavior.
type Config struct {
	// CheckTimeout is the maximum time to wait for a health check to complete.
	// Zero means no timeout.
	CheckTimeout time.Duration

	// OnCheck is called when a health check completes.
	OnCheck func(result *Result)

	// OnCheckAsync is called asynchronously when a health check completes.
	OnCheckAsync func(result *Result)

	// IncludeMemoryCheck determines if memory health checks should be performed.
	IncludeMemoryCheck bool

	// IncludeGCCheck determines if GC health checks should be performed.
	IncludeGCCheck bool

	// IncludeGoroutineCheck determines if goroutine health checks should be performed.
	IncludeGoroutineCheck bool

	// ParallelCheck controls whether health checkers are run in parallel.
	// Defaults to true.
	ParallelCheck bool
}

// DefaultConfig returns the default health check configuration.
func DefaultConfig() Config {
	return Config{
		CheckTimeout:          5 * time.Second,
		IncludeMemoryCheck:    true,
		IncludeGCCheck:        true,
		IncludeGoroutineCheck: true,
		ParallelCheck:         true,
	}
}

// Result represents the result of a health check.
type Result struct {
	// Timestamp is when the health check was performed.
	Timestamp time.Time

	// Overall is the overall health status.
	Overall Status

	// Healthy is true if all checks passed.
	Healthy bool

	// Checks contains results from individual checkers.
	Checks map[string]CheckResult

	// Runtime contains runtime health information.
	Runtime *RuntimeHealth

	// Message provides additional information about the health status.
	Message string
}

// CheckResult represents the result of a single health check.
type CheckResult struct {
	// Status is the health status.
	Status Status

	// Message provides additional information.
	Message string

	// Duration is how long the check took.
	Duration time.Duration

	// Error is the error if the check failed.
	Error error
}

// Status represents health check status.
type Status string

const (
	// StatusHealthy indicates the component is healthy.
	StatusHealthy Status = "healthy"

	// StatusUnhealthy indicates the component is unhealthy.
	StatusUnhealthy Status = "unhealthy"

	// StatusDegraded indicates the component is degraded but still functional.
	StatusDegraded Status = "degraded"

	// StatusUnknown indicates the health status is unknown.
	StatusUnknown Status = "unknown"
)

// String returns the string representation of the status.
func (s Status) String() string {
	return string(s)
}

// RuntimeHealth contains runtime health information.
type RuntimeHealth struct {
	// NumGoroutines is the number of goroutines.
	NumGoroutines int

	// Memory contains memory health information.
	Memory *MemoryHealth

	// GC contains GC health information.
	GC *GCHealth

	// GoVersion is the Go version.
	GoVersion string

	// NumCPU is the number of logical CPUs.
	NumCPU int
}

// MemoryHealth contains memory health information.
type MemoryHealth struct {
	// Alloc is bytes of allocated heap objects.
	Alloc uint64

	// Sys is the total bytes of memory obtained from the OS.
	Sys uint64

	// HeapAlloc is bytes of allocated heap objects.
	HeapAlloc uint64

	// HeapSys is bytes of heap memory obtained from the OS.
	HeapSys uint64

	// NumGC is the number of completed GC cycles.
	NumGC uint32

	// LastGC is when the last garbage collection finished.
	LastGC time.Time

	// Healthy indicates if memory usage is within thresholds.
	Healthy bool

	// Message provides additional information about memory health.
	Message string
}

// GCHealth contains GC health information.
type GCHealth struct {
	// NumGC is the number of completed GC cycles.
	NumGC uint64

	// PauseTotal is the cumulative time spent in GC pauses.
	PauseTotal time.Duration

	// LastGC is when the last garbage collection finished.
	LastGC time.Time

	// MaxPause is the maximum GC pause duration in recent GC cycles.
	MaxPause time.Duration

	// Healthy indicates if GC is healthy.
	Healthy bool

	// Message provides additional information about GC health.
	Message string
}

// Stats contains statistics about health checks.
type Stats struct {
	// TotalChecks is the total number of health checks performed.
	TotalChecks int64

	// TotalHealthy is the total number of times health checks passed.
	TotalHealthy int64

	// TotalUnhealthy is the total number of times health checks failed.
	TotalUnhealthy int64

	// LastCheckTime is when the last health check completed.
	LastCheckTime time.Time

	// LastCheckDuration is how long the last health check took.
	LastCheckDuration time.Duration

	// TotalCheckers is the number of registered custom checkers.
	TotalCheckers int64

	// TotalThresholds is the number of registered thresholds.
	TotalThresholds int64
}

// healthManager implements the Manager interface.
type healthManager struct {
	config Config

	mu         sync.RWMutex
	checkers   map[string]CheckerInfo
	thresholds map[string]Threshold
	stats      Stats
}

// NewManager creates a new health manager with the given configuration.
func NewManager(config Config) Manager {
	if config.CheckTimeout == 0 {
		config.CheckTimeout = 5 * time.Second
	}

	return &healthManager{
		config:     config,
		checkers:   make(map[string]CheckerInfo),
		thresholds: make(map[string]Threshold),
	}
}

// Check performs a health check and returns the result.
func (hm *healthManager) Check(ctx context.Context) (*Result, error) {
	if ctx == nil {
		return nil, NewError(ErrCodeNilContext, "context cannot be nil")
	}

	start := time.Now()

	// Use context timeout or default timeout
	timeout := hm.config.CheckTimeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
		if timeout <= 0 {
			timeout = hm.config.CheckTimeout
		}
	}

	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result := &Result{
		Timestamp: time.Now(),
		Overall:   StatusUnknown,
		Healthy:   true,
		Checks:    make(map[string]CheckResult),
	}

	// Collect runtime health information
	hm.collectRuntimeHealth(result, checkCtx)

	// Run custom checkers
	hm.runCheckers(checkCtx, result)

	// Check thresholds
	hm.checkThresholds(result)

	// Determine overall status
	hm.determineOverallStatus(result)

	// Update statistics
	duration := time.Since(start)
	atomic.AddInt64(&hm.stats.TotalChecks, 1)
	hm.mu.Lock()
	if result.Healthy {
		atomic.AddInt64(&hm.stats.TotalHealthy, 1)
	} else {
		atomic.AddInt64(&hm.stats.TotalUnhealthy, 1)
	}
	hm.stats.LastCheckTime = time.Now()
	hm.stats.LastCheckDuration = duration
	hm.mu.Unlock()

	// Call callbacks
	if hm.config.OnCheck != nil {
		hm.config.OnCheck(result)
	}

	if hm.config.OnCheckAsync != nil {
		go hm.config.OnCheckAsync(result)
	}

	return result, nil
}

// collectRuntimeHealth collects runtime health information.
func (hm *healthManager) collectRuntimeHealth(result *Result, ctx context.Context) {
	runtimeHealth := &RuntimeHealth{
		GoVersion:     runtime.Version(),
		NumGoroutines: runtime.NumGoroutine(),
		NumCPU:        runtime.NumCPU(),
	}

	if hm.config.IncludeMemoryCheck || hm.config.IncludeGCCheck {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		var lastGC time.Time
		if m.LastGC > 0 {
			lastGC = time.Unix(0, int64(m.LastGC))
		}

		if hm.config.IncludeMemoryCheck {
			runtimeHealth.Memory = &MemoryHealth{
				Alloc:     m.Alloc,
				Sys:       m.Sys,
				HeapAlloc: m.HeapAlloc,
				HeapSys:   m.HeapSys,
				NumGC:     m.NumGC,
				LastGC:    lastGC,
				Healthy:   true,
			}
		}

		if hm.config.IncludeGCCheck {
			var gcStats debug.GCStats
			debug.ReadGCStats(&gcStats)

			maxPause := time.Duration(0)
			if len(gcStats.Pause) > 0 {
				for _, pause := range gcStats.Pause {
					if pause > maxPause {
						maxPause = pause
					}
				}
			}

			runtimeHealth.GC = &GCHealth{
				NumGC:     uint64(gcStats.NumGC),
				PauseTotal: gcStats.PauseTotal,
				LastGC:    gcStats.LastGC,
				MaxPause:  maxPause,
				Healthy:   true,
			}
		}
	}

	result.Runtime = runtimeHealth
}

// runCheckers runs all registered custom checkers.
func (hm *healthManager) runCheckers(ctx context.Context, result *Result) {
	hm.mu.RLock()
	checkers := make(map[string]CheckerInfo)
	for name, info := range hm.checkers {
		checkers[name] = info
	}
	hm.mu.RUnlock()

	if len(checkers) == 0 {
		return
	}

	if hm.config.ParallelCheck {
		var wg sync.WaitGroup
		var mu sync.Mutex

		for name, checkerInfo := range checkers {
			wg.Add(1)
			go func(name string, info CheckerInfo) {
				defer wg.Done()
				checkResult := hm.runCheckerWithInfo(ctx, name, info)
				mu.Lock()
				result.Checks[name] = checkResult
				if checkResult.Status == StatusUnhealthy {
					result.Healthy = false
				} else if checkResult.Status == StatusDegraded {
					if result.Overall == StatusHealthy {
						result.Overall = StatusDegraded
					}
				}
				mu.Unlock()
			}(name, checkerInfo)
		}

		wg.Wait()
	} else {
		for name, checkerInfo := range checkers {
			select {
			case <-ctx.Done():
				result.Checks[name] = CheckResult{
					Status:   StatusUnknown,
					Message:  "health check timeout",
					Duration: 0,
					Error:    ctx.Err(),
				}
				result.Healthy = false
				return
			default:
				checkResult := hm.runCheckerWithInfo(ctx, name, checkerInfo)
				result.Checks[name] = checkResult
				if checkResult.Status == StatusUnhealthy {
					result.Healthy = false
				} else if checkResult.Status == StatusDegraded {
					if result.Overall == StatusHealthy {
						result.Overall = StatusDegraded
					}
				}
			}
		}
	}
}

// runCheckerWithInfo runs a single health checker with its metadata.
func (hm *healthManager) runCheckerWithInfo(ctx context.Context, name string, info CheckerInfo) CheckResult {
	// Use checker-specific timeout or default timeout
	timeout := info.Timeout
	if timeout == 0 {
		timeout = hm.config.CheckTimeout
	}

	// Create timeout context if needed
	checkCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		checkCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// Run checker in goroutine to properly handle timeouts
	type result struct {
		err      error
		duration time.Duration
	}

	resultChan := make(chan result, 1)
	start := time.Now()

	go func() {
		err := info.Checker(checkCtx)
		duration := time.Since(start)
		resultChan <- result{err: err, duration: duration}
	}()

	select {
	case <-checkCtx.Done():
		duration := time.Since(start)
		return CheckResult{
			Status:   StatusUnknown,
			Message:  fmt.Sprintf("health check timeout: %v", checkCtx.Err()),
			Duration: duration,
			Error:    checkCtx.Err(),
		}
	case res := <-resultChan:
		if res.err != nil {
			return CheckResult{
				Status:   StatusUnhealthy,
				Message:  res.err.Error(),
				Duration: res.duration,
				Error:    res.err,
			}
		}

		return CheckResult{
			Status:   StatusHealthy,
			Message:  "OK",
			Duration: res.duration,
		}
	}
}

// checkThresholds checks runtime metrics against registered thresholds.
func (hm *healthManager) checkThresholds(result *Result) {
	hm.mu.RLock()
	thresholds := make(map[string]Threshold)
	for name, threshold := range hm.thresholds {
		thresholds[name] = threshold
	}
	hm.mu.RUnlock()

	if len(thresholds) == 0 || result.Runtime == nil {
		return
	}

	for name, threshold := range thresholds {
		checkResult := hm.checkThreshold(name, threshold, result.Runtime)
		result.Checks[name] = checkResult
		if checkResult.Status == StatusUnhealthy {
			result.Healthy = false
		}
	}
}

// checkThreshold checks a single threshold.
func (hm *healthManager) checkThreshold(name string, threshold Threshold, runtimeHealth *RuntimeHealth) CheckResult {
	messages := make([]string, 0)

	// Check goroutine threshold
	if threshold.MaxGoroutines > 0 && runtimeHealth.NumGoroutines > threshold.MaxGoroutines {
		messages = append(messages, fmt.Sprintf("goroutines (%d) exceeds threshold (%d)",
			runtimeHealth.NumGoroutines, threshold.MaxGoroutines))
		// Don't modify memory health - goroutines are not memory!
	}

	// Check memory thresholds
	if runtimeHealth.Memory != nil {
		if threshold.MaxMemoryAlloc > 0 && runtimeHealth.Memory.Alloc > threshold.MaxMemoryAlloc {
			messages = append(messages, fmt.Sprintf("memory alloc (%d bytes) exceeds threshold (%d bytes)",
				runtimeHealth.Memory.Alloc, threshold.MaxMemoryAlloc))
			runtimeHealth.Memory.Healthy = false
		}

		if threshold.MaxMemorySys > 0 && runtimeHealth.Memory.Sys > threshold.MaxMemorySys {
			messages = append(messages, fmt.Sprintf("memory sys (%d bytes) exceeds threshold (%d bytes)",
				runtimeHealth.Memory.Sys, threshold.MaxMemorySys))
			runtimeHealth.Memory.Healthy = false
		}

		if threshold.MinNumGC > 0 && runtimeHealth.Memory.NumGC < threshold.MinNumGC {
			messages = append(messages, fmt.Sprintf("GC cycles (%d) below minimum (%d)",
				runtimeHealth.Memory.NumGC, threshold.MinNumGC))
		}
	}

	// Check GC thresholds
	if runtimeHealth.GC != nil {
		if threshold.MaxGCPause > 0 && runtimeHealth.GC.MaxPause > threshold.MaxGCPause {
			messages = append(messages, fmt.Sprintf("GC pause (%v) exceeds threshold (%v)",
				runtimeHealth.GC.MaxPause, threshold.MaxGCPause))
			runtimeHealth.GC.Healthy = false
		}
	}

	if len(messages) > 0 {
		message := fmt.Sprintf("threshold %s violated: %v", name, messages)
		return CheckResult{
			Status:  StatusUnhealthy,
			Message: message,
		}
	}

	return CheckResult{
		Status:  StatusHealthy,
		Message: "OK",
	}
}

// determineOverallStatus determines the overall health status.
func (hm *healthManager) determineOverallStatus(result *Result) {
	if len(result.Checks) == 0 {
		result.Overall = StatusUnknown
		result.Message = "no health checks configured"
		return
	}

	unhealthyCount := 0
	degradedCount := 0

	for _, check := range result.Checks {
		switch check.Status {
		case StatusUnhealthy:
			unhealthyCount++
		case StatusDegraded:
			degradedCount++
		}
	}

	if unhealthyCount > 0 {
		result.Overall = StatusUnhealthy
		result.Healthy = false
		result.Message = fmt.Sprintf("%d health checks failed", unhealthyCount)
	} else if degradedCount > 0 {
		result.Overall = StatusDegraded
		result.Message = fmt.Sprintf("%d health checks degraded", degradedCount)
	} else {
		result.Overall = StatusHealthy
		result.Message = "all health checks passed"
	}

	// Also check runtime health
	if result.Runtime != nil {
		if result.Runtime.Memory != nil && !result.Runtime.Memory.Healthy {
			result.Overall = StatusUnhealthy
			result.Healthy = false
			result.Message = result.Runtime.Memory.Message
		}
		if result.Runtime.GC != nil && !result.Runtime.GC.Healthy {
			if result.Overall == StatusHealthy {
				result.Overall = StatusDegraded
			}
			if result.Message == "" {
				result.Message = result.Runtime.GC.Message
			}
		}
	}
}

// RegisterChecker registers a custom health checker.
func (hm *healthManager) RegisterChecker(name string, checker Checker) error {
	return hm.RegisterCheckerWithTimeout(name, checker, 0)
}

// RegisterCheckerWithTimeout registers a custom health checker with a specific timeout.
func (hm *healthManager) RegisterCheckerWithTimeout(name string, checker Checker, timeout time.Duration) error {
	if name == "" {
		return NewError(ErrCodeInvalidChecker, "checker name cannot be empty")
	}
	if checker == nil {
		return NewError(ErrCodeInvalidChecker, "checker cannot be nil")
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	if _, exists := hm.checkers[name]; exists {
		return NewError(ErrCodeCheckerExists, fmt.Sprintf("checker %s already registered", name))
	}

	hm.checkers[name] = CheckerInfo{
		Name:         name,
		Checker:      checker,
		Timeout:      timeout,
		RegisteredAt: time.Now(),
	}
	atomic.AddInt64(&hm.stats.TotalCheckers, 1)

	return nil
}

// UnregisterChecker unregisters a health checker.
func (hm *healthManager) UnregisterChecker(name string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if _, exists := hm.checkers[name]; exists {
		delete(hm.checkers, name)
		atomic.AddInt64(&hm.stats.TotalCheckers, -1)
	}
}

// RegisterThreshold registers a threshold-based health checker.
func (hm *healthManager) RegisterThreshold(name string, threshold Threshold) error {
	if name == "" {
		return NewError(ErrCodeInvalidThreshold, "threshold name cannot be empty")
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	if _, exists := hm.thresholds[name]; exists {
		return NewError(ErrCodeThresholdExists, fmt.Sprintf("threshold %s already registered", name))
	}

	hm.thresholds[name] = threshold
	atomic.AddInt64(&hm.stats.TotalThresholds, 1)

	return nil
}

// UnregisterThreshold unregisters a threshold-based health checker.
func (hm *healthManager) UnregisterThreshold(name string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if _, exists := hm.thresholds[name]; exists {
		delete(hm.thresholds, name)
		atomic.AddInt64(&hm.stats.TotalThresholds, -1)
	}
}

// IsHealthy returns true if all health checks pass.
func (hm *healthManager) IsHealthy(ctx context.Context) (bool, error) {
	result, err := hm.Check(ctx)
	if err != nil {
		return false, err
	}
	return result.Healthy, nil
}

// ListCheckers returns the names of all registered checkers.
func (hm *healthManager) ListCheckers() []string {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	names := make([]string, 0, len(hm.checkers))
	for name := range hm.checkers {
		names = append(names, name)
	}

	return names
}

// ListThresholds returns the names of all registered thresholds.
func (hm *healthManager) ListThresholds() []string {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	names := make([]string, 0, len(hm.thresholds))
	for name := range hm.thresholds {
		names = append(names, name)
	}

	return names
}

// HasChecker checks if a checker with the given name exists.
func (hm *healthManager) HasChecker(name string) bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	_, exists := hm.checkers[name]
	return exists
}

// HasThreshold checks if a threshold with the given name exists.
func (hm *healthManager) HasThreshold(name string) bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	_, exists := hm.thresholds[name]
	return exists
}

// Stats returns statistics about health checks.
func (hm *healthManager) Stats() Stats {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	stats := hm.stats
	stats.TotalCheckers = atomic.LoadInt64(&hm.stats.TotalCheckers)
	stats.TotalThresholds = atomic.LoadInt64(&hm.stats.TotalThresholds)

	return stats
}
