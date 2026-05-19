package mesh

import (
	"sync"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	mu           sync.RWMutex
	state        State
	failures     int
	threshold    int
	resetTimeout time.Duration
	lastFailure  time.Time
}

func NewCircuitBreaker(threshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:        StateClosed,
		threshold:    threshold,
		resetTimeout: resetTimeout,
	}
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.RLock()
	state := cb.state
	lastFailure := cb.lastFailure
	cb.mu.RUnlock()

	if state == StateClosed {
		return true
	}

	if state == StateOpen {
		if time.Since(lastFailure) > cb.resetTimeout {
			cb.mu.Lock()
			// Double check
			if cb.state == StateOpen {
				cb.state = StateHalfOpen
				cb.failures = 0
			}
			cb.mu.Unlock()
			return true
		}
		return false
	}

	// StateHalfOpen: allow one request to probe
	return true
}

func (cb *CircuitBreaker) Success() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		cb.state = StateClosed
		cb.failures = 0
	} else if cb.state == StateClosed {
		cb.failures = 0
	}
}

func (cb *CircuitBreaker) Failure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.state == StateClosed && cb.failures >= cb.threshold {
		cb.state = StateOpen
	} else if cb.state == StateHalfOpen {
		cb.state = StateOpen
	}
}
