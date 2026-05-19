package fallback

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Executable is a function that can be executed with fallback.
type Executable func(ctx context.Context) error

// FallbackPredicate determines if a fallback should be executed based on the error.
// Returns true if fallback should be executed, false otherwise.
type FallbackPredicate func(err error) bool

// Config configures fallback behavior.
type Config struct {
	// Fallbacks is a list of fallback functions to try in order if primary fails.
	// Each fallback is tried sequentially until one succeeds or all fail.
	Fallbacks []Executable

	// Predicate determines if fallback should be executed based on the error.
	// If nil, fallback is always executed on any error.
	// Defaults to nil (always fallback on error).
	Predicate FallbackPredicate

	// OnPrimaryError is called when the primary function fails (before fallback).
	OnPrimaryError func(ctx context.Context, err error)

	// OnPrimaryErrorAsync is called asynchronously when the primary function fails.
	OnPrimaryErrorAsync func(ctx context.Context, err error)

	// OnFallbackAttempted is called when a fallback function is attempted.
	OnFallbackAttempted func(ctx context.Context, index int, fn Executable)

	// OnFallbackAttemptedAsync is called asynchronously when a fallback is attempted.
	OnFallbackAttemptedAsync func(ctx context.Context, index int, fn Executable)

	// OnFallbackSucceeded is called when a fallback function succeeds.
	OnFallbackSucceeded func(ctx context.Context, index int, result error)

	// OnFallbackSucceededAsync is called asynchronously when a fallback succeeds.
	OnFallbackSucceededAsync func(ctx context.Context, index int, result error)

	// OnFallbackFailed is called when a fallback function fails.
	OnFallbackFailed func(ctx context.Context, index int, err error)

	// OnFallbackFailedAsync is called asynchronously when a fallback fails.
	OnFallbackFailedAsync func(ctx context.Context, index int, err error)

	// OnAllFallbacksExhausted is called when all fallbacks have been exhausted.
	OnAllFallbacksExhausted func(ctx context.Context, primaryErr error, fallbackErrors []error)

	// OnAllFallbacksExhaustedAsync is called asynchronously when all fallbacks are exhausted.
	OnAllFallbacksExhaustedAsync func(ctx context.Context, primaryErr error, fallbackErrors []error)

	// EnableMetrics enables detailed metrics collection.
	EnableMetrics bool
}

// DefaultConfig returns the default fallback configuration.
func DefaultConfig() Config {
	return Config{
		Predicate: nil, // Always fallback on any error
	}
}

// Manager provides fallback functionality.
type Manager interface {
	// Execute executes a primary function with fallback using default config.
	Execute(ctx context.Context, primary Executable) error

	// ExecuteWithConfig executes a primary function with the specified fallback config.
	ExecuteWithConfig(ctx context.Context, primary Executable, config Config) error

	// ExecuteWithFallback is a convenience method that executes a primary function with a single fallback.
	ExecuteWithFallback(ctx context.Context, primary Executable, fallback Executable) error

	// Stats returns statistics about fallback operations.
	Stats() Stats
}

// Stats contains statistics about fallback operations.
type Stats struct {
	// TotalExecutions is the total number of primary function executions.
	TotalExecutions int64

	// TotalPrimarySuccesses is the total number of successful primary executions (no fallback needed).
	TotalPrimarySuccesses int64

	// TotalPrimaryFailures is the total number of failed primary executions (fallback triggered).
	TotalPrimaryFailures int64

	// TotalFallbackAttempts is the total number of fallback function attempts.
	TotalFallbackAttempts int64

	// TotalFallbackSuccesses is the total number of successful fallback executions.
	TotalFallbackSuccesses int64

	// TotalFallbackFailures is the total number of failed fallback executions.
	TotalFallbackFailures int64

	// TotalAllFallbacksExhausted is the total number of times all fallbacks were exhausted.
	TotalAllFallbacksExhausted int64

	// LastExecutionTime is when the last execution completed.
	LastExecutionTime time.Time
}

// fallbackManager implements the Manager interface.
type fallbackManager struct {
	stats Stats
	mu    sync.RWMutex
}

// NewManager creates a new fallback manager with default config.
func NewManager() Manager {
	return &fallbackManager{}
}

// Execute executes a primary function with fallback using default config.
func (m *fallbackManager) Execute(ctx context.Context, primary Executable) error {
	return m.ExecuteWithConfig(ctx, primary, DefaultConfig())
}

// ExecuteWithFallback is a convenience method that executes a primary function with a single fallback.
func (m *fallbackManager) ExecuteWithFallback(ctx context.Context, primary Executable, fallback Executable) error {
	config := DefaultConfig()
	config.Fallbacks = []Executable{fallback}
	return m.ExecuteWithConfig(ctx, primary, config)
}

// ExecuteWithConfig executes a primary function with the specified fallback config.
func (m *fallbackManager) ExecuteWithConfig(ctx context.Context, primary Executable, config Config) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}
	if primary == nil {
		return NewError(ErrCodeNilPrimaryFunction, "primary function cannot be nil")
	}

	atomic.AddInt64(&m.stats.TotalExecutions, 1)

	// Execute primary function
	primaryErr := primary(ctx)

	// Update statistics
	m.mu.Lock()
	m.stats.LastExecutionTime = time.Now()
	m.mu.Unlock()

	// If primary succeeded, return success
	if primaryErr == nil {
		atomic.AddInt64(&m.stats.TotalPrimarySuccesses, 1)
		return nil
	}

	// Primary failed
	atomic.AddInt64(&m.stats.TotalPrimaryFailures, 1)
	m.onPrimaryError(ctx, primaryErr, config)

	// Check if fallback should be executed
	if config.Predicate != nil && !config.Predicate(primaryErr) {
		// Predicate says don't fallback
		return primaryErr
	}

	// Execute fallbacks
	if len(config.Fallbacks) == 0 {
		// No fallbacks configured, return primary error
		return primaryErr
	}

	var fallbackErrors []error
	for i, fallback := range config.Fallbacks {
		if fallback == nil {
			continue // Skip nil fallback
		}

		atomic.AddInt64(&m.stats.TotalFallbackAttempts, 1)
		m.onFallbackAttempted(ctx, i, fallback, config)

		// Execute fallback
		fallbackErr := fallback(ctx)

		if fallbackErr == nil {
			// Fallback succeeded
			atomic.AddInt64(&m.stats.TotalFallbackSuccesses, 1)
			m.onFallbackSucceeded(ctx, i, nil, config)
			return nil
		}

		// Fallback failed
		atomic.AddInt64(&m.stats.TotalFallbackFailures, 1)
		fallbackErrors = append(fallbackErrors, fallbackErr)
		m.onFallbackFailed(ctx, i, fallbackErr, config)
	}

	// All fallbacks exhausted
	atomic.AddInt64(&m.stats.TotalAllFallbacksExhausted, 1)
	m.onAllFallbacksExhausted(ctx, primaryErr, fallbackErrors, config)

	return NewError(
		ErrCodeAllFallbacksExhausted,
		fmt.Sprintf("all %d fallback(s) exhausted; primary error: %v; fallback errors: %v", len(config.Fallbacks), primaryErr, fallbackErrors),
	)
}

// onPrimaryError invokes primary error callbacks.
func (m *fallbackManager) onPrimaryError(ctx context.Context, err error, config Config) {
	if config.OnPrimaryError != nil {
		config.OnPrimaryError(ctx, err)
	}

	if config.OnPrimaryErrorAsync != nil {
		go config.OnPrimaryErrorAsync(ctx, err)
	}
}

// onFallbackAttempted invokes fallback attempted callbacks.
func (m *fallbackManager) onFallbackAttempted(ctx context.Context, index int, fn Executable, config Config) {
	if config.OnFallbackAttempted != nil {
		config.OnFallbackAttempted(ctx, index, fn)
	}

	if config.OnFallbackAttemptedAsync != nil {
		go config.OnFallbackAttemptedAsync(ctx, index, fn)
	}
}

// onFallbackSucceeded invokes fallback succeeded callbacks.
func (m *fallbackManager) onFallbackSucceeded(ctx context.Context, index int, result error, config Config) {
	if config.OnFallbackSucceeded != nil {
		config.OnFallbackSucceeded(ctx, index, result)
	}

	if config.OnFallbackSucceededAsync != nil {
		go config.OnFallbackSucceededAsync(ctx, index, result)
	}
}

// onFallbackFailed invokes fallback failed callbacks.
func (m *fallbackManager) onFallbackFailed(ctx context.Context, index int, err error, config Config) {
	if config.OnFallbackFailed != nil {
		config.OnFallbackFailed(ctx, index, err)
	}

	if config.OnFallbackFailedAsync != nil {
		go config.OnFallbackFailedAsync(ctx, index, err)
	}
}

// onAllFallbacksExhausted invokes all fallbacks exhausted callbacks.
func (m *fallbackManager) onAllFallbacksExhausted(ctx context.Context, primaryErr error, fallbackErrors []error, config Config) {
	if config.OnAllFallbacksExhausted != nil {
		config.OnAllFallbacksExhausted(ctx, primaryErr, fallbackErrors)
	}

	if config.OnAllFallbacksExhaustedAsync != nil {
		go config.OnAllFallbacksExhaustedAsync(ctx, primaryErr, fallbackErrors)
	}
}

// Stats returns statistics about fallback operations.
func (m *fallbackManager) Stats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return Stats{
		TotalExecutions:           atomic.LoadInt64(&m.stats.TotalExecutions),
		TotalPrimarySuccesses:     atomic.LoadInt64(&m.stats.TotalPrimarySuccesses),
		TotalPrimaryFailures:      atomic.LoadInt64(&m.stats.TotalPrimaryFailures),
		TotalFallbackAttempts:     atomic.LoadInt64(&m.stats.TotalFallbackAttempts),
		TotalFallbackSuccesses:    atomic.LoadInt64(&m.stats.TotalFallbackSuccesses),
		TotalFallbackFailures:     atomic.LoadInt64(&m.stats.TotalFallbackFailures),
		TotalAllFallbacksExhausted: atomic.LoadInt64(&m.stats.TotalAllFallbacksExhausted),
		LastExecutionTime:         m.stats.LastExecutionTime,
	}
}
