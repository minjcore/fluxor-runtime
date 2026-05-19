package eventloop

import (
	"context"
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/fluxorio/fluxor/pkg/hft/protocol"
)

// EventLoop represents a single event loop running on one goroutine.
// The internal queue is a lock-free LMAX Disruptor RingBuffer.
type EventLoop struct {
	id         int
	ringBuffer *protocol.RingBuffer
	ctx        context.Context
	cancel     context.CancelFunc
	config     EventLoopConfig
	metrics    *LoopMetrics
	closed     int32
}

// Event represents a message to be processed
type Event struct {
	Key     string
	Address string
	Body    interface{}
	Headers map[string]string
	Handler func(ctx context.Context, event *Event) error
}

// nextPow2 rounds n up to the nearest power of 2 (minimum 2).
func nextPow2(n int) int {
	if n < 2 {
		return 2
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++
	return n
}

// NewEventLoop creates a new event loop backed by a Disruptor RingBuffer.
// config.QueueSize is rounded up to the next power of 2 as required by the ring buffer.
func NewEventLoop(id int, parentCtx context.Context, config EventLoopConfig) *EventLoop {
	ctx, cancel := context.WithCancel(parentCtx)

	size := nextPow2(config.QueueSize)
	rb := protocol.NewRingBuffer(size, protocol.YieldWait)

	loop := &EventLoop{
		id:         id,
		ringBuffer: rb,
		ctx:        ctx,
		cancel:     cancel,
		config:     config,
		metrics: &LoopMetrics{
			LoopID: id,
		},
	}

	go loop.run()
	return loop
}

// run is the main event loop goroutine.
func (l *EventLoop) run() {
	if l.config.CPUAffinity {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	next := int64(0)
	for {
		// WaitFor returns when sequence `next` (or higher) is published.
		// Returns error only on context cancellation.
		avail, err := l.ringBuffer.WaitFor(next, l.ctx)
		if err != nil {
			return
		}

		// Process all events from `next` to `avail`, consuming each slot
		// immediately after handling so producers can reclaim capacity.
		// Consuming per-event (not per-batch) means a blocking handler does
		// not starve the producer beyond the single occupied slot.
		for seq := next; seq <= avail; seq++ {
			event := (*Event)(l.ringBuffer.Get(seq))

			startTime := time.Now()
			if event != nil && event.Handler != nil {
				_ = event.Handler(l.ctx, event)
			}

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

			l.ringBuffer.Consume(seq)
		}

		next = avail + 1
	}
}

// Dispatch sends an event to this loop's ring buffer.
//
// BackpressureDrop:  returns QUEUE_FULL error immediately if the ring is full.
// BackpressureBlock: spins (with Gosched yields) until a slot is free or the
// context is cancelled — equivalent to blocking on a buffered channel.
func (l *EventLoop) Dispatch(event *Event) error {
	if atomic.LoadInt32(&l.closed) == 1 {
		return &EventLoopError{Code: "CLOSED", Message: "event loop is closed"}
	}

	ptr := unsafe.Pointer(event)

	switch l.config.Backpressure {
	case BackpressureDrop:
		seq := l.ringBuffer.Next()
		if seq < 0 {
			if l.config.Metrics {
				atomic.AddInt64(&l.metrics.DroppedMessages, 1)
			}
			return &EventLoopError{Code: "QUEUE_FULL", Message: "queue is full, message dropped"}
		}
		l.ringBuffer.Publish(seq, ptr)

	default: // BackpressureBlock
		for {
			if atomic.LoadInt32(&l.closed) == 1 || l.ctx.Err() != nil {
				return l.ctx.Err()
			}
			seq := l.ringBuffer.Next()
			if seq >= 0 {
				l.ringBuffer.Publish(seq, ptr)
				break
			}
			runtime.Gosched()
		}
	}

	return nil
}

// Close stops the event loop. Idempotent.
func (l *EventLoop) Close() error {
	if !atomic.CompareAndSwapInt32(&l.closed, 0, 1) {
		return nil
	}
	l.cancel()
	return nil
}

// Stats returns current loop statistics.
func (l *EventLoop) Stats() LoopMetrics {
	if !l.config.Metrics {
		return LoopMetrics{LoopID: l.id}
	}

	queueLen := l.ringBuffer.Size() - l.ringBuffer.AvailableCapacity()

	return LoopMetrics{
		LoopID:            l.id,
		QueueLength:       queueLen,
		DroppedMessages:   atomic.LoadInt64(&l.metrics.DroppedMessages),
		ProcessedMessages: atomic.LoadInt64(&l.metrics.ProcessedMessages),
		AvgLatency:        time.Duration(atomic.LoadInt64((*int64)(&l.metrics.AvgLatency))),
		P50Latency:        l.metrics.P50Latency,
		P95Latency:        l.metrics.P95Latency,
		P99Latency:        l.metrics.P99Latency,
		Throughput:        l.metrics.Throughput,
	}
}
