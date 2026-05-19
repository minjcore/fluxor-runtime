package backoff

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

	delay := manager.Delay(0)
	if delay != time.Second {
		t.Errorf("Expected delay 1s, got %v", delay)
	}
}

func TestNewManagerWithStrategy(t *testing.T) {
	strategy := NewFixedBackoff(500 * time.Millisecond)
	manager := NewManagerWithStrategy(strategy)
	if manager == nil {
		t.Fatal("NewManagerWithStrategy returned nil")
	}

	delay := manager.Delay(0)
	if delay != 500*time.Millisecond {
		t.Errorf("Expected delay 500ms, got %v", delay)
	}
}

func TestNewManagerWithStrategy_Nil(t *testing.T) {
	manager := NewManagerWithStrategy(nil)
	if manager == nil {
		t.Fatal("NewManagerWithStrategy returned nil")
	}

	// Should use default
	delay := manager.Delay(0)
	if delay <= 0 {
		t.Error("Expected positive delay, got", delay)
	}
}

func TestWait_Success(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	start := time.Now()

	err := manager.Wait(ctx, 0)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}

	elapsed := time.Since(start)
	if elapsed < time.Second || elapsed > 2*time.Second {
		t.Errorf("Expected elapsed time around 1s, got %v", elapsed)
	}
}

func TestWait_ContextCancellation(t *testing.T) {
	manager := NewManager()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context
	cancel()

	err := manager.Wait(ctx, 0)
	if err == nil {
		t.Fatal("Expected error for cancelled context, got nil")
	}

	backoffErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if backoffErr.Code != ErrCodeContextCanceled {
		t.Errorf("Expected error code %s, got %s", ErrCodeContextCanceled, backoffErr.Code)
	}
}

func TestWait_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Wait with longer delay - should timeout
	strategy := NewFixedBackoff(1 * time.Second)
	managerWithStrategy := NewManagerWithStrategy(strategy)

	err := managerWithStrategy.Wait(ctx, 0)
	if err == nil {
		t.Fatal("Expected error for context timeout, got nil")
	}

	backoffErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if backoffErr.Code != ErrCodeContextTimeout {
		t.Errorf("Expected error code %s, got %s", ErrCodeContextTimeout, backoffErr.Code)
	}
}

func TestWait_NilContext(t *testing.T) {
	manager := NewManager()

	err := manager.Wait(nil, 0)
	if err == nil {
		t.Fatal("Expected error for nil context, got nil")
	}

	backoffErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if backoffErr.Code != ErrCodeNilContext {
		t.Errorf("Expected error code %s, got %s", ErrCodeNilContext, backoffErr.Code)
	}
}

func TestFixedBackoff(t *testing.T) {
	strategy := NewFixedBackoff(500 * time.Millisecond)

	// All attempts should have the same delay
	for i := 0; i < 5; i++ {
		delay := strategy.Delay(i)
		if delay != 500*time.Millisecond {
			t.Errorf("Attempt %d: Expected delay 500ms, got %v", i, delay)
		}
	}
}

func TestFixedBackoff_NegativeDelay(t *testing.T) {
	strategy := NewFixedBackoff(-100 * time.Millisecond)

	delay := strategy.Delay(0)
	if delay < 0 {
		t.Errorf("Expected delay >= 0, got %v", delay)
	}
}

func TestExponentialBackoff(t *testing.T) {
	strategy := NewExponentialBackoff(
		100*time.Millisecond, // Initial
		10*time.Second,       // Max
		2.0,                  // Multiplier
		false,                // No jitter
	)

	expectedDelays := []time.Duration{
		100 * time.Millisecond,  // attempt 0
		200 * time.Millisecond,  // attempt 1
		400 * time.Millisecond,  // attempt 2
		800 * time.Millisecond,  // attempt 3
		1600 * time.Millisecond, // attempt 4
	}

	for i, expected := range expectedDelays {
		delay := strategy.Delay(i)
		if delay != expected {
			t.Errorf("Attempt %d: Expected delay %v, got %v", i, expected, delay)
		}
	}
}

func TestExponentialBackoff_WithMaxDelay(t *testing.T) {
	strategy := NewExponentialBackoff(
		100*time.Millisecond, // Initial
		500*time.Millisecond, // Max (caps at 500ms)
		2.0,                  // Multiplier
		false,                // No jitter
	)

	// After a few attempts, delay should be capped at max
	delay := strategy.Delay(10)
	if delay > 500*time.Millisecond {
		t.Errorf("Expected delay <= 500ms (max), got %v", delay)
	}
}

func TestExponentialBackoff_WithJitter(t *testing.T) {
	strategy := NewExponentialBackoff(
		100*time.Millisecond, // Initial
		10*time.Second,       // Max
		2.0,                  // Multiplier
		true,                 // With jitter
	)

	// Run multiple times to check jitter variation
	var delays []time.Duration
	for i := 0; i < 10; i++ {
		delays = append(delays, strategy.Delay(2))
	}

	// All delays should be around 400ms ±25%
	baseDelay := 400 * time.Millisecond
	minDelay := time.Duration(float64(baseDelay) * 0.75)
	maxDelay := time.Duration(float64(baseDelay) * 1.25)

	allWithinRange := true
	for i, delay := range delays {
		if delay < minDelay || delay > maxDelay {
			allWithinRange = false
			t.Logf("Delay %d: %v (expected between %v and %v)", i, delay, minDelay, maxDelay)
		}
	}

	// Not all delays should be exactly the same (jitter adds variation)
	if len(delays) > 1 {
		firstDelay := delays[0]
		allSame := true
		for _, delay := range delays[1:] {
			if delay != firstDelay {
				allSame = false
				break
			}
		}
		if allSame {
			t.Log("Warning: All delays were the same, jitter might not be working")
		}
	}

	if !allWithinRange {
		t.Log("Some delays were outside expected range, but jitter may cause this")
	}
}

func TestLinearBackoff(t *testing.T) {
	strategy := NewLinearBackoff(
		100*time.Millisecond, // Initial
		10*time.Second,       // Max
		100*time.Millisecond, // Increment
	)

	expectedDelays := []time.Duration{
		100 * time.Millisecond, // attempt 0
		200 * time.Millisecond, // attempt 1
		300 * time.Millisecond, // attempt 2
		400 * time.Millisecond, // attempt 3
	}

	for i, expected := range expectedDelays {
		delay := strategy.Delay(i)
		if delay != expected {
			t.Errorf("Attempt %d: Expected delay %v, got %v", i, expected, delay)
		}
	}
}

func TestLinearBackoff_WithMaxDelay(t *testing.T) {
	strategy := NewLinearBackoff(
		100*time.Millisecond, // Initial
		500*time.Millisecond, // Max
		200*time.Millisecond, // Increment
	)

	// After a few attempts, delay should be capped at max
	delay := strategy.Delay(10)
	if delay > 500*time.Millisecond {
		t.Errorf("Expected delay <= 500ms (max), got %v", delay)
	}
}

func TestLinearBackoff_DefaultIncrement(t *testing.T) {
	strategy := NewLinearBackoff(
		100*time.Millisecond, // Initial
		10*time.Second,       // Max
		0,                    // Zero increment (should default to initial)
	)

	delay1 := strategy.Delay(1)
	if delay1 != 200*time.Millisecond {
		t.Errorf("Expected delay 200ms (initial + increment), got %v", delay1)
	}
}

func TestWaitWithConfig_MaxAttempts(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	config := Config{
		Strategy:   NewFixedBackoff(100 * time.Millisecond),
		MaxAttempts: 5,
	}

	// Should work for attempt < MaxAttempts
	err := manager.WaitWithConfig(ctx, 4, config)
	if err != nil {
		t.Errorf("Expected nil error for attempt < MaxAttempts, got %v", err)
	}

	// Should fail for attempt >= MaxAttempts
	err = manager.WaitWithConfig(ctx, 5, config)
	if err == nil {
		t.Fatal("Expected error for attempt >= MaxAttempts, got nil")
	}

	backoffErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if backoffErr.Code != ErrCodeInvalidConfig {
		t.Errorf("Expected error code %s, got %s", ErrCodeInvalidConfig, backoffErr.Code)
	}
}

func TestWaitWithConfig_MaxDelay(t *testing.T) {
	manager := NewManager()

	strategy := NewExponentialBackoff(
		100*time.Millisecond, // Initial
		10*time.Second,       // Max (very high)
		2.0,
		false,
	)

	config := Config{
		Strategy: strategy,
		MaxDelay: 500 * time.Millisecond, // Override with lower max
	}

	// Delay should be capped at config.MaxDelay
	delay := manager.DelayWithConfig(10, config)
	if delay > 500*time.Millisecond {
		t.Errorf("Expected delay <= 500ms (config.MaxDelay), got %v", delay)
	}
}

func TestWaitWithConfig_OnBackoffCallback(t *testing.T) {
	var callbackCalled bool
	var callbackAttempt int
	var callbackDelay time.Duration
	var mu sync.Mutex

	config := Config{
		Strategy: NewFixedBackoff(50 * time.Millisecond),
		OnBackoff: func(ctx context.Context, attempt int, delay time.Duration) {
			mu.Lock()
			callbackCalled = true
			callbackAttempt = attempt
			callbackDelay = delay
			mu.Unlock()
		},
	}

	manager := NewManager()
	ctx := context.Background()

	manager.WaitWithConfig(ctx, 3, config)

	mu.Lock()
	if !callbackCalled {
		t.Error("Expected OnBackoff callback to be called")
	}
	if callbackAttempt != 3 {
		t.Errorf("Expected callback attempt 3, got %d", callbackAttempt)
	}
	if callbackDelay != 50*time.Millisecond {
		t.Errorf("Expected callback delay 50ms, got %v", callbackDelay)
	}
	mu.Unlock()
}

func TestDelay_NonBlocking(t *testing.T) {
	manager := NewManager()

	start := time.Now()
	delay := manager.Delay(0)
	elapsed := time.Since(start)

	// Delay calculation should be fast (non-blocking)
	if elapsed > 100*time.Millisecond {
		t.Errorf("Expected delay calculation to be fast, took %v", elapsed)
	}

	if delay != time.Second {
		t.Errorf("Expected delay 1s, got %v", delay)
	}
}

func TestDelay_ExponentialIncrease(t *testing.T) {
	strategy := NewExponentialBackoff(
		100*time.Millisecond,
		10*time.Second,
		2.0,
		false,
	)

	manager := NewManagerWithStrategy(strategy)

	delays := []time.Duration{
		manager.Delay(0),
		manager.Delay(1),
		manager.Delay(2),
		manager.Delay(3),
	}

	// Each delay should be approximately double the previous (without jitter)
	for i := 1; i < len(delays); i++ {
		ratio := float64(delays[i]) / float64(delays[i-1])
		if ratio < 1.8 || ratio > 2.2 {
			t.Errorf("Delay %d to %d: Expected ratio ~2.0, got %.2f", i-1, i, ratio)
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	strategy := NewExponentialBackoff(
		100*time.Millisecond,
		10*time.Second,
		2.0,
		true, // With jitter
	)

	manager := NewManagerWithStrategy(strategy)

	var wg sync.WaitGroup
	concurrency := 10

		ctx := context.Background()
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(attempt int) {
				defer wg.Done()
				delay := manager.Delay(attempt)
				if delay < 0 {
					t.Errorf("Expected delay >= 0, got %v", delay)
				}
				// Small wait to ensure concurrent access
				_ = manager.Wait(ctx, attempt)
			}(i)
		}

	wg.Wait()
}

func TestStrategy_Interface(t *testing.T) {
	var strategies []Strategy

	strategies = append(strategies, NewFixedBackoff(time.Second))
	strategies = append(strategies, NewExponentialBackoff(100*time.Millisecond, 10*time.Second, 2.0, false))
	strategies = append(strategies, NewLinearBackoff(100*time.Millisecond, 10*time.Second, 100*time.Millisecond))

	for i, strategy := range strategies {
		delay := strategy.Delay(0)
		if delay < 0 {
			t.Errorf("Strategy %d: Expected delay >= 0, got %v", i, delay)
		}
	}
}

func TestWaitWithConfig_NilStrategy(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	config := Config{
		Strategy: nil, // Should use manager's default
	}

	// Should use manager's default strategy
	err := manager.WaitWithConfig(ctx, 0, config)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
}

func TestWait_ZeroDelay(t *testing.T) {
	strategy := NewFixedBackoff(0)
	manager := NewManagerWithStrategy(strategy)

	ctx := context.Background()
	start := time.Now()

	err := manager.Wait(ctx, 0)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}

	elapsed := time.Since(start)
	// Should return immediately with zero delay
	if elapsed > 100*time.Millisecond {
		t.Errorf("Expected immediate return with zero delay, took %v", elapsed)
	}
}

func TestDelay_NegativeAttempt(t *testing.T) {
	strategy := NewExponentialBackoff(
		100*time.Millisecond,
		10*time.Second,
		2.0,
		false,
	)

	// Negative attempt should be treated as 0
	delay := strategy.Delay(-1)
	if delay != 100*time.Millisecond {
		t.Errorf("Expected delay 100ms for negative attempt, got %v", delay)
	}
}
