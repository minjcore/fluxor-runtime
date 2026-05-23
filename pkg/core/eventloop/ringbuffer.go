package eventloop

import (
	"context"
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"
)

// WaitStrategy defines how to wait for new events.
type WaitStrategy int

const (
	BusySpinWait WaitStrategy = 0
	YieldWait    WaitStrategy = 1
	BlockWait    WaitStrategy = 2
)

// seqPadded pads a sequence number to one cache line (64 bytes) to prevent
// false sharing between adjacent available[] slots under multi-producer load.
type seqPadded struct {
	v int64
	_ [7]int64
}

// RingBuffer implements the LMAX Disruptor pattern for lock-free high-throughput messaging.
//
// Publish protocol:
//  1. Next()     — claim a sequence via CAS on cursor
//  2. Publish()  — write pointer into buffer[], then mark available[slot] = seq
//  3. WaitFor()  — consumer polls available[slot] == expected seq
//
// This separates "slot claimed" from "slot ready to read", eliminating the race
// where cursor was advanced but data hadn't been written yet.
type RingBuffer struct {
	buffer     []unsafe.Pointer
	available  []seqPadded // per-slot published sequence; -1 = unpublished
	bufferSize int64
	indexMask  int64

	_      [7]int64
	cursor int64 // highest claimed sequence (atomic)
	_      [7]int64

	_      [7]int64
	gating int64 // highest consumed sequence (atomic)
	_      [7]int64

	waitStrategy WaitStrategy
}

// NewRingBuffer creates a ring buffer. size must be a power of 2.
func NewRingBuffer(size int, wait WaitStrategy) *RingBuffer {
	if size <= 0 || size&(size-1) != 0 {
		panic("ring buffer size must be a positive power of 2")
	}
	avail := make([]seqPadded, size)
	for i := range avail {
		avail[i].v = -1
	}
	return &RingBuffer{
		buffer:       make([]unsafe.Pointer, size),
		available:    avail,
		bufferSize:   int64(size),
		indexMask:    int64(size - 1),
		cursor:       -1,
		gating:       -1,
		waitStrategy: wait,
	}
}

// Next claims the next sequence number for publishing.
// Returns -1 if the buffer is full (non-blocking).
func (rb *RingBuffer) Next() int64 {
	for {
		current := atomic.LoadInt64(&rb.cursor)
		next := current + 1
		if next-rb.bufferSize > atomic.LoadInt64(&rb.gating) {
			return -1
		}
		if atomic.CompareAndSwapInt64(&rb.cursor, current, next) {
			return next
		}
	}
}

// Publish writes event at the given sequence and marks the slot as ready.
func (rb *RingBuffer) Publish(sequence int64, event unsafe.Pointer) {
	slot := sequence & rb.indexMask
	atomic.StorePointer(&rb.buffer[slot], event)
	atomic.StoreInt64(&rb.available[slot].v, sequence)
}

// Get retrieves the event at the given sequence.
func (rb *RingBuffer) Get(sequence int64) unsafe.Pointer {
	return atomic.LoadPointer(&rb.buffer[sequence&rb.indexMask])
}

// WaitFor waits until sequence is published by a producer.
func (rb *RingBuffer) WaitFor(sequence int64, ctx context.Context) (int64, error) {
	switch rb.waitStrategy {
	case YieldWait:
		return rb.waitForYield(sequence, ctx)
	case BlockWait:
		return rb.waitForBlock(sequence, ctx)
	default:
		return rb.waitForBusySpin(sequence, ctx)
	}
}

func (rb *RingBuffer) waitForBusySpin(sequence int64, ctx context.Context) (int64, error) {
	slot := sequence & rb.indexMask
	for {
		if ctx.Err() != nil {
			return -1, ctx.Err()
		}
		if atomic.LoadInt64(&rb.available[slot].v) == sequence {
			return sequence, nil
		}
	}
}

func (rb *RingBuffer) waitForYield(sequence int64, ctx context.Context) (int64, error) {
	slot := sequence & rb.indexMask
	spins := 100
	for {
		if ctx.Err() != nil {
			return -1, ctx.Err()
		}
		if atomic.LoadInt64(&rb.available[slot].v) == sequence {
			return sequence, nil
		}
		spins--
		if spins <= 0 {
			runtime.Gosched()
			spins = 100
		}
	}
}

func (rb *RingBuffer) waitForBlock(sequence int64, ctx context.Context) (int64, error) {
	slot := sequence & rb.indexMask
	for {
		if ctx.Err() != nil {
			return -1, ctx.Err()
		}
		if atomic.LoadInt64(&rb.available[slot].v) == sequence {
			return sequence, nil
		}
		time.Sleep(time.Microsecond)
	}
}

// Consume updates the gating sequence, telling producers slots up to sequence
// have been processed and may be reused.
func (rb *RingBuffer) Consume(sequence int64) {
	atomic.StoreInt64(&rb.gating, sequence)
}

// AvailableCapacity returns the number of slots producers can still claim.
func (rb *RingBuffer) AvailableCapacity() int64 {
	cursor := atomic.LoadInt64(&rb.cursor)
	gating := atomic.LoadInt64(&rb.gating)
	used := cursor - gating
	if used < 0 {
		used = 0
	}
	return rb.bufferSize - used
}

// Size returns the total buffer capacity.
func (rb *RingBuffer) Size() int64 { return rb.bufferSize }

// Metrics returns current utilization stats.
func (rb *RingBuffer) Metrics() RingBufferMetrics {
	cursor := atomic.LoadInt64(&rb.cursor)
	gating := atomic.LoadInt64(&rb.gating)
	used := cursor - gating
	if used < 0 {
		used = 0
	}
	return RingBufferMetrics{
		Cursor:      cursor,
		Gating:      gating,
		Utilization: float64(used) / float64(rb.bufferSize) * 100.0,
		Capacity:    rb.bufferSize,
	}
}

// RingBufferMetrics holds snapshot stats for a RingBuffer.
type RingBufferMetrics struct {
	Cursor      int64
	Gating      int64
	Utilization float64
	Capacity    int64
}
