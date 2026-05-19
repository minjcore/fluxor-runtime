package breaker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// State represents the circuit breaker state.
type State int32

const (
	// StateClosed is the normal operating state - requests pass through.
	StateClosed State = iota

	// StateOpen is the failing state - requests are rejected immediately.
	StateOpen

	// StateHalfOpen is the testing state - allows limited requests to test if service recovered.
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// Executable is a function that can be executed with circuit breaker protection.
type Executable func(ctx context.Context) error

// Config configures circuit breaker behavior.
type Config struct {
	// FailureThreshold is the number of consecutive failures before opening the circuit.
	// Must be greater than 0. Defaults to 5.
	FailureThreshold int

	// SuccessThreshold is the number of consecutive successes needed to close from HalfOpen.
	// Must be greater than 0. Defaults to 1.
	SuccessThreshold int

	// Timeout is the duration to wait before transitioning from Open to HalfOpen.
	// Must be greater than 0. Defaults to 60 seconds.
	Timeout time.Duration

	// HalfOpenMaxRequests is the maximum number of requests allowed in HalfOpen state.
	// Must be greater than 0. Defaults to 1.
	HalfOpenMaxRequests int

	// OnStateChanged is called when the circuit breaker state changes.
	OnStateChanged func(ctx context.Context, from State, to State)

	// OnStateChangedAsync is called asynchronously when state changes.
	OnStateChangedAsync func(ctx context.Context, from State, to State)

	// OnOpened is called when the circuit breaker opens (transitions to Open).
	OnOpened func(ctx context.Context)

	// OnOpenedAsync is called asynchronously when circuit breaker opens.
	OnOpenedAsync func(ctx context.Context)

	// OnClosed is called when the circuit breaker closes (transitions to Closed).
	OnClosed func(ctx context.Context)

	// OnClosedAsync is called asynchronously when circuit breaker closes.
	OnClosedAsync func(ctx context.Context)

	// OnHalfOpened is called when the circuit breaker half-opens (transitions to HalfOpen).
	OnHalfOpened func(ctx context.Context)

	// OnHalfOpenedAsync is called asynchronously when circuit breaker half-opens.
	OnHalfOpenedAsync func(ctx context.Context)

	// OnExecutionRejected is called when an execution is rejected (circuit open).
	OnExecutionRejected func(ctx context.Context)

	// OnExecutionRejectedAsync is called asynchronously when execution is rejected.
	OnExecutionRejectedAsync func(ctx context.Context)

	// EnableMetrics enables detailed metrics collection.
	EnableMetrics bool
}

// DefaultConfig returns the default circuit breaker configuration.
func DefaultConfig() Config {
	return Config{
		FailureThreshold:    5,
		SuccessThreshold:    1,
		Timeout:             60 * time.Second,
		HalfOpenMaxRequests: 1,
	}
}

// Manager provides circuit breaker functionality.
type Manager interface {
	// Execute executes a function with circuit breaker protection using default config.
	Execute(ctx context.Context, fn Executable) error

	// ExecuteWithConfig executes a function with the specified circuit breaker config.
	ExecuteWithConfig(ctx context.Context, fn Executable, config Config) error

	// State returns the current circuit breaker state.
	State() State

	// Allow checks if execution is allowed (non-blocking).
	Allow(ctx context.Context) bool

	// AllowWithConfig checks if execution is allowed with the specified config (non-blocking).
	AllowWithConfig(ctx context.Context, config Config) bool

	// Success records a successful execution.
	Success()

	// Failure records a failed execution.
	Failure()

	// Reset resets the circuit breaker to Closed state.
	Reset()

	// Stats returns statistics about circuit breaker operations.
	Stats() Stats
}

// Stats contains statistics about circuit breaker operations.
type Stats struct {
	// TotalExecutions is the total number of executions attempted.
	TotalExecutions int64

	// TotalAllowed is the total number of executions that were allowed.
	TotalAllowed int64

	// TotalRejected is the total number of executions that were rejected (circuit open).
	TotalRejected int64

	// TotalSuccessful is the total number of successful executions.
	TotalSuccessful int64

	// TotalFailed is the total number of failed executions.
	TotalFailed int64

	// ConsecutiveFailures is the current number of consecutive failures.
	ConsecutiveFailures int

	// ConsecutiveSuccesses is the current number of consecutive successes.
	ConsecutiveSuccesses int

	// CurrentState is the current circuit breaker state.
	CurrentState State

	// LastFailureTime is when the last failure occurred.
	LastFailureTime time.Time

	// LastSuccessTime is when the last success occurred.
	LastSuccessTime time.Time

	// LastStateChangeTime is when the state last changed.
	LastStateChangeTime time.Time
}

// breakerManager implements the Manager interface.
type breakerManager struct {
	config Config

	// Circuit breaker state
	state             State     // Current state (atomic)
	consecutiveFailures int32   // Consecutive failures (atomic)
	consecutiveSuccesses int32  // Consecutive successes (atomic)
	halfOpenRequests  int32     // Current requests in HalfOpen state (atomic)
	lastFailureTime   time.Time // Last failure time
	lastSuccessTime   time.Time // Last success time
	lastStateChangeTime time.Time // Last state change time
	stateMu           sync.RWMutex // Protects time fields

	// Statistics
	stats Stats
	statsMu sync.RWMutex
}

// NewManager creates a new circuit breaker manager with default config.
func NewManager() Manager {
	return NewManagerWithConfig(DefaultConfig())
}

// NewManagerWithConfig creates a new circuit breaker manager with the specified config.
func NewManagerWithConfig(config Config) Manager {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = DefaultConfig().FailureThreshold
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = DefaultConfig().SuccessThreshold
	}
	if config.Timeout <= 0 {
		config.Timeout = DefaultConfig().Timeout
	}
	if config.HalfOpenMaxRequests <= 0 {
		config.HalfOpenMaxRequests = DefaultConfig().HalfOpenMaxRequests
	}

	return &breakerManager{
		config:            config,
		state:             StateClosed,
		lastStateChangeTime: time.Now(),
	}
}

// State returns the current circuit breaker state.
func (m *breakerManager) State() State {
	return State(atomic.LoadInt32((*int32)(&m.state)))
}

// Execute executes a function with circuit breaker protection using default config.
func (m *breakerManager) Execute(ctx context.Context, fn Executable) error {
	return m.ExecuteWithConfig(ctx, fn, m.config)
}

// ExecuteWithConfig executes a function with the specified circuit breaker config.
func (m *breakerManager) ExecuteWithConfig(ctx context.Context, fn Executable, config Config) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}
	if fn == nil {
		return NewError(ErrCodeNilFunction, "function cannot be nil")
	}

	// Normalize config
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = m.config.FailureThreshold
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = m.config.SuccessThreshold
	}
	if config.Timeout <= 0 {
		config.Timeout = m.config.Timeout
	}
	if config.HalfOpenMaxRequests <= 0 {
		config.HalfOpenMaxRequests = m.config.HalfOpenMaxRequests
	}

	atomic.AddInt64(&m.stats.TotalExecutions, 1)

	// Check if execution is allowed
	if !m.allowExecution(ctx, config) {
		atomic.AddInt64(&m.stats.TotalRejected, 1)
		m.onExecutionRejected(ctx, config)
		return NewError(ErrCodeCircuitOpen, fmt.Sprintf("circuit breaker is %s", m.State().String()))
	}

	atomic.AddInt64(&m.stats.TotalAllowed, 1)

	// Execute function
	err := fn(ctx)

	// Record result
	if err == nil {
		m.recordSuccess(config)
	} else {
		m.recordFailure(config)
	}

	return err
}

// allowExecution checks if execution is allowed based on current state.
func (m *breakerManager) allowExecution(ctx context.Context, config Config) bool {
	currentState := m.State()

	switch currentState {
	case StateClosed:
		return true

	case StateOpen:
		// Check if timeout has elapsed
		m.stateMu.RLock()
		lastFailureTime := m.lastFailureTime
		lastStateChangeTime := m.lastStateChangeTime
		m.stateMu.RUnlock()

		// Use lastFailureTime or lastStateChangeTime (whichever is later) as the transition time
		transitionTime := lastFailureTime
		if lastStateChangeTime.After(lastFailureTime) {
			transitionTime = lastStateChangeTime
		}

		if !transitionTime.IsZero() && time.Since(transitionTime) >= config.Timeout {
			// Transition to HalfOpen
			if atomic.CompareAndSwapInt32((*int32)(&m.state), int32(StateOpen), int32(StateHalfOpen)) {
				m.stateMu.Lock()
				m.lastStateChangeTime = time.Now()
				atomic.StoreInt32(&m.consecutiveSuccesses, 0)
				atomic.StoreInt32(&m.consecutiveFailures, 0)
				atomic.StoreInt32(&m.halfOpenRequests, 0)
				m.stateMu.Unlock()

				m.onStateChanged(ctx, StateOpen, StateHalfOpen, config)
				m.onHalfOpened(ctx, config)
				// Fall through to HalfOpen logic
			} else {
				// Another goroutine transitioned - recheck state
				currentState = m.State()
				if currentState != StateHalfOpen {
					return false
				}
				// Fall through to HalfOpen case
			}
		} else {
			return false
		}
		fallthrough

	case StateHalfOpen:
		// Check if we've reached max requests
		requests := atomic.LoadInt32(&m.halfOpenRequests)
		if requests >= int32(config.HalfOpenMaxRequests) {
			return false
		}

		// Increment requests atomically
		atomic.AddInt32(&m.halfOpenRequests, 1)
		return true

	default:
		return false
	}
}

// Allow checks if execution is allowed (non-blocking).
func (m *breakerManager) Allow(ctx context.Context) bool {
	return m.AllowWithConfig(ctx, m.config)
}

// AllowWithConfig checks if execution is allowed with the specified config (non-blocking).
func (m *breakerManager) AllowWithConfig(ctx context.Context, config Config) bool {
	if ctx == nil {
		return false
	}

	// Normalize config
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = m.config.FailureThreshold
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = m.config.SuccessThreshold
	}
	if config.Timeout <= 0 {
		config.Timeout = m.config.Timeout
	}
	if config.HalfOpenMaxRequests <= 0 {
		config.HalfOpenMaxRequests = m.config.HalfOpenMaxRequests
	}

	return m.allowExecution(ctx, config)
}

// recordSuccess records a successful execution.
func (m *breakerManager) recordSuccess(config Config) {
	atomic.AddInt64(&m.stats.TotalSuccessful, 1)

	atomic.StoreInt32(&m.consecutiveFailures, 0) // Reset failures
	atomic.AddInt32(&m.consecutiveSuccesses, 1)

	m.stateMu.Lock()
	m.lastSuccessTime = time.Now()
	consecutiveSuccesses := atomic.LoadInt32(&m.consecutiveSuccesses)
	m.stateMu.Unlock()

	currentState := m.State()

	switch currentState {
	case StateClosed:
		// Reset failures in Closed state
		atomic.StoreInt32(&m.consecutiveFailures, 0)

	case StateHalfOpen:
		// Check if we've reached success threshold
		if consecutiveSuccesses >= int32(config.SuccessThreshold) {
			// Transition to Closed
			if atomic.CompareAndSwapInt32((*int32)(&m.state), int32(StateHalfOpen), int32(StateClosed)) {
				m.stateMu.Lock()
				m.lastStateChangeTime = time.Now()
				m.consecutiveFailures = 0
				m.consecutiveSuccesses = 0
				m.halfOpenRequests = 0
				m.stateMu.Unlock()

				m.onStateChanged(context.Background(), StateHalfOpen, StateClosed, config)
				m.onClosed(context.Background(), config)
			}
		}
	}
}

// recordFailure records a failed execution.
func (m *breakerManager) recordFailure(config Config) {
	atomic.AddInt64(&m.stats.TotalFailed, 1)

	atomic.StoreInt32(&m.consecutiveSuccesses, 0) // Reset successes
	atomic.AddInt32(&m.consecutiveFailures, 1)

	m.stateMu.Lock()
	m.lastFailureTime = time.Now()
	consecutiveFailures := atomic.LoadInt32(&m.consecutiveFailures)
	m.stateMu.Unlock()

	currentState := m.State()

	switch currentState {
	case StateClosed:
		// Check if we've reached failure threshold
		if consecutiveFailures >= int32(config.FailureThreshold) {
			// Transition to Open
			if atomic.CompareAndSwapInt32((*int32)(&m.state), int32(StateClosed), int32(StateOpen)) {
				m.stateMu.Lock()
				m.lastStateChangeTime = time.Now()
				m.stateMu.Unlock()

				m.onStateChanged(context.Background(), StateClosed, StateOpen, config)
				m.onOpened(context.Background(), config)
			}
		}

	case StateHalfOpen:
		// Any failure in HalfOpen transitions back to Open
		if atomic.CompareAndSwapInt32((*int32)(&m.state), int32(StateHalfOpen), int32(StateOpen)) {
			m.stateMu.Lock()
			m.lastStateChangeTime = time.Now()
			m.consecutiveSuccesses = 0
			m.halfOpenRequests = 0
			m.stateMu.Unlock()

			m.onStateChanged(context.Background(), StateHalfOpen, StateOpen, config)
			m.onOpened(context.Background(), config)
		}
	}
}

// Success records a successful execution.
func (m *breakerManager) Success() {
	m.recordSuccess(m.config)
}

// Failure records a failed execution.
func (m *breakerManager) Failure() {
	m.recordFailure(m.config)
}

// Reset resets the circuit breaker to Closed state.
func (m *breakerManager) Reset() {
	currentState := m.State()
	if currentState == StateClosed {
		return // Already closed
	}

	if atomic.CompareAndSwapInt32((*int32)(&m.state), int32(currentState), int32(StateClosed)) {
		m.stateMu.Lock()
		m.lastStateChangeTime = time.Now()
		m.consecutiveFailures = 0
		m.consecutiveSuccesses = 0
		m.halfOpenRequests = 0
		m.stateMu.Unlock()

		m.onStateChanged(context.Background(), currentState, StateClosed, m.config)
		m.onClosed(context.Background(), m.config)
	}
}

// onStateChanged invokes state change callbacks.
func (m *breakerManager) onStateChanged(ctx context.Context, from State, to State, config Config) {
	if config.OnStateChanged != nil {
		config.OnStateChanged(ctx, from, to)
	}

	if config.OnStateChangedAsync != nil {
		go config.OnStateChangedAsync(ctx, from, to)
	}
}

// onOpened invokes opened callbacks.
func (m *breakerManager) onOpened(ctx context.Context, config Config) {
	if config.OnOpened != nil {
		config.OnOpened(ctx)
	}

	if config.OnOpenedAsync != nil {
		go config.OnOpenedAsync(ctx)
	}
}

// onClosed invokes closed callbacks.
func (m *breakerManager) onClosed(ctx context.Context, config Config) {
	if config.OnClosed != nil {
		config.OnClosed(ctx)
	}

	if config.OnClosedAsync != nil {
		go config.OnClosedAsync(ctx)
	}
}

// onHalfOpened invokes half-opened callbacks.
func (m *breakerManager) onHalfOpened(ctx context.Context, config Config) {
	if config.OnHalfOpened != nil {
		config.OnHalfOpened(ctx)
	}

	if config.OnHalfOpenedAsync != nil {
		go config.OnHalfOpenedAsync(ctx)
	}
}

// onExecutionRejected invokes execution rejected callbacks.
func (m *breakerManager) onExecutionRejected(ctx context.Context, config Config) {
	if config.OnExecutionRejected != nil {
		config.OnExecutionRejected(ctx)
	}

	if config.OnExecutionRejectedAsync != nil {
		go config.OnExecutionRejectedAsync(ctx)
	}
}

// Stats returns statistics about circuit breaker operations.
func (m *breakerManager) Stats() Stats {
	m.statsMu.RLock()
	defer m.statsMu.RUnlock()

	m.stateMu.RLock()
	defer m.stateMu.RUnlock()

	return Stats{
		TotalExecutions:      atomic.LoadInt64(&m.stats.TotalExecutions),
		TotalAllowed:         atomic.LoadInt64(&m.stats.TotalAllowed),
		TotalRejected:        atomic.LoadInt64(&m.stats.TotalRejected),
		TotalSuccessful:      atomic.LoadInt64(&m.stats.TotalSuccessful),
		TotalFailed:          atomic.LoadInt64(&m.stats.TotalFailed),
		ConsecutiveFailures:  int(atomic.LoadInt32(&m.consecutiveFailures)),
		ConsecutiveSuccesses: int(atomic.LoadInt32(&m.consecutiveSuccesses)),
		CurrentState:         m.State(),
		LastFailureTime:      m.lastFailureTime,
		LastSuccessTime:      m.lastSuccessTime,
		LastStateChangeTime:  m.lastStateChangeTime,
	}
}
