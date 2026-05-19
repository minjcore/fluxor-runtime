package mesh

import (
	"context"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

// ServiceMesh defines the interface for the internal service mesh
type ServiceMesh interface {
	// Register registers a service with the mesh
	Register(serviceName string) error

	// Unregister unregisters a service from the mesh
	Unregister(serviceName string) error

	// Call performs a service call with mesh features (retries, circuit breaking, etc.)
	Call(ctx context.Context, serviceName string, action string, payload interface{}, opts CallOptions) (core.Message, error)
}

// CallOptions defines options for a service call
type CallOptions struct {
	Timeout        time.Duration
	RetryPolicy    *RetryPolicy
	CircuitBreaker *CircuitBreakerConfig
}

// RetryPolicy defines how to retry failed calls
type RetryPolicy struct {
	MaxRetries      int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
}

// CircuitBreakerConfig defines circuit breaker settings
type CircuitBreakerConfig struct {
	FailureThreshold int
	ResetTimeout     time.Duration
}

// DefaultRetryPolicy returns a default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:      3,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
	}
}
