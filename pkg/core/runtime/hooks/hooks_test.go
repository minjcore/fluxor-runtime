package hooks

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.StopOnError {
		t.Error("DefaultConfig should have StopOnError = false")
	}
	if config.Parallel {
		t.Error("DefaultConfig should have Parallel = false")
	}
}

func TestNewRegistry(t *testing.T) {
	config := DefaultConfig()
	registry := NewRegistry(config)

	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	stats := registry.Stats()
	if stats.TotalHooks != 0 {
		t.Errorf("Expected 0 hooks, got %d", stats.TotalHooks)
	}
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	hook := func(ctx context.Context) error {
		return nil
	}

	err := registry.Register("test-hook", hook)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	stats := registry.Stats()
	if stats.TotalHooks != 1 {
		t.Errorf("Expected 1 hook, got %d", stats.TotalHooks)
	}
}

func TestRegistry_Register_EmptyName(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	err := registry.Register("", func(ctx context.Context) error { return nil })
	if err == nil {
		t.Error("Register() with empty name should fail")
	}
	if err, ok := err.(*Error); ok && err.Code != ErrCodeNilHook {
		t.Errorf("Expected error code %q, got %q", ErrCodeNilHook, err.Code)
	}
}

func TestRegistry_Register_NilHook(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	err := registry.Register("test", nil)
	if err == nil {
		t.Error("Register() with nil hook should fail")
	}
	if err, ok := err.(*Error); ok && err.Code != ErrCodeNilHook {
		t.Errorf("Expected error code %q, got %q", ErrCodeNilHook, err.Code)
	}
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	hook := func(ctx context.Context) error { return nil }

	err := registry.Register("test", hook)
	if err != nil {
		t.Fatalf("First Register() error = %v", err)
	}

	err = registry.Register("test", hook)
	if err == nil {
		t.Error("Register() with duplicate name should fail")
	}
	if err, ok := err.(*Error); ok && err.Code != ErrCodeHookExists {
		t.Errorf("Expected error code %q, got %q", ErrCodeHookExists, err.Code)
	}
}

func TestRegistry_Register_WithPriority(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	hook1 := func(ctx context.Context) error { return nil }
	hook2 := func(ctx context.Context) error { return nil }

	registry.Register("hook1", hook1, WithPriority(10))
	registry.Register("hook2", hook2, WithPriority(5))

	hooks := registry.ListHooks()
	if len(hooks) != 2 {
		t.Fatalf("Expected 2 hooks, got %d", len(hooks))
	}

	// Should be sorted by priority (lower first)
	if hooks[0].Name != "hook2" || hooks[0].Priority != 5 {
		t.Errorf("Expected first hook to be hook2 with priority 5, got %s with priority %d", hooks[0].Name, hooks[0].Priority)
	}
	if hooks[1].Name != "hook1" || hooks[1].Priority != 10 {
		t.Errorf("Expected second hook to be hook1 with priority 10, got %s with priority %d", hooks[1].Name, hooks[1].Priority)
	}
}

func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	hook := func(ctx context.Context) error { return nil }
	registry.Register("test", hook)

	err := registry.Unregister("test")
	if err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}

	stats := registry.Stats()
	if stats.TotalHooks != 0 {
		t.Errorf("Expected 0 hooks after unregister, got %d", stats.TotalHooks)
	}
}

func TestRegistry_Unregister_NotFound(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	err := registry.Unregister("nonexistent")
	if err == nil {
		t.Error("Unregister() with nonexistent hook should fail")
	}
	if err, ok := err.(*Error); ok && err.Code != ErrCodeHookNotFound {
		t.Errorf("Expected error code %q, got %q", ErrCodeHookNotFound, err.Code)
	}
}

func TestRegistry_Execute_Sequential(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	executionOrder := make([]string, 0)
	var mu sync.Mutex

	hook1 := func(ctx context.Context) error {
		mu.Lock()
		executionOrder = append(executionOrder, "hook1")
		mu.Unlock()
		return nil
	}

	hook2 := func(ctx context.Context) error {
		mu.Lock()
		executionOrder = append(executionOrder, "hook2")
		mu.Unlock()
		return nil
	}

	registry.Register("hook1", hook1, WithPriority(2))
	registry.Register("hook2", hook2, WithPriority(1))

	ctx := context.Background()
	err := registry.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(executionOrder) != 2 {
		t.Fatalf("Expected 2 hooks executed, got %d", len(executionOrder))
	}

	// Should execute in priority order (lower first)
	if executionOrder[0] != "hook2" || executionOrder[1] != "hook1" {
		t.Errorf("Expected execution order [hook2, hook1], got %v", executionOrder)
	}
}

func TestRegistry_Execute_Parallel(t *testing.T) {
	config := DefaultConfig()
	config.Parallel = true
	registry := NewRegistry(config)

	executionCount := int32(0)

	hook1 := func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&executionCount, 1)
		return nil
	}

	hook2 := func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&executionCount, 1)
		return nil
	}

	registry.Register("hook1", hook1)
	registry.Register("hook2", hook2)

	ctx := context.Background()
	start := time.Now()
	err := registry.Execute(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// With parallel execution, both hooks should complete in ~10ms (not ~20ms)
	if duration > 20*time.Millisecond {
		t.Errorf("Parallel execution took too long: %v", duration)
	}

	if atomic.LoadInt32(&executionCount) != 2 {
		t.Errorf("Expected 2 hooks executed, got %d", atomic.LoadInt32(&executionCount))
	}
}

func TestRegistry_Execute_WithError(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	testErr := errors.New("test error")
	hook := func(ctx context.Context) error {
		return testErr
	}

	registry.Register("failing-hook", hook)

	ctx := context.Background()
	err := registry.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute() should not return error when StopOnError is false, got %v", err)
	}

	stats := registry.Stats()
	if stats.TotalFailures != 1 {
		t.Errorf("Expected 1 failure, got %d", stats.TotalFailures)
	}
}

func TestRegistry_Execute_StopOnError(t *testing.T) {
	config := DefaultConfig()
	config.StopOnError = true
	registry := NewRegistry(config)

	executionOrder := make([]string, 0)
	var mu sync.Mutex

	hook1 := func(ctx context.Context) error {
		mu.Lock()
		executionOrder = append(executionOrder, "hook1")
		mu.Unlock()
		return errors.New("hook1 error")
	}

	hook2 := func(ctx context.Context) error {
		mu.Lock()
		executionOrder = append(executionOrder, "hook2")
		mu.Unlock()
		return nil
	}

	registry.Register("hook1", hook1, WithPriority(1))
	registry.Register("hook2", hook2, WithPriority(2))

	ctx := context.Background()
	err := registry.Execute(ctx)
	if err == nil {
		t.Error("Execute() should return error when StopOnError is true and hook fails")
	}

	// Only hook1 should have executed
	if len(executionOrder) != 1 || executionOrder[0] != "hook1" {
		t.Errorf("Expected only hook1 to execute, got %v", executionOrder)
	}
}

func TestRegistry_Execute_NilContext(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	hook := func(ctx context.Context) error { return nil }
	registry.Register("test", hook)

	err := registry.Execute(nil)
	if err == nil {
		t.Error("Execute(nil) should fail")
	}
	if err, ok := err.(*Error); ok && err.Code != ErrCodeNilContext {
		t.Errorf("Expected error code %q, got %q", ErrCodeNilContext, err.Code)
	}
}

func TestRegistry_ExecuteByName(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	executed := false
	hook := func(ctx context.Context) error {
		executed = true
		return nil
	}

	registry.Register("test-hook", hook)

	ctx := context.Background()
	err := registry.ExecuteByName(ctx, "test-hook")
	if err != nil {
		t.Fatalf("ExecuteByName() error = %v", err)
	}

	if !executed {
		t.Error("Hook should have been executed")
	}
}

func TestRegistry_ExecuteByName_NotFound(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	ctx := context.Background()
	err := registry.ExecuteByName(ctx, "nonexistent")
	if err == nil {
		t.Error("ExecuteByName() with nonexistent hook should fail")
	}
	if err, ok := err.(*Error); ok && err.Code != ErrCodeHookNotFound {
		t.Errorf("Expected error code %q, got %q", ErrCodeHookNotFound, err.Code)
	}
}

func TestRegistry_ExecuteByPriority(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	executionOrder := make([]string, 0)
	var mu sync.Mutex

	hook1 := func(ctx context.Context) error {
		mu.Lock()
		executionOrder = append(executionOrder, "hook1")
		mu.Unlock()
		return nil
	}

	hook2 := func(ctx context.Context) error {
		mu.Lock()
		executionOrder = append(executionOrder, "hook2")
		mu.Unlock()
		return nil
	}

	hook3 := func(ctx context.Context) error {
		mu.Lock()
		executionOrder = append(executionOrder, "hook3")
		mu.Unlock()
		return nil
	}

	registry.Register("hook1", hook1, WithPriority(5))
	registry.Register("hook2", hook2, WithPriority(10))
	registry.Register("hook3", hook3, WithPriority(10))

	ctx := context.Background()
	err := registry.ExecuteByPriority(ctx, 10)
	if err != nil {
		t.Fatalf("ExecuteByPriority() error = %v", err)
	}

	// Only hooks with priority 10 should execute
	if len(executionOrder) != 2 {
		t.Fatalf("Expected 2 hooks executed, got %d", len(executionOrder))
	}

	if executionOrder[0] != "hook2" || executionOrder[1] != "hook3" {
		t.Errorf("Expected execution order [hook2, hook3], got %v", executionOrder)
	}
}

func TestRegistry_GetHook(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	hook := func(ctx context.Context) error { return nil }
	registry.Register("test-hook", hook, WithPriority(5))

	info, err := registry.GetHook("test-hook")
	if err != nil {
		t.Fatalf("GetHook() error = %v", err)
	}

	if info.Name != "test-hook" {
		t.Errorf("Expected name 'test-hook', got %q", info.Name)
	}
	if info.Priority != 5 {
		t.Errorf("Expected priority 5, got %d", info.Priority)
	}
}

func TestRegistry_GetHook_NotFound(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	_, err := registry.GetHook("nonexistent")
	if err == nil {
		t.Error("GetHook() with nonexistent hook should fail")
	}
	if err, ok := err.(*Error); ok && err.Code != ErrCodeHookNotFound {
		t.Errorf("Expected error code %q, got %q", ErrCodeHookNotFound, err.Code)
	}
}

func TestRegistry_ListHooks(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	hook1 := func(ctx context.Context) error { return nil }
	hook2 := func(ctx context.Context) error { return nil }
	hook3 := func(ctx context.Context) error { return nil }

	registry.Register("hook1", hook1, WithPriority(10))
	registry.Register("hook2", hook2, WithPriority(5))
	registry.Register("hook3", hook3, WithPriority(15))

	hooks := registry.ListHooks()
	if len(hooks) != 3 {
		t.Fatalf("Expected 3 hooks, got %d", len(hooks))
	}

	// Should be sorted by priority
	if hooks[0].Name != "hook2" || hooks[0].Priority != 5 {
		t.Errorf("Expected first hook to be hook2 with priority 5")
	}
	if hooks[1].Name != "hook1" || hooks[1].Priority != 10 {
		t.Errorf("Expected second hook to be hook1 with priority 10")
	}
	if hooks[2].Name != "hook3" || hooks[2].Priority != 15 {
		t.Errorf("Expected third hook to be hook3 with priority 15")
	}
}

func TestRegistry_Stats(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	hook1 := func(ctx context.Context) error { return nil }
	hook2 := func(ctx context.Context) error { return errors.New("error") }

	registry.Register("hook1", hook1)
	registry.Register("hook2", hook2)

	stats := registry.Stats()
	if stats.TotalHooks != 2 {
		t.Errorf("Expected 2 hooks, got %d", stats.TotalHooks)
	}

	ctx := context.Background()
	registry.Execute(ctx)

	stats = registry.Stats()
	if stats.TotalExecutions != 2 {
		t.Errorf("Expected 2 executions, got %d", stats.TotalExecutions)
	}
	if stats.TotalSuccesses != 1 {
		t.Errorf("Expected 1 success, got %d", stats.TotalSuccesses)
	}
	if stats.TotalFailures != 1 {
		t.Errorf("Expected 1 failure, got %d", stats.TotalFailures)
	}
}

func TestRegistry_Clear(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	hook := func(ctx context.Context) error { return nil }
	registry.Register("hook1", hook)
	registry.Register("hook2", hook)

	stats := registry.Stats()
	if stats.TotalHooks != 2 {
		t.Errorf("Expected 2 hooks, got %d", stats.TotalHooks)
	}

	registry.Clear()

	stats = registry.Stats()
	if stats.TotalHooks != 0 {
		t.Errorf("Expected 0 hooks after Clear(), got %d", stats.TotalHooks)
	}

	hooks := registry.ListHooks()
	if len(hooks) != 0 {
		t.Errorf("Expected 0 hooks in list, got %d", len(hooks))
	}
}

func TestRegistry_Execute_WithTimeout(t *testing.T) {
	config := DefaultConfig()
	config.DefaultTimeout = 50 * time.Millisecond
	registry := NewRegistry(config)

	hook := func(ctx context.Context) error {
		// Check context to respect timeout - use a longer sleep that will be interrupted
		select {
		case <-time.After(200 * time.Millisecond):
			return nil
		case <-ctx.Done():
			// Context was cancelled (timeout)
			return ctx.Err()
		}
	}

	registry.Register("slow-hook", hook)

	ctx := context.Background()
	err := registry.Execute(ctx)
	
	// With StopOnError=false (default), Execute() doesn't return error even if hooks fail
	// But we should verify the hook failed by checking stats
	stats := registry.Stats()
	if stats.TotalFailures == 0 {
		t.Error("Expected hook to fail due to timeout, but TotalFailures is 0")
	}
	if stats.TotalExecutions == 0 {
		t.Error("Expected hook to be executed, but TotalExecutions is 0")
	}
	
	// Verify hook info shows the error
	info, getErr := registry.GetHook("slow-hook")
	if getErr != nil {
		t.Fatalf("GetHook() error = %v", getErr)
	}
	if info.LastError == nil {
		t.Error("Expected hook to have LastError set due to timeout")
	}
	
	t.Logf("Hook execution completed. Error returned by Execute(): %v, Hook LastError: %v", err, info.LastError)
}

func TestRegistry_Execute_WithContextTimeout(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	hook := func(ctx context.Context) error {
		// Check context to respect timeout
		select {
		case <-time.After(100 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	registry.Register("slow-hook", hook)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := registry.Execute(ctx)
	
	// With StopOnError=false (default), Execute() doesn't return error even if hooks fail
	// But we should verify the hook failed by checking stats
	stats := registry.Stats()
	if stats.TotalFailures == 0 {
		t.Error("Expected hook to fail due to context timeout, but TotalFailures is 0")
	}
	
	// Verify hook info shows the error
	info, getErr := registry.GetHook("slow-hook")
	if getErr != nil {
		t.Fatalf("GetHook() error = %v", getErr)
	}
	if info.LastError == nil {
		t.Error("Expected hook to have LastError set due to context timeout")
	}
	
	t.Logf("Hook execution completed. Error returned by Execute(): %v, Hook LastError: %v", err, info.LastError)
}

func TestRegistry_Execute_AsyncHook(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	executed := make(chan bool, 1)
	hook := func(ctx context.Context) error {
		executed <- true
		return nil
	}

	registry.Register("async-hook", hook, WithAsync(true))

	ctx := context.Background()
	err := registry.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Async hook should execute in background
	select {
	case <-executed:
		// Hook executed
	case <-time.After(100 * time.Millisecond):
		t.Error("Async hook should have executed")
	}
}

func TestRegistry_Execute_WithCallbacks(t *testing.T) {
	config := DefaultConfig()
	started := make([]string, 0)
	completed := make([]string, 0)
	var mu sync.Mutex

	config.OnHookStart = func(name string) {
		mu.Lock()
		started = append(started, name)
		mu.Unlock()
	}

	config.OnHookComplete = func(name string, err error) {
		mu.Lock()
		completed = append(completed, name)
		mu.Unlock()
	}

	registry := NewRegistry(config)

	hook := func(ctx context.Context) error { return nil }
	registry.Register("test-hook", hook)

	ctx := context.Background()
	err := registry.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	mu.Lock()
	if len(started) != 1 || started[0] != "test-hook" {
		t.Errorf("Expected OnHookStart to be called with 'test-hook', got %v", started)
	}
	if len(completed) != 1 || completed[0] != "test-hook" {
		t.Errorf("Expected OnHookComplete to be called with 'test-hook', got %v", completed)
	}
	mu.Unlock()
}

func TestRegistry_Execute_WithErrorCallback(t *testing.T) {
	config := DefaultConfig()
	errorNames := make([]string, 0)
	var mu sync.Mutex

	config.OnHookError = func(name string, err error) {
		mu.Lock()
		errorNames = append(errorNames, name)
		mu.Unlock()
	}

	registry := NewRegistry(config)

	testErr := errors.New("test error")
	hook := func(ctx context.Context) error {
		return testErr
	}

	registry.Register("failing-hook", hook)

	ctx := context.Background()
	err := registry.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute() should not return error when StopOnError is false, got %v", err)
	}

	mu.Lock()
	if len(errorNames) != 1 || errorNames[0] != "failing-hook" {
		t.Errorf("Expected OnHookError to be called with 'failing-hook', got %v", errorNames)
	}
	mu.Unlock()
}

func TestRegistry_Execute_MaxConcurrency(t *testing.T) {
	config := DefaultConfig()
	config.Parallel = true
	config.MaxConcurrency = 2
	registry := NewRegistry(config)

	running := int32(0)
	maxRunning := int32(0)
	var mu sync.Mutex

	hook := func(ctx context.Context) error {
		current := atomic.AddInt32(&running, 1)
		defer atomic.AddInt32(&running, -1)

		mu.Lock()
		if current > maxRunning {
			maxRunning = current
		}
		mu.Unlock()

		time.Sleep(10 * time.Millisecond)
		return nil
	}

	// Register 5 hooks
	for i := 0; i < 5; i++ {
		registry.Register("hook", hook)
	}

	ctx := context.Background()
	err := registry.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	mu.Lock()
	if maxRunning > 2 {
		t.Errorf("Expected max concurrency of 2, got %d", maxRunning)
	}
	mu.Unlock()
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewRegistry(DefaultConfig())

	hook := func(ctx context.Context) error { return nil }

	// Concurrent registration
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			registry.Register("hook", hook)
		}(i)
	}

	// Concurrent execution
	ctx := context.Background()
	go func() {
		for i := 0; i < 10; i++ {
			registry.Execute(ctx)
		}
	}()

	wg.Wait()

	// Should not panic
	stats := registry.Stats()
	_ = stats
}
