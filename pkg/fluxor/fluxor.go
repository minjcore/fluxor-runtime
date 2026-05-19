package entrypoint

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/google/uuid"
)

// ReactorRuntime provides a reactor-based runtime (alternative to Verticle-based)
// NOTE: This is experimental and uses a different API pattern
type ReactorRuntime struct {
	gocmd       core.GoCMD
	deployments map[string]Reactor // Registry
	mu          sync.RWMutex
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

// Reactor interface for reactor-based components
type Reactor interface {
	OnStart(ctx core.FluxorContext) error
	OnStop() error
}

func New() *ReactorRuntime {
	ctx, cancel := context.WithCancel(context.Background())
	gocmd := core.NewGoCMD(ctx)
	return &ReactorRuntime{
		gocmd:       gocmd,
		deployments: make(map[string]Reactor),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// EventBus returns the event bus
func (r *ReactorRuntime) EventBus() core.EventBus {
	return r.gocmd.EventBus()
}

func (r *ReactorRuntime) Deploy(reactor Reactor, config map[string]any) string {
	id := uuid.New().String()

	// Create FluxorContext using gocmd
	fctx := newFluxorContext(r.ctx, r.gocmd)
	for k, v := range config {
		fctx.SetConfig(k, v)
	}

	r.mu.Lock()
	r.deployments[id] = reactor
	r.mu.Unlock()

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		// Start Reactor
		if err := reactor.OnStart(fctx); err != nil {
			slog.Error("Failed to deploy reactor", "id", id, "error", err)
			r.Undeploy(id)
			return
		}
	}()

	return id
}

func (r *ReactorRuntime) Undeploy(id string) {
	r.mu.Lock()
	reactor, exists := r.deployments[id]
	if !exists {
		r.mu.Unlock()
		return
	}
	delete(r.deployments, id)
	r.mu.Unlock()

	if err := reactor.OnStop(); err != nil {
		slog.Error("Error stopping reactor", "id", id, "error", err)
	}
}

func (r *ReactorRuntime) Shutdown() {
	slog.Info("System shutting down...")
	r.cancel() // Signal context cancellation

	r.mu.Lock()
	for id, reactor := range r.deployments {
		if err := reactor.OnStop(); err != nil {
			slog.Error("Error stopping reactor", "id", id, "error", err)
		}
		delete(r.deployments, id)
	}
	r.mu.Unlock()

	// Wait with Timeout
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("Shutdown graceful complete")
	case <-time.After(5 * time.Second):
		slog.Info("Shutdown timed out")
	}

	if err := r.gocmd.Close(); err != nil {
		slog.Error("Error closing gocmd", "error", err)
	}
}

// newFluxorContext creates a FluxorContext for the ReactorRuntime.
// This is a local implementation since core.newFluxorContext is not exported.
//
// Parameters:
//   - goCtx: the Go context.Context (not FluxorContext)
//   - gocmd: the GoCMD instance
func newFluxorContext(goCtx context.Context, gocmd core.GoCMD) core.FluxorContext {
	return &fluxorContextWrapper{
		goCtx:  goCtx,
		gocmd:  gocmd,
		config: make(map[string]interface{}),
	}
}

// fluxorContextWrapper implements core.FluxorContext for ReactorRuntime
type fluxorContextWrapper struct {
	goCtx  context.Context // renamed from 'ctx' for clarity: this is Go's context.Context
	gocmd  core.GoCMD
	config map[string]interface{}
}

func (c *fluxorContextWrapper) Context() context.Context                { return c.goCtx }
func (c *fluxorContextWrapper) EventBus() core.EventBus                 { return c.gocmd.EventBus() }
func (c *fluxorContextWrapper) GoCMD() core.GoCMD                       { return c.gocmd }
func (c *fluxorContextWrapper) Config() map[string]interface{}          { return c.config }
func (c *fluxorContextWrapper) SetConfig(key string, value interface{}) { c.config[key] = value }
func (c *fluxorContextWrapper) Deploy(verticle core.Verticle) (string, error) {
	return c.gocmd.DeployVerticle(verticle)
}
func (c *fluxorContextWrapper) Undeploy(deploymentID string) error {
	return c.gocmd.UndeployVerticle(deploymentID)
}
