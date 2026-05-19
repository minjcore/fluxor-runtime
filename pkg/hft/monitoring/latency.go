package monitoring

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// LatencyMonitor tracks latency metrics with histogram
type LatencyMonitor struct {
	name string
	
	// Histogram buckets (nanoseconds)
	buckets      []int64
	bucketCounts []uint64
	
	// Summary statistics (cache-line padded)
	_            [7]uint64
	count        uint64
	_            [7]uint64
	sum          uint64
	_            [7]uint64
	min          uint64
	_            [7]uint64
	max          uint64
	_            [7]uint64
	
	// Lock for percentile calculation
	mu           sync.RWMutex
}

// NewLatencyMonitor creates a new latency monitor
func NewLatencyMonitor(name string) *LatencyMonitor {
	// Buckets: 100ns, 500ns, 1µs, 5µs, 10µs, 50µs, 100µs, 500µs, 1ms, 5ms, 10ms, 50ms, 100ms
	buckets := []int64{
		100,      // 100ns
		500,      // 500ns
		1000,     // 1µs
		5000,     // 5µs
		10000,    // 10µs
		50000,    // 50µs
		100000,   // 100µs
		500000,   // 500µs
		1000000,  // 1ms
		5000000,  // 5ms
		10000000, // 10ms
		50000000, // 50ms
		100000000, // 100ms
	}
	
	return &LatencyMonitor{
		name:         name,
		buckets:      buckets,
		bucketCounts: make([]uint64, len(buckets)+1),
		min:          ^uint64(0), // Max uint64
	}
}

// Record records a latency measurement
func (lm *LatencyMonitor) Record(latencyNs uint64) {
	// Update count and sum
	atomic.AddUint64(&lm.count, 1)
	atomic.AddUint64(&lm.sum, latencyNs)
	
	// Update min
	for {
		oldMin := atomic.LoadUint64(&lm.min)
		if latencyNs >= oldMin {
			break
		}
		if atomic.CompareAndSwapUint64(&lm.min, oldMin, latencyNs) {
			break
		}
	}
	
	// Update max
	for {
		oldMax := atomic.LoadUint64(&lm.max)
		if latencyNs <= oldMax {
			break
		}
		if atomic.CompareAndSwapUint64(&lm.max, oldMax, latencyNs) {
			break
		}
	}
	
	// Update histogram
	for i, bucket := range lm.buckets {
		if int64(latencyNs) <= bucket {
			atomic.AddUint64(&lm.bucketCounts[i], 1)
			return
		}
	}
	
	// Overflow bucket
	atomic.AddUint64(&lm.bucketCounts[len(lm.buckets)], 1)
}

// RecordDuration records a duration
func (lm *LatencyMonitor) RecordDuration(d time.Duration) {
	lm.Record(uint64(d.Nanoseconds()))
}

// GetStats returns current statistics
func (lm *LatencyMonitor) GetStats() LatencyStats {
	count := atomic.LoadUint64(&lm.count)
	sum := atomic.LoadUint64(&lm.sum)
	min := atomic.LoadUint64(&lm.min)
	max := atomic.LoadUint64(&lm.max)
	
	var avg uint64
	if count > 0 {
		avg = sum / count
	}
	
	return LatencyStats{
		Name:  lm.name,
		Count: count,
		Min:   min,
		Max:   max,
		Avg:   avg,
		Sum:   sum,
		P50:   lm.getPercentile(0.50),
		P95:   lm.getPercentile(0.95),
		P99:   lm.getPercentile(0.99),
		P999:  lm.getPercentile(0.999),
	}
}

// getPercentile calculates percentile from histogram
func (lm *LatencyMonitor) getPercentile(p float64) uint64 {
	count := atomic.LoadUint64(&lm.count)
	if count == 0 {
		return 0
	}
	
	targetCount := uint64(float64(count) * p)
	var cumulative uint64
	
	for i, bucket := range lm.buckets {
		cumulative += atomic.LoadUint64(&lm.bucketCounts[i])
		if cumulative >= targetCount {
			return uint64(bucket)
		}
	}
	
	// Return max bucket if not found
	return uint64(lm.buckets[len(lm.buckets)-1])
}

// Reset resets all statistics
func (lm *LatencyMonitor) Reset() {
	atomic.StoreUint64(&lm.count, 0)
	atomic.StoreUint64(&lm.sum, 0)
	atomic.StoreUint64(&lm.min, ^uint64(0))
	atomic.StoreUint64(&lm.max, 0)
	
	for i := range lm.bucketCounts {
		atomic.StoreUint64(&lm.bucketCounts[i], 0)
	}
}

// LatencyStats contains latency statistics
type LatencyStats struct {
	Name  string
	Count uint64
	Min   uint64
	Max   uint64
	Avg   uint64
	Sum   uint64
	P50   uint64
	P95   uint64
	P99   uint64
	P999  uint64
}

// String returns formatted statistics
func (ls LatencyStats) String() string {
	return fmt.Sprintf("%s: count=%d min=%dns avg=%dns p50=%dns p95=%dns p99=%dns p999=%dns max=%dns",
		ls.Name, ls.Count, ls.Min, ls.Avg, ls.P50, ls.P95, ls.P99, ls.P999, ls.Max)
}

// FormatMicroseconds returns formatted statistics in microseconds
func (ls LatencyStats) FormatMicroseconds() string {
	return fmt.Sprintf("%s: count=%d min=%.1fµs avg=%.1fµs p50=%.1fµs p95=%.1fµs p99=%.1fµs p999=%.1fµs max=%.1fµs",
		ls.Name, ls.Count,
		float64(ls.Min)/1000, float64(ls.Avg)/1000,
		float64(ls.P50)/1000, float64(ls.P95)/1000,
		float64(ls.P99)/1000, float64(ls.P999)/1000,
		float64(ls.Max)/1000)
}

// SLAChecker checks latency against SLA targets
type SLAChecker struct {
	targets map[string]SLATarget
	mu      sync.RWMutex
}

// SLATarget defines SLA targets for a metric
type SLATarget struct {
	P50Target  uint64 // nanoseconds
	P95Target  uint64
	P99Target  uint64
	P999Target uint64
}

// NewSLAChecker creates a new SLA checker
func NewSLAChecker() *SLAChecker {
	return &SLAChecker{
		targets: make(map[string]SLATarget),
	}
}

// SetTarget sets SLA target for a metric
func (sc *SLAChecker) SetTarget(name string, target SLATarget) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.targets[name] = target
}

// Check checks if latency meets SLA
func (sc *SLAChecker) Check(stats LatencyStats) SLAResult {
	sc.mu.RLock()
	target, exists := sc.targets[stats.Name]
	sc.mu.RUnlock()
	
	if !exists {
		return SLAResult{
			Name:   stats.Name,
			Passed: true,
			Reason: "No SLA target defined",
		}
	}
	
	// Check each percentile
	violations := make([]string, 0)
	
	if stats.P50 > target.P50Target {
		violations = append(violations, fmt.Sprintf("P50: %dns > %dns", stats.P50, target.P50Target))
	}
	
	if stats.P95 > target.P95Target {
		violations = append(violations, fmt.Sprintf("P95: %dns > %dns", stats.P95, target.P95Target))
	}
	
	if stats.P99 > target.P99Target {
		violations = append(violations, fmt.Sprintf("P99: %dns > %dns", stats.P99, target.P99Target))
	}
	
	if stats.P999 > target.P999Target {
		violations = append(violations, fmt.Sprintf("P999: %dns > %dns", stats.P999, target.P999Target))
	}
	
	passed := len(violations) == 0
	reason := "All targets met"
	if !passed {
		reason = fmt.Sprintf("Violations: %v", violations)
	}
	
	return SLAResult{
		Name:       stats.Name,
		Passed:     passed,
		Violations: violations,
		Reason:     reason,
	}
}

// SLAResult contains SLA check result
type SLAResult struct {
	Name       string
	Passed     bool
	Violations []string
	Reason     string
}

// LatencyTracker tracks multiple latency monitors
type LatencyTracker struct {
	monitors map[string]*LatencyMonitor
	mu       sync.RWMutex
}

// NewLatencyTracker creates a new latency tracker
func NewLatencyTracker() *LatencyTracker {
	return &LatencyTracker{
		monitors: make(map[string]*LatencyMonitor),
	}
}

// GetMonitor gets or creates a latency monitor
func (lt *LatencyTracker) GetMonitor(name string) *LatencyMonitor {
	lt.mu.RLock()
	monitor, exists := lt.monitors[name]
	lt.mu.RUnlock()
	
	if exists {
		return monitor
	}
	
	lt.mu.Lock()
	defer lt.mu.Unlock()
	
	// Double-check after acquiring write lock
	monitor, exists = lt.monitors[name]
	if exists {
		return monitor
	}
	
	monitor = NewLatencyMonitor(name)
	lt.monitors[name] = monitor
	return monitor
}

// GetAllStats returns statistics for all monitors
func (lt *LatencyTracker) GetAllStats() []LatencyStats {
	lt.mu.RLock()
	defer lt.mu.RUnlock()
	
	stats := make([]LatencyStats, 0, len(lt.monitors))
	for _, monitor := range lt.monitors {
		stats = append(stats, monitor.GetStats())
	}
	
	return stats
}

// ResetAll resets all monitors
func (lt *LatencyTracker) ResetAll() {
	lt.mu.RLock()
	defer lt.mu.RUnlock()
	
	for _, monitor := range lt.monitors {
		monitor.Reset()
	}
}

// Timer provides convenient latency measurement
type Timer struct {
	monitor *LatencyMonitor
	start   time.Time
}

// Start starts a new timer
func (lm *LatencyMonitor) Start() *Timer {
	return &Timer{
		monitor: lm,
		start:   time.Now(),
	}
}

// Stop stops the timer and records latency
func (t *Timer) Stop() time.Duration {
	elapsed := time.Since(t.start)
	t.monitor.RecordDuration(elapsed)
	return elapsed
}

// Global latency tracker
var globalTracker = NewLatencyTracker()

// Track records latency for a named operation
func Track(name string, latencyNs uint64) {
	monitor := globalTracker.GetMonitor(name)
	monitor.Record(latencyNs)
}

// StartTimer starts a timer for a named operation
func StartTimer(name string) *Timer {
	monitor := globalTracker.GetMonitor(name)
	return monitor.Start()
}

// GetStats returns statistics for a named operation
func GetStats(name string) LatencyStats {
	monitor := globalTracker.GetMonitor(name)
	return monitor.GetStats()
}

// GetAllStats returns all statistics
func GetAllStats() []LatencyStats {
	return globalTracker.GetAllStats()
}

// ResetAll resets all monitors
func ResetAll() {
	globalTracker.ResetAll()
}
