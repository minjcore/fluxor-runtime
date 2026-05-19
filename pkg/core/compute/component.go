package compute

import (
	"context"
	"fmt"
	"sync"

	"github.com/fluxorio/fluxor/pkg/core"
)

// ComputeComponent is a Fluxor component that runs compute tasks in a worker pool
// This is the framework-level pattern for CPU-bound work (LLM, FFmpeg, ML, crypto, etc.)
type ComputeComponent[T any] struct {
	*core.BaseComponent
	pool    *ComputePool[T]
	config  Config
	handler func(context.Context, T) (interface{}, error)
	// Override state management (Go doesn't support method overriding)
	mu      sync.RWMutex
	started bool
}

// NewComputeComponent creates a new compute component
func NewComputeComponent[T any](name string, handler func(context.Context, T) (interface{}, error), config Config) *ComputeComponent[T] {
	return &ComputeComponent[T]{
		BaseComponent: core.NewBaseComponent(name),
		config:        config,
		handler:       handler,
	}
}

// Start initializes the component.
// Note: Go does not support virtual dispatch for embedded methods, so we must
// override Start/Stop and call our doStart/doStop explicitly.
func (c *ComputeComponent[T]) Start(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return &core.EventBusError{Code: "ALREADY_STARTED", Message: "component already started"}
	}
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}
	if err := c.doStart(ctx); err != nil {
		return err
	}

	c.started = true
	return nil
}

// Stop stops the component.
func (c *ComputeComponent[T]) Stop(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil
	}
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}
	if err := c.doStop(ctx); err != nil {
		return err
	}

	c.started = false
	return nil
}

// IsStarted returns whether the component is started.
func (c *ComputeComponent[T]) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

// doStart initializes the compute pool
func (c *ComputeComponent[T]) doStart(ctx core.FluxorContext) error {
	// Create handler wrapper that converts interface{} to T
	// The handler returns interface{} which can be any type (e.g., ChatResponse for LLM)
	wrappedHandler := func(goCtx context.Context, payload interface{}) (interface{}, error) {
		// Type assertion
		typedPayload, ok := payload.(T)
		if !ok {
			return nil, fmt.Errorf("invalid payload type")
		}

		// Call original handler (returns interface{}, e.g., *ChatResponse)
		result, err := c.handler(goCtx, typedPayload)
		// Return result as-is (will be type-asserted by caller if needed)
		return result, err
	}

	// Create pool (explicitly specify type parameter for inference)
	gocmdCtx := ctx.GoCMD().Context()
	pool, err := NewComputePool[T](gocmdCtx, wrappedHandler, c.config)
	if err != nil {
		return &core.EventBusError{Code: "COMPUTE_POOL_ERROR", Message: err.Error()}
	}

	c.pool = pool

	// Start pool
	if err := pool.Start(); err != nil {
		return &core.EventBusError{Code: "COMPUTE_POOL_START_ERROR", Message: err.Error()}
	}

	// Notify via EventBus
	eventBus := c.EventBus()
	if eventBus != nil {
		_ = eventBus.Publish("compute.component.ready", map[string]interface{}{
			"component": c.Name(),
			"workers":   pool.Stats().Workers,
		})
	}

	return nil
}

// doStop stops the compute pool
func (c *ComputeComponent[T]) doStop(ctx core.FluxorContext) error {
	if c.pool != nil {
		gocmdCtx := ctx.GoCMD().Context()
		if err := c.pool.Stop(gocmdCtx); err != nil {
			return &core.EventBusError{Code: "COMPUTE_POOL_STOP_ERROR", Message: err.Error()}
		}
	}
	c.pool = nil

	// Notify via EventBus
	eventBus := c.EventBus()
	if eventBus != nil {
		_ = eventBus.Publish("compute.component.stopped", map[string]interface{}{
			"component": c.Name(),
		})
	}

	return nil
}

// Submit submits a compute job asynchronously
// Returns immediately with a Future
func (c *ComputeComponent[T]) Submit(ctx core.FluxorContext, key string, payload T) (*Future[T], error) {
	if c.pool == nil || !c.pool.IsRunning() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "compute component is not started"}
	}

	return c.pool.Submit(ctx.Context(), key, payload)
}

// SubmitSync submits a job and waits for the result
// WARNING: This blocks! Use only when necessary
func (c *ComputeComponent[T]) SubmitSync(ctx core.FluxorContext, key string, payload T) (T, error) {
	future, err := c.Submit(ctx, key, payload)
	if err != nil {
		var zero T
		return zero, err
	}

	return future.Get(ctx.Context())
}

// Pool returns the underlying compute pool (for advanced usage)
func (c *ComputeComponent[T]) Pool() *ComputePool[T] {
	return c.pool
}

// Stats returns pool statistics
func (c *ComputeComponent[T]) Stats() PoolStats {
	if c.pool == nil {
		return PoolStats{}
	}
	return c.pool.Stats()
}

// ComputeVerticle is a Fluxor verticle that provides compute capabilities
// This is the recommended pattern for CPU-bound work in event-driven systems
type ComputeVerticle[T any] struct {
	*core.BaseVerticle
	component *ComputeComponent[T]
}

// NewComputeVerticle creates a new compute verticle
func NewComputeVerticle[T any](name string, handler func(context.Context, T) (interface{}, error), config Config) *ComputeVerticle[T] {
	component := NewComputeComponent(name, handler, config)
	return &ComputeVerticle[T]{
		BaseVerticle: core.NewBaseVerticle(name),
		component:    component,
	}
}

// Start overrides BaseVerticle.Start() to initialize compute component
func (v *ComputeVerticle[T]) Start(ctx core.FluxorContext) error {
	// Call base Start() first
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}

	// Set parent and start component
	v.component.SetParent(v.BaseVerticle)
	return v.component.Start(ctx)
}

// Stop overrides BaseVerticle.Stop() to stop compute component
func (v *ComputeVerticle[T]) Stop(ctx core.FluxorContext) error {
	// Stop component first
	if err := v.component.Stop(ctx); err != nil {
		return err
	}

	// Call base Stop()
	return v.BaseVerticle.Stop(ctx)
}

// Submit submits a compute job asynchronously
func (v *ComputeVerticle[T]) Submit(ctx core.FluxorContext, key string, payload T) (*Future[T], error) {
	return v.component.Submit(ctx, key, payload)
}

// SubmitSync submits a job and waits for the result
// WARNING: This blocks the EventLoop! Use only when necessary
func (v *ComputeVerticle[T]) SubmitSync(ctx core.FluxorContext, key string, payload T) (T, error) {
	return v.component.SubmitSync(ctx, key, payload)
}

// Component returns the underlying compute component
func (v *ComputeVerticle[T]) Component() *ComputeComponent[T] {
	return v.component
}

// Stats returns pool statistics
func (v *ComputeVerticle[T]) Stats() PoolStats {
	return v.component.Stats()
}
