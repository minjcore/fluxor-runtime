package hedge

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Executable is a function that can be executed as part of a hedge.
type Executable func(ctx context.Context) error

// Config configures hedge behavior.
type Config struct {
	// MaxConcurrency is the maximum number of parallel hedge executions.
	// Must be greater than 0. Defaults to 3.
	MaxConcurrency int

	// CancelRemainingOnSuccess cancels remaining hedge requests when one succeeds.
	// Defaults to true.
	CancelRemainingOnSuccess bool

	// OnHedgeAttempted is called when a hedge execution is attempted.
	OnHedgeAttempted func(ctx context.Context, index int, fn Executable)

	// OnHedgeAttemptedAsync is called asynchronously when a hedge is attempted.
	OnHedgeAttemptedAsync func(ctx context.Context, index int, fn Executable)

	// OnHedgeSucceeded is called when a hedge execution succeeds (first success).
	OnHedgeSucceeded func(ctx context.Context, index int, result error)

	// OnHedgeSucceededAsync is called asynchronously when a hedge succeeds.
	OnHedgeSucceededAsync func(ctx context.Context, index int, result error)

	// OnHedgeFailed is called when a hedge execution fails.
	OnHedgeFailed func(ctx context.Context, index int, err error)

	// OnHedgeFailedAsync is called asynchronously when a hedge fails.
	OnHedgeFailedAsync func(ctx context.Context, index int, err error)

	// OnAllHedgesFailed is called when all hedge executions have failed.
	OnAllHedgesFailed func(ctx context.Context, errors []error)

	// OnAllHedgesFailedAsync is called asynchronously when all hedges fail.
	OnAllHedgesFailedAsync func(ctx context.Context, errors []error)

	// EnableMetrics enables detailed metrics collection.
	EnableMetrics bool
}

// DefaultConfig returns the default hedge configuration.
func DefaultConfig() Config {
	return Config{
		MaxConcurrency:           3,
		CancelRemainingOnSuccess: true,
	}
}

// Manager provides hedge functionality.
type Manager interface {
	// Execute executes multiple functions in parallel and returns the first successful result.
	Execute(ctx context.Context, functions []Executable) error

	// ExecuteWithConfig executes functions with the specified hedge config.
	ExecuteWithConfig(ctx context.Context, functions []Executable, config Config) error

	// Stats returns statistics about hedge operations.
	Stats() Stats
}

// Stats contains statistics about hedge operations.
type Stats struct {
	// TotalExecutions is the total number of hedge executions attempted.
	TotalExecutions int64

	// TotalHedgeAttempts is the total number of individual hedge function attempts.
	TotalHedgeAttempts int64

	// TotalHedgeSuccesses is the total number of successful hedge executions.
	TotalHedgeSuccesses int64

	// TotalHedgeFailures is the total number of failed hedge executions.
	TotalHedgeFailures int64

	// TotalAllHedgesFailed is the total number of times all hedges failed.
	TotalAllHedgesFailed int64

	// TotalCancelled is the total number of hedge executions that were cancelled (due to first success).
	TotalCancelled int64

	// LastExecutionTime is when the last execution completed.
	LastExecutionTime time.Time
}

// hedgeManager implements the Manager interface.
type hedgeManager struct {
	stats Stats
	mu    sync.RWMutex
}

// NewManager creates a new hedge manager with default config.
func NewManager() Manager {
	return &hedgeManager{}
}

// Execute executes multiple functions in parallel and returns the first successful result.
func (m *hedgeManager) Execute(ctx context.Context, functions []Executable) error {
	return m.ExecuteWithConfig(ctx, functions, DefaultConfig())
}

// ExecuteWithConfig executes functions with the specified hedge config.
func (m *hedgeManager) ExecuteWithConfig(ctx context.Context, functions []Executable, config Config) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}

	if len(functions) == 0 {
		return NewError(ErrCodeNilFunction, "at least one function is required")
	}

	for i, fn := range functions {
		if fn == nil {
			return NewError(ErrCodeNilFunction, fmt.Sprintf("function at index %d is nil", i))
		}
	}

	atomic.AddInt64(&m.stats.TotalExecutions, 1)

	// Set defaults
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = DefaultConfig().MaxConcurrency
	}
	if config.MaxConcurrency > len(functions) {
		config.MaxConcurrency = len(functions)
	}

	// Execute hedges in parallel
	return m.executeHedges(ctx, functions, config)
}

// executeHedges executes the functions in parallel and returns the first success.
func (m *hedgeManager) executeHedges(ctx context.Context, functions []Executable, config Config) error {
	// Create context for cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Result channels
	type result struct {
		index int
		err   error
	}

	resultCh := make(chan result, len(functions))
	var wg sync.WaitGroup
	var once sync.Once
	var successIndex atomic.Int32
	successIndex.Store(-1)

	// Execute functions in parallel (up to MaxConcurrency)
	semaphore := make(chan struct{}, config.MaxConcurrency)

	for i, fn := range functions {
		wg.Add(1)
		go func(index int, f Executable) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check if we should continue (another hedge might have succeeded)
			if config.CancelRemainingOnSuccess {
				select {
				case <-ctx.Done():
					atomic.AddInt64(&m.stats.TotalCancelled, 1)
					return
				default:
				}
			}

			atomic.AddInt64(&m.stats.TotalHedgeAttempts, 1)
			m.onHedgeAttempted(ctx, index, f, config)

			// Execute function
			err := f(ctx)

			// Check if we were cancelled (but still process result if function completed)
			if config.CancelRemainingOnSuccess {
				select {
				case <-ctx.Done():
					atomic.AddInt64(&m.stats.TotalCancelled, 1)
					// If cancelled and function already completed (err == nil), still process the result
					// If cancelled and function failed, return early
					if err != nil {
						return // Cancelled before completion, return early
					}
				default:
				}
			}

			if err == nil {
				// Success - mark as first success if not already set
				if config.CancelRemainingOnSuccess {
					once.Do(func() {
						successIndex.Store(int32(index))
						cancel() // Cancel remaining hedges
					})
				}

				atomic.AddInt64(&m.stats.TotalHedgeSuccesses, 1)
				m.onHedgeSucceeded(ctx, index, nil, config)

				// Always send success result, even if context is cancelled
				// The context cancellation happens after we've already succeeded
				select {
				case resultCh <- result{index: index, err: nil}:
					// Success result sent
				default:
					// Channel full or closed - shouldn't happen with buffered channel
					// but handle gracefully
				}
				return
			}

			// Failure
			atomic.AddInt64(&m.stats.TotalHedgeFailures, 1)
			m.onHedgeFailed(ctx, index, err, config)

			select {
			case resultCh <- result{index: index, err: err}:
			case <-ctx.Done():
			}
		}(i, fn)
	}

	// Collect results and return first success
	var errors []error
	errorMap := make(map[int]error)

	// Wait for all goroutines to complete or first success
	resultChanClosed := make(chan struct{})
	go func() {
		wg.Wait()
		close(resultCh)
		close(resultChanClosed)
	}()

	// Wait for first success or all failures
	successReceived := false
	for {
		select {
		case result, ok := <-resultCh:
			if !ok {
				// Channel closed - all results received
				goto processResults
			}
			if result.err == nil {
				// First success
				if !successReceived {
					successReceived = true
					if config.CancelRemainingOnSuccess {
						// Don't wait for remaining results if we cancel on success
						// Drain channel in background to avoid goroutine leak
						go func() {
							for range resultCh {
							}
						}()
						// Update stats and return success immediately
						m.mu.Lock()
						m.stats.LastExecutionTime = time.Now()
						m.mu.Unlock()
						return nil
					}
				}
			} else {
				errorMap[result.index] = result.err
			}
		case <-resultChanClosed:
			// All goroutines completed - drain remaining results
			for result := range resultCh {
				if result.err == nil {
					if !successReceived {
						successReceived = true
					}
				} else {
					errorMap[result.index] = result.err
				}
			}
			goto processResults
		}
	}

processResults:

	// Update statistics
	m.mu.Lock()
	m.stats.LastExecutionTime = time.Now()
	m.mu.Unlock()

	if successReceived {
		// At least one hedge succeeded
		return nil
	}

	// All hedges failed - collect errors in order
	for i := 0; i < len(functions); i++ {
		if err, ok := errorMap[i]; ok {
			errors = append(errors, err)
		}
	}

	atomic.AddInt64(&m.stats.TotalAllHedgesFailed, 1)
	m.onAllHedgesFailed(ctx, errors, config)

	return NewError(
		ErrCodeAllHedgesFailed,
		fmt.Sprintf("all %d hedge(s) failed; errors: %v", len(functions), errors),
	)
}

// onHedgeAttempted invokes hedge attempted callbacks.
func (m *hedgeManager) onHedgeAttempted(ctx context.Context, index int, fn Executable, config Config) {
	if config.OnHedgeAttempted != nil {
		config.OnHedgeAttempted(ctx, index, fn)
	}

	if config.OnHedgeAttemptedAsync != nil {
		go config.OnHedgeAttemptedAsync(ctx, index, fn)
	}
}

// onHedgeSucceeded invokes hedge succeeded callbacks.
func (m *hedgeManager) onHedgeSucceeded(ctx context.Context, index int, result error, config Config) {
	if config.OnHedgeSucceeded != nil {
		config.OnHedgeSucceeded(ctx, index, result)
	}

	if config.OnHedgeSucceededAsync != nil {
		go config.OnHedgeSucceededAsync(ctx, index, result)
	}
}

// onHedgeFailed invokes hedge failed callbacks.
func (m *hedgeManager) onHedgeFailed(ctx context.Context, index int, err error, config Config) {
	if config.OnHedgeFailed != nil {
		config.OnHedgeFailed(ctx, index, err)
	}

	if config.OnHedgeFailedAsync != nil {
		go config.OnHedgeFailedAsync(ctx, index, err)
	}
}

// onAllHedgesFailed invokes all hedges failed callbacks.
func (m *hedgeManager) onAllHedgesFailed(ctx context.Context, errors []error, config Config) {
	if config.OnAllHedgesFailed != nil {
		config.OnAllHedgesFailed(ctx, errors)
	}

	if config.OnAllHedgesFailedAsync != nil {
		go config.OnAllHedgesFailedAsync(ctx, errors)
	}
}

// Stats returns statistics about hedge operations.
func (m *hedgeManager) Stats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return Stats{
		TotalExecutions:      atomic.LoadInt64(&m.stats.TotalExecutions),
		TotalHedgeAttempts:   atomic.LoadInt64(&m.stats.TotalHedgeAttempts),
		TotalHedgeSuccesses:  atomic.LoadInt64(&m.stats.TotalHedgeSuccesses),
		TotalHedgeFailures:   atomic.LoadInt64(&m.stats.TotalHedgeFailures),
		TotalAllHedgesFailed: atomic.LoadInt64(&m.stats.TotalAllHedgesFailed),
		TotalCancelled:       atomic.LoadInt64(&m.stats.TotalCancelled),
		LastExecutionTime:    m.stats.LastExecutionTime,
	}
}
