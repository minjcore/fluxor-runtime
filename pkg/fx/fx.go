// Package fx provides the "Sugar Dev" pattern - syntactic sugar that makes Fluxor
// more accessible to Node.js developers by offering function-based APIs similar to
// Express.js, instead of the more verbose struct/interface-based Verticle pattern.
//
// The fx package provides dependency injection and lifecycle management through
// a function-based API, making it easier for developers familiar with Node.js/Express.js
// to get started with Fluxor. Under the hood, fx uses the same Fluxor Stream
// (GoCMD, EventBus), but provides a sweeter syntax for better developer experience.
//
// Example:
//
//	func setupApplication(deps map[reflect.Type]interface{}) error {
//	    // Setup your application
//	    return nil
//	}
//
//	app, _ := fx.New(ctx, fx.Invoke(fx.NewInvoker(setupApplication)))
//	app.Start()
//
// This is much simpler than implementing the Verticle interface, making it
// ideal for Node.js developers transitioning to Go.
package fx

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/fluxorio/fluxor/pkg/core"
)

// JSON is a convenience alias for core.JSON (Dev UX)
// This allows using fx.JSON instead of core.JSON in fx package
type JSON = core.JSON

// Error represents an FX framework error
type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

// Fluxor is the dependency injection and lifecycle management framework
type Fluxor struct {
	gocmd     core.GoCMD
	providers []Provider
	invokers  []Invoker
	lifecycle *lifecycle
	mu        sync.RWMutex
}

// Provider provides a value to the dependency injection container
type Provider interface {
	// Provide returns the value and optionally an error
	Provide() (interface{}, error)
}

// Invoker is a function that will be invoked after all providers are initialized
type Invoker interface {
	// Invoke is called with the provided dependencies
	Invoke(deps map[reflect.Type]interface{}) error
}

// New creates a new Fluxor instance
func New(ctx context.Context, options ...Option) (*Fluxor, error) {
	gocmd := core.NewGoCMD(ctx)

	fx := &Fluxor{
		gocmd:     gocmd,
		providers: make([]Provider, 0),
		invokers:  make([]Invoker, 0),
		lifecycle: newLifecycle(),
	}

	for _, opt := range options {
		if err := opt(fx); err != nil {
			return nil, err
		}
	}

	return fx, nil
}

// Option configures a Fluxor instance
type Option func(*Fluxor) error

// Provide registers a provider
func Provide(provider Provider) Option {
	return func(fx *Fluxor) error {
		fx.mu.Lock()
		defer fx.mu.Unlock()
		fx.providers = append(fx.providers, provider)
		return nil
	}
}

// Invoke registers an invoker
func Invoke(invoker Invoker) Option {
	return func(fx *Fluxor) error {
		fx.mu.Lock()
		defer fx.mu.Unlock()
		fx.invokers = append(fx.invokers, invoker)
		return nil
	}
}

// Start starts the Fluxor application
func (fx *Fluxor) Start() error {
	fx.mu.Lock()
	defer fx.mu.Unlock()

	// Build dependency map
	deps := make(map[reflect.Type]interface{})

	// Provide all dependencies
	for _, provider := range fx.providers {
		value, err := provider.Provide()
		if err != nil {
			return fmt.Errorf("provider error: %w", err)
		}

		valueType := reflect.TypeOf(value)
		if valueType != nil {
			deps[valueType] = value
		}
	}

	// Add GoCMD to dependencies
	deps[reflect.TypeOf((*core.GoCMD)(nil)).Elem()] = fx.gocmd
	deps[reflect.TypeOf((*core.EventBus)(nil)).Elem()] = fx.gocmd.EventBus()

	// Invoke all invokers
	for _, invoker := range fx.invokers {
		if err := invoker.Invoke(deps); err != nil {
			return fmt.Errorf("invoker error: %w", err)
		}
	}

	fx.lifecycle.start()
	return nil
}

// Stop stops the Fluxor application
func (fx *Fluxor) Stop() error {
	fx.mu.Lock()
	defer fx.mu.Unlock()

	fx.lifecycle.stop()
	return fx.gocmd.Close()
}

// GoCMD returns the GoCMD instance (kept as GoCMD for backward compatibility)
func (fx *Fluxor) GoCMD() core.GoCMD {
	return fx.gocmd
}

// KeepAndServe blocks until the application is stopped (e.g. by signal or Stop()).
func (fx *Fluxor) KeepAndServe() error {
	return fx.lifecycle.wait()
}

// lifecycle manages application lifecycle
type lifecycle struct {
	started chan struct{}
	stopped chan struct{}
	mu      sync.Mutex
}

func newLifecycle() *lifecycle {
	return &lifecycle{
		started: make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

func (l *lifecycle) start() {
	l.mu.Lock()
	defer l.mu.Unlock()
	close(l.started)
}

func (l *lifecycle) stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	select {
	case <-l.stopped:
	default:
		close(l.stopped)
	}
}

func (l *lifecycle) wait() error {
	<-l.stopped
	return nil
}
