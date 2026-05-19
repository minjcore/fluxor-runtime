package mesh

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

type meshImpl struct {
	eventBus core.EventBus
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
}

func NewServiceMesh(eventBus core.EventBus) ServiceMesh {
	return &meshImpl{
		eventBus: eventBus,
		breakers: make(map[string]*CircuitBreaker),
	}
}

func (m *meshImpl) Register(serviceName string) error {
	// In a real mesh, this might register with a central discovery service
	// For now, we just ensure we have a circuit breaker map entry
	m.getCircuitBreaker(serviceName)
	return nil
}

func (m *meshImpl) Unregister(serviceName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.breakers, serviceName)
	return nil
}

func (m *meshImpl) Call(ctx context.Context, serviceName string, action string, payload interface{}, opts CallOptions) (core.Message, error) {
	cb := m.getCircuitBreaker(serviceName)
	if !cb.Allow() {
		return nil, fmt.Errorf("circuit breaker open for service %s", serviceName)
	}

	// Determine address (simple convention: service.action)
	address := fmt.Sprintf("%s.%s", serviceName, action)

	var lastErr error
	retryPolicy := opts.RetryPolicy
	if retryPolicy == nil {
		retryPolicy = DefaultRetryPolicy()
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	for i := 0; i <= retryPolicy.MaxRetries; i++ {
		// If context is canceled, stop
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Perform request
		resp, err := m.eventBus.Request(address, payload, timeout)
		if err == nil {
			cb.Success()
			return resp, nil
		}

		lastErr = err

		// If it's a timeout or network error, we might want to fail the breaker
		// For simplicity, we count all errors as failures
		cb.Failure()

		// Wait before retry
		if i < retryPolicy.MaxRetries {
			sleepDuration := retryPolicy.InitialInterval * time.Duration(1<<uint(i)) // Exponential backoff
			if sleepDuration > retryPolicy.MaxInterval {
				sleepDuration = retryPolicy.MaxInterval
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(sleepDuration):
				// continue
			}
		}
	}

	return nil, fmt.Errorf("call failed after %d retries: %w", retryPolicy.MaxRetries, lastErr)
}

func (m *meshImpl) getCircuitBreaker(serviceName string) *CircuitBreaker {
	m.mu.RLock()
	cb, ok := m.breakers[serviceName]
	m.mu.RUnlock()

	if ok {
		return cb
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double check
	if cb, ok := m.breakers[serviceName]; ok {
		return cb
	}

	// Default circuit breaker config
	cb = NewCircuitBreaker(5, 10*time.Second)
	m.breakers[serviceName] = cb
	return cb
}
