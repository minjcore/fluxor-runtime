package backoff

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// Strategy calculates the delay for backoff operations.
type Strategy interface {
	// Delay returns the duration to wait before the next attempt.
	// attempt is the current attempt number (0-indexed, so first attempt is 0).
	Delay(attempt int) time.Duration
}

// Config configures backoff behavior.
type Config struct {
	// Strategy is the backoff strategy to use.
	Strategy Strategy

	// MaxAttempts is the maximum number of attempts (0 means unlimited).
	// Defaults to 0 (unlimited).
	MaxAttempts int

	// MaxDelay is the maximum delay allowed (overrides strategy's max if set).
	// Zero means use strategy's max delay. Defaults to 0.
	MaxDelay time.Duration

	// OnBackoff is called before each backoff delay.
	OnBackoff func(ctx context.Context, attempt int, delay time.Duration)

	// OnBackoffAsync is called asynchronously before each backoff delay.
	OnBackoffAsync func(ctx context.Context, attempt int, delay time.Duration)
}

// DefaultConfig returns the default backoff configuration.
func DefaultConfig() Config {
	return Config{
		Strategy:   NewFixedBackoff(time.Second),
		MaxAttempts: 0,
		MaxDelay:   0,
	}
}

// Manager provides backoff functionality.
type Manager interface {
	// Wait waits for the backoff delay for the given attempt.
	Wait(ctx context.Context, attempt int) error

	// WaitWithConfig waits for the backoff delay using the specified config.
	WaitWithConfig(ctx context.Context, attempt int, config Config) error

	// Delay returns the delay duration for the given attempt (non-blocking).
	Delay(attempt int) time.Duration

	// DelayWithConfig returns the delay duration using the specified config (non-blocking).
	DelayWithConfig(attempt int, config Config) time.Duration

	// Strategy returns the default backoff strategy.
	Strategy() Strategy
}

// ManagerWithConfig provides backoff functionality with config.
type ManagerWithConfig interface {
	// Wait waits for the backoff delay for the given attempt.
	Wait(ctx context.Context, attempt int) error

	// Delay returns the delay duration for the given attempt (non-blocking).
	Delay(attempt int) time.Duration

	// Strategy returns the backoff strategy.
	Strategy() Strategy
}

// backoffManager implements the Manager interface.
type backoffManager struct {
	strategy Strategy
}

// NewManager creates a new backoff manager with default strategy (fixed 1 second).
func NewManager() Manager {
	return &backoffManager{
		strategy: NewFixedBackoff(time.Second),
	}
}

// NewManagerWithStrategy creates a new backoff manager with the specified strategy.
func NewManagerWithStrategy(strategy Strategy) Manager {
	if strategy == nil {
		strategy = NewFixedBackoff(time.Second)
	}
	return &backoffManager{
		strategy: strategy,
	}
}

// Wait waits for the backoff delay for the given attempt.
func (m *backoffManager) Wait(ctx context.Context, attempt int) error {
	config := DefaultConfig()
	config.Strategy = m.strategy // Use manager's strategy
	return m.WaitWithConfig(ctx, attempt, config)
}

// WaitWithConfig waits for the backoff delay using the specified config.
func (m *backoffManager) WaitWithConfig(ctx context.Context, attempt int, config Config) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}

	// Use config strategy or manager's default
	strategy := config.Strategy
	if strategy == nil {
		strategy = m.strategy
	}

	// Get delay
	delay := strategy.Delay(attempt)

	// Apply max delay if configured
	if config.MaxDelay > 0 && delay > config.MaxDelay {
		delay = config.MaxDelay
	}

	// Check max attempts (only if configured)
	if config.MaxAttempts > 0 && attempt >= config.MaxAttempts {
		return NewError(ErrCodeInvalidConfig, fmt.Sprintf("max attempts (%d) exceeded", config.MaxAttempts))
	}

	// If delay is 0, return immediately
	if delay <= 0 {
		return nil
	}

	// Invoke callbacks
	if config.OnBackoff != nil {
		config.OnBackoff(ctx, attempt, delay)
	}

	if config.OnBackoffAsync != nil {
		go config.OnBackoffAsync(ctx, attempt, delay)
	}

	// Wait for delay or context cancellation
	if delay > 0 {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return NewError(ErrCodeContextTimeout, fmt.Sprintf("context timeout: %v", ctx.Err()))
			}
			return NewError(ErrCodeContextCanceled, fmt.Sprintf("context cancelled: %v", ctx.Err()))
		case <-time.After(delay):
			return nil
		}
	}

	return nil
}

// Delay returns the delay duration for the given attempt (non-blocking).
func (m *backoffManager) Delay(attempt int) time.Duration {
	return m.strategy.Delay(attempt)
}

// DelayWithConfig returns the delay duration using the specified config (non-blocking).
func (m *backoffManager) DelayWithConfig(attempt int, config Config) time.Duration {
	strategy := config.Strategy
	if strategy == nil {
		strategy = m.strategy
	}

	delay := strategy.Delay(attempt)

	// Apply max delay if configured
	if config.MaxDelay > 0 && delay > config.MaxDelay {
		delay = config.MaxDelay
	}

	return delay
}

// Strategy returns the default backoff strategy.
func (m *backoffManager) Strategy() Strategy {
	return m.strategy
}

// FixedBackoff provides a constant delay between attempts.
type FixedBackoff struct {
	DelayDuration time.Duration
}

// NewFixedBackoff creates a new fixed backoff strategy.
func NewFixedBackoff(delay time.Duration) Strategy {
	if delay < 0 {
		delay = 0
	}
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
	mu           sync.Mutex
	rng          *rand.Rand
}

// NewExponentialBackoff creates a new exponential backoff strategy.
// initialDelay is the delay for the first attempt (attempt 0).
// maxDelay is the maximum delay allowed.
// multiplier is the factor by which delay increases each attempt (default: 2.0).
// jitter determines if random jitter is added to prevent thundering herd.
func NewExponentialBackoff(initialDelay, maxDelay time.Duration, multiplier float64, jitter bool) Strategy {
	if initialDelay < 0 {
		initialDelay = 0
	}
	if maxDelay < 0 {
		maxDelay = 0
	}
	if multiplier <= 0 {
		multiplier = 2.0
	}

	eb := &ExponentialBackoff{
		InitialDelay: initialDelay,
		MaxDelay:     maxDelay,
		Multiplier:   multiplier,
		Jitter:       jitter,
	}

	if jitter {
		eb.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	return eb
}

// Delay returns the exponential backoff delay with optional jitter.
func (e *ExponentialBackoff) Delay(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// Calculate exponential delay: initial * multiplier^attempt
	delay := float64(e.InitialDelay) * math.Pow(e.Multiplier, float64(attempt))

	// Cap at max delay
	if e.MaxDelay > 0 && delay > float64(e.MaxDelay) {
		delay = float64(e.MaxDelay)
	}

	// Add jitter if enabled (up to 25% random variation)
	if e.Jitter && e.rng != nil {
		e.mu.Lock()
		jitterAmount := delay * 0.25 * (e.rng.Float64()*2.0 - 1.0) // ±25% jitter
		e.mu.Unlock()

		delay += jitterAmount

		// Ensure delay doesn't go negative
		if delay < 0 {
			delay = 0
		}

		// Cap at max delay if set
		if e.MaxDelay > 0 && delay > float64(e.MaxDelay) {
			delay = float64(e.MaxDelay)
		}
	}

	return time.Duration(delay)
}

// LinearBackoff provides linear backoff (delay increases linearly with each attempt).
type LinearBackoff struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Increment    time.Duration
}

// NewLinearBackoff creates a new linear backoff strategy.
// initialDelay is the delay for the first attempt (attempt 0).
// maxDelay is the maximum delay allowed.
// increment is the amount to add to delay each attempt (default: same as initialDelay).
func NewLinearBackoff(initialDelay, maxDelay time.Duration, increment time.Duration) Strategy {
	if initialDelay < 0 {
		initialDelay = 0
	}
	if maxDelay < 0 {
		maxDelay = 0
	}
	if increment <= 0 {
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
	if attempt < 0 {
		attempt = 0
	}

	delay := l.InitialDelay + time.Duration(attempt)*l.Increment

	if l.MaxDelay > 0 && delay > l.MaxDelay {
		return l.MaxDelay
	}

	return delay
}
