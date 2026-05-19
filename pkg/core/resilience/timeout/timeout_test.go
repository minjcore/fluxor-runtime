package timeout

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
	if stats.TotalExecutions != 0 {
		t.Errorf("Expected TotalExecutions 0, got %d", stats.TotalExecutions)
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
	if stats.TotalSuccesses != 1 {
		t.Errorf("Expected TotalSuccesses 1, got %d", stats.TotalSuccesses)
	}
	if stats.TotalExecutions != 1 {
		t.Errorf("Expected TotalExecutions 1, got %d", stats.TotalExecutions)
	}
}

func TestExecute_Error(t *testing.T) {
	manager := NewManager()

	testErr := errors.New("test error")
	ctx := context.Background()
	err := manager.Execute(ctx, func(ctx context.Context) error {
		return testErr
	})

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, err)
	}

	stats := manager.Stats()
	if stats.TotalExecutions != 1 {
		t.Errorf("Expected TotalExecutions 1, got %d", stats.TotalExecutions)
	}
	if stats.TotalSuccesses != 0 {
		t.Errorf("Expected TotalSuccesses 0, got %d", stats.TotalSuccesses)
	}
}

func TestExecuteWithTimeout_Timeout(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	err := manager.ExecuteWithTimeout(ctx, func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond) // Longer than timeout
		return nil
	}, 50*time.Millisecond)

	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	timeoutErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if timeoutErr.Code != ErrCodeTimeoutExceeded {
		t.Errorf("Expected error code %s, got %s", ErrCodeTimeoutExceeded, timeoutErr.Code)
	}

	stats := manager.Stats()
	if stats.TotalTimeouts != 1 {
		t.Errorf("Expected TotalTimeouts 1, got %d", stats.TotalTimeouts)
	}
}

func TestExecuteWithTimeout_SuccessWithinTimeout(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	err := manager.ExecuteWithTimeout(ctx, func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond) // Shorter than timeout
		return nil
	}, 100*time.Millisecond)

	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}

	stats := manager.Stats()
	if stats.TotalSuccesses != 1 {
		t.Errorf("Expected TotalSuccesses 1, got %d", stats.TotalSuccesses)
	}
	if stats.TotalTimeouts != 0 {
		t.Errorf("Expected TotalTimeouts 0, got %d", stats.TotalTimeouts)
	}
}

func TestExecuteWithTimeout_ContextCancellation(t *testing.T) {
	manager := NewManager()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel before timeout
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	err := manager.ExecuteWithTimeout(ctx, func(ctx context.Context) error {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return nil
		}
	}, 100*time.Millisecond)

	if err == nil {
		t.Fatal("Expected error for cancelled context, got nil")
	}

	timeoutErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	// Could be cancellation or timeout depending on timing
	// Both are acceptable behaviors
	if timeoutErr.Code != ErrCodeContextCanceled && timeoutErr.Code != ErrCodeTimeoutExceeded {
		t.Errorf("Expected error code %s or %s, got %s", ErrCodeContextCanceled, ErrCodeTimeoutExceeded, timeoutErr.Code)
	}
}

func TestExecuteWithTimeout_ContextAlreadyExpired(t *testing.T) {
	manager := NewManager()

	ctx, cancel := context.WithTimeout(context.Background(), -time.Second)
	defer cancel()

	err := manager.ExecuteWithTimeout(ctx, func(ctx context.Context) error {
		return nil
	}, 100*time.Millisecond)

	if err == nil {
		t.Fatal("Expected error for expired context, got nil")
	}

	timeoutErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	// Context with negative timeout expires immediately with DeadlineExceeded
	if timeoutErr.Code != ErrCodeTimeoutExceeded {
		t.Errorf("Expected error code %s, got %s", ErrCodeTimeoutExceeded, timeoutErr.Code)
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

	timeoutErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if timeoutErr.Code != ErrCodeNilContext {
		t.Errorf("Expected error code %s, got %s", ErrCodeNilContext, timeoutErr.Code)
	}
}

func TestExecute_NilFunction(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	err := manager.Execute(ctx, nil)

	if err == nil {
		t.Fatal("Expected error for nil function, got nil")
	}

	timeoutErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if timeoutErr.Code != ErrCodeNilFunction {
		t.Errorf("Expected error code %s, got %s", ErrCodeNilFunction, timeoutErr.Code)
	}
}

func TestExecuteWithConfig_ZeroTimeout(t *testing.T) {
	manager := NewManager()

	// Zero timeout should use context timeout if available
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	}, Config{Timeout: 0})

	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	stats := manager.Stats()
	if stats.TotalTimeouts != 1 {
		t.Errorf("Expected TotalTimeouts 1, got %d", stats.TotalTimeouts)
	}
}

func TestExecuteWithConfig_ZeroTimeoutNoContextDeadline(t *testing.T) {
	manager := NewManager()

	// Zero timeout with no context deadline should execute without timeout
	ctx := context.Background()
	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		return nil
	}, Config{Timeout: 0})

	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}

	stats := manager.Stats()
	if stats.TotalSuccesses != 1 {
		t.Errorf("Expected TotalSuccesses 1, got %d", stats.TotalSuccesses)
	}
}

func TestExecuteWithConfig_OnTimeoutCallback(t *testing.T) {
	manager := NewManager()

	var timeoutCalled bool
	var timeoutCtx context.Context
	var timeoutDuration time.Duration
	var mu sync.Mutex

	config := Config{
		Timeout: 50 * time.Millisecond,
		OnTimeout: func(ctx context.Context, timeout time.Duration) {
			mu.Lock()
			timeoutCalled = true
			timeoutCtx = ctx
			timeoutDuration = timeout
			mu.Unlock()
		},
	}

	ctx := context.Background()
	manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}, config)

	mu.Lock()
	if !timeoutCalled {
		t.Error("Expected OnTimeout callback to be called")
	}
	if timeoutDuration != 50*time.Millisecond {
		t.Errorf("Expected timeout duration %v, got %v", 50*time.Millisecond, timeoutDuration)
	}
	if timeoutCtx == nil {
		t.Error("Expected timeout context to be set")
	}
	mu.Unlock()
}

func TestExecuteWithConfig_OnTimeoutAsyncCallback(t *testing.T) {
	manager := NewManager()

	var timeoutCalled int32
	var mu sync.Mutex

	config := Config{
		Timeout: 50 * time.Millisecond,
		OnTimeoutAsync: func(ctx context.Context, timeout time.Duration) {
			atomic.AddInt32(&timeoutCalled, 1)
		},
	}

	ctx := context.Background()
	manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}, config)

	// Wait a bit for async callback
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	calls := atomic.LoadInt32(&timeoutCalled)
	if calls != 1 {
		t.Errorf("Expected OnTimeoutAsync callback to be called once, got %d", calls)
	}
	mu.Unlock()
}

func TestExecuteWithConfig_ContextDeadlineBeforeTimeout(t *testing.T) {
	manager := NewManager()

	// Context timeout is shorter than config timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	config := Config{
		Timeout: 200 * time.Millisecond, // Longer than context timeout
	}

	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	}, config)

	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	// Should respect context timeout (shorter one)
	stats := manager.Stats()
	if stats.TotalTimeouts != 1 {
		t.Errorf("Expected TotalTimeouts 1, got %d", stats.TotalTimeouts)
	}
}

func TestExecuteWithConfig_InvalidTimeout(t *testing.T) {
	manager := NewManager()

	config := Config{
		Timeout: -1 * time.Second, // Negative timeout
	}

	ctx := context.Background()
	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		return nil
	}, config)

	if err == nil {
		t.Fatal("Expected error for invalid timeout, got nil")
	}

	timeoutErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if timeoutErr.Code != ErrCodeInvalidTimeout {
		t.Errorf("Expected error code %s, got %s", ErrCodeInvalidTimeout, timeoutErr.Code)
	}
}

func TestExecute_FunctionRespectsContextCancellation(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	err := manager.ExecuteWithTimeout(ctx, func(ctx context.Context) error {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return nil
		}
	}, 50*time.Millisecond)

	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	// Function should detect context cancellation
	timeoutErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	// Could be timeout or cancellation depending on timing
	if timeoutErr.Code != ErrCodeTimeoutExceeded && timeoutErr.Code != ErrCodeContextCanceled {
		t.Errorf("Expected error code %s or %s, got %s", ErrCodeTimeoutExceeded, ErrCodeContextCanceled, timeoutErr.Code)
	}
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
			err := manager.ExecuteWithTimeout(ctx, func(ctx context.Context) error {
				return nil
			}, 100*time.Millisecond)
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
	if stats.TotalSuccesses != int64(concurrency) {
		t.Errorf("Expected TotalSuccesses %d, got %d", concurrency, stats.TotalSuccesses)
	}
	if stats.TotalExecutions != int64(concurrency) {
		t.Errorf("Expected TotalExecutions %d, got %d", concurrency, stats.TotalExecutions)
	}
}

func TestStats(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()

	// One success
	manager.ExecuteWithTimeout(ctx, func(ctx context.Context) error {
		return nil
	}, 100*time.Millisecond)

	// One timeout
	manager.ExecuteWithTimeout(ctx, func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}, 50*time.Millisecond)

	// One cancellation
	ctxCancel, cancel := context.WithCancel(ctx)
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	manager.ExecuteWithTimeout(ctxCancel, func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	}, 200*time.Millisecond)

	stats := manager.Stats()

	if stats.TotalExecutions != 3 {
		t.Errorf("Expected TotalExecutions 3, got %d", stats.TotalExecutions)
	}

	if stats.TotalSuccesses != 1 {
		t.Errorf("Expected TotalSuccesses 1, got %d", stats.TotalSuccesses)
	}

	if stats.TotalTimeouts != 1 {
		t.Errorf("Expected TotalTimeouts 1, got %d", stats.TotalTimeouts)
	}

	if stats.TotalCancellations != 1 {
		t.Errorf("Expected TotalCancellations 1, got %d", stats.TotalCancellations)
	}

	if stats.LastExecutionTime.IsZero() {
		t.Error("Expected LastExecutionTime to be set")
	}
}

func TestExecuteWithTimeout_ErrorReturned(t *testing.T) {
	manager := NewManager()

	testErr := errors.New("operation error")
	ctx := context.Background()
	err := manager.ExecuteWithTimeout(ctx, func(ctx context.Context) error {
		return testErr
	}, 100*time.Millisecond)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, err)
	}

	stats := manager.Stats()
	if stats.TotalExecutions != 1 {
		t.Errorf("Expected TotalExecutions 1, got %d", stats.TotalExecutions)
	}
	if stats.TotalSuccesses != 0 {
		t.Errorf("Expected TotalSuccesses 0, got %d", stats.TotalSuccesses)
	}
}
