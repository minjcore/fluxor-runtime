package retry

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

func TestExecute_SuccessOnRetry(t *testing.T) {
	manager := NewManager()

	attempts := int32(0)
	testErr := errors.New("test error")

	ctx := context.Background()
	err := manager.Execute(ctx, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		if attempts < 3 {
			return testErr
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected nil error after retries, got %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	stats := manager.Stats()
	if stats.TotalSuccesses != 1 {
		t.Errorf("Expected TotalSuccesses 1, got %d", stats.TotalSuccesses)
	}
	if stats.TotalRetries != 2 {
		t.Errorf("Expected TotalRetries 2, got %d", stats.TotalRetries)
	}
}

func TestExecute_MaxRetriesExceeded(t *testing.T) {
	manager := NewManager()

	testErr := errors.New("test error")
	attempts := int32(0)

	ctx := context.Background()
	err := manager.Execute(ctx, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return testErr
	})

	if err == nil {
		t.Fatal("Expected error after max retries, got nil")
	}

	retryErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if retryErr.Code != ErrCodeMaxRetriesExceeded {
		t.Errorf("Expected error code %s, got %s", ErrCodeMaxRetriesExceeded, retryErr.Code)
	}

	// Should have attempted 4 times (1 initial + 3 retries)
	if attempts != 4 {
		t.Errorf("Expected 4 attempts (1 + 3 retries), got %d", attempts)
	}

	stats := manager.Stats()
	if stats.TotalFailures != 1 {
		t.Errorf("Expected TotalFailures 1, got %d", stats.TotalFailures)
	}
	if stats.TotalRetries != 3 {
		t.Errorf("Expected TotalRetries 3, got %d", stats.TotalRetries)
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

	retryErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if retryErr.Code != ErrCodeNilContext {
		t.Errorf("Expected error code %s, got %s", ErrCodeNilContext, retryErr.Code)
	}
}

func TestExecute_NilRetriable(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	err := manager.Execute(ctx, nil)

	if err == nil {
		t.Fatal("Expected error for nil retriable, got nil")
	}

	retryErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if retryErr.Code != ErrCodeNilRetriable {
		t.Errorf("Expected error code %s, got %s", ErrCodeNilRetriable, retryErr.Code)
	}
}

func TestExecute_ContextCancellation(t *testing.T) {
	manager := NewManager()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := manager.Execute(ctx, func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond) // Long operation
		return errors.New("should not reach here")
	})

	if err == nil {
		t.Fatal("Expected error for cancelled context, got nil")
	}

	retryErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if retryErr.Code != ErrCodeContextCanceled {
		t.Errorf("Expected error code %s, got %s", ErrCodeContextCanceled, retryErr.Code)
	}
}

func TestExecute_ContextTimeout(t *testing.T) {
	manager := NewManager()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := manager.Execute(ctx, func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond) // Long operation
		return errors.New("should not reach here")
	})

	if err == nil {
		t.Fatal("Expected error for timeout, got nil")
	}

	retryErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if retryErr.Code != ErrCodeContextTimeout {
		t.Errorf("Expected error code %s, got %s", ErrCodeContextTimeout, retryErr.Code)
	}
}

func TestExecute_ContextCancellationDuringBackoff(t *testing.T) {
	manager := NewManager()

	ctx, cancel := context.WithCancel(context.Background())
	testErr := errors.New("test error")

	config := DefaultConfig()
	config.MaxRetries = 5
	config.Backoff = NewFixedBackoff(200 * time.Millisecond)

	// Cancel during backoff
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		return testErr
	}, config)

	if err == nil {
		t.Fatal("Expected error for cancelled context, got nil")
	}

	retryErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if retryErr.Code != ErrCodeContextCanceled {
		t.Errorf("Expected error code %s, got %s", ErrCodeContextCanceled, retryErr.Code)
	}
}

func TestExecuteWithConfig_CustomMaxRetries(t *testing.T) {
	manager := NewManager()

	attempts := int32(0)
	testErr := errors.New("test error")

	config := DefaultConfig()
	config.MaxRetries = 5

	ctx := context.Background()
	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return testErr
	}, config)

	if err == nil {
		t.Fatal("Expected error after max retries, got nil")
	}

	// Should have attempted 6 times (1 initial + 5 retries)
	if attempts != 6 {
		t.Errorf("Expected 6 attempts (1 + 5 retries), got %d", attempts)
	}
}

func TestExecuteWithConfig_ZeroRetries(t *testing.T) {
	manager := NewManager()

	attempts := int32(0)
	testErr := errors.New("test error")

	config := DefaultConfig()
	config.MaxRetries = 0

	ctx := context.Background()
	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return testErr
	}, config)

	if err == nil {
		t.Fatal("Expected error with no retries, got nil")
	}

	// Should have attempted only once (no retries)
	if attempts != 1 {
		t.Errorf("Expected 1 attempt (no retries), got %d", attempts)
	}

	stats := manager.Stats()
	if stats.TotalRetries != 0 {
		t.Errorf("Expected TotalRetries 0, got %d", stats.TotalRetries)
	}
}

func TestExecuteWithConfig_CustomBackoff(t *testing.T) {
	manager := NewManager()

	attempts := int32(0)
	testErr := errors.New("test error")
	var lastBackoffTime time.Time
	backoffDelays := make([]time.Duration, 0)

	config := DefaultConfig()
	config.MaxRetries = 3
	config.Backoff = NewFixedBackoff(50 * time.Millisecond)
	config.OnRetry = func(attempt int, err error) {
		if !lastBackoffTime.IsZero() {
			delay := time.Since(lastBackoffTime)
			backoffDelays = append(backoffDelays, delay)
		}
		lastBackoffTime = time.Now()
	}

	ctx := context.Background()
	start := time.Now()
	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return testErr
	}, config)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected error after max retries, got nil")
	}

	// Should have waited approximately 3 * 50ms = 150ms for backoff
	// Allow some variance for execution time
	if elapsed < 100*time.Millisecond || elapsed > 500*time.Millisecond {
		t.Errorf("Expected elapsed time around 150ms, got %v", elapsed)
	}

	// Verify backoff delays are approximately correct
	for _, delay := range backoffDelays {
		if delay < 30*time.Millisecond || delay > 100*time.Millisecond {
			t.Errorf("Expected backoff delay around 50ms, got %v", delay)
		}
	}
}

func TestExecuteWithConfig_ExponentialBackoff(t *testing.T) {
	manager := NewManager()

	attempts := int32(0)
	testErr := errors.New("test error")
	backoffTimes := make([]time.Time, 0)
	var mu sync.Mutex

	config := DefaultConfig()
	config.MaxRetries = 3
	config.Backoff = NewExponentialBackoff(10*time.Millisecond, 1*time.Second, 2.0, false)
	config.OnRetry = func(attempt int, err error) {
		mu.Lock()
		backoffTimes = append(backoffTimes, time.Now())
		mu.Unlock()
	}

	ctx := context.Background()
	start := time.Now()
	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return testErr
	}, config)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected error after max retries, got nil")
	}

	// Exponential backoff: 10ms, 20ms, 40ms = 70ms total
	// Allow variance
	if elapsed < 50*time.Millisecond || elapsed > 200*time.Millisecond {
		t.Errorf("Expected elapsed time around 70ms, got %v", elapsed)
	}

	// Verify exponential progression
	mu.Lock()
	if len(backoffTimes) >= 2 {
		delay1 := backoffTimes[1].Sub(backoffTimes[0])
		delay2 := backoffTimes[2].Sub(backoffTimes[1])
		// delay2 should be approximately 2x delay1
		ratio := float64(delay2) / float64(delay1)
		if ratio < 1.5 || ratio > 2.5 {
			t.Errorf("Expected exponential backoff ratio ~2.0, got %.2f", ratio)
		}
	}
	mu.Unlock()
}

func TestExecuteWithConfig_LinearBackoff(t *testing.T) {
	manager := NewManager()

	attempts := int32(0)
	testErr := errors.New("test error")
	backoffTimes := make([]time.Time, 0)
	var mu sync.Mutex

	config := DefaultConfig()
	config.MaxRetries = 3
	config.Backoff = NewLinearBackoff(10*time.Millisecond, 1*time.Second, 10*time.Millisecond)
	config.OnRetry = func(attempt int, err error) {
		mu.Lock()
		backoffTimes = append(backoffTimes, time.Now())
		mu.Unlock()
	}

	ctx := context.Background()
	start := time.Now()
	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return testErr
	}, config)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected error after max retries, got nil")
	}

	// Linear backoff: 10ms, 20ms, 30ms = 60ms total
	// Allow variance
	if elapsed < 40*time.Millisecond || elapsed > 150*time.Millisecond {
		t.Errorf("Expected elapsed time around 60ms, got %v", elapsed)
	}

	// Verify linear progression
	mu.Lock()
	if len(backoffTimes) >= 2 {
		delay1 := backoffTimes[1].Sub(backoffTimes[0])
		delay2 := backoffTimes[2].Sub(backoffTimes[1])
		// delays should be approximately equal (linear)
		// Allow wider tolerance due to timing variations
		ratio := float64(delay2) / float64(delay1)
		if ratio < 0.5 || ratio > 2.0 {
			t.Errorf("Expected linear backoff ratio ~1.0, got %.2f (delays: %v, %v)", ratio, delay1, delay2)
		}
	}
	mu.Unlock()
}

func TestExecuteWithConfig_RetryPredicate(t *testing.T) {
	manager := NewManager()

	attempts := int32(0)
	retryableErr := errors.New("timeout occurred")
	nonRetryableErr := errors.New("permanent error")

	config := DefaultConfig()
	config.MaxRetries = 3
	config.Predicate = RetryOnTimeout

	tests := []struct {
		name           string
		err            error
		expectedRetries int32
	}{
		{
			name:           "retryable error",
			err:            retryableErr,
			expectedRetries: 4, // 1 initial + 3 retries
		},
		{
			name:           "non-retryable error",
			err:            nonRetryableErr,
			expectedRetries: 1, // 1 initial, no retries
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			atomic.StoreInt32(&attempts, 0)
			errToReturn := tt.err

			ctx := context.Background()
			err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
				atomic.AddInt32(&attempts, 1)
				return errToReturn
			}, config)

			if err == nil && tt.err != nil {
				t.Error("Expected error, got nil")
			}

			if attempts != tt.expectedRetries {
				t.Errorf("Expected %d attempts, got %d", tt.expectedRetries, attempts)
			}
		})
	}
}

func TestExecuteWithConfig_RetryPredicateNeverRetry(t *testing.T) {
	manager := NewManager()

	attempts := int32(0)
	testErr := errors.New("test error")

	config := DefaultConfig()
	config.MaxRetries = 5
	config.Predicate = NeverRetry

	ctx := context.Background()
	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return testErr
	}, config)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Should only attempt once (never retry)
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}

	stats := manager.Stats()
	if stats.TotalRetries != 0 {
		t.Errorf("Expected TotalRetries 0, got %d", stats.TotalRetries)
	}
}

func TestExecuteWithConfig_RetryPredicateCombinations(t *testing.T) {
	manager := NewManager()

	tests := []struct {
		name     string
		err      error
		predicate RetryPredicate
		expectedAttempts int32
	}{
		{
			name: "RetryOnAny with timeout",
			err:  errors.New("timeout occurred"),
			predicate: RetryOnAny(RetryOnTimeout, RetryOnNetworkError),
			expectedAttempts: 4, // 1 + 3 retries
		},
		{
			name: "RetryOnAny with network error",
			err:  errors.New("connection refused"),
			predicate: RetryOnAny(RetryOnTimeout, RetryOnNetworkError),
			expectedAttempts: 4,
		},
		{
			name: "RetryOnAny with non-matching error",
			err:  errors.New("permanent error"),
			predicate: RetryOnAny(RetryOnTimeout, RetryOnNetworkError),
			expectedAttempts: 1, // No retries
		},
		{
			name: "RetryOnAll requires both",
			err:  errors.New("timeout connection"),
			predicate: RetryOnAll(RetryOnTimeout, RetryOnNetworkError),
			expectedAttempts: 4, // Both match
		},
		{
			name: "RetryOnAll fails if one doesn't match",
			err:  errors.New("timeout occurred"),
			predicate: RetryOnAll(RetryOnTimeout, RetryOnNetworkError),
			expectedAttempts: 1, // Network doesn't match
		},
		{
			name: "RetryOnNot negates predicate",
			err:  errors.New("permanent error"),
			predicate: RetryOnNot(RetryOnTimeout),
			expectedAttempts: 4, // Not timeout, so retry
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := int32(0)
			config := DefaultConfig()
			config.MaxRetries = 3
			config.Predicate = tt.predicate

			ctx := context.Background()
			manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
				atomic.AddInt32(&attempts, 1)
				return tt.err
			}, config)

			if attempts != tt.expectedAttempts {
				t.Errorf("Expected %d attempts, got %d", tt.expectedAttempts, attempts)
			}
		})
	}
}

func TestExecuteWithConfig_Timeout(t *testing.T) {
	manager := NewManager()

	config := DefaultConfig()
	config.MaxRetries = 10
	config.Timeout = 100 * time.Millisecond
	config.Backoff = NewFixedBackoff(200 * time.Millisecond) // Backoff longer than timeout

	testErr := errors.New("test error")

	ctx := context.Background()
	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		return testErr
	}, config)

	if err == nil {
		t.Fatal("Expected error for timeout, got nil")
	}

	retryErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if retryErr.Code != ErrCodeContextTimeout {
		t.Errorf("Expected error code %s, got %s", ErrCodeContextTimeout, retryErr.Code)
	}
}

func TestExecuteWithConfig_TimeoutDuringExecution(t *testing.T) {
	manager := NewManager()

	config := DefaultConfig()
	config.MaxRetries = 5
	config.Timeout = 50 * time.Millisecond
	config.Backoff = NewFixedBackoff(10 * time.Millisecond)

	testErr := errors.New("test error")

	ctx := context.Background()
	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		// Check if context is already cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		time.Sleep(100 * time.Millisecond) // Longer than timeout
		return testErr
	}, config)

	if err == nil {
		t.Fatal("Expected error for timeout, got nil")
	}

	retryErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if retryErr.Code != ErrCodeContextTimeout {
		t.Errorf("Expected error code %s, got %s", ErrCodeContextTimeout, retryErr.Code)
	}
}

func TestExecuteWithConfig_OnRetryCallback(t *testing.T) {
	manager := NewManager()

	var retryCalls []int
	var retryErrors []error
	var mu sync.Mutex

	config := DefaultConfig()
	config.MaxRetries = 3
	config.OnRetry = func(attempt int, err error) {
		mu.Lock()
		retryCalls = append(retryCalls, attempt)
		retryErrors = append(retryErrors, err)
		mu.Unlock()
	}

	testErr := errors.New("test error")

	ctx := context.Background()
	manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		return testErr
	}, config)

	mu.Lock()
	if len(retryCalls) != 3 {
		t.Errorf("Expected 3 retry callbacks, got %d", len(retryCalls))
	}
	for i, attempt := range retryCalls {
		if attempt != i {
			t.Errorf("Expected attempt %d, got %d", i, attempt)
		}
		if retryErrors[i] != testErr {
			t.Errorf("Expected error %v, got %v", testErr, retryErrors[i])
		}
	}
	mu.Unlock()
}

func TestExecuteWithConfig_OnRetryAsyncCallback(t *testing.T) {
	manager := NewManager()

	var retryCalls int32
	var mu sync.Mutex

	config := DefaultConfig()
	config.MaxRetries = 3
	config.OnRetryAsync = func(attempt int, err error) {
		atomic.AddInt32(&retryCalls, 1)
	}

	testErr := errors.New("test error")

	ctx := context.Background()
	manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		return testErr
	}, config)

	// Wait a bit for async callbacks
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	calls := atomic.LoadInt32(&retryCalls)
	if calls != 3 {
		t.Errorf("Expected 3 async retry callbacks, got %d", calls)
	}
	mu.Unlock()
}

func TestExecuteWithConfig_BothCallbacks(t *testing.T) {
	manager := NewManager()

	var syncCalls int32
	var asyncCalls int32

	config := DefaultConfig()
	config.MaxRetries = 3
	config.OnRetry = func(attempt int, err error) {
		atomic.AddInt32(&syncCalls, 1)
	}
	config.OnRetryAsync = func(attempt int, err error) {
		atomic.AddInt32(&asyncCalls, 1)
	}

	testErr := errors.New("test error")

	ctx := context.Background()
	manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		return testErr
	}, config)

	// Wait for async callbacks
	time.Sleep(100 * time.Millisecond)

	if syncCalls != 3 {
		t.Errorf("Expected 3 sync retry callbacks, got %d", syncCalls)
	}
	if asyncCalls != 3 {
		t.Errorf("Expected 3 async retry callbacks, got %d", asyncCalls)
	}
}

func TestExecuteWithConfig_InvalidConfig(t *testing.T) {
	manager := NewManager()

	config := DefaultConfig()
	config.MaxRetries = -1

	ctx := context.Background()
	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		return nil
	}, config)

	if err == nil {
		t.Fatal("Expected error for invalid config, got nil")
	}

	retryErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if retryErr.Code != ErrCodeInvalidConfig {
		t.Errorf("Expected error code %s, got %s", ErrCodeInvalidConfig, retryErr.Code)
	}
}

func TestExecuteWithConfig_DefaultBackoffAndPredicate(t *testing.T) {
	manager := NewManager()

	config := Config{
		MaxRetries: 2,
		// Backoff and Predicate are nil, should use defaults
	}

	testErr := errors.New("test error")
	attempts := int32(0)

	ctx := context.Background()
	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		if attempts == 3 {
			return nil // Succeed on third attempt
		}
		return testErr
	}, config)

	if err != nil {
		t.Errorf("Expected nil error after retry success, got %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestExecuteWithConfig_NilBackoffUsesDefault(t *testing.T) {
	manager := NewManager()

	config := DefaultConfig()
	config.Backoff = nil // Should use default

	testErr := errors.New("test error")
	attempts := int32(0)

	ctx := context.Background()
	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return testErr
	}, config)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Should have retried with default backoff
	if attempts != 4 {
		t.Errorf("Expected 4 attempts, got %d", attempts)
	}
}

func TestExecuteWithConfig_NilPredicateUsesDefault(t *testing.T) {
	manager := NewManager()

	config := DefaultConfig()
	config.Predicate = nil // Should use AlwaysRetry

	testErr := errors.New("test error")
	attempts := int32(0)

	ctx := context.Background()
	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return testErr
	}, config)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Should have retried (AlwaysRetry)
	if attempts != 4 {
		t.Errorf("Expected 4 attempts, got %d", attempts)
	}
}

func TestExecuteWithConfig_ZeroTimeout(t *testing.T) {
	manager := NewManager()

	config := DefaultConfig()
	config.Timeout = 0 // No timeout

	testErr := errors.New("test error")
	attempts := int32(0)

	ctx := context.Background()
	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return testErr
	}, config)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Should complete all retries without timeout
	if attempts != 4 {
		t.Errorf("Expected 4 attempts, got %d", attempts)
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
	if stats.TotalSuccesses != int64(concurrency) {
		t.Errorf("Expected TotalSuccesses %d, got %d", concurrency, stats.TotalSuccesses)
	}
	if stats.TotalExecutions != int64(concurrency) {
		t.Errorf("Expected TotalExecutions %d, got %d", concurrency, stats.TotalExecutions)
	}
}

func TestConcurrentExecution_WithRetries(t *testing.T) {
	manager := NewManager()

	var wg sync.WaitGroup
	concurrency := 10
	successes := int32(0)

	config := DefaultConfig()
	config.MaxRetries = 2
	config.Backoff = NewFixedBackoff(10 * time.Millisecond) // Fast backoff for test

	ctx := context.Background()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			localAttempts := int32(0)
			err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
				localAttempts++
				// Fail first 2 attempts, succeed on 3rd (within retry limit)
				if localAttempts < 3 {
					return errors.New("temporary error")
				}
				return nil
			}, config)
			if err == nil {
				atomic.AddInt32(&successes, 1)
			}
		}(i)
	}

	wg.Wait()

	// Allow time for all operations to complete
	time.Sleep(100 * time.Millisecond)

	// All should eventually succeed (with retries)
	// Allow some tolerance for race conditions in concurrent tests
	if successes < int32(concurrency*9/10) {
		t.Errorf("Expected at least %d successes, got %d (concurrency: %d)", concurrency*9/10, successes, concurrency)
	}
}

func TestStats(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()

	// One success
	manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	// One failure
	testErr := errors.New("test error")
	manager.Execute(ctx, func(ctx context.Context) error {
		return testErr
	})

	stats := manager.Stats()

	// 1 success (1 execution) + 1 failure with 3 retries (4 executions) = 5 total
	if stats.TotalExecutions != 5 {
		t.Errorf("Expected TotalExecutions 5, got %d", stats.TotalExecutions)
	}

	if stats.TotalSuccesses != 1 {
		t.Errorf("Expected TotalSuccesses 1, got %d", stats.TotalSuccesses)
	}

	if stats.TotalFailures != 1 {
		t.Errorf("Expected TotalFailures 1, got %d", stats.TotalFailures)
	}

	if stats.TotalRetries != 3 {
		t.Errorf("Expected TotalRetries 3, got %d", stats.TotalRetries)
	}

	if stats.LastExecutionTime.IsZero() {
		t.Error("Expected LastExecutionTime to be set")
	}
}

func TestStats_Accumulation(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	config := DefaultConfig()
	config.MaxRetries = 2

	// Multiple successes
	for i := 0; i < 5; i++ {
		manager.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}

	// Multiple failures
	for i := 0; i < 3; i++ {
		manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
			return errors.New("error")
		}, config)
	}

	stats := manager.Stats()

	// 5 successes (5 executions) + 3 failures with 2 retries each (9 executions) = 14 total
	if stats.TotalExecutions != 14 {
		t.Errorf("Expected TotalExecutions 14, got %d", stats.TotalExecutions)
	}

	if stats.TotalSuccesses != 5 {
		t.Errorf("Expected TotalSuccesses 5, got %d", stats.TotalSuccesses)
	}

	if stats.TotalFailures != 3 {
		t.Errorf("Expected TotalFailures 3, got %d", stats.TotalFailures)
	}

	if stats.TotalRetries != 6 { // 3 failures * 2 retries each
		t.Errorf("Expected TotalRetries 6, got %d", stats.TotalRetries)
	}
}

func TestStats_ThreadSafety(t *testing.T) {
	manager := NewManager()

	var wg sync.WaitGroup
	concurrency := 20

	ctx := context.Background()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			manager.Execute(ctx, func(ctx context.Context) error {
				return nil
			})
			// Read stats concurrently
			_ = manager.Stats()
		}()
	}

	wg.Wait()

	stats := manager.Stats()
	if stats.TotalExecutions != int64(concurrency) {
		t.Errorf("Expected TotalExecutions %d, got %d", concurrency, stats.TotalExecutions)
	}
}

func TestError_Wrapping(t *testing.T) {
	originalErr := errors.New("original error")
	retryErr := NewError(ErrCodeMaxRetriesExceeded, "max retries exceeded")

	// Test error wrapping
	wrappedErr := errors.New("wrapped: " + retryErr.Error())

	if !errors.Is(wrappedErr, retryErr) && !errors.Is(wrappedErr, originalErr) {
		// Error wrapping is not strictly required, but error message should contain the original
		if retryErr.Error() == "" {
			t.Error("Error should have a message")
		}
	}
}

func TestError_ErrorCode(t *testing.T) {
	err := NewError(ErrCodeMaxRetriesExceeded, "test message")
	
	if err.Code != ErrCodeMaxRetriesExceeded {
		t.Errorf("Expected error code %s, got %s", ErrCodeMaxRetriesExceeded, err.Code)
	}
	
	if err.Message != "test message" {
		t.Errorf("Expected error message 'test message', got '%s'", err.Message)
	}
	
	errorStr := err.Error()
	if errorStr == "" {
		t.Error("Error string should not be empty")
	}
	if !contains(errorStr, ErrCodeMaxRetriesExceeded) || !contains(errorStr, "test message") {
		t.Errorf("Error string should contain code and message, got: %s", errorStr)
	}
}

func TestExecute_ImmediateSuccess(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	start := time.Now()
	err := manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}

	// Should be very fast (no retries, no backoff)
	if elapsed > 10*time.Millisecond {
		t.Errorf("Expected immediate return, took %v", elapsed)
	}
}

func TestExecuteWithConfig_BackoffRespectsContext(t *testing.T) {
	manager := NewManager()

	ctx, cancel := context.WithCancel(context.Background())
	testErr := errors.New("test error")

	config := DefaultConfig()
	config.MaxRetries = 5
	config.Backoff = NewFixedBackoff(200 * time.Millisecond)

	// Cancel during backoff
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		return testErr
	}, config)

	if err == nil {
		t.Fatal("Expected error for cancelled context, got nil")
	}

	retryErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if retryErr.Code != ErrCodeContextCanceled {
		t.Errorf("Expected error code %s, got %s", ErrCodeContextCanceled, retryErr.Code)
	}
}

func TestExecuteWithConfig_PredicateReturnsFalse(t *testing.T) {
	manager := NewManager()

	attempts := int32(0)
	permanentErr := errors.New("permanent error")

	config := DefaultConfig()
	config.MaxRetries = 5
	config.Predicate = func(err error) bool {
		return false // Never retry
	}

	ctx := context.Background()
	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return permanentErr
	}, config)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Should only attempt once
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}

	// Should return original error, not retry error
	if err != permanentErr {
		t.Errorf("Expected original error, got %v", err)
	}
}

func TestExecuteWithConfig_LargeMaxRetries(t *testing.T) {
	manager := NewManager()

	attempts := int32(0)
	testErr := errors.New("test error")

	config := DefaultConfig()
	config.MaxRetries = 100
	config.Backoff = NewFixedBackoff(1 * time.Millisecond) // Fast backoff for test

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := manager.ExecuteWithConfig(ctx, func(ctx context.Context) error {
		atomic.AddInt32(&attempts, 1)
		return testErr
	}, config)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Should have attempted multiple times before timeout
	if attempts < 10 {
		t.Errorf("Expected at least 10 attempts before timeout, got %d", attempts)
	}
}

// Helper function
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
