package fallback

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()
	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	stats := manager.Stats()
	if stats.TotalExecutions != 0 {
		t.Errorf("Expected TotalExecutions 0, got %d", stats.TotalExecutions)
	}
}

func TestExecute_PrimarySuccess(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	err := manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}

	stats := manager.Stats()
	if stats.TotalPrimarySuccesses != 1 {
		t.Errorf("Expected TotalPrimarySuccesses 1, got %d", stats.TotalPrimarySuccesses)
	}
	if stats.TotalFallbackAttempts != 0 {
		t.Errorf("Expected TotalFallbackAttempts 0, got %d", stats.TotalFallbackAttempts)
	}
}

func TestExecute_PrimaryFailure_NoFallback(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	testErr := errors.New("primary error")
	err := manager.Execute(ctx, func(ctx context.Context) error {
		return testErr
	})

	if err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, err)
	}

	stats := manager.Stats()
	if stats.TotalPrimaryFailures != 1 {
		t.Errorf("Expected TotalPrimaryFailures 1, got %d", stats.TotalPrimaryFailures)
	}
	if stats.TotalFallbackAttempts != 0 {
		t.Errorf("Expected TotalFallbackAttempts 0, got %d", stats.TotalFallbackAttempts)
	}
}

func TestExecuteWithFallback_FallbackSuccess(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	primaryErr := errors.New("primary error")

	err := manager.ExecuteWithFallback(ctx,
		func(ctx context.Context) error {
			return primaryErr
		},
		func(ctx context.Context) error {
			return nil // Fallback succeeds
		},
	)

	if err != nil {
		t.Errorf("Expected nil error (fallback succeeded), got %v", err)
	}

	stats := manager.Stats()
	if stats.TotalPrimaryFailures != 1 {
		t.Errorf("Expected TotalPrimaryFailures 1, got %d", stats.TotalPrimaryFailures)
	}
	if stats.TotalFallbackAttempts != 1 {
		t.Errorf("Expected TotalFallbackAttempts 1, got %d", stats.TotalFallbackAttempts)
	}
	if stats.TotalFallbackSuccesses != 1 {
		t.Errorf("Expected TotalFallbackSuccesses 1, got %d", stats.TotalFallbackSuccesses)
	}
}

func TestExecuteWithFallback_BothFail(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	primaryErr := errors.New("primary error")
	fallbackErr := errors.New("fallback error")

	err := manager.ExecuteWithFallback(ctx,
		func(ctx context.Context) error {
			return primaryErr
		},
		func(ctx context.Context) error {
			return fallbackErr
		},
	)

	if err == nil {
		t.Fatal("Expected error (all fallbacks exhausted), got nil")
	}

	fallbackError, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if fallbackError.Code != ErrCodeAllFallbacksExhausted {
		t.Errorf("Expected error code %s, got %s", ErrCodeAllFallbacksExhausted, fallbackError.Code)
	}

	stats := manager.Stats()
	if stats.TotalFallbackAttempts != 1 {
		t.Errorf("Expected TotalFallbackAttempts 1, got %d", stats.TotalFallbackAttempts)
	}
	if stats.TotalFallbackFailures != 1 {
		t.Errorf("Expected TotalFallbackFailures 1, got %d", stats.TotalFallbackFailures)
	}
	if stats.TotalAllFallbacksExhausted != 1 {
		t.Errorf("Expected TotalAllFallbacksExhausted 1, got %d", stats.TotalAllFallbacksExhausted)
	}
}

func TestExecuteWithConfig_MultipleFallbacks(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	var executionOrder []int
	var mu sync.Mutex

	config := Config{
		Fallbacks: []Executable{
			func(ctx context.Context) error {
				mu.Lock()
				executionOrder = append(executionOrder, 1)
				mu.Unlock()
				return errors.New("fallback 1 failed")
			},
			func(ctx context.Context) error {
				mu.Lock()
				executionOrder = append(executionOrder, 2)
				mu.Unlock()
				return nil // Second fallback succeeds
			},
			func(ctx context.Context) error {
				mu.Lock()
				executionOrder = append(executionOrder, 3)
				mu.Unlock()
				return errors.New("fallback 3 failed")
			},
		},
	}

	err := manager.ExecuteWithConfig(ctx,
		func(ctx context.Context) error {
			return errors.New("primary failed")
		},
		config,
	)

	if err != nil {
		t.Errorf("Expected nil error (fallback succeeded), got %v", err)
	}

	mu.Lock()
	if len(executionOrder) != 2 {
		t.Errorf("Expected 2 fallback executions, got %d", len(executionOrder))
	}
	if executionOrder[0] != 1 || executionOrder[1] != 2 {
		t.Errorf("Expected execution order [1, 2], got %v", executionOrder)
	}
	mu.Unlock()

	stats := manager.Stats()
	if stats.TotalFallbackAttempts != 2 {
		t.Errorf("Expected TotalFallbackAttempts 2, got %d", stats.TotalFallbackAttempts)
	}
	if stats.TotalFallbackSuccesses != 1 {
		t.Errorf("Expected TotalFallbackSuccesses 1, got %d", stats.TotalFallbackSuccesses)
	}
}

func TestExecuteWithConfig_AllFallbacksFail(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	config := Config{
		Fallbacks: []Executable{
			func(ctx context.Context) error {
				return errors.New("fallback 1 failed")
			},
			func(ctx context.Context) error {
				return errors.New("fallback 2 failed")
			},
		},
	}

	err := manager.ExecuteWithConfig(ctx,
		func(ctx context.Context) error {
			return errors.New("primary failed")
		},
		config,
	)

	if err == nil {
		t.Fatal("Expected error (all fallbacks exhausted), got nil")
	}

	fallbackError, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if fallbackError.Code != ErrCodeAllFallbacksExhausted {
		t.Errorf("Expected error code %s, got %s", ErrCodeAllFallbacksExhausted, fallbackError.Code)
	}

	stats := manager.Stats()
	if stats.TotalFallbackAttempts != 2 {
		t.Errorf("Expected TotalFallbackAttempts 2, got %d", stats.TotalFallbackAttempts)
	}
	if stats.TotalFallbackFailures != 2 {
		t.Errorf("Expected TotalFallbackFailures 2, got %d", stats.TotalFallbackFailures)
	}
	if stats.TotalAllFallbacksExhausted != 1 {
		t.Errorf("Expected TotalAllFallbacksExhausted 1, got %d", stats.TotalAllFallbacksExhausted)
	}
}

func TestExecuteWithConfig_WithPredicate(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	var fallbackAttempted1 bool
	var fallbackAttempted2 bool

	// Test with timeout error
	config1 := Config{
		Fallbacks: []Executable{
			func(ctx context.Context) error {
				fallbackAttempted1 = true
				return nil
			},
		},
		Predicate: FallbackOnTimeout, // Only fallback on timeout
	}

	err := manager.ExecuteWithConfig(ctx,
		func(ctx context.Context) error {
			return context.DeadlineExceeded
		},
		config1,
	)

	if err != nil {
		t.Errorf("Expected nil error (fallback succeeded), got %v", err)
	}
	if !fallbackAttempted1 {
		t.Error("Expected fallback to be attempted on timeout error")
	}

	// Test with non-timeout error - use separate config to avoid closure capture issues
	config2 := Config{
		Fallbacks: []Executable{
			func(ctx context.Context) error {
				fallbackAttempted2 = true
				return nil
			},
		},
		Predicate: FallbackOnTimeout, // Only fallback on timeout
	}

	err = manager.ExecuteWithConfig(ctx,
		func(ctx context.Context) error {
			return errors.New("database connection failed")
		},
		config2,
	)

	if fallbackAttempted2 {
		t.Error("Expected fallback not to be attempted on non-timeout error")
	}
}

func TestExecuteWithConfig_NilFallbackSkipped(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	config := Config{
		Fallbacks: []Executable{
			nil, // Nil fallback should be skipped
			func(ctx context.Context) error {
				return nil // Second fallback succeeds
			},
		},
	}

	err := manager.ExecuteWithConfig(ctx,
		func(ctx context.Context) error {
			return errors.New("primary failed")
		},
		config,
	)

	if err != nil {
		t.Errorf("Expected nil error (fallback succeeded), got %v", err)
	}

	stats := manager.Stats()
	if stats.TotalFallbackAttempts != 1 {
		t.Errorf("Expected TotalFallbackAttempts 1 (nil skipped), got %d", stats.TotalFallbackAttempts)
	}
}

func TestExecute_NilContext(t *testing.T) {
	manager := NewManager()

	err := manager.Execute(nil, func(ctx context.Context) error {
		return nil
	})

	if err == nil {
		t.Fatal("Expected error for nil context, got nil")
	}

	fallbackError, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if fallbackError.Code != ErrCodeNilContext {
		t.Errorf("Expected error code %s, got %s", ErrCodeNilContext, fallbackError.Code)
	}
}

func TestExecute_NilPrimaryFunction(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	err := manager.Execute(ctx, nil)

	if err == nil {
		t.Fatal("Expected error for nil primary function, got nil")
	}

	fallbackError, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if fallbackError.Code != ErrCodeNilPrimaryFunction {
		t.Errorf("Expected error code %s, got %s", ErrCodeNilPrimaryFunction, fallbackError.Code)
	}
}

func TestOnPrimaryErrorCallback(t *testing.T) {
	var callbackCalled bool
	var callbackErr error
	var mu sync.Mutex

	config := Config{
		Fallbacks: []Executable{
			func(ctx context.Context) error {
				return nil
			},
		},
		OnPrimaryError: func(ctx context.Context, err error) {
			mu.Lock()
			callbackCalled = true
			callbackErr = err
			mu.Unlock()
		},
	}

	manager := NewManager()
	ctx := context.Background()
	primaryErr := errors.New("primary error")

	manager.ExecuteWithConfig(ctx,
		func(ctx context.Context) error {
			return primaryErr
		},
		config,
	)

	mu.Lock()
	if !callbackCalled {
		t.Error("Expected OnPrimaryError callback to be called")
	}
	if callbackErr != primaryErr {
		t.Errorf("Expected callback error %v, got %v", primaryErr, callbackErr)
	}
	mu.Unlock()
}

func TestOnFallbackAttemptedCallback(t *testing.T) {
	var callbackCalled bool
	var callbackIndex int
	var mu sync.Mutex

	config := Config{
		Fallbacks: []Executable{
			func(ctx context.Context) error {
				return nil
			},
		},
		OnFallbackAttempted: func(ctx context.Context, index int, fn Executable) {
			mu.Lock()
			callbackCalled = true
			callbackIndex = index
			mu.Unlock()
		},
	}

	manager := NewManager()
	ctx := context.Background()

	manager.ExecuteWithConfig(ctx,
		func(ctx context.Context) error {
			return errors.New("primary failed")
		},
		config,
	)

	mu.Lock()
	if !callbackCalled {
		t.Error("Expected OnFallbackAttempted callback to be called")
	}
	if callbackIndex != 0 {
		t.Errorf("Expected callback index 0, got %d", callbackIndex)
	}
	mu.Unlock()
}

func TestOnFallbackSucceededCallback(t *testing.T) {
	var callbackCalled bool
	var callbackIndex int
	var mu sync.Mutex

	config := Config{
		Fallbacks: []Executable{
			func(ctx context.Context) error {
				return nil
			},
		},
		OnFallbackSucceeded: func(ctx context.Context, index int, result error) {
			mu.Lock()
			callbackCalled = true
			callbackIndex = index
			mu.Unlock()
		},
	}

	manager := NewManager()
	ctx := context.Background()

	manager.ExecuteWithConfig(ctx,
		func(ctx context.Context) error {
			return errors.New("primary failed")
		},
		config,
	)

	mu.Lock()
	if !callbackCalled {
		t.Error("Expected OnFallbackSucceeded callback to be called")
	}
	if callbackIndex != 0 {
		t.Errorf("Expected callback index 0, got %d", callbackIndex)
	}
	mu.Unlock()
}

func TestOnFallbackFailedCallback(t *testing.T) {
	var callbackCalled bool
	var callbackIndex int
	var callbackErr error
	var mu sync.Mutex

	fallbackErr := errors.New("fallback error")

	config := Config{
		Fallbacks: []Executable{
			func(ctx context.Context) error {
				return fallbackErr
			},
		},
		OnFallbackFailed: func(ctx context.Context, index int, err error) {
			mu.Lock()
			callbackCalled = true
			callbackIndex = index
			callbackErr = err
			mu.Unlock()
		},
	}

	manager := NewManager()
	ctx := context.Background()

	manager.ExecuteWithConfig(ctx,
		func(ctx context.Context) error {
			return errors.New("primary failed")
		},
		config,
	)

	mu.Lock()
	if !callbackCalled {
		t.Error("Expected OnFallbackFailed callback to be called")
	}
	if callbackIndex != 0 {
		t.Errorf("Expected callback index 0, got %d", callbackIndex)
	}
	if callbackErr != fallbackErr {
		t.Errorf("Expected callback error %v, got %v", fallbackErr, callbackErr)
	}
	mu.Unlock()
}

func TestOnAllFallbacksExhaustedCallback(t *testing.T) {
	var callbackCalled bool
	var callbackPrimaryErr error
	var callbackFallbackErrors []error
	var mu sync.Mutex

	primaryErr := errors.New("primary error")
	fallbackErr1 := errors.New("fallback 1 error")
	fallbackErr2 := errors.New("fallback 2 error")

	config := Config{
		Fallbacks: []Executable{
			func(ctx context.Context) error {
				return fallbackErr1
			},
			func(ctx context.Context) error {
				return fallbackErr2
			},
		},
		OnAllFallbacksExhausted: func(ctx context.Context, primary error, fallbackErrors []error) {
			mu.Lock()
			callbackCalled = true
			callbackPrimaryErr = primary
			callbackFallbackErrors = fallbackErrors
			mu.Unlock()
		},
	}

	manager := NewManager()
	ctx := context.Background()

	manager.ExecuteWithConfig(ctx,
		func(ctx context.Context) error {
			return primaryErr
		},
		config,
	)

	mu.Lock()
	if !callbackCalled {
		t.Error("Expected OnAllFallbacksExhausted callback to be called")
	}
	if callbackPrimaryErr != primaryErr {
		t.Errorf("Expected primary error %v, got %v", primaryErr, callbackPrimaryErr)
	}
	if len(callbackFallbackErrors) != 2 {
		t.Errorf("Expected 2 fallback errors, got %d", len(callbackFallbackErrors))
	}
	if callbackFallbackErrors[0] != fallbackErr1 {
		t.Errorf("Expected fallback error 1 %v, got %v", fallbackErr1, callbackFallbackErrors[0])
	}
	if callbackFallbackErrors[1] != fallbackErr2 {
		t.Errorf("Expected fallback error 2 %v, got %v", fallbackErr2, callbackFallbackErrors[1])
	}
	mu.Unlock()
}

func TestConcurrentExecution(t *testing.T) {
	manager := NewManager()

	var wg sync.WaitGroup
	concurrency := 10
	successes := int32(0)

	ctx := context.Background()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := manager.ExecuteWithFallback(ctx,
				func(ctx context.Context) error {
					return errors.New("primary failed")
				},
				func(ctx context.Context) error {
					return nil // Fallback succeeds
				},
			)
			if err == nil {
				atomic.AddInt32(&successes, 1)
			}
		}()
	}

	wg.Wait()

	if successes != int32(concurrency) {
		t.Errorf("Expected %d successes, got %d", concurrency, successes)
	}

	stats := manager.Stats()
	if stats.TotalExecutions != int64(concurrency) {
		t.Errorf("Expected TotalExecutions %d, got %d", concurrency, stats.TotalExecutions)
	}
	if stats.TotalPrimaryFailures != int64(concurrency) {
		t.Errorf("Expected TotalPrimaryFailures %d, got %d", concurrency, stats.TotalPrimaryFailures)
	}
	if stats.TotalFallbackSuccesses != int64(concurrency) {
		t.Errorf("Expected TotalFallbackSuccesses %d, got %d", concurrency, stats.TotalFallbackSuccesses)
	}
}

func TestStats(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()

	// Primary success
	manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	// Primary failure, fallback success
	manager.ExecuteWithFallback(ctx,
		func(ctx context.Context) error {
			return errors.New("primary failed")
		},
		func(ctx context.Context) error {
			return nil
		},
	)

	// Primary failure, fallback failure
	manager.ExecuteWithFallback(ctx,
		func(ctx context.Context) error {
			return errors.New("primary failed")
		},
		func(ctx context.Context) error {
			return errors.New("fallback failed")
		},
	)

	stats := manager.Stats()

	if stats.TotalExecutions != 3 {
		t.Errorf("Expected TotalExecutions 3, got %d", stats.TotalExecutions)
	}

	if stats.TotalPrimarySuccesses != 1 {
		t.Errorf("Expected TotalPrimarySuccesses 1, got %d", stats.TotalPrimarySuccesses)
	}

	if stats.TotalPrimaryFailures != 2 {
		t.Errorf("Expected TotalPrimaryFailures 2, got %d", stats.TotalPrimaryFailures)
	}

	if stats.TotalFallbackAttempts != 2 {
		t.Errorf("Expected TotalFallbackAttempts 2, got %d", stats.TotalFallbackAttempts)
	}

	if stats.TotalFallbackSuccesses != 1 {
		t.Errorf("Expected TotalFallbackSuccesses 1, got %d", stats.TotalFallbackSuccesses)
	}

	if stats.TotalFallbackFailures != 1 {
		t.Errorf("Expected TotalFallbackFailures 1, got %d", stats.TotalFallbackFailures)
	}

	if stats.TotalAllFallbacksExhausted != 1 {
		t.Errorf("Expected TotalAllFallbacksExhausted 1, got %d", stats.TotalAllFallbacksExhausted)
	}
}
