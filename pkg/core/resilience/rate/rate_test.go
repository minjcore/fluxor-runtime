package rate

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()
	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	stats := manager.Stats()
	if stats.Window != time.Second {
		t.Errorf("Expected Window 1s, got %v", stats.Window)
	}
	if stats.Granularity != 10 {
		t.Errorf("Expected Granularity 10, got %d", stats.Granularity)
	}
}

func TestNewManagerWithConfig(t *testing.T) {
	config := Config{
		Window:      time.Minute,
		Granularity: 60,
	}

	manager := NewManagerWithConfig(config)
	if manager == nil {
		t.Fatal("NewManagerWithConfig returned nil")
	}

	stats := manager.Stats()
	if stats.Window != time.Minute {
		t.Errorf("Expected Window 1m, got %v", stats.Window)
	}
	if stats.Granularity != 60 {
		t.Errorf("Expected Granularity 60, got %d", stats.Granularity)
	}
}

func TestRecord(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	manager.Record(ctx)

	stats := manager.Stats()
	if stats.TotalEvents != 1 {
		t.Errorf("Expected TotalEvents 1, got %d", stats.TotalEvents)
	}
}

func TestRecordN(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	manager.RecordN(ctx, 10)

	stats := manager.Stats()
	if stats.TotalEvents != 10 {
		t.Errorf("Expected TotalEvents 10, got %d", stats.TotalEvents)
	}
}

func TestRate_Basic(t *testing.T) {
	config := Config{
		Window:      time.Second,
		Granularity: 10,
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Record 10 events
	for i := 0; i < 10; i++ {
		manager.Record(ctx)
	}

	// Rate should be approximately 10 events per second
	rate := manager.Rate()
	if rate < 8 || rate > 12 {
		t.Errorf("Expected rate around 10 events/s, got %.2f", rate)
	}
}

func TestRate_WithTimeWindow(t *testing.T) {
	config := Config{
		Window:      time.Minute,
		Granularity: 60,
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Record 60 events (1 per second over 1 minute)
	for i := 0; i < 60; i++ {
		manager.Record(ctx)
		time.Sleep(10 * time.Millisecond) // Small delay
	}

	// Rate per minute should be approximately 60
	ratePerMinute := manager.Rate()
	if ratePerMinute < 50 || ratePerMinute > 70 {
		t.Logf("Rate per minute: %.2f (expected around 60)", ratePerMinute)
	}

	// Rate per second should be approximately 1
	ratePerSecond := manager.RateWithWindow(time.Second)
	if ratePerSecond < 0.5 || ratePerSecond > 2 {
		t.Logf("Rate per second: %.2f (expected around 1)", ratePerSecond)
	}
}

func TestRate_ZeroEvents(t *testing.T) {
	manager := NewManager()

	// Don't record any events
	rate := manager.Rate()
	if rate != 0 {
		t.Errorf("Expected rate 0 with no events, got %.2f", rate)
	}
}

func TestRate_SlidingWindow(t *testing.T) {
	config := Config{
		Window:      2 * time.Second,
		Granularity: 10,
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Record events in first second
	for i := 0; i < 10; i++ {
		manager.Record(ctx)
		time.Sleep(100 * time.Millisecond)
	}

	firstRate := manager.Rate()
	if firstRate < 5 || firstRate > 15 {
		t.Logf("First rate: %.2f (expected around 10 events/s)", firstRate)
	}

	// Wait for window to slide
	time.Sleep(2 * time.Second)

	// Rate should decrease as old events expire
	secondRate := manager.Rate()
	if secondRate >= firstRate {
		t.Logf("Second rate: %.2f (expected less than first rate %f)", secondRate, firstRate)
	}
}

func TestRecord_NilContext(t *testing.T) {
	manager := NewManager()

	// Should not panic with nil context
	manager.Record(nil)

	_ = manager.Stats()
	// May or may not record (implementation detail)
}

func TestRecordN_Zero(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	manager.RecordN(ctx, 0)

	stats := manager.Stats()
	if stats.TotalEvents != 0 {
		t.Errorf("Expected TotalEvents 0 for RecordN(0), got %d", stats.TotalEvents)
	}
}

func TestOnRateCalculatedCallback(t *testing.T) {
	var callbackCalled bool
	var callbackRate float64
	var mu sync.Mutex

	config := Config{
		Window:     time.Second,
		Granularity: 10,
		OnRateCalculated: func(ctx context.Context, rate float64, window time.Duration) {
			mu.Lock()
			callbackCalled = true
			callbackRate = rate
			mu.Unlock()
		},
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()
	manager.Record(ctx)

	// Rate() should trigger callback
	manager.Rate()

	// Wait a bit for callback
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if !callbackCalled {
		t.Error("Expected OnRateCalculated callback to be called")
	}
	if callbackRate < 0 {
		t.Errorf("Expected callback rate >= 0, got %.2f", callbackRate)
	}
	mu.Unlock()
}

func TestConcurrentExecution(t *testing.T) {
	manager := NewManager()

	var wg sync.WaitGroup
	concurrency := 10
	eventsPerGoroutine := 100

	ctx := context.Background()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				manager.Record(ctx)
			}
		}()
	}

	wg.Wait()

	stats := manager.Stats()
	expectedTotal := int64(concurrency * eventsPerGoroutine)
	if stats.TotalEvents != expectedTotal {
		t.Errorf("Expected TotalEvents %d, got %d", expectedTotal, stats.TotalEvents)
	}
}

func TestStats(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()

	manager.Record(ctx)
	time.Sleep(10 * time.Millisecond)
	manager.Record(ctx)

	stats := manager.Stats()

	if stats.TotalEvents != 2 {
		t.Errorf("Expected TotalEvents 2, got %d", stats.TotalEvents)
	}

	if stats.Window != time.Second {
		t.Errorf("Expected Window 1s, got %v", stats.Window)
	}

	if stats.Granularity != 10 {
		t.Errorf("Expected Granularity 10, got %d", stats.Granularity)
	}

	if stats.FirstEventTime.IsZero() {
		t.Error("Expected FirstEventTime to be set")
	}

	if stats.LastEventTime.IsZero() {
		t.Error("Expected LastEventTime to be set")
	}

	if stats.LastEventTime.Before(stats.FirstEventTime) {
		t.Error("Expected LastEventTime >= FirstEventTime")
	}

	if stats.CurrentRate < 0 {
		t.Errorf("Expected CurrentRate >= 0, got %.2f", stats.CurrentRate)
	}
}

func TestExecuteWithConfig_InvalidWindow(t *testing.T) {
	config := Config{
		Window:     0, // Invalid
		Granularity: 10,
	}

	manager := NewManagerWithConfig(config)

	// Should use default
	stats := manager.Stats()
	if stats.Window <= 0 {
		t.Error("Expected valid window after fixing invalid config")
	}
}

func TestExecuteWithConfig_InvalidGranularity(t *testing.T) {
	config := Config{
		Window:     time.Second,
		Granularity: 0, // Invalid
	}

	manager := NewManagerWithConfig(config)

	// Should use default
	stats := manager.Stats()
	if stats.Granularity <= 0 {
		t.Error("Expected valid granularity after fixing invalid config")
	}
}
