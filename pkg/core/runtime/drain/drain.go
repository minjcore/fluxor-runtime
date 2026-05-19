package drain

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Drainable represents a component that can be drained.
// Draining means stopping new work from being accepted while allowing
// in-flight work to complete gracefully.
type Drainable interface {
	// Drain gracefully drains the component, allowing in-flight work to complete.
	// It should stop accepting new work and wait for existing work to finish.
	// Returns an error if draining fails or times out.
	Drain(ctx context.Context) error
}

// Config configures drain behavior.
type Config struct {
	// DefaultTimeout is the default timeout for drain operations.
	// If zero, defaults to 30 seconds.
	DefaultTimeout time.Duration

	// OnDrainStart is called when draining starts for a component.
	OnDrainStart func(name string)

	// OnDrainComplete is called when draining completes for a component.
	OnDrainComplete func(name string, err error)

	// OnDrainError is called when a drain operation encounters an error.
	OnDrainError func(name string, err error)

	// Parallel controls whether components are drained in parallel or sequentially.
	// Defaults to false (sequential).
	Parallel bool
}

// DefaultConfig returns the default drain configuration.
func DefaultConfig() Config {
	return Config{
		DefaultTimeout: 30 * time.Second,
		Parallel:       false,
	}
}

// Stats contains statistics about drain operations.
type Stats struct {
	// TotalDrained is the total number of components successfully drained.
	TotalDrained int64

	// TotalFailed is the total number of components that failed to drain.
	TotalFailed int64

	// TotalTimeouts is the total number of drain operations that timed out.
	TotalTimeouts int64

	// LastDrainTime is when the last drain operation completed.
	LastDrainTime time.Time

	// IsDraining indicates if a drain operation is currently in progress.
	IsDraining bool

	// IsDrained indicates if all components have been drained.
	IsDrained bool
}

// Component represents a drainable component with a name.
type Component struct {
	Name     string
	Drainable Drainable
}

// Drainer manages draining of multiple components.
type Drainer struct {
	config     Config
	components []Component
	mu         sync.RWMutex
	stats      Stats
	isDraining int32 // atomic: 1 if draining, 0 otherwise
	isDrained  int64 // atomic: 1 if drained, 0 otherwise
}

// NewDrainer creates a new drainer with the given configuration.
func NewDrainer(config Config) *Drainer {
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 30 * time.Second
	}
	return &Drainer{
		config:     config,
		components: make([]Component, 0),
	}
}

// Register registers a drainable component with the drainer.
func (d *Drainer) Register(name string, drainable Drainable) error {
	if name == "" {
		return NewError(ErrCodeNilDrainable, "component name cannot be empty")
	}
	if drainable == nil {
		return NewError(ErrCodeNilDrainable, "drainable cannot be nil")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if already registered
	for _, comp := range d.components {
		if comp.Name == name {
			return NewError(ErrCodeAlreadyDrained, fmt.Sprintf("component %s already registered", name))
		}
	}

	d.components = append(d.components, Component{
		Name:     name,
		Drainable: drainable,
	})

	return nil
}

// Unregister removes a component from the drainer.
func (d *Drainer) Unregister(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for i, comp := range d.components {
		if comp.Name == name {
			d.components = append(d.components[:i], d.components[i+1:]...)
			return
		}
	}
}

// DrainAll drains all registered components.
// If a timeout is provided in the context, it's used. Otherwise, DefaultTimeout is used.
func (d *Drainer) DrainAll(ctx context.Context) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}

	d.mu.Lock()
	if atomic.LoadInt32(&d.isDraining) == 1 {
		d.mu.Unlock()
		return NewError(ErrCodeAlreadyDraining, "drain operation already in progress")
	}

	atomic.StoreInt32(&d.isDraining, 1)
	components := make([]Component, len(d.components))
	copy(components, d.components)
	d.mu.Unlock()

	defer atomic.StoreInt32(&d.isDraining, 0)

	// Use context timeout or default timeout
	timeout := d.config.DefaultTimeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
		if timeout <= 0 {
			timeout = d.config.DefaultTimeout
		}
	}

	drainCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	drainComponent := func(comp Component) {
		if d.config.OnDrainStart != nil {
			d.config.OnDrainStart(comp.Name)
		}

		err := comp.Drainable.Drain(drainCtx)
		if err != nil {
			atomic.AddInt64(&d.stats.TotalFailed, 1)
			if err == context.DeadlineExceeded || err == context.Canceled {
				atomic.AddInt64(&d.stats.TotalTimeouts, 1)
			}

			mu.Lock()
			errors = append(errors, fmt.Errorf("component %s: %w", comp.Name, err))
			mu.Unlock()

			if d.config.OnDrainError != nil {
				d.config.OnDrainError(comp.Name, err)
			}
		} else {
			atomic.AddInt64(&d.stats.TotalDrained, 1)
		}

		if d.config.OnDrainComplete != nil {
			d.config.OnDrainComplete(comp.Name, err)
		}
	}

	if d.config.Parallel {
		// Drain all components in parallel
		wg.Add(len(components))
		for _, comp := range components {
			comp := comp // capture loop variable
			go func() {
				defer wg.Done()
				drainComponent(comp)
			}()
		}
		wg.Wait()
	} else {
		// Drain components sequentially
		for _, comp := range components {
			drainComponent(comp)
		}
	}

	atomic.StoreInt64(&d.isDrained, 1)
	d.mu.Lock()
	d.stats.LastDrainTime = time.Now()
	d.mu.Unlock()

	if len(errors) > 0 {
		return fmt.Errorf("drain completed with %d error(s): %v", len(errors), errors)
	}

	return nil
}

// Drain drains a specific component by name.
func (d *Drainer) Drain(ctx context.Context, name string) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}

	d.mu.RLock()
	var component *Component
	for _, comp := range d.components {
		if comp.Name == name {
			component = &comp
			break
		}
	}
	d.mu.RUnlock()

	if component == nil {
		return NewError(ErrCodeDrainFailed, fmt.Sprintf("component %s not found", name))
	}

	timeout := d.config.DefaultTimeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
		if timeout <= 0 {
			timeout = d.config.DefaultTimeout
		}
	}

	drainCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if d.config.OnDrainStart != nil {
		d.config.OnDrainStart(component.Name)
	}

	err := component.Drainable.Drain(drainCtx)
	if err != nil {
		atomic.AddInt64(&d.stats.TotalFailed, 1)
		if err == context.DeadlineExceeded || err == context.Canceled {
			atomic.AddInt64(&d.stats.TotalTimeouts, 1)
		}
		if d.config.OnDrainError != nil {
			d.config.OnDrainError(component.Name, err)
		}
	} else {
		atomic.AddInt64(&d.stats.TotalDrained, 1)
	}

	if d.config.OnDrainComplete != nil {
		d.config.OnDrainComplete(component.Name, err)
	}

	return err
}

// Stats returns statistics about drain operations.
func (d *Drainer) Stats() Stats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := d.stats
	stats.IsDraining = atomic.LoadInt32(&d.isDraining) == 1
	stats.IsDrained = atomic.LoadInt64(&d.isDrained) == 1

	return stats
}

// Components returns a list of all registered component names.
func (d *Drainer) Components() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	names := make([]string, len(d.components))
	for i, comp := range d.components {
		names[i] = comp.Name
	}
	return names
}

// Reset resets the drainer statistics.
func (d *Drainer) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()

	atomic.StoreInt64(&d.stats.TotalDrained, 0)
	atomic.StoreInt64(&d.stats.TotalFailed, 0)
	atomic.StoreInt64(&d.stats.TotalTimeouts, 0)
	atomic.StoreInt64(&d.isDrained, 0)
	d.stats.LastDrainTime = time.Time{}
}
