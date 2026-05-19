package tcp

import (
	"sync/atomic"
	"time"
)

// BackpressureController manages backpressure for the TCP server.
// Normal capacity is set to target utilization baseline (e.g., queue + workers),
// rejecting overflow connections fail-fast to protect the runtime.
type BackpressureController struct {
	normalCapacity int64 // Normal capacity (target utilization baseline)
	currentLoad    int64 // Current load (atomic)
	rejectedCount  int64 // Rejected connections count
	lastReset      int64 // Last reset time (unix timestamp)
	resetInterval  int64 // Reset interval in seconds
}

// NewBackpressureController creates a new backpressure controller.
func NewBackpressureController(normalCapacity int, resetIntervalSeconds int64) *BackpressureController {
	if normalCapacity < 1 {
		normalCapacity = 1
	}
	if resetIntervalSeconds < 1 {
		resetIntervalSeconds = 60
	}
	return &BackpressureController{
		normalCapacity: int64(normalCapacity),
		currentLoad:    0,
		rejectedCount:  0,
		lastReset:      time.Now().Unix(),
		resetInterval:  resetIntervalSeconds,
	}
}

// TryAcquire attempts to acquire capacity (fail-fast).
// Returns true if normal capacity is available, false if it should reject.
func (bc *BackpressureController) TryAcquire() bool {
	now := time.Now().Unix()
	if now-bc.lastReset > bc.resetInterval {
		atomic.StoreInt64(&bc.currentLoad, 0)
		atomic.StoreInt64(&bc.lastReset, now)
	}

	current := atomic.LoadInt64(&bc.currentLoad)
	if current >= bc.normalCapacity {
		atomic.AddInt64(&bc.rejectedCount, 1)
		return false
	}

	atomic.AddInt64(&bc.currentLoad, 1)
	return true
}

// Release releases capacity.
func (bc *BackpressureController) Release() {
	atomic.AddInt64(&bc.currentLoad, -1)
}

// GetMetrics returns current backpressure metrics.
func (bc *BackpressureController) GetMetrics() BackpressureMetrics {
	currentLoad := atomic.LoadInt64(&bc.currentLoad)
	normal := atomic.LoadInt64(&bc.normalCapacity)
	util := 0.0
	if normal > 0 {
		util = float64(currentLoad) / float64(normal) * 100
	}
	return BackpressureMetrics{
		NormalCapacity: normal,
		CurrentLoad:    currentLoad,
		RejectedCount:  atomic.LoadInt64(&bc.rejectedCount),
		Utilization:    util,
	}
}

// BackpressureMetrics provides backpressure statistics.
type BackpressureMetrics struct {
	NormalCapacity int64   // Normal capacity (target utilization)
	CurrentLoad    int64   // Current load
	RejectedCount  int64   // Total rejected connections
	Utilization    float64 // Utilization percentage (relative to normal capacity)
}
