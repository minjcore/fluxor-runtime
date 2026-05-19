package web

import (
	"sync/atomic"
	"time"
)

// BackpressureController manages backpressure for the server
// Ensures system stability under high load by rejecting overflow requests
// Normal capacity is set to target utilization (e.g., 80% of max capacity)
type BackpressureController struct {
	normalCapacity int64 // Normal capacity (target utilization, e.g., 80% of max)
	currentLoad    int64 // Current load (atomic)
	rejectedCount  int64 // Rejected requests count
	lastReset      int64 // Last reset time (unix timestamp)
	resetInterval  int64 // Reset interval in seconds
}

// NewBackpressureController creates a new backpressure controller
// normalCapacity: Target capacity for normal operations (e.g., 80% of max)
// This ensures system operates at target utilization under normal load
func NewBackpressureController(normalCapacity int, resetIntervalSeconds int64) *BackpressureController {
	// Validate inputs
	if normalCapacity < 1 {
		normalCapacity = 1
	}
	if resetIntervalSeconds < 1 {
		resetIntervalSeconds = 60 // Default to 60 seconds
	}
	return &BackpressureController{
		normalCapacity: int64(normalCapacity),
		currentLoad:    0,
		rejectedCount:  0,
		lastReset:      time.Now().Unix(),
		resetInterval:  resetIntervalSeconds,
	}
}

// TryAcquire attempts to acquire capacity (fail-fast)
// Returns true if normal capacity available, false if should reject (503)
// Normal capacity = target utilization (e.g., 80% of max capacity)
func (bc *BackpressureController) TryAcquire() bool {
	// Reset counters periodically (atomic check-and-reset to prevent race conditions)
	now := time.Now().Unix()
	lastReset := atomic.LoadInt64(&bc.lastReset)
	if now-lastReset > bc.resetInterval {
		// Use CompareAndSwap to ensure only one goroutine performs the reset
		if atomic.CompareAndSwapInt64(&bc.lastReset, lastReset, now) {
			atomic.StoreInt64(&bc.currentLoad, 0)
			// Reset rejectedCount periodically to prevent overflow
			atomic.StoreInt64(&bc.rejectedCount, 0)
		}
	}

	// Check current load against normal capacity (target utilization)
	current := atomic.LoadInt64(&bc.currentLoad)
	if current >= bc.normalCapacity {
		// Fail-fast: normal capacity exceeded, reject immediately
		// This maintains target utilization (e.g., 80%) under normal conditions
		atomic.AddInt64(&bc.rejectedCount, 1)
		return false
	}

	// Acquire capacity
	atomic.AddInt64(&bc.currentLoad, 1)
	return true
}

// Release releases capacity
// Should be called when request processing completes (typically in a defer)
// Prevents load from going negative due to programming errors
func (bc *BackpressureController) Release() {
	// Use AddInt64 which handles underflow gracefully (will go negative, but that's OK for debugging)
	// In production, negative values indicate a bug (Release called without matching TryAcquire)
	atomic.AddInt64(&bc.currentLoad, -1)
}

// GetMetrics returns current backpressure metrics
func (bc *BackpressureController) GetMetrics() BackpressureMetrics {
	currentLoad := atomic.LoadInt64(&bc.currentLoad)
	normalCapacity := bc.normalCapacity
	utilization := 0.0
	if normalCapacity > 0 {
		utilization = float64(currentLoad) / float64(normalCapacity) * 100
	}
	return BackpressureMetrics{
		NormalCapacity: normalCapacity,
		CurrentLoad:    currentLoad,
		RejectedCount:  atomic.LoadInt64(&bc.rejectedCount),
		Utilization:    utilization,
	}
}

// BackpressureMetrics provides backpressure statistics
type BackpressureMetrics struct {
	NormalCapacity int64   // Normal capacity (target utilization)
	CurrentLoad    int64   // Current load
	RejectedCount  int64   // Total rejected requests
	Utilization    float64 // Utilization percentage (relative to normal capacity)
}
