package retry

import (
	"testing"
	"time"
)

func TestFixedBackoff(t *testing.T) {
	delay := 100 * time.Millisecond
	backoff := NewFixedBackoff(delay)

	// Should return the same delay for all attempts
	for i := 0; i < 10; i++ {
		result := backoff.Delay(i)
		if result != delay {
			t.Errorf("Expected delay %v, got %v at attempt %d", delay, result, i)
		}
	}
}

func TestExponentialBackoff(t *testing.T) {
	initialDelay := 100 * time.Millisecond
	maxDelay := 10 * time.Second
	multiplier := 2.0

	tests := []struct {
		name         string
		jitter       bool
		attempt      int
		minExpected  time.Duration
		maxExpected  time.Duration
	}{
		{
			name:        "No jitter attempt 0",
			jitter:      false,
			attempt:     0,
			minExpected: 100 * time.Millisecond,
			maxExpected: 100 * time.Millisecond,
		},
		{
			name:        "No jitter attempt 1",
			jitter:      false,
			attempt:     1,
			minExpected: 200 * time.Millisecond,
			maxExpected: 200 * time.Millisecond,
		},
		{
			name:        "No jitter attempt 2",
			jitter:      false,
			attempt:     2,
			minExpected: 400 * time.Millisecond,
			maxExpected: 400 * time.Millisecond,
		},
		{
			name:        "With jitter attempt 0",
			jitter:      true,
			attempt:     0,
			minExpected: 75 * time.Millisecond,  // 100 - 25%
			maxExpected: 125 * time.Millisecond, // 100 + 25%
		},
		{
			name:        "Capped at max delay",
			jitter:      false,
			attempt:     10,
			minExpected: 100 * time.Millisecond,
			maxExpected: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backoff := NewExponentialBackoff(initialDelay, maxDelay, multiplier, tt.jitter)
			delay := backoff.Delay(tt.attempt)

			if delay < tt.minExpected || delay > tt.maxExpected {
				// For jitter case, allow some variance
				if !tt.jitter && delay != tt.minExpected {
					t.Errorf("Expected delay between %v and %v, got %v", tt.minExpected, tt.maxExpected, delay)
				} else if tt.jitter {
					// With jitter, just verify it's in reasonable range
					if delay < tt.minExpected || delay > tt.maxExpected {
						t.Errorf("Expected delay between %v and %v (with jitter), got %v", tt.minExpected, tt.maxExpected, delay)
					}
				}
			}
		})
	}
}

func TestExponentialBackoff_DefaultMultiplier(t *testing.T) {
	backoff := NewExponentialBackoff(100*time.Millisecond, 10*time.Second, 0, false).(*ExponentialBackoff)
	if backoff.Multiplier != 2.0 {
		t.Errorf("Expected default multiplier 2.0, got %v", backoff.Multiplier)
	}
}

func TestLinearBackoff(t *testing.T) {
	initialDelay := 100 * time.Millisecond
	maxDelay := 1 * time.Second
	increment := 50 * time.Millisecond

	backoff := NewLinearBackoff(initialDelay, maxDelay, increment)

	tests := []struct {
		attempt int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},  // initial
		{1, 150 * time.Millisecond},  // initial + increment
		{2, 200 * time.Millisecond},  // initial + 2*increment
		{3, 250 * time.Millisecond},  // initial + 3*increment
		{20, 1 * time.Second},        // capped at max
	}

	for _, tt := range tests {
		result := backoff.Delay(tt.attempt)
		if result != tt.expected {
			t.Errorf("Attempt %d: expected %v, got %v", tt.attempt, tt.expected, result)
		}
	}
}

func TestLinearBackoff_DefaultIncrement(t *testing.T) {
	initialDelay := 100 * time.Millisecond
	maxDelay := 1 * time.Second

	backoff := NewLinearBackoff(initialDelay, maxDelay, 0).(*LinearBackoff)
	if backoff.Increment != initialDelay {
		t.Errorf("Expected default increment %v, got %v", initialDelay, backoff.Increment)
	}
}

func TestLinearBackoff_CappedAtMax(t *testing.T) {
	initialDelay := 100 * time.Millisecond
	maxDelay := 500 * time.Millisecond
	increment := 200 * time.Millisecond

	backoff := NewLinearBackoff(initialDelay, maxDelay, increment)

	// Attempt 2 would be 100 + 2*200 = 500ms (exactly max)
	result := backoff.Delay(2)
	if result != maxDelay {
		t.Errorf("Expected %v (max), got %v", maxDelay, result)
	}

	// Attempt 3 would exceed max, so should be capped
	result = backoff.Delay(3)
	if result != maxDelay {
		t.Errorf("Expected %v (capped), got %v", maxDelay, result)
	}
}

func TestExponentialBackoff_CappedAtMax(t *testing.T) {
	initialDelay := 100 * time.Millisecond
	maxDelay := 1 * time.Second
	multiplier := 10.0 // Large multiplier to quickly exceed max

	backoff := NewExponentialBackoff(initialDelay, maxDelay, multiplier, false)

	// Attempt 0: 100ms (okay)
	result := backoff.Delay(0)
	if result != initialDelay {
		t.Errorf("Expected %v, got %v", initialDelay, result)
	}

	// Attempt 1: 1000ms (exceeds max, should be capped)
	result = backoff.Delay(1)
	if result != maxDelay {
		t.Errorf("Expected %v (capped), got %v", maxDelay, result)
	}

	// Attempt 2: would be 10000ms, should still be capped
	result = backoff.Delay(2)
	if result != maxDelay {
		t.Errorf("Expected %v (capped), got %v", maxDelay, result)
	}
}
