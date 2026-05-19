package retry

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Retriable is a function that can be retried.
// It takes a context and returns an error.
type Retriable func(ctx context.Context) error

// Config configures retry behavior.
type Config struct {
	// MaxRetries is the maximum number of retry attempts (0 means no retries, only initial attempt).
	// Defaults to 3.
	MaxRetries int

	// Backoff is the backoff strategy to use between retries.
	// If nil, uses DefaultBackoff().
	Backoff BackoffStrategy

	// Predicate determines if an error should trigger a retry.
	// If nil, uses AlwaysRetry.
	Predicate RetryPredicate

	// Timeout is the maximum total time for all retry attempts.
	// Zero means no timeout.
	Timeout time.Duration

	// OnRetry is called before each retry attempt (including the first retry).
	// attempt is 0-indexed (0 = first retry after initial failure).
	// This is useful for logging or metrics.
	OnRetry func(attempt int, err error)

	// OnRetryAsync is called asynchronously before each retry attempt.
	OnRetryAsync func(attempt int, err error)

	// EnableMetrics enables detailed metrics collection.
	EnableMetrics bool
}

// DefaultConfig returns the default retry configuration.
func DefaultConfig() Config {
	return Config{
		MaxRetries: 3,
		Backoff:    NewExponentialBackoff(100*time.Millisecond, 30*time.Second, 2.0, true),
		Predicate:  AlwaysRetry,
		Timeout:    0, // No timeout by default
	}
}

// Manager provides retry functionality.
type Manager interface {
	// Execute executes the retriable function with retry logic using default config.
	Execute(ctx context.Context, fn Retriable) error

	// ExecuteWithConfig executes the retriable function with the specified config.
	ExecuteWithConfig(ctx context.Context, fn Retriable, config Config) error

	// Stats returns statistics about retry operations.
	Stats() Stats
}

// Stats contains statistics about retry operations.
type Stats struct {
	// TotalExecutions is the total number of function executions (including retries).
	TotalExecutions int64

	// TotalSuccesses is the total number of successful executions (after retries).
	TotalSuccesses int64

	// TotalFailures is the total number of failed executions (after all retries).
	TotalFailures int64

	// TotalRetries is the total number of retry attempts.
	TotalRetries int64

	// LastExecutionTime is when the last execution completed.
	LastExecutionTime time.Time
}

// retryManager implements the Manager interface.
type retryManager struct {
	stats Stats
	mu    sync.RWMutex
}

// NewManager creates a new retry manager.
func NewManager() Manager {
	return &retryManager{}
}

// Execute executes the retriable function with retry logic using default config.
func (m *retryManager) Execute(ctx context.Context, fn Retriable) error {
	return m.ExecuteWithConfig(ctx, fn, DefaultConfig())
}

// ExecuteWithConfig executes the retriable function with the specified config.
func (m *retryManager) ExecuteWithConfig(ctx context.Context, fn Retriable, config Config) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}
	if fn == nil {
		return NewError(ErrCodeNilRetriable, "retriable function cannot be nil")
	}

	// Validate config
	if config.MaxRetries < 0 {
		return NewError(ErrCodeInvalidConfig, "MaxRetries cannot be negative")
	}

	// Use defaults if not specified
	if config.Backoff == nil {
		config.Backoff = DefaultConfig().Backoff
	}
	if config.Predicate == nil {
		config.Predicate = AlwaysRetry
	}

	// Create context with timeout if specified
	execCtx := ctx
	var cancel context.CancelFunc
	if config.Timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
	}

	var lastErr error
	maxAttempts := config.MaxRetries + 1 // +1 for initial attempt

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Check if context is cancelled or timed out
		select {
		case <-execCtx.Done():
			atomic.AddInt64(&m.stats.TotalFailures, 1)
			m.mu.Lock()
			m.stats.LastExecutionTime = time.Now()
			m.mu.Unlock()

			if execCtx.Err() == context.DeadlineExceeded {
				return NewError(ErrCodeContextTimeout, fmt.Sprintf("retry timeout after %v: %v", config.Timeout, execCtx.Err()))
			}
			return NewError(ErrCodeContextCanceled, fmt.Sprintf("retry cancelled: %v", execCtx.Err()))
		default:
		}

		// Execute the function
		err := fn(execCtx)
		atomic.AddInt64(&m.stats.TotalExecutions, 1)

		if err == nil {
			// Success!
			atomic.AddInt64(&m.stats.TotalSuccesses, 1)
			m.mu.Lock()
			m.stats.LastExecutionTime = time.Now()
			m.mu.Unlock()
			return nil
		}

		lastErr = err

		// Check if we should retry this error
		if !config.Predicate(err) {
			// Error is not retryable
			atomic.AddInt64(&m.stats.TotalFailures, 1)
			m.mu.Lock()
			m.stats.LastExecutionTime = time.Now()
			m.mu.Unlock()
			return err
		}

		// Check if we have more retries available
		if attempt < config.MaxRetries {
			atomic.AddInt64(&m.stats.TotalRetries, 1)

			// Call retry callbacks
			if config.OnRetry != nil {
				config.OnRetry(attempt, err)
			}
			if config.OnRetryAsync != nil {
				go config.OnRetryAsync(attempt, err)
			}

			// Calculate backoff delay
			delay := config.Backoff.Delay(attempt)

			// Wait before retry (respecting context cancellation)
			select {
			case <-execCtx.Done():
				atomic.AddInt64(&m.stats.TotalFailures, 1)
				m.mu.Lock()
				m.stats.LastExecutionTime = time.Now()
				m.mu.Unlock()

				if execCtx.Err() == context.DeadlineExceeded {
					return NewError(ErrCodeContextTimeout, fmt.Sprintf("retry timeout during backoff: %v", execCtx.Err()))
				}
				return NewError(ErrCodeContextCanceled, fmt.Sprintf("retry cancelled during backoff: %v", execCtx.Err()))
			case <-time.After(delay):
				// Continue to next retry
			}
		}
	}

	// All retries exhausted
	atomic.AddInt64(&m.stats.TotalFailures, 1)
	m.mu.Lock()
	m.stats.LastExecutionTime = time.Now()
	m.mu.Unlock()

	return NewError(ErrCodeMaxRetriesExceeded, fmt.Sprintf("max retries (%d) exceeded: %v", config.MaxRetries, lastErr))
}

// Stats returns statistics about retry operations.
func (m *retryManager) Stats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return Stats{
		TotalExecutions:  atomic.LoadInt64(&m.stats.TotalExecutions),
		TotalSuccesses:   atomic.LoadInt64(&m.stats.TotalSuccesses),
		TotalFailures:    atomic.LoadInt64(&m.stats.TotalFailures),
		TotalRetries:     atomic.LoadInt64(&m.stats.TotalRetries),
		LastExecutionTime: m.stats.LastExecutionTime,
	}
}
