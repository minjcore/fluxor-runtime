package breaker

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

	if manager.State() != StateClosed {
		t.Errorf("Expected initial state %s, got %s", StateClosed, manager.State())
	}

	stats := manager.Stats()
	if stats.CurrentState != StateClosed {
		t.Errorf("Expected CurrentState %s, got %s", StateClosed, stats.CurrentState)
	}
}

func TestNewManagerWithConfig(t *testing.T) {
	config := Config{
		FailureThreshold:    3,
		SuccessThreshold:    2,
		Timeout:             10 * time.Second,
		HalfOpenMaxRequests: 2,
	}

	manager := NewManagerWithConfig(config)
	if manager == nil {
		t.Fatal("NewManagerWithConfig returned nil")
	}

	if manager.State() != StateClosed {
		t.Errorf("Expected initial state %s, got %s", StateClosed, manager.State())
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

func TestExecute_Failure(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	testErr := errors.New("test error")
	err := manager.Execute(ctx, func(ctx context.Context) error {
		return testErr
	})

	if err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, err)
	}

	stats := manager.Stats()
	if stats.TotalAllowed != 1 {
		t.Errorf("Expected TotalAllowed 1, got %d", stats.TotalAllowed)
	}
	if stats.TotalFailed != 1 {
		t.Errorf("Expected TotalFailed 1, got %d", stats.TotalFailed)
	}
	if stats.ConsecutiveFailures != 1 {
		t.Errorf("Expected ConsecutiveFailures 1, got %d", stats.ConsecutiveFailures)
	}
}

func TestExecute_CircuitOpens(t *testing.T) {
	config := Config{
		FailureThreshold: 3,
		Timeout:          100 * time.Millisecond,
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Fail enough times to open circuit
	for i := 0; i < 3; i++ {
		manager.Execute(ctx, func(ctx context.Context) error {
			return errors.New("failure")
		})
	}

	// Circuit should be open now
	if manager.State() != StateOpen {
		t.Errorf("Expected state %s, got %s", StateOpen, manager.State())
	}

	// Next execution should be rejected
	err := manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	if err == nil {
		t.Fatal("Expected error for circuit open, got nil")
	}

	breakerErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if breakerErr.Code != ErrCodeCircuitOpen {
		t.Errorf("Expected error code %s, got %s", ErrCodeCircuitOpen, breakerErr.Code)
	}

	stats := manager.Stats()
	if stats.TotalRejected != 1 {
		t.Errorf("Expected TotalRejected 1, got %d", stats.TotalRejected)
	}
}

func TestExecute_CircuitHalfOpen(t *testing.T) {
	config := Config{
		FailureThreshold:    2,
		SuccessThreshold:    1,
		Timeout:             50 * time.Millisecond,
		HalfOpenMaxRequests: 1,
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Open circuit
	for i := 0; i < 2; i++ {
		manager.Execute(ctx, func(ctx context.Context) error {
			return errors.New("failure")
		})
	}

	if manager.State() != StateOpen {
		t.Errorf("Expected state %s, got %s", StateOpen, manager.State())
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Execute success - should transition to HalfOpen, then close circuit on success
	// The transition to HalfOpen happens in allowExecution when Execute is called
	err := manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}

	// Circuit should be closed after success in HalfOpen
	if manager.State() != StateClosed {
		t.Errorf("Expected state %s after success, got %s", StateClosed, manager.State())
	}
}

func TestExecute_HalfOpenFailure(t *testing.T) {
	config := Config{
		FailureThreshold:    2,
		SuccessThreshold:    1,
		Timeout:             50 * time.Millisecond,
		HalfOpenMaxRequests: 1,
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Open circuit
	for i := 0; i < 2; i++ {
		manager.Execute(ctx, func(ctx context.Context) error {
			return errors.New("failure")
		})
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Execute failure in HalfOpen - should open again
	err := manager.Execute(ctx, func(ctx context.Context) error {
		return errors.New("failure")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	if manager.State() != StateOpen {
		t.Errorf("Expected state %s after HalfOpen failure, got %s", StateOpen, manager.State())
	}
}

func TestAllow_Closed(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()

	if !manager.Allow(ctx) {
		t.Error("Expected Allow to return true in Closed state")
	}
}

func TestAllow_Open(t *testing.T) {
	config := Config{
		FailureThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Open circuit
	for i := 0; i < 2; i++ {
		manager.Execute(ctx, func(ctx context.Context) error {
			return errors.New("failure")
		})
	}

	// Allow should return false in Open state (before timeout)
	if manager.Allow(ctx) {
		t.Error("Expected Allow to return false in Open state")
	}
}

func TestAllow_HalfOpen(t *testing.T) {
	config := Config{
		FailureThreshold:    2,
		Timeout:             50 * time.Millisecond,
		HalfOpenMaxRequests: 2,
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Open circuit
	for i := 0; i < 2; i++ {
		manager.Execute(ctx, func(ctx context.Context) error {
			return errors.New("failure")
		})
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Should allow up to HalfOpenMaxRequests
	if !manager.Allow(ctx) {
		t.Error("Expected Allow to return true in HalfOpen state (first request)")
	}

	if !manager.Allow(ctx) {
		t.Error("Expected Allow to return true in HalfOpen state (second request)")
	}

	// Third request should be rejected (max reached)
	if manager.Allow(ctx) {
		t.Error("Expected Allow to return false in HalfOpen state (max requests reached)")
	}
}

func TestSuccess_Manual(t *testing.T) {
	manager := NewManager()

	manager.Success()

	stats := manager.Stats()
	if stats.ConsecutiveSuccesses != 1 {
		t.Errorf("Expected ConsecutiveSuccesses 1, got %d", stats.ConsecutiveSuccesses)
	}
}

func TestFailure_Manual(t *testing.T) {
	manager := NewManager()

	manager.Failure()

	stats := manager.Stats()
	if stats.ConsecutiveFailures != 1 {
		t.Errorf("Expected ConsecutiveFailures 1, got %d", stats.ConsecutiveFailures)
	}
}

func TestReset(t *testing.T) {
	config := Config{
		FailureThreshold: 2,
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Open circuit
	for i := 0; i < 2; i++ {
		manager.Execute(ctx, func(ctx context.Context) error {
			return errors.New("failure")
		})
	}

	if manager.State() != StateOpen {
		t.Errorf("Expected state %s, got %s", StateOpen, manager.State())
	}

	// Reset
	manager.Reset()

	if manager.State() != StateClosed {
		t.Errorf("Expected state %s after reset, got %s", StateClosed, manager.State())
	}

	stats := manager.Stats()
	if stats.ConsecutiveFailures != 0 {
		t.Errorf("Expected ConsecutiveFailures 0 after reset, got %d", stats.ConsecutiveFailures)
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

	breakerErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if breakerErr.Code != ErrCodeNilContext {
		t.Errorf("Expected error code %s, got %s", ErrCodeNilContext, breakerErr.Code)
	}
}

func TestExecute_NilFunction(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	err := manager.Execute(ctx, nil)

	if err == nil {
		t.Fatal("Expected error for nil function, got nil")
	}

	breakerErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if breakerErr.Code != ErrCodeNilFunction {
		t.Errorf("Expected error code %s, got %s", ErrCodeNilFunction, breakerErr.Code)
	}
}

func TestOnStateChangedCallback(t *testing.T) {
	var callbackCalled bool
	var fromState, toState State
	var mu sync.Mutex

	config := Config{
		FailureThreshold: 2,
		OnStateChanged: func(ctx context.Context, from State, to State) {
			mu.Lock()
			callbackCalled = true
			fromState = from
			toState = to
			mu.Unlock()
		},
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Open circuit
	for i := 0; i < 2; i++ {
		manager.Execute(ctx, func(ctx context.Context) error {
			return errors.New("failure")
		})
	}

	// Wait a bit for callback
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if !callbackCalled {
		t.Error("Expected OnStateChanged callback to be called")
	}
	if fromState != StateClosed {
		t.Errorf("Expected fromState %s, got %s", StateClosed, fromState)
	}
	if toState != StateOpen {
		t.Errorf("Expected toState %s, got %s", StateOpen, toState)
	}
	mu.Unlock()
}

func TestOnOpenedCallback(t *testing.T) {
	var callbackCalled bool
	var mu sync.Mutex

	config := Config{
		FailureThreshold: 2,
		OnOpened: func(ctx context.Context) {
			mu.Lock()
			callbackCalled = true
			mu.Unlock()
		},
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Open circuit
	for i := 0; i < 2; i++ {
		manager.Execute(ctx, func(ctx context.Context) error {
			return errors.New("failure")
		})
	}

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if !callbackCalled {
		t.Error("Expected OnOpened callback to be called")
	}
	mu.Unlock()
}

func TestOnClosedCallback(t *testing.T) {
	var callbackCalled bool
	var mu sync.Mutex

	config := Config{
		FailureThreshold: 2,
		SuccessThreshold: 1,
		Timeout:          50 * time.Millisecond,
		OnClosed: func(ctx context.Context) {
			mu.Lock()
			callbackCalled = true
			mu.Unlock()
		},
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Open circuit
	for i := 0; i < 2; i++ {
		manager.Execute(ctx, func(ctx context.Context) error {
			return errors.New("failure")
		})
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Close circuit with success
	manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if !callbackCalled {
		t.Error("Expected OnClosed callback to be called")
	}
	mu.Unlock()
}

func TestOnExecutionRejectedCallback(t *testing.T) {
	var callbackCalled bool
	var mu sync.Mutex

	config := Config{
		FailureThreshold: 2,
		OnExecutionRejected: func(ctx context.Context) {
			mu.Lock()
			callbackCalled = true
			mu.Unlock()
		},
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Open circuit
	for i := 0; i < 2; i++ {
		manager.Execute(ctx, func(ctx context.Context) error {
			return errors.New("failure")
		})
	}

	// Try to execute (should be rejected)
	manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	mu.Lock()
	if !callbackCalled {
		t.Error("Expected OnExecutionRejected callback to be called")
	}
	mu.Unlock()
}

func TestConcurrentExecution(t *testing.T) {
	config := Config{
		FailureThreshold: 5,
	}

	manager := NewManagerWithConfig(config)

	var wg sync.WaitGroup
	concurrency := 10
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

	if successes != int32(concurrency) {
		t.Errorf("Expected %d successes, got %d", concurrency, successes)
	}

	stats := manager.Stats()
	if stats.TotalExecutions != int64(concurrency) {
		t.Errorf("Expected TotalExecutions %d, got %d", concurrency, stats.TotalExecutions)
	}
}

func TestStats(t *testing.T) {
	config := Config{
		FailureThreshold: 3,
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Success
	manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	// Failures
	for i := 0; i < 2; i++ {
		manager.Execute(ctx, func(ctx context.Context) error {
			return errors.New("failure")
		})
	}

	stats := manager.Stats()

	if stats.TotalExecutions != 3 {
		t.Errorf("Expected TotalExecutions 3, got %d", stats.TotalExecutions)
	}

	if stats.TotalAllowed != 3 {
		t.Errorf("Expected TotalAllowed 3, got %d", stats.TotalAllowed)
	}

	if stats.TotalSuccessful != 1 {
		t.Errorf("Expected TotalSuccessful 1, got %d", stats.TotalSuccessful)
	}

	if stats.TotalFailed != 2 {
		t.Errorf("Expected TotalFailed 2, got %d", stats.TotalFailed)
	}

	if stats.ConsecutiveFailures != 2 {
		t.Errorf("Expected ConsecutiveFailures 2, got %d", stats.ConsecutiveFailures)
	}

	if stats.CurrentState != StateClosed {
		t.Errorf("Expected CurrentState %s, got %s", StateClosed, stats.CurrentState)
	}
}

func TestExecuteWithConfig_InvalidThreshold(t *testing.T) {
	config := Config{
		FailureThreshold: 0, // Invalid
		SuccessThreshold: 0, // Invalid
	}

	manager := NewManagerWithConfig(config)

	// Should use defaults
	stats := manager.Stats()
	if stats.CurrentState != StateClosed {
		t.Error("Expected valid state after fixing invalid config")
	}
}

func TestSuccessThreshold(t *testing.T) {
	config := Config{
		FailureThreshold:    2,
		SuccessThreshold:    3, // Need 3 successes to close
		Timeout:             50 * time.Millisecond,
		HalfOpenMaxRequests: 5,
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Open circuit
	for i := 0; i < 2; i++ {
		manager.Execute(ctx, func(ctx context.Context) error {
			return errors.New("failure")
		})
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Execute 2 successes (not enough)
	for i := 0; i < 2; i++ {
		manager.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}

	// Circuit should still be HalfOpen
	if manager.State() != StateHalfOpen {
		t.Errorf("Expected state %s after 2 successes, got %s", StateHalfOpen, manager.State())
	}

	// Third success should close circuit
	manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	if manager.State() != StateClosed {
		t.Errorf("Expected state %s after 3 successes, got %s", StateClosed, manager.State())
	}
}
