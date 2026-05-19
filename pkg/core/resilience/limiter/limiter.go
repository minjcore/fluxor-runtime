package limiter

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Executable is a function that can be executed with rate limiting.
type Executable func(ctx context.Context) error

// Config configures rate limiting behavior.
type Config struct {
	// Rate is the maximum number of executions per interval.
	// Must be greater than 0.
	Rate int

	// Interval is the time window for the rate limit.
	// Must be greater than 0.
	// Defaults to 1 second.
	Interval time.Duration

	// Burst is the maximum burst size (number of tokens that can accumulate).
	// If zero, Burst equals Rate.
	// Defaults to 0 (equals Rate).
	Burst int

	// OnRateLimitExceeded is called when rate limit is exceeded.
	OnRateLimitExceeded func(ctx context.Context, rate int, interval time.Duration)

	// OnRateLimitExceededAsync is called asynchronously when rate limit is exceeded.
	OnRateLimitExceededAsync func(ctx context.Context, rate int, interval time.Duration)

	// OnAllowed is called when execution is allowed (before execution).
	OnAllowed func(ctx context.Context)

	// OnAllowedAsync is called asynchronously when execution is allowed.
	OnAllowedAsync func(ctx context.Context)

	// OnCompleted is called when execution completes.
	OnCompleted func(ctx context.Context, err error)

	// OnCompletedAsync is called asynchronously when execution completes.
	OnCompletedAsync func(ctx context.Context, err error)

	// EnableMetrics enables detailed metrics collection.
	EnableMetrics bool
}

// DefaultConfig returns the default rate limiter configuration.
func DefaultConfig() Config {
	return Config{
		Rate:     10,           // 10 executions per second
		Interval: time.Second,  // 1 second interval
		Burst:    0,            // Same as Rate
	}
}

// Manager provides rate limiting functionality.
type Manager interface {
	// Execute executes a function with rate limiting using default config.
	Execute(ctx context.Context, fn Executable) error

	// ExecuteWithConfig executes a function with the specified rate limiter config.
	ExecuteWithConfig(ctx context.Context, fn Executable, config Config) error

	// Allow checks if an execution would be allowed (non-blocking).
	Allow(ctx context.Context) bool

	// AllowWithConfig checks if an execution would be allowed with the specified config (non-blocking).
	AllowWithConfig(ctx context.Context, config Config) bool

	// Wait blocks until an execution is allowed.
	Wait(ctx context.Context) error

	// WaitWithConfig blocks until an execution is allowed with the specified config.
	WaitWithConfig(ctx context.Context, config Config) error

	// Stats returns statistics about rate limiter operations.
	Stats() Stats
}

// Stats contains statistics about rate limiter operations.
type Stats struct {
	// TotalExecutions is the total number of executions attempted.
	TotalExecutions int64

	// TotalAllowed is the total number of executions that were allowed.
	TotalAllowed int64

	// TotalRateLimited is the total number of executions that were rate limited.
	TotalRateLimited int64

	// TotalSuccessful is the total number of successful executions.
	TotalSuccessful int64

	// TotalFailed is the total number of failed executions.
	TotalFailed int64

	// LastExecutionTime is when the last execution completed.
	LastExecutionTime time.Time

	// CurrentTokens is the current number of available tokens.
	CurrentTokens int

	// CurrentRate is the current rate limit (executions per interval).
	CurrentRate int

	// CurrentInterval is the current rate limit interval.
	CurrentInterval time.Duration
}

// limiterManager implements the Manager interface using token bucket algorithm.
type limiterManager struct {
	config Config

	// Token bucket state
	tokens     int64         // Current number of tokens (atomic)
	lastRefill time.Time     // Last time tokens were refilled
	mu         sync.RWMutex  // Protects lastRefill

	// Statistics
	stats Stats
	statsMu sync.RWMutex
}

// NewManager creates a new rate limiter manager with default config.
func NewManager() Manager {
	return NewManagerWithConfig(DefaultConfig())
}

// NewManagerWithConfig creates a new rate limiter manager with the specified config.
func NewManagerWithConfig(config Config) Manager {
	if config.Rate <= 0 {
		config.Rate = DefaultConfig().Rate
	}
	if config.Interval <= 0 {
		config.Interval = DefaultConfig().Interval
	}
	if config.Burst <= 0 {
		config.Burst = config.Rate
	}

	return &limiterManager{
		config:    config,
		tokens:    int64(config.Burst), // Start with full bucket
		lastRefill: time.Now(),
	}
}

// Execute executes a function with rate limiting using default config.
func (m *limiterManager) Execute(ctx context.Context, fn Executable) error {
	return m.ExecuteWithConfig(ctx, fn, m.config)
}

// ExecuteWithConfig executes a function with the specified rate limiter config.
func (m *limiterManager) ExecuteWithConfig(ctx context.Context, fn Executable, config Config) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}
	if fn == nil {
		return NewError(ErrCodeNilFunction, "function cannot be nil")
	}

	// Normalize config
	if config.Rate <= 0 {
		config.Rate = m.config.Rate
	}
	if config.Interval <= 0 {
		config.Interval = m.config.Interval
	}
	if config.Burst <= 0 {
		config.Burst = config.Rate
	}

	atomic.AddInt64(&m.stats.TotalExecutions, 1)

	// Check if token is available (non-blocking first)
	if !m.consumeToken(config) {
		// No token available - wait for token (will respect context timeout/cancel)
		if err := m.waitForToken(ctx, config); err != nil {
			atomic.AddInt64(&m.stats.TotalRateLimited, 1)
			m.onRateLimitExceeded(ctx, config.Rate, config.Interval, config)
			if err == context.DeadlineExceeded {
				return NewError(ErrCodeContextTimeout, fmt.Sprintf("context timeout while waiting for rate limit: %v", err))
			}
			if err == context.Canceled {
				return NewError(ErrCodeContextCanceled, fmt.Sprintf("context cancelled while waiting for rate limit: %v", err))
			}
			return NewError(ErrCodeRateLimitExceeded, fmt.Sprintf("rate limit exceeded: %d per %v", config.Rate, config.Interval))
		}
	}

	atomic.AddInt64(&m.stats.TotalAllowed, 1)
	m.onAllowed(ctx, config)

	// Execute function
	err := fn(ctx)

	// Update statistics
	m.statsMu.Lock()
	m.stats.LastExecutionTime = time.Now()
	m.statsMu.Unlock()

	if err == nil {
		atomic.AddInt64(&m.stats.TotalSuccessful, 1)
	} else {
		atomic.AddInt64(&m.stats.TotalFailed, 1)
	}

	m.onCompleted(ctx, err, config)

	return err
}

// Allow checks if an execution would be allowed (non-blocking).
func (m *limiterManager) Allow(ctx context.Context) bool {
	return m.AllowWithConfig(ctx, m.config)
}

// AllowWithConfig checks if an execution would be allowed with the specified config (non-blocking).
func (m *limiterManager) AllowWithConfig(ctx context.Context, config Config) bool {
	if ctx == nil {
		return false
	}

	// Normalize config
	if config.Rate <= 0 {
		config.Rate = m.config.Rate
	}
	if config.Interval <= 0 {
		config.Interval = m.config.Interval
	}
	if config.Burst <= 0 {
		config.Burst = config.Rate
	}

	return m.consumeToken(config)
}

// Wait blocks until an execution is allowed.
func (m *limiterManager) Wait(ctx context.Context) error {
	return m.WaitWithConfig(ctx, m.config)
}

// WaitWithConfig blocks until an execution is allowed with the specified config.
func (m *limiterManager) WaitWithConfig(ctx context.Context, config Config) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}

	// Normalize config
	if config.Rate <= 0 {
		config.Rate = m.config.Rate
	}
	if config.Interval <= 0 {
		config.Interval = m.config.Interval
	}
	if config.Burst <= 0 {
		config.Burst = config.Rate
	}

	return m.waitForToken(ctx, config)
}

// waitForToken waits until a token is available or context is cancelled/timeout.
func (m *limiterManager) waitForToken(ctx context.Context, config Config) error {
	// Try to consume token immediately
	if m.consumeToken(config) {
		return nil
	}

	// Calculate refill rate
	refillRate := config.Interval / time.Duration(config.Rate)
	if refillRate <= 0 {
		refillRate = time.Millisecond
	}

	// Wait with exponential backoff
	waitTime := refillRate
	maxWaitTime := config.Interval

	for {
		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try to consume token
		if m.consumeToken(config) {
			return nil
		}

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Exponential backoff up to max wait time
			waitTime *= 2
			if waitTime > maxWaitTime {
				waitTime = maxWaitTime
			}
		}
	}
}

// consumeToken consumes a token if available (non-blocking).
func (m *limiterManager) consumeToken(config Config) bool {
	// Refill tokens if needed
	m.refillTokens(config)

	// Try to consume token
	for {
		current := atomic.LoadInt64(&m.tokens)
		if current <= 0 {
			return false
		}

		// Try to decrement atomically
		if atomic.CompareAndSwapInt64(&m.tokens, current, current-1) {
			return true
		}

		// Retry if CAS failed (loop continues until success or tokens <= 0)
	}
}

// refillTokens refills tokens based on elapsed time.
func (m *limiterManager) refillTokens(config Config) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(m.lastRefill)

	if elapsed < config.Interval {
		return // Not enough time has passed
	}

	// Calculate tokens to add
	refillRate := config.Interval / time.Duration(config.Rate)
	if refillRate <= 0 {
		refillRate = time.Millisecond
	}

	tokensToAdd := int64(elapsed / refillRate)
	if tokensToAdd <= 0 {
		return
	}

	// Refill tokens (up to burst size)
	current := atomic.LoadInt64(&m.tokens)
	newTokens := current + tokensToAdd
	burst := int64(config.Burst)
	if newTokens > burst {
		newTokens = burst
	}

	atomic.StoreInt64(&m.tokens, newTokens)
	m.lastRefill = now
}

// onRateLimitExceeded invokes rate limit exceeded callbacks.
func (m *limiterManager) onRateLimitExceeded(ctx context.Context, rate int, interval time.Duration, config Config) {
	if config.OnRateLimitExceeded != nil {
		config.OnRateLimitExceeded(ctx, rate, interval)
	}

	if config.OnRateLimitExceededAsync != nil {
		go config.OnRateLimitExceededAsync(ctx, rate, interval)
	}
}

// onAllowed invokes allowed callbacks.
func (m *limiterManager) onAllowed(ctx context.Context, config Config) {
	if config.OnAllowed != nil {
		config.OnAllowed(ctx)
	}

	if config.OnAllowedAsync != nil {
		go config.OnAllowedAsync(ctx)
	}
}

// onCompleted invokes completion callbacks.
func (m *limiterManager) onCompleted(ctx context.Context, err error, config Config) {
	if config.OnCompleted != nil {
		config.OnCompleted(ctx, err)
	}

	if config.OnCompletedAsync != nil {
		go config.OnCompletedAsync(ctx, err)
	}
}

// Stats returns statistics about rate limiter operations.
func (m *limiterManager) Stats() Stats {
	m.statsMu.RLock()
	defer m.statsMu.RUnlock()

	// Refill tokens to get accurate current count
	m.refillTokens(m.config)

	return Stats{
		TotalExecutions:  atomic.LoadInt64(&m.stats.TotalExecutions),
		TotalAllowed:     atomic.LoadInt64(&m.stats.TotalAllowed),
		TotalRateLimited: atomic.LoadInt64(&m.stats.TotalRateLimited),
		TotalSuccessful:  atomic.LoadInt64(&m.stats.TotalSuccessful),
		TotalFailed:      atomic.LoadInt64(&m.stats.TotalFailed),
		LastExecutionTime: m.stats.LastExecutionTime,
		CurrentTokens:    int(atomic.LoadInt64(&m.tokens)),
		CurrentRate:      m.config.Rate,
		CurrentInterval:  m.config.Interval,
	}
}
