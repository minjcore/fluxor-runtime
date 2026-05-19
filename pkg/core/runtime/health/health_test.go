package health

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.CheckTimeout != 5*time.Second {
		t.Errorf("Expected CheckTimeout %v, got %v", 5*time.Second, config.CheckTimeout)
	}

	if !config.IncludeMemoryCheck {
		t.Error("Expected IncludeMemoryCheck to be true by default")
	}

	if !config.IncludeGCCheck {
		t.Error("Expected IncludeGCCheck to be true by default")
	}

	if !config.IncludeGoroutineCheck {
		t.Error("Expected IncludeGoroutineCheck to be true by default")
	}

	if !config.ParallelCheck {
		t.Error("Expected ParallelCheck to be true by default")
	}
}

func TestNewManager(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	stats := manager.Stats()
	if stats.TotalChecks != 0 {
		t.Errorf("Expected TotalChecks to be 0, got %d", stats.TotalChecks)
	}
}

func TestCheckWithNilContext(t *testing.T) {
	manager := NewManager(DefaultConfig())

	_, err := manager.Check(nil)
	if err == nil {
		t.Fatal("Expected error for nil context")
	}

	if e, ok := err.(*Error); !ok || e.Code != ErrCodeNilContext {
		t.Errorf("Expected ErrCodeNilContext, got %v", err)
	}
}

func TestCheckBasic(t *testing.T) {
	manager := NewManager(DefaultConfig())

	result, err := manager.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() returned error: %v", err)
	}

	if result == nil {
		t.Fatal("Check() returned nil result")
	}

	if result.Runtime == nil {
		t.Error("Expected Runtime to be populated")
	}

	if result.Runtime.NumGoroutines <= 0 {
		t.Error("Expected NumGoroutines to be positive")
	}

	if result.Runtime.NumCPU <= 0 {
		t.Error("Expected NumCPU to be positive")
	}

	if result.Runtime.GoVersion == "" {
		t.Error("Expected GoVersion to be set")
	}

	stats := manager.Stats()
	if stats.TotalChecks != 1 {
		t.Errorf("Expected TotalChecks to be 1, got %d", stats.TotalChecks)
	}

	if stats.LastCheckDuration <= 0 {
		t.Error("Expected LastCheckDuration to be positive")
	}
}

func TestCheckWithMemoryCheck(t *testing.T) {
	config := DefaultConfig()
	config.IncludeMemoryCheck = true
	manager := NewManager(config)

	result, err := manager.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() returned error: %v", err)
	}

	if result.Runtime == nil {
		t.Fatal("Expected Runtime to be populated")
	}

	if result.Runtime.Memory == nil {
		t.Error("Expected Memory to be populated when IncludeMemoryCheck is true")
	}

	if result.Runtime.Memory.Alloc == 0 {
		t.Error("Expected Memory.Alloc to be non-zero")
	}
}

func TestCheckWithGCCheck(t *testing.T) {
	config := DefaultConfig()
	config.IncludeGCCheck = true
	manager := NewManager(config)

	result, err := manager.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() returned error: %v", err)
	}

	if result.Runtime == nil {
		t.Fatal("Expected Runtime to be populated")
	}

	if result.Runtime.GC == nil {
		t.Error("Expected GC to be populated when IncludeGCCheck is true")
	}
}

func TestCheckWithoutMemoryCheck(t *testing.T) {
	config := DefaultConfig()
	config.IncludeMemoryCheck = false
	manager := NewManager(config)

	result, err := manager.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() returned error: %v", err)
	}

	if result.Runtime != nil && result.Runtime.Memory != nil {
		t.Error("Expected Memory to be nil when IncludeMemoryCheck is false")
	}
}

func TestCheckWithoutGCCheck(t *testing.T) {
	config := DefaultConfig()
	config.IncludeGCCheck = false
	manager := NewManager(config)

	result, err := manager.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() returned error: %v", err)
	}

	if result.Runtime != nil && result.Runtime.GC != nil {
		t.Error("Expected GC to be nil when IncludeGCCheck is false")
	}
}

func TestRegisterChecker(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.RegisterChecker("test", func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("RegisterChecker() returned error: %v", err)
	}

	stats := manager.Stats()
	if stats.TotalCheckers != 1 {
		t.Errorf("Expected TotalCheckers to be 1, got %d", stats.TotalCheckers)
	}
}

func TestRegisterCheckerWithEmptyName(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.RegisterChecker("", func(ctx context.Context) error {
		return nil
	})
	if err == nil {
		t.Fatal("Expected error for empty checker name")
	}

	if e, ok := err.(*Error); !ok || e.Code != ErrCodeInvalidChecker {
		t.Errorf("Expected ErrCodeInvalidChecker, got %v", err)
	}
}

func TestRegisterCheckerWithNilChecker(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.RegisterChecker("test", nil)
	if err == nil {
		t.Fatal("Expected error for nil checker")
	}

	if e, ok := err.(*Error); !ok || e.Code != ErrCodeInvalidChecker {
		t.Errorf("Expected ErrCodeInvalidChecker, got %v", err)
	}
}

func TestRegisterCheckerDuplicate(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.RegisterChecker("test", func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("RegisterChecker() returned error: %v", err)
	}

	err = manager.RegisterChecker("test", func(ctx context.Context) error {
		return nil
	})
	if err == nil {
		t.Fatal("Expected error for duplicate checker")
	}

	if e, ok := err.(*Error); !ok || e.Code != ErrCodeCheckerExists {
		t.Errorf("Expected ErrCodeCheckerExists, got %v", err)
	}
}

func TestUnregisterChecker(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.RegisterChecker("test", func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("RegisterChecker() returned error: %v", err)
	}

	manager.UnregisterChecker("test")

	stats := manager.Stats()
	if stats.TotalCheckers != 0 {
		t.Errorf("Expected TotalCheckers to be 0, got %d", stats.TotalCheckers)
	}
}

func TestCheckWithCustomChecker(t *testing.T) {
	manager := NewManager(DefaultConfig())

	checkerCalled := false
	err := manager.RegisterChecker("test", func(ctx context.Context) error {
		checkerCalled = true
		return nil
	})
	if err != nil {
		t.Fatalf("RegisterChecker() returned error: %v", err)
	}

	result, err := manager.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() returned error: %v", err)
	}

	if !checkerCalled {
		t.Error("Expected checker to be called")
	}

	checkResult, ok := result.Checks["test"]
	if !ok {
		t.Fatal("Expected check result for 'test'")
	}

	if checkResult.Status != StatusHealthy {
		t.Errorf("Expected StatusHealthy, got %s", checkResult.Status)
	}

	if !result.Healthy {
		t.Error("Expected result to be healthy")
	}
}

func TestCheckWithFailingChecker(t *testing.T) {
	manager := NewManager(DefaultConfig())

	testErr := errors.New("test error")
	err := manager.RegisterChecker("test", func(ctx context.Context) error {
		return testErr
	})
	if err != nil {
		t.Fatalf("RegisterChecker() returned error: %v", err)
	}

	result, err := manager.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() returned error: %v", err)
	}

	checkResult, ok := result.Checks["test"]
	if !ok {
		t.Fatal("Expected check result for 'test'")
	}

	if checkResult.Status != StatusUnhealthy {
		t.Errorf("Expected StatusUnhealthy, got %s", checkResult.Status)
	}

	if result.Healthy {
		t.Error("Expected result to be unhealthy")
	}

	if result.Overall != StatusUnhealthy {
		t.Errorf("Expected Overall StatusUnhealthy, got %s", result.Overall)
	}
}

func TestCheckWithParallelCheckers(t *testing.T) {
	config := DefaultConfig()
	config.ParallelCheck = true
	manager := NewManager(config)

	var mu sync.Mutex
	order := make([]string, 0)

	err := manager.RegisterChecker("checker1", func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		order = append(order, "checker1")
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("RegisterChecker() returned error: %v", err)
	}

	err = manager.RegisterChecker("checker2", func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		order = append(order, "checker2")
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("RegisterChecker() returned error: %v", err)
	}

	start := time.Now()
	result, err := manager.Check(context.Background())
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Check() returned error: %v", err)
	}

	// With parallel checks, total time should be less than sum of individual times
	if duration > 150*time.Millisecond {
		t.Errorf("Expected parallel checks to complete faster, took %v", duration)
	}

	if len(result.Checks) != 2 {
		t.Errorf("Expected 2 checks, got %d", len(result.Checks))
	}
}

func TestCheckWithSequentialCheckers(t *testing.T) {
	config := DefaultConfig()
	config.ParallelCheck = false
	manager := NewManager(config)

	var mu sync.Mutex
	order := make([]string, 0)

	err := manager.RegisterChecker("checker1", func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		order = append(order, "checker1")
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("RegisterChecker() returned error: %v", err)
	}

	err = manager.RegisterChecker("checker2", func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)
		mu.Lock()
		order = append(order, "checker2")
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("RegisterChecker() returned error: %v", err)
	}

	result, err := manager.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() returned error: %v", err)
	}

	if len(result.Checks) != 2 {
		t.Errorf("Expected 2 checks, got %d", len(result.Checks))
	}
}

func TestCheckWithTimeout(t *testing.T) {
	config := DefaultConfig()
	config.CheckTimeout = 100 * time.Millisecond
	manager := NewManager(config)

	err := manager.RegisterChecker("slow", func(ctx context.Context) error {
		select {
		case <-time.After(200 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	if err != nil {
		t.Fatalf("RegisterChecker() returned error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := manager.Check(ctx)
	if err != nil {
		t.Fatalf("Check() returned error: %v", err)
	}

	checkResult, ok := result.Checks["slow"]
	if !ok {
		t.Fatal("Expected check result for 'slow'")
	}

	if checkResult.Error == nil {
		t.Error("Expected timeout error")
	}
}

func TestRegisterThreshold(t *testing.T) {
	manager := NewManager(DefaultConfig())

	threshold := Threshold{
		MaxGoroutines: 1000,
		MaxMemoryAlloc: 100 * 1024 * 1024, // 100MB
	}

	err := manager.RegisterThreshold("test", threshold)
	if err != nil {
		t.Fatalf("RegisterThreshold() returned error: %v", err)
	}

	stats := manager.Stats()
	if stats.TotalThresholds != 1 {
		t.Errorf("Expected TotalThresholds to be 1, got %d", stats.TotalThresholds)
	}
}

func TestRegisterThresholdWithEmptyName(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.RegisterThreshold("", Threshold{})
	if err == nil {
		t.Fatal("Expected error for empty threshold name")
	}

	if e, ok := err.(*Error); !ok || e.Code != ErrCodeInvalidThreshold {
		t.Errorf("Expected ErrCodeInvalidThreshold, got %v", err)
	}
}

func TestRegisterThresholdDuplicate(t *testing.T) {
	manager := NewManager(DefaultConfig())

	threshold := Threshold{
		MaxGoroutines: 1000,
	}

	err := manager.RegisterThreshold("test", threshold)
	if err != nil {
		t.Fatalf("RegisterThreshold() returned error: %v", err)
	}

	err = manager.RegisterThreshold("test", threshold)
	if err == nil {
		t.Fatal("Expected error for duplicate threshold")
	}

	if e, ok := err.(*Error); !ok || e.Code != ErrCodeThresholdExists {
		t.Errorf("Expected ErrCodeThresholdExists, got %v", err)
	}
}

func TestUnregisterThreshold(t *testing.T) {
	manager := NewManager(DefaultConfig())

	threshold := Threshold{
		MaxGoroutines: 1000,
	}

	err := manager.RegisterThreshold("test", threshold)
	if err != nil {
		t.Fatalf("RegisterThreshold() returned error: %v", err)
	}

	manager.UnregisterThreshold("test")

	stats := manager.Stats()
	if stats.TotalThresholds != 0 {
		t.Errorf("Expected TotalThresholds to be 0, got %d", stats.TotalThresholds)
	}
}

func TestCheckWithGoroutineThreshold(t *testing.T) {
	config := DefaultConfig()
	config.IncludeMemoryCheck = true
	manager := NewManager(config)

	// Set a very low threshold that should fail
	threshold := Threshold{
		MaxGoroutines: 1,
	}

	err := manager.RegisterThreshold("goroutines", threshold)
	if err != nil {
		t.Fatalf("RegisterThreshold() returned error: %v", err)
	}

	result, err := manager.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() returned error: %v", err)
	}

	// The threshold check should fail if we have more than 1 goroutine
	if result.Runtime.NumGoroutines > 1 {
		checkResult, ok := result.Checks["goroutines"]
		if ok && checkResult.Status == StatusUnhealthy {
			if result.Healthy {
				t.Error("Expected result to be unhealthy due to goroutine threshold")
			}
		}
	}
}

func TestCheckWithMemoryThreshold(t *testing.T) {
	config := DefaultConfig()
	config.IncludeMemoryCheck = true
	manager := NewManager(config)

	// Set a very low memory threshold that should fail
	threshold := Threshold{
		MaxMemoryAlloc: 1, // 1 byte - should always fail
	}

	err := manager.RegisterThreshold("memory", threshold)
	if err != nil {
		t.Fatalf("RegisterThreshold() returned error: %v", err)
	}

	result, err := manager.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() returned error: %v", err)
	}

	checkResult, ok := result.Checks["memory"]
	if !ok {
		t.Fatal("Expected check result for 'memory'")
	}

	if checkResult.Status != StatusUnhealthy {
		t.Errorf("Expected StatusUnhealthy, got %s", checkResult.Status)
	}

	if result.Healthy {
		t.Error("Expected result to be unhealthy due to memory threshold")
	}
}

func TestIsHealthy(t *testing.T) {
	manager := NewManager(DefaultConfig())

	healthy, err := manager.IsHealthy(context.Background())
	if err != nil {
		t.Fatalf("IsHealthy() returned error: %v", err)
	}

	if !healthy {
		t.Error("Expected system to be healthy with no checkers")
	}
}

func TestIsHealthyWithFailingChecker(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.RegisterChecker("test", func(ctx context.Context) error {
		return errors.New("test error")
	})
	if err != nil {
		t.Fatalf("RegisterChecker() returned error: %v", err)
	}

	healthy, err := manager.IsHealthy(context.Background())
	if err != nil {
		t.Fatalf("IsHealthy() returned error: %v", err)
	}

	if healthy {
		t.Error("Expected system to be unhealthy with failing checker")
	}
}

func TestStats(t *testing.T) {
	manager := NewManager(DefaultConfig())

	stats := manager.Stats()
	if stats.TotalChecks != 0 {
		t.Errorf("Expected TotalChecks to be 0, got %d", stats.TotalChecks)
	}

	_, _ = manager.Check(context.Background())

	stats = manager.Stats()
	if stats.TotalChecks != 1 {
		t.Errorf("Expected TotalChecks to be 1, got %d", stats.TotalChecks)
	}

	if stats.LastCheckTime.IsZero() {
		t.Error("Expected LastCheckTime to be set")
	}
}

func TestError(t *testing.T) {
	err := NewError(ErrCodeNilContext, "test message")

	if err.Code != ErrCodeNilContext {
		t.Errorf("Expected Code %s, got %s", ErrCodeNilContext, err.Code)
	}

	if err.Message != "test message" {
		t.Errorf("Expected Message 'test message', got '%s'", err.Message)
	}

	errStr := err.Error()
	expected := ErrCodeNilContext + ": test message"
	if errStr != expected {
		t.Errorf("Expected Error() to return '%s', got '%s'", expected, errStr)
	}
}

func TestStatusString(t *testing.T) {
	testCases := []struct {
		status Status
		want   string
	}{
		{StatusHealthy, "healthy"},
		{StatusUnhealthy, "unhealthy"},
		{StatusDegraded, "degraded"},
		{StatusUnknown, "unknown"},
	}

	for _, tc := range testCases {
		if got := tc.status.String(); got != tc.want {
			t.Errorf("Status.String() = %s, want %s", got, tc.want)
		}
	}
}

func TestOnCheckCallback(t *testing.T) {
	config := DefaultConfig()
	var callbackCalled bool
	var callbackResult *Result

	config.OnCheck = func(result *Result) {
		callbackCalled = true
		callbackResult = result
	}

	manager := NewManager(config)

	result, err := manager.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() returned error: %v", err)
	}

	if !callbackCalled {
		t.Error("Expected OnCheck callback to be called")
	}

	if callbackResult != result {
		t.Error("Expected callback result to match returned result")
	}
}

func TestOnCheckAsyncCallback(t *testing.T) {
	config := DefaultConfig()
	var callbackCalled bool
	var callbackResult *Result
	var mu sync.Mutex

	config.OnCheckAsync = func(result *Result) {
		mu.Lock()
		callbackCalled = true
		callbackResult = result
		mu.Unlock()
	}

	manager := NewManager(config)

	result, err := manager.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() returned error: %v", err)
	}

	// Wait a bit for async callback
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	called := callbackCalled
	res := callbackResult
	mu.Unlock()

	if !called {
		t.Error("Expected OnCheckAsync callback to be called")
	}

	if res != result {
		t.Error("Expected async callback result to match returned result")
	}
}
