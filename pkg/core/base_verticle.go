package core

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/concurrency"
)

// BaseVerticle provides a Java-style abstract base class for verticles
// It implements common lifecycle management and provides hook methods for customization
// Similar to Java's AbstractVerticle pattern
// Each verticle has its own event loop for sequential event processing
type BaseVerticle struct {
	// Name of the verticle (can be set by subclasses)
	name string

	// Context reference (set during Start)
	ctx FluxorContext

	// EventBus reference (cached for convenience)
	eventBus EventBus

	// GoCMD reference (cached for convenience) - GoCMD instance
	gocmd GoCMD

	// Event loop for this verticle (sequential processing)
	// Each verticle has its own event loop - events are processed sequentially
	eventLoop concurrency.Executor

	// Worker pool for blocking work (CPU-bound operations)
	// Separated from event loop to ensure event loop never blocks > 20µs
	workerPool concurrency.WorkerPool

	// State management
	mu      sync.RWMutex
	started bool
	stopped bool

	// Consumers registered by this verticle (for cleanup)
	consumers []Consumer
}

// NewBaseVerticle creates a new BaseVerticle
func NewBaseVerticle(name string) *BaseVerticle {
	return &BaseVerticle{
		name:      name,
		consumers: make([]Consumer, 0),
	}
}

// Start implements Verticle.Start
// This is the single entry point - subclasses override Start() directly
// For async operations, use the event loop or implement AsyncVerticle
func (bv *BaseVerticle) Start(ctx FluxorContext) error {
	bv.mu.Lock()
	defer bv.mu.Unlock()

	if bv.started {
		return &EventBusError{Code: "ALREADY_STARTED", Message: "verticle already started"}
	}

	// Set context and references
	bv.ctx = ctx
	bv.eventBus = ctx.EventBus()
	bv.gocmd = ctx.GoCMD()

	// Create event loop for this verticle (1 worker = sequential processing)
	// Each verticle has its own event loop - events are processed sequentially
	//
	// Race-Free Guarantee:
	// - Workers: 1 ensures only one goroutine processes tasks from the queue
	// - Go channels guarantee only one receiver gets each message (serialization)
	// - All tasks execute sequentially in the same goroutine (no concurrent access)
	// - This matches the explicit channel-based event loop pattern:
	//     for {
	//         select {
	//         case task := <-taskChan:
	//             task.Execute()  // Sequential - no race conditions
	//         case <-ctx.Done():
	//             return
	//         }
	//     }
	gocmdCtx := ctx.GoCMD().Context()
	eventLoopConfig := concurrency.ExecutorConfig{
		Workers:   1,    // Single worker = sequential processing (event loop) - ensures race-free operation
		QueueSize: 1000, // Queue size for events
	}
	bv.eventLoop = concurrency.NewExecutor(gocmdCtx, eventLoopConfig)

	// Create worker pool for blocking work (CPU-bound operations)
	// Event loop should only do IO + dispatch (< 20µs per task)
	// Blocking work goes to worker pool
	workerPoolConfig := concurrency.WorkerPoolConfig{
		Workers:   runtime.NumCPU(), // Use CPU cores for worker pool
		QueueSize: 1000,             // Queue size for blocking tasks
	}
	bv.workerPool = concurrency.NewWorkerPool(gocmdCtx, workerPoolConfig)
	if err := bv.workerPool.Start(); err != nil {
		return err
	}

	bv.started = true
	return nil
}

// Stop implements Verticle.Stop with template method pattern
// Subclasses should override doStop() for custom cleanup
func (bv *BaseVerticle) Stop(ctx FluxorContext) error {
	bv.mu.Lock()
	defer bv.mu.Unlock()

	if bv.stopped {
		return nil // Already stopped
	}

	// Call hook method for subclass customization
	if err := bv.doStop(ctx); err != nil {
		return err
	}

	// Cleanup registered consumers
	for _, consumer := range bv.consumers {
		_ = consumer.Unregister()
	}
	bv.consumers = nil

	// Shutdown event loop gracefully
	if bv.eventLoop != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = bv.eventLoop.Shutdown(shutdownCtx)
		cancel()
		bv.eventLoop = nil
	}

	// Shutdown worker pool gracefully
	if bv.workerPool != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = bv.workerPool.Stop(shutdownCtx)
		cancel()
		bv.workerPool = nil
	}

	bv.stopped = true
	return nil
}

// doStart is deprecated - subclasses should override Start() directly
// Kept for backward compatibility
func (bv *BaseVerticle) doStart(ctx FluxorContext) error {
	return nil
}

// doStop is deprecated - subclasses should override Stop() directly
// Kept for backward compatibility
func (bv *BaseVerticle) doStop(ctx FluxorContext) error {
	return nil
}

// ExecuteOn executes a function asynchronously in a goroutine
// This hides goroutine creation from application code, similar to Vert.x patterns
// The function runs in its own goroutine and should handle context cancellation
func (bv *BaseVerticle) ExecuteOn(fn func()) {
	if fn == nil {
		return
	}
	go fn()
}
