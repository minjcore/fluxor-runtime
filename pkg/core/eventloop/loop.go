package eventloop

import (
	"context"
	"runtime"
	"sync/atomic"
	"time"
)

// EventLoop represents a single event loop running on one goroutine
type EventLoop struct {
	id      int
	queue   chan *Event
	ctx     context.Context
	cancel  context.CancelFunc
	config  EventLoopConfig
	metrics *LoopMetrics
	closed  int32
}

// Event represents a message to be processed
type Event struct {
	Key     string
	Address string
	Body    interface{}
	Headers map[string]string
	Handler func(ctx context.Context, event *Event) error
}

// NewEventLoop creates a new event loop
func NewEventLoop(id int, parentCtx context.Context, config EventLoopConfig) *EventLoop {
	ctx, cancel := context.WithCancel(parentCtx)

	loop := &EventLoop{
		id:     id,
		queue:  make(chan *Event, config.QueueSize),
		ctx:    ctx,
		cancel: cancel,
		config: config,
		metrics: &LoopMetrics{
			LoopID: id,
		},
	}

	// Start the loop goroutine
	go loop.run()

	return loop
}

// run is the main event loop goroutine
func (l *EventLoop) run() {
	// Optional CPU affinity: pin goroutine to CPU core
	if l.config.CPUAffinity {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	for {
		select {
		case event, ok := <-l.queue:
			if !ok {
				return // Channel closed
			}

			// Update queue length
			if l.config.Metrics {
				atomic.AddInt64(&l.metrics.QueueLength, -1)
			}

			// Process event
			startTime := time.Now()
			if event.Handler != nil {
				if err := event.Handler(l.ctx, event); err != nil {
					// Log error but don't crash
					// Error handling is up to the handler
				}
			}

			// Update metrics
			if l.config.Metrics {
				latency := time.Since(startTime)
				atomic.AddInt64(&l.metrics.ProcessedMessages, 1)
				// EWMA (α=0.125): new = old*7/8 + sample/8
				old := atomic.LoadInt64((*int64)(&l.metrics.AvgLatency))
				var updated int64
				if old == 0 {
					updated = int64(latency)
				} else {
					updated = (old*7 + int64(latency)) / 8
				}
				atomic.StoreInt64((*int64)(&l.metrics.AvgLatency), updated)
			}

		case <-l.ctx.Done():
			return // Context cancelled
		}
	}
}

// Dispatch dispatches an event to this loop
func (l *EventLoop) Dispatch(event *Event) error {
	if atomic.LoadInt32(&l.closed) == 1 {
		return &EventLoopError{Code: "CLOSED", Message: "event loop is closed"}
	}

	// Update queue length
	if l.config.Metrics {
		atomic.AddInt64(&l.metrics.QueueLength, 1)
	}

	// Handle backpressure
	switch l.config.Backpressure {
	case BackpressureBlock:
		// Block until queue has space or context cancelled
		select {
		case l.queue <- event:
			return nil
		case <-l.ctx.Done():
			return l.ctx.Err()
		}

	case BackpressureDrop:
		// Non-blocking: drop if queue is full
		select {
		case l.queue <- event:
			return nil
		default:
			// Queue full, drop message
			if l.config.Metrics {
				atomic.AddInt64(&l.metrics.DroppedMessages, 1)
			}
			return &EventLoopError{Code: "QUEUE_FULL", Message: "queue is full, message dropped"}
		}

	default:
		// Default to block
		select {
		case l.queue <- event:
			return nil
		case <-l.ctx.Done():
			return l.ctx.Err()
		}
	}
}

// Close closes the event loop
func (l *EventLoop) Close() error {
	if !atomic.CompareAndSwapInt32(&l.closed, 0, 1) {
		return nil // Already closed
	}

	// Cancel context to stop the loop
	l.cancel()

	// Close queue channel
	close(l.queue)

	return nil
}

// Stats returns current loop statistics
func (l *EventLoop) Stats() LoopMetrics {
	if !l.config.Metrics {
		return LoopMetrics{LoopID: l.id}
	}

	return LoopMetrics{
		LoopID:            l.id,
		QueueLength:       atomic.LoadInt64(&l.metrics.QueueLength),
		DroppedMessages:   atomic.LoadInt64(&l.metrics.DroppedMessages),
		ProcessedMessages: atomic.LoadInt64(&l.metrics.ProcessedMessages),
		AvgLatency:        time.Duration(atomic.LoadInt64((*int64)(&l.metrics.AvgLatency))),
		P50Latency:        l.metrics.P50Latency,
		P95Latency:        l.metrics.P95Latency,
		P99Latency:        l.metrics.P99Latency,
		Throughput:        l.metrics.Throughput,
	}
}
