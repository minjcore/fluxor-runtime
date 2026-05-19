package proxy

import (
	"sync"
	"time"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu                sync.RWMutex
	state             CircuitBreakerState
	failureCount      int
	successCount      int
	lastFailureTime   time.Time
	openDuration      time.Duration
	failureThreshold  int
	successThreshold  int
	halfOpenMaxCalls   int
	halfOpenCalls      int
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold int, openDuration time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		openDuration:     openDuration,
		successThreshold: 2, // Need 2 successes to close from half-open
		halfOpenMaxCalls: 3, // Allow 3 calls in half-open state
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	state := cb.state
	cb.mu.Unlock()

	// Check if circuit is open
	if state == StateOpen {
		cb.mu.Lock()
		// Check if we should transition to half-open
		if time.Since(cb.lastFailureTime) >= cb.openDuration {
			cb.state = StateHalfOpen
			cb.halfOpenCalls = 0
			cb.successCount = 0
			state = StateHalfOpen
		}
		cb.mu.Unlock()

		if state == StateOpen {
			return NewProxyError(ErrCodeCircuitBreakerOpen, "circuit breaker is open")
		}
	}

	// Check half-open call limit
	if state == StateHalfOpen {
		cb.mu.Lock()
		if cb.halfOpenCalls >= cb.halfOpenMaxCalls {
			cb.mu.Unlock()
			return NewProxyError(ErrCodeCircuitBreakerOpen, "circuit breaker half-open call limit reached")
		}
		cb.halfOpenCalls++
		cb.mu.Unlock()
	}

	// Execute function
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		// Failure
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		if cb.state == StateHalfOpen {
			// Half-open -> Open
			cb.state = StateOpen
			cb.halfOpenCalls = 0
		} else if cb.failureCount >= cb.failureThreshold {
			// Closed -> Open
			cb.state = StateOpen
		}
	} else {
		// Success
		cb.failureCount = 0

		if cb.state == StateHalfOpen {
			cb.successCount++
			if cb.successCount >= cb.successThreshold {
				// Half-open -> Closed
				cb.state = StateClosed
				cb.halfOpenCalls = 0
				cb.successCount = 0
			}
		}
	}

	return err
}

// State returns the current circuit breaker state
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.halfOpenCalls = 0
}
