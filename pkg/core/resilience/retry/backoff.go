package retry

import (
	"math"
	"math/rand"
	"time"
)

// BackoffStrategy calculates the delay before the next retry attempt.
type BackoffStrategy interface {
	// Delay returns the duration to wait before the next retry.
	// attempt is the current attempt number (0-indexed, so first retry is attempt 0).
	Delay(attempt int) time.Duration
}

// FixedBackoff provides a constant delay between retries.
type FixedBackoff struct {
	DelayDuration time.Duration
}

// NewFixedBackoff creates a new fixed backoff strategy.
func NewFixedBackoff(delay time.Duration) BackoffStrategy {
	return &FixedBackoff{DelayDuration: delay}
}

// Delay returns the fixed delay duration.
func (f *FixedBackoff) Delay(attempt int) time.Duration {
	return f.DelayDuration
}

// ExponentialBackoff provides exponential backoff with optional jitter.
type ExponentialBackoff struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       bool
}

// NewExponentialBackoff creates a new exponential backoff strategy.
// initialDelay is the delay for the first retry.
// maxDelay is the maximum delay allowed.
// multiplier is the factor by which delay increases each retry (default: 2.0).
// jitter determines if random jitter is added to prevent thundering herd.
func NewExponentialBackoff(initialDelay, maxDelay time.Duration, multiplier float64, jitter bool) BackoffStrategy {
	if multiplier <= 0 {
		multiplier = 2.0
	}
	return &ExponentialBackoff{
		InitialDelay: initialDelay,
		MaxDelay:     maxDelay,
		Multiplier:   multiplier,
		Jitter:       jitter,
	}
}

// Delay returns the exponential backoff delay with optional jitter.
func (e *ExponentialBackoff) Delay(attempt int) time.Duration {
	// Calculate exponential delay: initial * multiplier^attempt
	delay := float64(e.InitialDelay) * math.Pow(e.Multiplier, float64(attempt))
	
	// Cap at max delay
	if delay > float64(e.MaxDelay) {
		delay = float64(e.MaxDelay)
	}
	
	// Add jitter if enabled (up to 25% random variation)
	if e.Jitter {
		jitterAmount := delay * 0.25 * (rand.Float64() - 0.5) // ±25% jitter
		delay += jitterAmount
		// Ensure delay doesn't go negative or exceed max
		if delay < 0 {
			delay = 0
		}
		if delay > float64(e.MaxDelay) {
			delay = float64(e.MaxDelay)
		}
	}
	
	return time.Duration(delay)
}

// LinearBackoff provides linear backoff (delay increases linearly with each retry).
type LinearBackoff struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Increment    time.Duration
}

// NewLinearBackoff creates a new linear backoff strategy.
// initialDelay is the delay for the first retry.
// maxDelay is the maximum delay allowed.
// increment is the amount to add to delay each retry (default: same as initialDelay).
func NewLinearBackoff(initialDelay, maxDelay time.Duration, increment time.Duration) BackoffStrategy {
	if increment == 0 {
		increment = initialDelay
	}
	return &LinearBackoff{
		InitialDelay: initialDelay,
		MaxDelay:     maxDelay,
		Increment:    increment,
	}
}

// Delay returns the linear backoff delay.
func (l *LinearBackoff) Delay(attempt int) time.Duration {
	delay := l.InitialDelay + time.Duration(attempt)*l.Increment
	if delay > l.MaxDelay {
		return l.MaxDelay
	}
	return delay
}
