package timeout

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// TimedFunction is a function that can be executed with a timeout.
type TimedFunction func(ctx context.Context) error

// Config configures timeout behavior.
type Config struct {
	// Timeout is the maximum duration for the function execution.
	// Zero means no timeout (use context timeout if available).
	Timeout time.Duration

	// OnTimeout is called when a timeout occurs.
	// It receives the context and the timeout duration.
	OnTimeout func(ctx context.Context, timeout time.Duration)

	// OnTimeoutAsync is called asynchronously when a timeout occurs.
	OnTimeoutAsync func(ctx context.Context, timeout time.Duration)

	// EnableMetrics enables detailed metrics collection.
	EnableMetrics bool
}

// DefaultConfig returns the default timeout configuration.
func DefaultConfig() Config {
	return Config{
		Timeout: 0, // No default timeout - use context
	}
}

// Manager provides timeout functionality.
type Manager interface {
	// Execute executes a function with a timeout using default config.
	Execute(ctx context.Context, fn TimedFunction) error

	// ExecuteWithConfig executes a function with the specified timeout config.
	ExecuteWithConfig(ctx context.Context, fn TimedFunction, config Config) error

	// ExecuteWithTimeout is a convenience method that executes a function with a specific timeout.
	ExecuteWithTimeout(ctx context.Context, fn TimedFunction, timeout time.Duration) error

	// Stats returns statistics about timeout operations.
	Stats() Stats
}

// Stats contains statistics about timeout operations.
type Stats struct {
	// TotalExecutions is the total number of function executions.
	TotalExecutions int64

	// TotalSuccesses is the total number of successful executions (completed before timeout).
	TotalSuccesses int64

	// TotalTimeouts is the total number of timeouts.
	TotalTimeouts int64

	// TotalCancellations is the total number of cancellations (context cancelled).
	TotalCancellations int64

	// LastExecutionTime is when the last execution completed.
	LastExecutionTime time.Time
}

// timeoutManager implements the Manager interface.
type timeoutManager struct {
	stats Stats
	mu    sync.RWMutex
}

// NewManager creates a new timeout manager.
func NewManager() Manager {
	return &timeoutManager{}
}

// Execute executes a function with a timeout using default config.
func (m *timeoutManager) Execute(ctx context.Context, fn TimedFunction) error {
	return m.ExecuteWithConfig(ctx, fn, DefaultConfig())
}

// ExecuteWithTimeout is a convenience method that executes a function with a specific timeout.
func (m *timeoutManager) ExecuteWithTimeout(ctx context.Context, fn TimedFunction, timeout time.Duration) error {
	config := DefaultConfig()
	config.Timeout = timeout
	return m.ExecuteWithConfig(ctx, fn, config)
}

// ExecuteWithConfig executes a function with the specified timeout config.
func (m *timeoutManager) ExecuteWithConfig(ctx context.Context, fn TimedFunction, config Config) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}
	if fn == nil {
		return NewError(ErrCodeNilFunction, "function cannot be nil")
	}

	// Check if context is already cancelled or expired
	select {
	case <-ctx.Done():
		atomic.AddInt64(&m.stats.TotalCancellations, 1)
		m.mu.Lock()
		m.stats.LastExecutionTime = time.Now()
		m.mu.Unlock()
		if ctx.Err() == context.DeadlineExceeded {
			return NewError(ErrCodeTimeoutExceeded, fmt.Sprintf("context deadline already exceeded: %v", ctx.Err()))
		}
		return NewError(ErrCodeContextCanceled, fmt.Sprintf("context already cancelled: %v", ctx.Err()))
	default:
	}

	// Determine timeout duration
	timeout := config.Timeout
	
	// If timeout is explicitly negative, it's invalid
	if timeout < 0 {
		return NewError(ErrCodeInvalidTimeout, "timeout cannot be negative")
	}
	
	if timeout == 0 {
		// Use context timeout if available
		if deadline, ok := ctx.Deadline(); ok {
			timeout = time.Until(deadline)
			if timeout <= 0 {
				// Context already expired
				atomic.AddInt64(&m.stats.TotalTimeouts, 1)
				m.mu.Lock()
				m.stats.LastExecutionTime = time.Now()
				m.mu.Unlock()
				return NewError(ErrCodeTimeoutExceeded, fmt.Sprintf("context deadline already exceeded: %v", ctx.Err()))
			}
		} else {
			// No timeout specified and no context deadline
			// Execute without timeout
			return m.executeWithoutTimeout(ctx, fn, config)
		}
	}

	// At this point, timeout should be positive
	if timeout <= 0 {
		return NewError(ErrCodeInvalidTimeout, "timeout must be positive")
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute function in goroutine
	type result struct {
		err error
	}

	resultChan := make(chan result, 1)

	go func() {
		err := fn(timeoutCtx)
		select {
		case resultChan <- result{err: err}:
			// Successfully sent result
		case <-timeoutCtx.Done():
			// Timeout occurred, result not needed
		}
	}()

	// Wait for either completion or timeout
	select {
	case res := <-resultChan:
		// Function completed
		atomic.AddInt64(&m.stats.TotalExecutions, 1)
		m.mu.Lock()
		m.stats.LastExecutionTime = time.Now()
		m.mu.Unlock()

		if res.err == nil {
			atomic.AddInt64(&m.stats.TotalSuccesses, 1)
			return nil
		}

		// Check if error is due to timeout or cancellation from our timeout context
		if timeoutCtx.Err() == context.DeadlineExceeded {
			atomic.AddInt64(&m.stats.TotalTimeouts, 1)
			m.onTimeout(config, timeoutCtx, timeout)
			return NewError(ErrCodeTimeoutExceeded, fmt.Sprintf("operation timed out after %v: %v", timeout, res.err))
		}

		if timeoutCtx.Err() == context.Canceled {
			atomic.AddInt64(&m.stats.TotalCancellations, 1)
			return NewError(ErrCodeContextCanceled, fmt.Sprintf("operation cancelled: %v", res.err))
		}

		// Check if error itself indicates timeout or cancellation
		if res.err == context.DeadlineExceeded {
			atomic.AddInt64(&m.stats.TotalTimeouts, 1)
			m.onTimeout(config, timeoutCtx, timeout)
			return NewError(ErrCodeTimeoutExceeded, fmt.Sprintf("operation timed out after %v: %v", timeout, res.err))
		}

		if res.err == context.Canceled {
			atomic.AddInt64(&m.stats.TotalCancellations, 1)
			return NewError(ErrCodeContextCanceled, fmt.Sprintf("operation cancelled: %v", res.err))
		}

		// Other error
		return res.err

	case <-timeoutCtx.Done():
		// Timeout occurred before function completed
		atomic.AddInt64(&m.stats.TotalExecutions, 1)
		m.mu.Lock()
		m.stats.LastExecutionTime = time.Now()
		m.mu.Unlock()

		if timeoutCtx.Err() == context.DeadlineExceeded {
			atomic.AddInt64(&m.stats.TotalTimeouts, 1)
			m.onTimeout(config, timeoutCtx, timeout)
			return NewError(ErrCodeTimeoutExceeded, fmt.Sprintf("operation timed out after %v", timeout))
		}

		atomic.AddInt64(&m.stats.TotalCancellations, 1)
		return NewError(ErrCodeContextCanceled, fmt.Sprintf("operation cancelled: %v", timeoutCtx.Err()))
	}
}

// executeWithoutTimeout executes a function without applying a timeout.
// The function still respects context cancellation.
func (m *timeoutManager) executeWithoutTimeout(ctx context.Context, fn TimedFunction, config Config) error {
	err := fn(ctx)

	atomic.AddInt64(&m.stats.TotalExecutions, 1)
	m.mu.Lock()
	m.stats.LastExecutionTime = time.Now()
	m.mu.Unlock()

	if err == nil {
		atomic.AddInt64(&m.stats.TotalSuccesses, 1)
		return nil
	}

	// Check if error is due to cancellation
	if err == context.Canceled || ctx.Err() == context.Canceled {
		atomic.AddInt64(&m.stats.TotalCancellations, 1)
		return NewError(ErrCodeContextCanceled, fmt.Sprintf("operation cancelled: %v", err))
	}

	if err == context.DeadlineExceeded || ctx.Err() == context.DeadlineExceeded {
		atomic.AddInt64(&m.stats.TotalTimeouts, 1)
		return NewError(ErrCodeTimeoutExceeded, fmt.Sprintf("context deadline exceeded: %v", err))
	}

	return err
}

// onTimeout invokes timeout callbacks.
func (m *timeoutManager) onTimeout(config Config, ctx context.Context, timeout time.Duration) {
	if config.OnTimeout != nil {
		config.OnTimeout(ctx, timeout)
	}

	if config.OnTimeoutAsync != nil {
		go config.OnTimeoutAsync(ctx, timeout)
	}
}

// Stats returns statistics about timeout operations.
func (m *timeoutManager) Stats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return Stats{
		TotalExecutions:   atomic.LoadInt64(&m.stats.TotalExecutions),
		TotalSuccesses:    atomic.LoadInt64(&m.stats.TotalSuccesses),
		TotalTimeouts:     atomic.LoadInt64(&m.stats.TotalTimeouts),
		TotalCancellations: atomic.LoadInt64(&m.stats.TotalCancellations),
		LastExecutionTime: m.stats.LastExecutionTime,
	}
}
