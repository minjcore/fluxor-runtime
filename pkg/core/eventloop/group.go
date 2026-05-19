package eventloop

import (
	"context"
	"runtime"
)

// EventLoopGroup manages a pool of event loops (one per CPU core)
type EventLoopGroup struct {
	loops      []*EventLoop
	dispatcher *Dispatcher
	config     EventLoopConfig
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewEventLoopGroup creates a new event loop group
func NewEventLoopGroup(parentCtx context.Context, config EventLoopConfig) (*EventLoopGroup, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(parentCtx)

	// Determine number of loops
	numLoops := config.NumLoops
	if numLoops == 0 {
		numLoops = runtime.GOMAXPROCS(0)
	}

	// Create event loops
	loops := make([]*EventLoop, numLoops)
	for i := 0; i < numLoops; i++ {
		loops[i] = NewEventLoop(i, ctx, config)
	}

	// Create dispatcher
	// Note: FloxID from context is checked at EventBus level before dispatch
	dispatcher := NewDispatcher(loops, DefaultKeyExtractor)

	return &EventLoopGroup{
		loops:      loops,
		dispatcher: dispatcher,
		config:     config,
		ctx:        ctx,
		cancel:     cancel,
	}, nil
}

// Dispatch dispatches an event to the appropriate loop
func (g *EventLoopGroup) Dispatch(ctx context.Context, event *Event) error {
	return g.dispatcher.Dispatch(ctx, event)
}

// DispatchEnvelope dispatches an envelope to the appropriate loop
func (g *EventLoopGroup) DispatchEnvelope(ctx context.Context, env *EnvelopeData) error {
	return g.dispatcher.DispatchEnvelope(ctx, env)
}

// DispatchByKey dispatches an event to a specific loop by key
func (g *EventLoopGroup) DispatchByKey(key string, event *Event) error {
	return g.dispatcher.DispatchByKey(key, event)
}

// Dispatcher returns the dispatcher (for advanced usage)
func (g *EventLoopGroup) Dispatcher() *Dispatcher {
	return g.dispatcher
}

// Stats returns statistics for all loops
func (g *EventLoopGroup) Stats() []LoopMetrics {
	return g.dispatcher.Stats()
}

// Close closes all event loops
func (g *EventLoopGroup) Close() error {
	g.cancel()
	return g.dispatcher.Close()
}
