package limiter

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()
	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	stats := manager.Stats()
	if stats.CurrentRate != 10 {
		t.Errorf("Expected CurrentRate 10, got %d", stats.CurrentRate)
	}
	if stats.CurrentInterval != time.Second {
		t.Errorf("Expected CurrentInterval 1s, got %v", stats.CurrentInterval)
	}
}

func TestNewManagerWithConfig(t *testing.T) {
	config := Config{
		Rate:     100,
		Interval: time.Minute,
		Burst:    150,
	}

	manager := NewManagerWithConfig(config)
	if manager == nil {
		t.Fatal("NewManagerWithConfig returned nil")
	}

	stats := manager.Stats()
	if stats.CurrentRate != 100 {
		t.Errorf("Expected CurrentRate 100, got %d", stats.CurrentRate)
	}
	if stats.CurrentInterval != time.Minute {
		t.Errorf("Expected CurrentInterval 1m, got %v", stats.CurrentInterval)
	}
}

func TestExecute_Success(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	err := manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}

	stats := manager.Stats()
	if stats.TotalAllowed != 1 {
		t.Errorf("Expected TotalAllowed 1, got %d", stats.TotalAllowed)
	}
	if stats.TotalSuccessful != 1 {
		t.Errorf("Expected TotalSuccessful 1, got %d", stats.TotalSuccessful)
	}
}

func TestExecute_RateLimited(t *testing.T) {
	config := Config{
		Rate:     1, // Only 1 execution per second
		Interval: time.Second,
		Burst:    1,
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// First execution should succeed
	err := manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected nil error for first execution, got %v", err)
	}

	// Second execution immediately after should be rate limited
	// Use a context with timeout to ensure it fails
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	err = manager.Execute(ctxWithTimeout, func(ctx context.Context) error {
		return nil
	})

	if err == nil {
		t.Fatal("Expected error for rate limited execution, got nil")
	}

	// Can be either rate limit error or timeout error
	if err == context.DeadlineExceeded {
		// Timeout is acceptable - rate limited execution waited and timed out
		stats := manager.Stats()
		if stats.TotalRateLimited < 1 {
			t.Errorf("Expected TotalRateLimited at least 1, got %d", stats.TotalRateLimited)
		}
		return
	}

	limiterErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error or context.DeadlineExceeded, got %T: %v", err, err)
	}

	if limiterErr.Code != ErrCodeRateLimitExceeded && limiterErr.Code != ErrCodeContextTimeout {
		t.Errorf("Expected error code %s or %s, got %s", ErrCodeRateLimitExceeded, ErrCodeContextTimeout, limiterErr.Code)
	}

	stats := manager.Stats()
	if stats.TotalRateLimited < 1 {
		t.Errorf("Expected TotalRateLimited at least 1, got %d", stats.TotalRateLimited)
	}
}

func TestExecute_AllowsBurst(t *testing.T) {
	config := Config{
		Rate:     10, // 10 executions per second
		Interval: time.Second,
		Burst:    5, // But burst of 5
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Should allow up to 5 executions immediately (burst)
	for i := 0; i < 5; i++ {
		err := manager.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
		if err != nil {
			t.Errorf("Expected nil error for burst execution %d, got %v", i, err)
		}
	}

	// 6th execution should be rate limited (no tokens available)
	// Use a context with timeout to ensure it fails
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	err := manager.Execute(ctxWithTimeout, func(ctx context.Context) error {
		return nil
	})

	if err == nil {
		t.Fatal("Expected error for rate limited execution after burst, got nil")
	}

	// Can be either rate limit error or timeout error
	if err != context.DeadlineExceeded {
		limiterErr, ok := err.(*Error)
		if !ok || (limiterErr.Code != ErrCodeRateLimitExceeded && limiterErr.Code != ErrCodeContextTimeout) {
			t.Errorf("Expected rate limit or timeout error, got %v", err)
		}
	}

	stats := manager.Stats()
	if stats.TotalAllowed != 5 {
		t.Errorf("Expected TotalAllowed 5, got %d", stats.TotalAllowed)
	}
	if stats.TotalRateLimited < 1 {
		t.Errorf("Expected TotalRateLimited at least 1, got %d", stats.TotalRateLimited)
	}
}

func TestAllow_NonBlocking(t *testing.T) {
	config := Config{
		Rate:     1,
		Interval: time.Second,
		Burst:    1,
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// First check should allow
	if !manager.Allow(ctx) {
		t.Error("Expected Allow to return true")
	}

	// Consume the token
	manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	// Second check immediately after should not allow
	if manager.Allow(ctx) {
		t.Error("Expected Allow to return false (rate limited)")
	}
}

func TestWait_BlocksUntilAllowed(t *testing.T) {
	config := Config{
		Rate:     2, // 2 executions per second
		Interval: time.Second,
		Burst:    1,
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Consume initial token
	manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	// Wait should block until token is available
	waitCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	start := time.Now()
	err := manager.Wait(waitCtx)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Expected nil error from Wait, got %v", err)
	}

	if elapsed < 400*time.Millisecond {
		t.Errorf("Expected Wait to take at least 400ms, got %v", elapsed)
	}
}

func TestWait_ContextTimeout(t *testing.T) {
	config := Config{
		Rate:     1,
		Interval: time.Second,
		Burst:    1,
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Consume initial token
	manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	// Wait with short timeout should fail
	waitCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	err := manager.Wait(waitCtx)

	if err == nil {
		t.Fatal("Expected error for timeout, got nil")
	}

	// Wait can return either context error or limiter error
	if err == context.DeadlineExceeded {
		// This is acceptable - context timeout
		return
	}

	limiterErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error or context.DeadlineExceeded, got %T: %v", err, err)
	}

	if limiterErr.Code != ErrCodeContextTimeout {
		t.Errorf("Expected error code %s, got %s", ErrCodeContextTimeout, limiterErr.Code)
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

	limiterErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if limiterErr.Code != ErrCodeNilContext {
		t.Errorf("Expected error code %s, got %s", ErrCodeNilContext, limiterErr.Code)
	}
}

func TestExecute_NilFunction(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	err := manager.Execute(ctx, nil)

	if err == nil {
		t.Fatal("Expected error for nil function, got nil")
	}

	limiterErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if limiterErr.Code != ErrCodeNilFunction {
		t.Errorf("Expected error code %s, got %s", ErrCodeNilFunction, limiterErr.Code)
	}
}

func TestOnRateLimitExceededCallback(t *testing.T) {
	var callbackCalled bool
	var callbackRate int
	var mu sync.Mutex

	config := Config{
		Rate:     1,
		Interval: time.Second,
		Burst:    1,
		OnRateLimitExceeded: func(ctx context.Context, rate int, interval time.Duration) {
			mu.Lock()
			callbackCalled = true
			callbackRate = rate
			mu.Unlock()
		},
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Consume initial token
	manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	// This should trigger rate limit - use timeout to ensure it fails
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	manager.Execute(ctxWithTimeout, func(ctx context.Context) error {
		return nil
	})

	// Wait a bit for callback
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if !callbackCalled {
		t.Error("Expected OnRateLimitExceeded callback to be called")
	}
	if callbackRate != 1 {
		t.Errorf("Expected callback rate 1, got %d", callbackRate)
	}
	mu.Unlock()
}

func TestOnAllowedCallback(t *testing.T) {
	var callbackCalled bool
	var mu sync.Mutex

	config := Config{
		Rate:     10,
		Interval: time.Second,
		OnAllowed: func(ctx context.Context) {
			mu.Lock()
			callbackCalled = true
			mu.Unlock()
		},
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()
	manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	mu.Lock()
	if !callbackCalled {
		t.Error("Expected OnAllowed callback to be called")
	}
	mu.Unlock()
}

func TestOnCompletedCallback(t *testing.T) {
	var callbackCalled bool
	var callbackErr error
	var mu sync.Mutex

	testErr := errors.New("test error")

	config := Config{
		Rate:     10,
		Interval: time.Second,
		OnCompleted: func(ctx context.Context, err error) {
			mu.Lock()
			callbackCalled = true
			callbackErr = err
			mu.Unlock()
		},
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()
	err := manager.Execute(ctx, func(ctx context.Context) error {
		return testErr
	})

	mu.Lock()
	if !callbackCalled {
		t.Error("Expected OnCompleted callback to be called")
	}
	if callbackErr != testErr {
		t.Errorf("Expected callback error %v, got %v", testErr, callbackErr)
	}
	mu.Unlock()

	if err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, err)
	}
}

func TestConcurrentExecution(t *testing.T) {
	config := Config{
		Rate:     10,
		Interval: time.Second,
		Burst:    10,
	}

	manager := NewManagerWithConfig(config)

	var wg sync.WaitGroup
	concurrency := 20
	successes := int32(0)

	ctx := context.Background()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := manager.Execute(ctx, func(ctx context.Context) error {
				return nil
			})
			if err == nil {
				atomic.AddInt32(&successes, 1)
			}
		}()
	}

	wg.Wait()

	// Should allow up to burst size immediately, then rate limit
	if successes < int32(config.Burst) {
		t.Errorf("Expected at least %d successes (burst size), got %d", config.Burst, successes)
	}

	stats := manager.Stats()
	if stats.TotalExecutions != int64(concurrency) {
		t.Errorf("Expected TotalExecutions %d, got %d", concurrency, stats.TotalExecutions)
	}
}

func TestStats(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()

	// Success case
	manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	// Failure case
	testErr := errors.New("test error")
	manager.Execute(ctx, func(ctx context.Context) error {
		return testErr
	})

	stats := manager.Stats()

	if stats.TotalExecutions != 2 {
		t.Errorf("Expected TotalExecutions 2, got %d", stats.TotalExecutions)
	}

	if stats.TotalAllowed != 2 {
		t.Errorf("Expected TotalAllowed 2, got %d", stats.TotalAllowed)
	}

	if stats.TotalSuccessful != 1 {
		t.Errorf("Expected TotalSuccessful 1, got %d", stats.TotalSuccessful)
	}

	if stats.TotalFailed != 1 {
		t.Errorf("Expected TotalFailed 1, got %d", stats.TotalFailed)
	}

	if stats.CurrentTokens < 0 {
		t.Errorf("Expected CurrentTokens >= 0, got %d", stats.CurrentTokens)
	}
}

func TestExecuteWithConfig_InvalidRate(t *testing.T) {
	config := Config{
		Rate:     0, // Invalid
		Interval: time.Second,
	}

	manager := NewManagerWithConfig(config)

	// Should use default
	stats := manager.Stats()
	if stats.CurrentRate <= 0 {
		t.Error("Expected valid rate after fixing invalid config")
	}
}

func TestExecuteWithConfig_InvalidInterval(t *testing.T) {
	config := Config{
		Rate:     10,
		Interval: 0, // Invalid
	}

	manager := NewManagerWithConfig(config)

	// Should use default
	stats := manager.Stats()
	if stats.CurrentInterval <= 0 {
		t.Error("Expected valid interval after fixing invalid config")
	}
}
