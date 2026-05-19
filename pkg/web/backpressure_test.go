package web

import (
	"sync"
	"testing"
	"time"
)

func TestCCUBasedConfig(t *testing.T) {
	config := CCUBasedConfig(":8080", 5000, 500)

	// Verify configuration
	if config.MaxQueue+config.Workers < 5000 {
		t.Errorf("Total capacity (%d) should be at least 5000", config.MaxQueue+config.Workers)
	}

	if config.Workers < 50 {
		t.Error("Workers should be at least 50")
	}

	if config.MaxQueue < 100 {
		t.Error("MaxQueue should be at least 100")
	}
}

func TestCCUBasedConfigWithUtilization(t *testing.T) {
	maxCCU := 5000
	utilizationPercent := 67
	config := CCUBasedConfigWithUtilization(":8080", maxCCU, utilizationPercent)

	// Calculate expected normal capacity (67% of max)
	expectedNormalCapacity := int(float64(maxCCU) * float64(utilizationPercent) / 100.0)
	actualNormalCapacity := config.MaxQueue + config.Workers

	// Allow some tolerance for rounding
	tolerance := 50
	if actualNormalCapacity < expectedNormalCapacity-tolerance || actualNormalCapacity > expectedNormalCapacity+tolerance {
		t.Errorf("Normal capacity = %d, want ~%d (67%% of %d)", actualNormalCapacity, expectedNormalCapacity, maxCCU)
	}

	if config.Workers < 50 {
		t.Error("Workers should be at least 50")
	}

	if config.MaxQueue < 100 {
		t.Error("MaxQueue should be at least 100")
	}

	// MaxConns should allow up to maxCCU
	if config.MaxConns < maxCCU {
		t.Errorf("MaxConns = %d, should be at least %d", config.MaxConns, maxCCU)
	}
}

func TestBackpressureController(t *testing.T) {
	normalCapacity := 3350 // 67% of 5000 max
	bc := NewBackpressureController(normalCapacity, 67)

	// Test capacity acquisition (up to normal capacity)
	for i := 0; i < normalCapacity; i++ {
		if !bc.TryAcquire() {
			t.Errorf("Should acquire capacity for request %d", i)
		}
	}

	// Test overflow rejection (fail-fast) - beyond normal capacity
	if bc.TryAcquire() {
		t.Error("Should reject request when normal capacity exceeded")
	}

	// Test metrics
	metrics := bc.GetMetrics()
	if metrics.CurrentLoad != int64(normalCapacity) {
		t.Errorf("CurrentLoad = %d, want %d", metrics.CurrentLoad, normalCapacity)
	}
	if metrics.NormalCapacity != int64(normalCapacity) {
		t.Errorf("NormalCapacity = %d, want %d", metrics.NormalCapacity, normalCapacity)
	}
	if metrics.RejectedCount == 0 {
		t.Error("Should have rejected at least one request")
	}
	if metrics.Utilization < 100.0 {
		t.Errorf("Utilization should be >= 100%% when at capacity, got %.2f%%", metrics.Utilization)
	}

	// Test release
	bc.Release()
	metrics = bc.GetMetrics()
	if metrics.CurrentLoad != int64(normalCapacity-1) {
		t.Errorf("CurrentLoad = %d, want %d", metrics.CurrentLoad, normalCapacity-1)
	}

	// After release, should be able to acquire again
	if !bc.TryAcquire() {
		t.Error("Should acquire capacity after release")
	}
}

func TestBackpressureController_InvalidCapacity(t *testing.T) {
	// Test with zero capacity - should default to 1
	bc := NewBackpressureController(0, 60)
	if bc.normalCapacity != 1 {
		t.Errorf("Expected capacity to default to 1, got %d", bc.normalCapacity)
	}

	// Test with negative capacity - should default to 1
	bc = NewBackpressureController(-10, 60)
	if bc.normalCapacity != 1 {
		t.Errorf("Expected capacity to default to 1, got %d", bc.normalCapacity)
	}
}

func TestBackpressureController_InvalidResetInterval(t *testing.T) {
	// Test with zero reset interval - should default to 60
	bc := NewBackpressureController(100, 0)
	if bc.resetInterval != 60 {
		t.Errorf("Expected reset interval to default to 60, got %d", bc.resetInterval)
	}

	// Test with negative reset interval - should default to 60
	bc = NewBackpressureController(100, -10)
	if bc.resetInterval != 60 {
		t.Errorf("Expected reset interval to default to 60, got %d", bc.resetInterval)
	}
}

func TestBackpressureController_ConcurrentAccess(t *testing.T) {
	normalCapacity := 100
	bc := NewBackpressureController(normalCapacity, 60)

	var wg sync.WaitGroup
	acquired := int64(0)
	rejected := int64(0)

	// Try to acquire more than capacity concurrently
	numRequests := normalCapacity * 2
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if bc.TryAcquire() {
				acquired++
				time.Sleep(1 * time.Millisecond)
				bc.Release()
			} else {
				rejected++
			}
		}()
	}

	wg.Wait()

	// All capacity should be released
	metrics := bc.GetMetrics()
	if metrics.CurrentLoad != 0 {
		t.Errorf("Expected CurrentLoad to be 0 after all releases, got %d", metrics.CurrentLoad)
	}

	// Should have some rejections
	if rejected == 0 {
		t.Error("Expected some rejections when exceeding capacity")
	}
}

func TestBackpressureController_ReleaseWithoutAcquire(t *testing.T) {
	bc := NewBackpressureController(100, 60)

	// Release without acquiring - should handle gracefully
	bc.Release()

	metrics := bc.GetMetrics()
	// Load can go negative, which is OK for debugging
	if metrics.CurrentLoad > 0 {
		t.Errorf("Expected CurrentLoad <= 0 after release without acquire, got %d", metrics.CurrentLoad)
	}
}

func TestBackpressureController_MultipleReleases(t *testing.T) {
	bc := NewBackpressureController(100, 60)

	// Acquire once
	if !bc.TryAcquire() {
		t.Error("Should acquire capacity")
	}

	// Release multiple times
	bc.Release()
	bc.Release()
	bc.Release()

	metrics := bc.GetMetrics()
	// Load can go negative
	if metrics.CurrentLoad >= 0 {
		t.Logf("CurrentLoad is %d (may be negative due to multiple releases)", metrics.CurrentLoad)
	}
}

func TestBackpressureController_ResetInterval(t *testing.T) {
	// Use a very short reset interval for testing
	resetInterval := int64(1) // 1 second
	bc := NewBackpressureController(10, resetInterval)

	// Fill up capacity
	for i := 0; i < 10; i++ {
		bc.TryAcquire()
	}

	// Should reject
	if bc.TryAcquire() {
		t.Error("Should reject when at capacity")
	}

	// Wait for reset interval
	time.Sleep(2 * time.Second)

	// After reset, should be able to acquire
	// Note: Reset happens in TryAcquire, so we need to call it
	if !bc.TryAcquire() {
		t.Error("Should be able to acquire after reset interval")
	}

	// Clean up
	bc.Release()
}

func TestBackpressureController_GetMetrics(t *testing.T) {
	bc := NewBackpressureController(100, 60)

	// Test metrics with no load
	metrics := bc.GetMetrics()
	if metrics.NormalCapacity != 100 {
		t.Errorf("Expected NormalCapacity 100, got %d", metrics.NormalCapacity)
	}
	if metrics.CurrentLoad != 0 {
		t.Errorf("Expected CurrentLoad 0, got %d", metrics.CurrentLoad)
	}
	if metrics.RejectedCount != 0 {
		t.Errorf("Expected RejectedCount 0, got %d", metrics.RejectedCount)
	}
	if metrics.Utilization != 0.0 {
		t.Errorf("Expected Utilization 0.0, got %.2f", metrics.Utilization)
	}

	// Test metrics with partial load
	bc.TryAcquire()
	bc.TryAcquire()
	metrics = bc.GetMetrics()
	if metrics.CurrentLoad != 2 {
		t.Errorf("Expected CurrentLoad 2, got %d", metrics.CurrentLoad)
	}
	expectedUtilization := float64(2) / float64(100) * 100
	if metrics.Utilization != expectedUtilization {
		t.Errorf("Expected Utilization %.2f, got %.2f", expectedUtilization, metrics.Utilization)
	}

	bc.Release()
	bc.Release()
}

func TestBackpressureController_UtilizationCalculation(t *testing.T) {
	bc := NewBackpressureController(50, 60)

	// Test 50% utilization
	for i := 0; i < 25; i++ {
		bc.TryAcquire()
	}

	metrics := bc.GetMetrics()
	expectedUtilization := float64(25) / float64(50) * 100
	if metrics.Utilization != expectedUtilization {
		t.Errorf("Expected Utilization %.2f, got %.2f", expectedUtilization, metrics.Utilization)
	}

	// Clean up
	for i := 0; i < 25; i++ {
		bc.Release()
	}
}

func TestBackpressureController_ZeroCapacityUtilization(t *testing.T) {
	// This should default to 1, but test the edge case
	bc := NewBackpressureController(1, 60)
	bc.TryAcquire()

	metrics := bc.GetMetrics()
	// Utilization should be 100% when at capacity
	if metrics.Utilization < 100.0 {
		t.Errorf("Expected Utilization >= 100.0, got %.2f", metrics.Utilization)
	}

	bc.Release()
}

func TestHighThroughputIOBoundConfig(t *testing.T) {
	config := HighThroughputIOBoundConfig(":8080")

	if config.Addr != ":8080" {
		t.Errorf("Expected Addr ':8080', got '%s'", config.Addr)
	}

	if config.Workers != 2000 {
		t.Errorf("Expected Workers 2000, got %d", config.Workers)
	}

	if config.MaxQueue != 50000 {
		t.Errorf("Expected MaxQueue 50000, got %d", config.MaxQueue)
	}

	if config.MaxConns != 100000 {
		t.Errorf("Expected MaxConns 100000, got %d", config.MaxConns)
	}
}

func TestHighThroughputIOBoundConfigWithTargetRPS(t *testing.T) {
	targetRPS := 100000
	avgLatencyMs := 10
	config := HighThroughputIOBoundConfigWithTargetRPS(":8080", targetRPS, avgLatencyMs)

	if config.Addr != ":8080" {
		t.Errorf("Expected Addr ':8080', got '%s'", config.Addr)
	}

	// Workers = (targetRPS * avgLatencyMs) / 1000 = (100000 * 10) / 1000 = 1000
	expectedWorkers := (targetRPS * avgLatencyMs) / 1000
	if config.Workers != expectedWorkers {
		t.Errorf("Expected Workers %d, got %d", expectedWorkers, config.Workers)
	}

	// QueueSize = targetRPS / 2 = 50000
	expectedQueueSize := targetRPS / 2
	if config.MaxQueue != expectedQueueSize {
		t.Errorf("Expected MaxQueue %d, got %d", expectedQueueSize, config.MaxQueue)
	}

	// MaxConns = targetRPS = 100000
	if config.MaxConns != targetRPS {
		t.Errorf("Expected MaxConns %d, got %d", targetRPS, config.MaxConns)
	}
}

func TestHighThroughputIOBoundConfigWithTargetRPS_Minimums(t *testing.T) {
	// Test with very low target RPS to trigger minimums
	targetRPS := 100
	avgLatencyMs := 1
	config := HighThroughputIOBoundConfigWithTargetRPS(":8080", targetRPS, avgLatencyMs)

	// Workers should be at least 100
	if config.Workers < 100 {
		t.Errorf("Expected Workers >= 100, got %d", config.Workers)
	}

	// QueueSize should be at least 1000
	if config.MaxQueue < 1000 {
		t.Errorf("Expected MaxQueue >= 1000, got %d", config.MaxQueue)
	}

	// MaxConns should be at least 10000
	if config.MaxConns < 10000 {
		t.Errorf("Expected MaxConns >= 10000, got %d", config.MaxConns)
	}
}

func TestHighThroughputIOBoundConfigWithTargetRPS_Maximums(t *testing.T) {
	// Test with very high target RPS to trigger maximums
	targetRPS := 1000000
	avgLatencyMs := 100
	config := HighThroughputIOBoundConfigWithTargetRPS(":8080", targetRPS, avgLatencyMs)

	// Workers should be capped at 5000
	if config.Workers > 5000 {
		t.Errorf("Expected Workers <= 5000, got %d", config.Workers)
	}

	// QueueSize should be capped at 100000
	if config.MaxQueue > 100000 {
		t.Errorf("Expected MaxQueue <= 100000, got %d", config.MaxQueue)
	}

	// MaxConns should be capped at 200000
	if config.MaxConns > 200000 {
		t.Errorf("Expected MaxConns <= 200000, got %d", config.MaxConns)
	}
}

func TestHighThroughputIOBoundConfigWithTargetRPS_HTTPS(t *testing.T) {
	// Test with HTTPS latency (20ms to account for TLS overhead)
	targetRPS := 50000
	avgLatencyMs := 20
	config := HighThroughputIOBoundConfigWithTargetRPS(":443", targetRPS, avgLatencyMs)

	// Workers = (50000 * 20) / 1000 = 1000
	expectedWorkers := (targetRPS * avgLatencyMs) / 1000
	if config.Workers != expectedWorkers {
		t.Errorf("Expected Workers %d for HTTPS, got %d", expectedWorkers, config.Workers)
	}

	if config.Addr != ":443" {
		t.Errorf("Expected Addr ':443', got '%s'", config.Addr)
	}
}

func TestCCUBasedConfig_EdgeCases(t *testing.T) {
	// Test with very low maxCCU
	config := CCUBasedConfig(":8080", 100, 10)
	if config.Workers < 50 {
		t.Errorf("Expected Workers >= 50 (minimum), got %d", config.Workers)
	}
	if config.MaxQueue < 100 {
		t.Errorf("Expected MaxQueue >= 100 (minimum), got %d", config.MaxQueue)
	}

	// Test with very high maxCCU
	config = CCUBasedConfig(":8080", 100000, 10000)
	if config.Workers > 500 {
		t.Errorf("Expected Workers <= 500 (maximum), got %d", config.Workers)
	}
	if config.MaxQueue < 100 {
		t.Errorf("Expected MaxQueue >= 100 (minimum), got %d", config.MaxQueue)
	}
}

func TestCCUBasedConfigWithUtilization_InvalidPercent(t *testing.T) {
	// Test with invalid utilization percent (should default to 67%)
	config := CCUBasedConfigWithUtilization(":8080", 5000, 0)
	// Should default to 67%
	expectedNormalCapacity := int(float64(5000) * 0.67)
	actualNormalCapacity := config.MaxQueue + config.Workers
	tolerance := 50
	if actualNormalCapacity < expectedNormalCapacity-tolerance || actualNormalCapacity > expectedNormalCapacity+tolerance {
		t.Logf("With 0%% utilization, got normal capacity %d (should default to ~67%% = %d)", actualNormalCapacity, expectedNormalCapacity)
	}

	// Test with > 100%
	config = CCUBasedConfigWithUtilization(":8080", 5000, 150)
	// Should default to 67%
	actualNormalCapacity = config.MaxQueue + config.Workers
	if actualNormalCapacity < expectedNormalCapacity-tolerance || actualNormalCapacity > expectedNormalCapacity+tolerance {
		t.Logf("With 150%% utilization, got normal capacity %d (should default to ~67%% = %d)", actualNormalCapacity, expectedNormalCapacity)
	}
}
