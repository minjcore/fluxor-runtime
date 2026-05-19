package quota

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.HistorySize != 0 {
		t.Errorf("Expected HistorySize 0, got %d", config.HistorySize)
	}

	if config.AutoResetInterval != 0 {
		t.Errorf("Expected AutoResetInterval 0, got %v", config.AutoResetInterval)
	}

	if config.EnableMetrics {
		t.Error("Expected EnableMetrics to be false")
	}
}

func TestNewManager(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	stats := manager.Stats()
	if stats.TotalQuotas != 0 {
		t.Errorf("Expected TotalQuotas 0, got %d", stats.TotalQuotas)
	}
}

func TestRegister(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test-quota", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	quotas := manager.ListQuotas()
	if len(quotas) != 1 || quotas[0].Name != "test-quota" {
		t.Errorf("Expected 1 quota named 'test-quota', got %v", quotas)
	}

	stats := manager.Stats()
	if stats.TotalQuotas != 1 {
		t.Errorf("Expected TotalQuotas 1, got %d", stats.TotalQuotas)
	}
}

func TestRegister_EmptyName(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("", QuotaTypeRequests, 100, time.Minute)
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

func TestRegister_NegativeLimit(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test", QuotaTypeRequests, -1, time.Minute)
	if err == nil {
		t.Error("Expected error for negative limit")
	}
}

func TestRegister_NegativeWindow(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test", QuotaTypeRequests, 100, -1*time.Second)
	if err == nil {
		t.Error("Expected error for negative window")
	}
}

func TestRegister_Duplicate(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("First Register failed: %v", err)
	}

	err = manager.Register("test", QuotaTypeRequests, 200, time.Minute)
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}
}

func TestUnregister(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if len(manager.ListQuotas()) != 1 {
		t.Fatal("Expected 1 quota before unregister")
	}

	err = manager.Unregister("test")
	if err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}

	if len(manager.ListQuotas()) != 0 {
		t.Error("Expected 0 quotas after unregister")
	}
}

func TestUnregister_NotFound(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Unregister("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent quota")
	}
}

func TestAcquire_Success(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	acquired, usage, remaining, err := manager.Acquire(ctx, "test", 10)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	if !acquired {
		t.Error("Expected quota to be acquired")
	}

	if usage != 10 {
		t.Errorf("Expected usage 10, got %d", usage)
	}

	if remaining != 90 {
		t.Errorf("Expected remaining 90, got %d", remaining)
	}

	currentUsage, err := manager.GetUsage("test")
	if err != nil {
		t.Fatalf("GetUsage failed: %v", err)
	}

	if currentUsage != 10 {
		t.Errorf("Expected current usage 10, got %d", currentUsage)
	}
}

func TestAcquire_Exceeded(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	acquired, _, _, err := manager.Acquire(ctx, "test", 101)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	if acquired {
		t.Error("Expected quota to be exceeded")
	}

	stats := manager.Stats()
	if stats.TotalExceeded != 1 {
		t.Errorf("Expected TotalExceeded 1, got %d", stats.TotalExceeded)
	}
}

func TestAcquire_ContextNil(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	_, _, _, err = manager.Acquire(nil, "test", 10)
	if err == nil {
		t.Error("Expected error for nil context")
	}
}

func TestAcquire_EmptyName(t *testing.T) {
	manager := NewManager(DefaultConfig())

	ctx := context.Background()
	_, _, _, err := manager.Acquire(ctx, "", 10)
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

func TestAcquire_NegativeAmount(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	_, _, _, err = manager.Acquire(ctx, "test", -1)
	if err == nil {
		t.Error("Expected error for negative amount")
	}
}

func TestAcquire_NotFound(t *testing.T) {
	manager := NewManager(DefaultConfig())

	ctx := context.Background()
	_, _, _, err := manager.Acquire(ctx, "nonexistent", 10)
	if err == nil {
		t.Error("Expected error for nonexistent quota")
	}
}

func TestAcquire_UnlimitedQuota(t *testing.T) {
	manager := NewManager(DefaultConfig())

	// Register quota with limit 0 (unlimited)
	err := manager.Register("unlimited", QuotaTypeRequests, 0, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	acquired, usage, remaining, err := manager.Acquire(ctx, "unlimited", 1000)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	if !acquired {
		t.Error("Expected quota to be acquired")
	}

	if usage != 1000 {
		t.Errorf("Expected usage 1000, got %d", usage)
	}

	if remaining != -1 {
		t.Errorf("Expected remaining -1 (unlimited), got %d", remaining)
	}
}

func TestRelease(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	manager.Acquire(ctx, "test", 50)
	manager.Acquire(ctx, "test", 30)

	err = manager.Release("test", 20)
	if err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	usage, err := manager.GetUsage("test")
	if err != nil {
		t.Fatalf("GetUsage failed: %v", err)
	}

	if usage != 60 {
		t.Errorf("Expected usage 60, got %d", usage)
	}

	stats := manager.Stats()
	if stats.TotalReleased != 1 {
		t.Errorf("Expected TotalReleased 1, got %d", stats.TotalReleased)
	}
}

func TestRelease_BelowZero(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	manager.Acquire(ctx, "test", 10)

	err = manager.Release("test", 20)
	if err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	usage, err := manager.GetUsage("test")
	if err != nil {
		t.Fatalf("GetUsage failed: %v", err)
	}

	// Usage should not go below 0
	if usage != 0 {
		t.Errorf("Expected usage 0, got %d", usage)
	}
}

func TestRelease_NotFound(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Release("nonexistent", 10)
	if err == nil {
		t.Error("Expected error for nonexistent quota")
	}
}

func TestGetUsage(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	usage, err := manager.GetUsage("test")
	if err != nil {
		t.Fatalf("GetUsage failed: %v", err)
	}

	if usage != 0 {
		t.Errorf("Expected initial usage 0, got %d", usage)
	}

	ctx := context.Background()
	manager.Acquire(ctx, "test", 25)

	usage, err = manager.GetUsage("test")
	if err != nil {
		t.Fatalf("GetUsage failed: %v", err)
	}

	if usage != 25 {
		t.Errorf("Expected usage 25, got %d", usage)
	}
}

func TestGetRemaining(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	remaining, err := manager.GetRemaining("test")
	if err != nil {
		t.Fatalf("GetRemaining failed: %v", err)
	}

	if remaining != 100 {
		t.Errorf("Expected remaining 100, got %d", remaining)
	}

	ctx := context.Background()
	manager.Acquire(ctx, "test", 30)

	remaining, err = manager.GetRemaining("test")
	if err != nil {
		t.Fatalf("GetRemaining failed: %v", err)
	}

	if remaining != 70 {
		t.Errorf("Expected remaining 70, got %d", remaining)
	}
}

func TestGetRemaining_Unlimited(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("unlimited", QuotaTypeRequests, 0, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	remaining, err := manager.GetRemaining("unlimited")
	if err != nil {
		t.Fatalf("GetRemaining failed: %v", err)
	}

	if remaining != -1 {
		t.Errorf("Expected remaining -1 (unlimited), got %d", remaining)
	}
}

func TestReset(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	manager.Acquire(ctx, "test", 50)

	usage, _ := manager.GetUsage("test")
	if usage != 50 {
		t.Fatalf("Expected usage 50 before reset, got %d", usage)
	}

	err = manager.Reset("test")
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	usage, err = manager.GetUsage("test")
	if err != nil {
		t.Fatalf("GetUsage failed: %v", err)
	}

	if usage != 0 {
		t.Errorf("Expected usage 0 after reset, got %d", usage)
	}

	stats := manager.Stats()
	if stats.TotalReset != 1 {
		t.Errorf("Expected TotalReset 1, got %d", stats.TotalReset)
	}
}

func TestResetAll(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test1", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = manager.Register("test2", QuotaTypeRequests, 200, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	manager.Acquire(ctx, "test1", 50)
	manager.Acquire(ctx, "test2", 75)

	err = manager.ResetAll()
	if err != nil {
		t.Fatalf("ResetAll failed: %v", err)
	}

	usage1, _ := manager.GetUsage("test1")
	usage2, _ := manager.GetUsage("test2")

	if usage1 != 0 || usage2 != 0 {
		t.Errorf("Expected both quotas to be reset, got usage1=%d, usage2=%d", usage1, usage2)
	}
}

func TestQuotaStats(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	manager.Acquire(ctx, "test", 30)

	stats, err := manager.QuotaStats("test")
	if err != nil {
		t.Fatalf("QuotaStats failed: %v", err)
	}

	if stats.Name != "test" {
		t.Errorf("Expected name 'test', got %s", stats.Name)
	}

	if stats.Type != QuotaTypeRequests {
		t.Errorf("Expected type %s, got %s", QuotaTypeRequests, stats.Type)
	}

	if stats.Limit != 100 {
		t.Errorf("Expected limit 100, got %d", stats.Limit)
	}

	if stats.Usage != 30 {
		t.Errorf("Expected usage 30, got %d", stats.Usage)
	}

	if stats.Remaining != 70 {
		t.Errorf("Expected remaining 70, got %d", stats.Remaining)
	}
}

func TestQuotaStats_NotFound(t *testing.T) {
	manager := NewManager(DefaultConfig())

	_, err := manager.QuotaStats("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent quota")
	}
}

func TestQuotaWindow_Reset(t *testing.T) {
	manager := NewManager(DefaultConfig())

	// Register quota with 100ms window
	err := manager.Register("test", QuotaTypeRequests, 100, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	manager.Acquire(ctx, "test", 50)

	usage, _ := manager.GetUsage("test")
	if usage != 50 {
		t.Fatalf("Expected usage 50, got %d", usage)
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Next acquire should reset the quota
	acquired, usage, _, err := manager.Acquire(ctx, "test", 10)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	if !acquired {
		t.Error("Expected quota to be acquired after window reset")
	}

	// Usage should be reset to the new amount, not cumulative
	if usage != 10 {
		t.Errorf("Expected usage 10 after window reset, got %d", usage)
	}
}

func TestQuotaWindow_NoReset(t *testing.T) {
	manager := NewManager(DefaultConfig())

	// Register quota with long window
	err := manager.Register("test", QuotaTypeRequests, 100, time.Hour)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	manager.Acquire(ctx, "test", 50)

	usage1, _ := manager.GetUsage("test")

	// Wait a bit but less than window
	time.Sleep(50 * time.Millisecond)

	manager.Acquire(ctx, "test", 20)

	usage2, _ := manager.GetUsage("test")

	// Usage should be cumulative (70)
	if usage2 != 70 {
		t.Errorf("Expected usage 70, got %d (previous: %d)", usage2, usage1)
	}
}

func TestOnQuotaExceeded_Callback(t *testing.T) {
	var called bool
	var calledName string
	var calledType QuotaType
	var calledLimit, calledUsage int64

	config := DefaultConfig()
	config.OnQuotaExceeded = func(name string, quotaType QuotaType, limit, usage int64) {
		called = true
		calledName = name
		calledType = quotaType
		calledLimit = limit
		calledUsage = usage
	}

	manager := NewManager(config)

	err := manager.Register("test", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	manager.Acquire(ctx, "test", 101) // Exceed quota

	if !called {
		t.Error("Expected OnQuotaExceeded callback to be called")
	}

	if calledName != "test" {
		t.Errorf("Expected name 'test', got %s", calledName)
	}

	if calledType != QuotaTypeRequests {
		t.Errorf("Expected type %s, got %s", QuotaTypeRequests, calledType)
	}

	if calledLimit != 100 {
		t.Errorf("Expected limit 100, got %d", calledLimit)
	}

	if calledUsage != 0 {
		t.Errorf("Expected usage 0 (before acquisition), got %d", calledUsage)
	}
}

func TestOnQuotaExceededAsync_Callback(t *testing.T) {
	var called bool
	var mu sync.Mutex

	config := DefaultConfig()
	config.OnQuotaExceededAsync = func(name string, quotaType QuotaType, limit, usage int64) {
		mu.Lock()
		called = true
		mu.Unlock()
	}

	manager := NewManager(config)

	err := manager.Register("test", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	manager.Acquire(ctx, "test", 101) // Exceed quota

	// Wait a bit for async callback
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	calledCheck := called
	mu.Unlock()

	if !calledCheck {
		t.Error("Expected OnQuotaExceededAsync callback to be called")
	}
}

func TestOnQuotaReset_Callback(t *testing.T) {
	var called bool
	var calledName string

	config := DefaultConfig()
	config.OnQuotaReset = func(name string) {
		called = true
		calledName = name
	}

	manager := NewManager(config)

	err := manager.Register("test", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	manager.Acquire(ctx, "test", 50)
	manager.Reset("test")

	if !called {
		t.Error("Expected OnQuotaReset callback to be called")
	}

	if calledName != "test" {
		t.Errorf("Expected name 'test', got %s", calledName)
	}
}

func TestHistory(t *testing.T) {
	config := DefaultConfig()
	config.HistorySize = 10
	manager := NewManager(config)

	err := manager.Register("test", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()

	// Exceed quota to generate events
	manager.Acquire(ctx, "test", 101)
	manager.Reset("test")
	manager.Acquire(ctx, "test", 101)

	// History is stored but not exposed via public API in this implementation
	// This test verifies the system works without errors
	stats := manager.Stats()
	if stats.TotalExceeded != 2 {
		t.Errorf("Expected TotalExceeded 2, got %d", stats.TotalExceeded)
	}
}

func TestConcurrentAcquire(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test", QuotaTypeRequests, 1000, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	var wg sync.WaitGroup
	concurrency := 100
	amountPerGoroutine := 10

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			manager.Acquire(ctx, "test", int64(amountPerGoroutine))
		}()
	}

	wg.Wait()

	usage, err := manager.GetUsage("test")
	if err != nil {
		t.Fatalf("GetUsage failed: %v", err)
	}

	expectedUsage := int64(concurrency * amountPerGoroutine)
	if usage != expectedUsage {
		t.Errorf("Expected usage %d, got %d", expectedUsage, usage)
	}

	stats := manager.Stats()
	if stats.TotalAcquired != int64(concurrency) {
		t.Errorf("Expected TotalAcquired %d, got %d", concurrency, stats.TotalAcquired)
	}
}

func TestMultipleQuotaTypes(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("requests", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = manager.Register("memory", QuotaTypeMemory, 1024, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = manager.Register("cpu", QuotaTypeCPU, 80, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	quotas := manager.ListQuotas()
	if len(quotas) != 3 {
		t.Errorf("Expected 3 quotas, got %d", len(quotas))
	}

	ctx := context.Background()
	manager.Acquire(ctx, "requests", 10)
	manager.Acquire(ctx, "memory", 100)
	manager.Acquire(ctx, "cpu", 20)

	reqUsage, _ := manager.GetUsage("requests")
	memUsage, _ := manager.GetUsage("memory")
	cpuUsage, _ := manager.GetUsage("cpu")

	if reqUsage != 10 {
		t.Errorf("Expected requests usage 10, got %d", reqUsage)
	}

	if memUsage != 100 {
		t.Errorf("Expected memory usage 100, got %d", memUsage)
	}

	if cpuUsage != 20 {
		t.Errorf("Expected cpu usage 20, got %d", cpuUsage)
	}
}

func TestAutoReset(t *testing.T) {
	config := DefaultConfig()
	config.AutoResetInterval = 100 * time.Millisecond
	manager := NewManager(config)

	err := manager.Register("test", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	manager.Acquire(ctx, "test", 50)

	usage1, _ := manager.GetUsage("test")
	if usage1 != 50 {
		t.Fatalf("Expected usage 50, got %d", usage1)
	}

	// Wait for auto-reset
	time.Sleep(150 * time.Millisecond)

	usage2, _ := manager.GetUsage("test")
	if usage2 != 0 {
		t.Errorf("Expected usage 0 after auto-reset, got %d", usage2)
	}

	// Stop the manager
	if qm, ok := manager.(*quotaManager); ok {
		qm.Stop()
	}
}

func TestGetQuota(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	manager.Acquire(ctx, "test", 30)

	info, err := manager.GetQuota("test")
	if err != nil {
		t.Fatalf("GetQuota failed: %v", err)
	}

	if info.Name != "test" {
		t.Errorf("Expected name 'test', got %s", info.Name)
	}

	if info.Type != QuotaTypeRequests {
		t.Errorf("Expected type %s, got %s", QuotaTypeRequests, info.Type)
	}

	if info.Limit != 100 {
		t.Errorf("Expected limit 100, got %d", info.Limit)
	}

	if info.Usage != 30 {
		t.Errorf("Expected usage 30, got %d", info.Usage)
	}

	if info.Remaining != 70 {
		t.Errorf("Expected remaining 70, got %d", info.Remaining)
	}

	if info.RegisteredAt.IsZero() {
		t.Error("Expected RegisteredAt to be set")
	}

	if info.LastReset.IsZero() {
		t.Error("Expected LastReset to be set")
	}

	if info.LastUpdate.IsZero() {
		t.Error("Expected LastUpdate to be set")
	}
}

func TestGetQuota_NotFound(t *testing.T) {
	manager := NewManager(DefaultConfig())

	_, err := manager.GetQuota("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent quota")
	}
}

func TestGetQuota_EmptyName(t *testing.T) {
	manager := NewManager(DefaultConfig())

	_, err := manager.GetQuota("")
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

func TestListQuotas_ReturnsInfo(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("quota1", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = manager.Register("quota2", QuotaTypeMemory, 200, time.Hour)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	quotas := manager.ListQuotas()
	if len(quotas) != 2 {
		t.Fatalf("Expected 2 quotas, got %d", len(quotas))
	}

	// Should be sorted by name
	if quotas[0].Name != "quota1" {
		t.Errorf("Expected first quota to be 'quota1', got %s", quotas[0].Name)
	}

	if quotas[1].Name != "quota2" {
		t.Errorf("Expected second quota to be 'quota2', got %s", quotas[1].Name)
	}

	// Verify info is populated
	if quotas[0].Type != QuotaTypeRequests {
		t.Errorf("Expected quota1 type %s, got %s", QuotaTypeRequests, quotas[0].Type)
	}

	if quotas[1].Type != QuotaTypeMemory {
		t.Errorf("Expected quota2 type %s, got %s", QuotaTypeMemory, quotas[1].Type)
	}
}

func TestClear(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test1", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = manager.Register("test2", QuotaTypeMemory, 200, time.Hour)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if len(manager.ListQuotas()) != 2 {
		t.Fatal("Expected 2 quotas before clear")
	}

	manager.Clear()

	quotas := manager.ListQuotas()
	if len(quotas) != 0 {
		t.Errorf("Expected 0 quotas after clear, got %d", len(quotas))
	}

	stats := manager.Stats()
	if stats.TotalQuotas != 0 {
		t.Errorf("Expected TotalQuotas 0 after clear, got %d", stats.TotalQuotas)
	}
}

func TestQuotaStats_WithMetrics(t *testing.T) {
	config := DefaultConfig()
	config.EnableMetrics = true
	manager := NewManager(config)

	err := manager.Register("test", QuotaTypeRequests, 100, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	manager.Acquire(ctx, "test", 30)  // Acquire 30 (succeeds)
	manager.Acquire(ctx, "test", 80)  // Should exceed (30 + 80 = 110 > 100)
	manager.Acquire(ctx, "test", 20)  // Acquire 20 more (succeeds: 30 + 20 = 50)
	manager.Release("test", 10)

	stats, err := manager.QuotaStats("test")
	if err != nil {
		t.Fatalf("QuotaStats failed: %v", err)
	}

	if stats.TotalAcquired != 2 {
		t.Errorf("Expected TotalAcquired 2, got %d", stats.TotalAcquired)
	}

	if stats.Exceeded != 1 {
		t.Errorf("Expected Exceeded 1, got %d", stats.Exceeded)
	}

	if stats.TotalReleased != 1 {
		t.Errorf("Expected TotalReleased 1, got %d", stats.TotalReleased)
	}
}

func TestQuotaStats_Unlimited(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("unlimited", QuotaTypeRequests, 0, time.Minute)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	ctx := context.Background()
	manager.Acquire(ctx, "unlimited", 1000)

	stats, err := manager.QuotaStats("unlimited")
	if err != nil {
		t.Fatalf("QuotaStats failed: %v", err)
	}

	if stats.Remaining != -1 {
		t.Errorf("Expected remaining -1 (unlimited), got %d", stats.Remaining)
	}
}
