package debug

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Enabled {
		t.Error("Expected Enabled to be false by default")
	}

	if config.CollectTimeout != 5*time.Second {
		t.Errorf("Expected CollectTimeout %v, got %v", 5*time.Second, config.CollectTimeout)
	}

	if !config.IncludeMemoryStats {
		t.Error("Expected IncludeMemoryStats to be true by default")
	}

	if !config.IncludeGCStats {
		t.Error("Expected IncludeGCStats to be true by default")
	}

	if !config.IncludeStackTrace {
		t.Error("Expected IncludeStackTrace to be true by default")
	}

	if !config.IncludeGoroutineDump {
		t.Error("Expected IncludeGoroutineDump to be true by default")
	}

	if config.ParallelCollect {
		t.Error("Expected ParallelCollect to be false by default")
	}

	if config.HistorySize != 0 {
		t.Error("Expected HistorySize to be 0 by default")
	}
}

func TestNewManager(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.IsEnabled() {
		t.Error("Expected debug mode to be disabled by default")
	}
}

func TestEnableDisable(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	if err := manager.Enable(); err != nil {
		t.Fatalf("Enable() returned error: %v", err)
	}

	if !manager.IsEnabled() {
		t.Error("Expected debug mode to be enabled")
	}

	if err := manager.Disable(); err != nil {
		t.Fatalf("Disable() returned error: %v", err)
	}

	if manager.IsEnabled() {
		t.Error("Expected debug mode to be disabled")
	}
}

func TestStackTrace(t *testing.T) {
	manager := NewManager(DefaultConfig())
	stack := manager.StackTrace()

	if len(stack) == 0 {
		t.Error("Expected non-empty stack trace")
	}

	// Verify it contains expected runtime information
	stackStr := string(stack)
	if len(stackStr) < 10 {
		t.Error("Stack trace seems too short")
	}
}

func TestGoroutineDump(t *testing.T) {
	manager := NewManager(DefaultConfig())
	dump := manager.GoroutineDump()

	if len(dump) == 0 {
		t.Error("Expected non-empty goroutine dump")
	}

	// Verify it contains goroutine information
	dumpStr := string(dump)
	if len(dumpStr) < 10 {
		t.Error("Goroutine dump seems too short")
	}
}

func TestMemoryStats(t *testing.T) {
	manager := NewManager(DefaultConfig())
	stats := manager.MemoryStats()

	// Basic sanity checks
	if stats.Sys == 0 {
		t.Error("Expected non-zero Sys value")
	}

	if stats.NumGC == 0 && stats.TotalAlloc > 0 {
		// This is possible if GC hasn't run yet, but TotalAlloc should be >= Alloc
		if stats.TotalAlloc < stats.Alloc {
			t.Error("TotalAlloc should be >= Alloc")
		}
	}
}

func TestGCStats(t *testing.T) {
	manager := NewManager(DefaultConfig())
	stats := manager.GCStats()

	// GC stats should be valid even if no GC has occurred
	if stats.NumGC < 0 {
		t.Error("NumGC should be non-negative")
	}
}

func TestCollect(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	ctx := context.Background()
	info, err := manager.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	if info == nil {
		t.Fatal("Collect() returned nil info")
	}

	if info.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}

	if info.GoVersion == "" {
		t.Error("Expected non-empty GoVersion")
	}

	if info.NumGoroutines <= 0 {
		t.Error("Expected positive NumGoroutines")
	}

	if info.NumCPU <= 0 {
		t.Error("Expected positive NumCPU")
	}

	if info.Duration == 0 {
		t.Error("Expected non-zero Duration")
	}

	if info.CollectorErrors == nil {
		t.Error("Expected non-nil CollectorErrors")
	}

	if info.MemoryStats == nil {
		t.Error("Expected non-nil MemoryStats")
	}

	if info.GCStats == nil {
		t.Error("Expected non-nil GCStats")
	}

	if len(info.StackTrace) == 0 {
		t.Error("Expected non-empty StackTrace")
	}

	if len(info.GoroutineDump) == 0 {
		t.Error("Expected non-empty GoroutineDump")
	}
}

func TestCollectWithNilContext(t *testing.T) {
	manager := NewManager(DefaultConfig())
	_, err := manager.Collect(nil)

	if err == nil {
		t.Error("Expected error for nil context")
	}

	var debugErr *Error
	if !errors.As(err, &debugErr) {
		t.Errorf("Expected *Error, got %T", err)
	}

	if debugErr.Code != ErrCodeNilContext {
		t.Errorf("Expected error code %s, got %s", ErrCodeNilContext, debugErr.Code)
	}
}

func TestCollectWithTimeout(t *testing.T) {
	config := DefaultConfig()
	config.CollectTimeout = 100 * time.Millisecond
	config.IncludeMemoryStats = true
	config.IncludeGCStats = true
	config.IncludeStackTrace = true
	config.IncludeGoroutineDump = true

	manager := NewManager(config)

	// Register a slow collector
	manager.RegisterCollector("slow", func(ctx context.Context) (interface{}, error) {
		time.Sleep(200 * time.Millisecond)
		return "done", nil
	})

	ctx := context.Background()
	_, err := manager.Collect(ctx)

	if err == nil {
		t.Error("Expected timeout error")
	}

	var debugErr *Error
	if !errors.As(err, &debugErr) {
		t.Errorf("Expected *Error, got %T", err)
	}

	if debugErr.Code != ErrCodeCollectTimeout {
		t.Errorf("Expected error code %s, got %s", ErrCodeCollectTimeout, debugErr.Code)
	}
}

func TestCollectWithCustomCollectors(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	// Register a custom collector
	err := manager.RegisterCollector("test", func(ctx context.Context) (interface{}, error) {
		return map[string]interface{}{
			"value": 42,
			"name":  "test",
		}, nil
	})
	if err != nil {
		t.Fatalf("RegisterCollector() returned error: %v", err)
	}

	ctx := context.Background()
	info, err := manager.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	if info.CustomData == nil {
		t.Fatal("Expected non-nil CustomData")
	}

	testData, exists := info.CustomData["test"]
	if !exists {
		t.Error("Expected custom data 'test' to exist")
	}

	data, ok := testData.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map[string]interface{}, got %T", testData)
	}

	if data["value"] != 42 {
		t.Errorf("Expected value 42, got %v", data["value"])
	}
}

func TestCollectWithFailingCollector(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	// Register a collector that fails
	manager.RegisterCollector("failing", func(ctx context.Context) (interface{}, error) {
		return nil, errors.New("collector error")
	})

	ctx := context.Background()
	info, err := manager.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	// Collection should continue even if one collector fails
	if info.CustomData == nil {
		t.Fatal("Expected non-nil CustomData")
	}

	failingData, exists := info.CustomData["failing"]
	if !exists {
		t.Error("Expected custom data 'failing' to exist")
	}

	data, ok := failingData.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map[string]interface{}, got %T", failingData)
	}

	if _, hasError := data["error"]; !hasError {
		t.Error("Expected error in failing collector data")
	}
}

func TestRegisterCollector(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.RegisterCollector("test", func(ctx context.Context) (interface{}, error) {
		return "data", nil
	})
	if err != nil {
		t.Fatalf("RegisterCollector() returned error: %v", err)
	}

	// Try to register the same collector again
	err = manager.RegisterCollector("test", func(ctx context.Context) (interface{}, error) {
		return "data", nil
	})
	if err == nil {
		t.Error("Expected error when registering duplicate collector")
	}

	var debugErr *Error
	if !errors.As(err, &debugErr) {
		t.Errorf("Expected *Error, got %T", err)
	}

	if debugErr.Code != ErrCodeCollectorExists {
		t.Errorf("Expected error code %s, got %s", ErrCodeCollectorExists, debugErr.Code)
	}
}

func TestRegisterCollectorWithEmptyName(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.RegisterCollector("", func(ctx context.Context) (interface{}, error) {
		return "data", nil
	})
	if err == nil {
		t.Error("Expected error for empty collector name")
	}

	var debugErr *Error
	if !errors.As(err, &debugErr) {
		t.Errorf("Expected *Error, got %T", err)
	}

	if debugErr.Code != ErrCodeInvalidCollector {
		t.Errorf("Expected error code %s, got %s", ErrCodeInvalidCollector, debugErr.Code)
	}
}

func TestRegisterCollectorWithNilCollector(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.RegisterCollector("test", nil)
	if err == nil {
		t.Error("Expected error for nil collector")
	}

	var debugErr *Error
	if !errors.As(err, &debugErr) {
		t.Errorf("Expected *Error, got %T", err)
	}

	if debugErr.Code != ErrCodeInvalidCollector {
		t.Errorf("Expected error code %s, got %s", ErrCodeInvalidCollector, debugErr.Code)
	}
}

func TestUnregisterCollector(t *testing.T) {
	manager := NewManager(DefaultConfig())

	manager.RegisterCollector("test", func(ctx context.Context) (interface{}, error) {
		return "data", nil
	})

	// Unregister the collector
	manager.UnregisterCollector("test")

	// Try to unregister again (should not panic)
	manager.UnregisterCollector("test")

	// Collect should not include the unregistered collector
	ctx := context.Background()
	info, err := manager.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	if _, exists := info.CustomData["test"]; exists {
		t.Error("Expected custom data 'test' to not exist after unregistering")
	}
}

func TestStats(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	stats := manager.Stats()
	if stats.TotalCollections != 0 {
		t.Errorf("Expected TotalCollections 0, got %d", stats.TotalCollections)
	}

	// Collect some debug info
	ctx := context.Background()
	_, err := manager.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	stats = manager.Stats()
	if stats.TotalCollections != 1 {
		t.Errorf("Expected TotalCollections 1, got %d", stats.TotalCollections)
	}

	if stats.LastCollectionTime.IsZero() {
		t.Error("Expected non-zero LastCollectionTime")
	}

	if stats.LastCollectionDuration == 0 {
		t.Error("Expected non-zero LastCollectionDuration")
	}
}

func TestCollectParallel(t *testing.T) {
	config := DefaultConfig()
	config.ParallelCollect = true
	manager := NewManager(config)

	// Register multiple collectors
	manager.RegisterCollector("collector1", func(ctx context.Context) (interface{}, error) {
		time.Sleep(10 * time.Millisecond)
		return "data1", nil
	})

	manager.RegisterCollector("collector2", func(ctx context.Context) (interface{}, error) {
		time.Sleep(10 * time.Millisecond)
		return "data2", nil
	})

	ctx := context.Background()
	info, err := manager.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	if len(info.CustomData) != 2 {
		t.Errorf("Expected 2 custom data entries, got %d", len(info.CustomData))
	}

	// Parallel collection should be faster than sequential
	if info.Duration > 50*time.Millisecond {
		t.Errorf("Parallel collection took too long: %v", info.Duration)
	}
}

func TestGoroutineFiltering(t *testing.T) {
	config := DefaultConfig()
	config.MaxGoroutines = 2
	manager := NewManager(config)

	dump := manager.GoroutineDump()
	if len(dump) == 0 {
		t.Error("Expected non-empty goroutine dump")
	}

	// Count goroutines in dump
	dumpStr := string(dump)
	goroutineCount := 0
	for _, line := range splitLinesForTest(dumpStr) {
		if isGoroutineHeaderForTest(line) {
			goroutineCount++
		}
	}

	// Should have at most MaxGoroutines goroutines
	if goroutineCount > config.MaxGoroutines {
		t.Errorf("Expected at most %d goroutines, got %d", config.MaxGoroutines, goroutineCount)
	}
}

func TestHistory(t *testing.T) {
	config := DefaultConfig()
	config.HistorySize = 3
	manager := NewManager(config)

	ctx := context.Background()

	// Collect multiple times
	for i := 0; i < 5; i++ {
		_, err := manager.Collect(ctx)
		if err != nil {
			t.Fatalf("Collect() returned error: %v", err)
		}
	}

	// Get history
	history := manager.History()
	if history == nil {
		t.Fatal("Expected non-nil history")
	}

	// Should have at most HistorySize entries
	if len(history) > config.HistorySize {
		t.Errorf("Expected at most %d history entries, got %d", config.HistorySize, len(history))
	}

	// Should have the most recent collections
	if len(history) < 2 {
		t.Error("Expected at least 2 history entries")
	}
}

func TestStatsWithDuration(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	ctx := context.Background()
	_, err := manager.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	stats := manager.Stats()
	if stats.LastCollectionDuration == 0 {
		t.Error("Expected non-zero LastCollectionDuration")
	}

	if stats.AverageCollectionDuration == 0 {
		t.Error("Expected non-zero AverageCollectionDuration")
	}
}

// Helper functions for testing
func splitLinesForTest(s string) []string {
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

func isGoroutineHeaderForTest(line string) bool {
	if len(line) < 9 {
		return false
	}
	return line[:9] == "goroutine "
}

func TestCollectWithCallbacks(t *testing.T) {
	config := DefaultConfig()
	var collectedInfo *Info
	var asyncCollectedInfo *Info
	var mu sync.Mutex

	config.OnCollect = func(info *Info) {
		collectedInfo = info
	}

	config.OnCollectAsync = func(info *Info) {
		mu.Lock()
		asyncCollectedInfo = info
		mu.Unlock()
	}

	manager := NewManager(config)

	ctx := context.Background()
	info, err := manager.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	if collectedInfo == nil {
		t.Error("Expected OnCollect to be called")
	}

	if collectedInfo != info {
		t.Error("OnCollect should receive the same info")
	}

	// Wait a bit for async callback
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if asyncCollectedInfo == nil {
		mu.Unlock()
		t.Error("Expected OnCollectAsync to be called")
	} else {
		if asyncCollectedInfo != info {
			t.Error("OnCollectAsync should receive the same info")
		}
		mu.Unlock()
	}
}

func TestCollectWithDisabledFeatures(t *testing.T) {
	config := DefaultConfig()
	config.IncludeMemoryStats = false
	config.IncludeGCStats = false
	config.IncludeStackTrace = false
	config.IncludeGoroutineDump = false

	manager := NewManager(config)

	ctx := context.Background()
	info, err := manager.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect() returned error: %v", err)
	}

	if info.MemoryStats != nil {
		t.Error("Expected nil MemoryStats when disabled")
	}

	if info.GCStats != nil {
		t.Error("Expected nil GCStats when disabled")
	}

	if len(info.StackTrace) != 0 {
		t.Error("Expected empty StackTrace when disabled")
	}

	if len(info.GoroutineDump) != 0 {
		t.Error("Expected empty GoroutineDump when disabled")
	}
}

func TestError(t *testing.T) {
	err := NewError(ErrCodeNilContext, "test message")
	if err == nil {
		t.Fatal("NewError returned nil")
	}

	if err.Code != ErrCodeNilContext {
		t.Errorf("Expected code %s, got %s", ErrCodeNilContext, err.Code)
	}

	if err.Message != "test message" {
		t.Errorf("Expected message 'test message', got %s", err.Message)
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() returned empty string")
	}

	if errStr != "NIL_CONTEXT: test message" {
		t.Errorf("Expected error string 'NIL_CONTEXT: test message', got %s", errStr)
	}
}

func TestCollectConcurrent(t *testing.T) {
	manager := NewManager(DefaultConfig())

	// Run multiple concurrent collections
	ctx := context.Background()
	done := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_, err := manager.Collect(ctx)
			done <- err
		}()
	}

	// Wait for all collections to complete
	for i := 0; i < 10; i++ {
		err := <-done
		if err != nil {
			t.Errorf("Collection %d returned error: %v", i, err)
		}
	}

	stats := manager.Stats()
	if stats.TotalCollections != 10 {
		t.Errorf("Expected TotalCollections 10, got %d", stats.TotalCollections)
	}
}

func BenchmarkCollect(b *testing.B) {
	config := DefaultConfig()
	manager := NewManager(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.Collect(ctx)
		if err != nil {
			b.Fatalf("Collect() returned error: %v", err)
		}
	}
}

func BenchmarkMemoryStats(b *testing.B) {
	manager := NewManager(DefaultConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.MemoryStats()
	}
}

func BenchmarkGoroutineDump(b *testing.B) {
	manager := NewManager(DefaultConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.GoroutineDump()
	}
}
