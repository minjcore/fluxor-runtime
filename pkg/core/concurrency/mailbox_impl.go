package concurrency

import (
	"context"
	"sync/atomic"
)

// boundedMailbox implements Mailbox using channels internally
// Hides chan type and select statements from public API
type boundedMailbox struct {
	ch       chan interface{} // Hidden: internal channel
	closed   int32            // Atomic flag for thread-safe close check
	capacity int
	size     int64 // Atomic counter for accurate size tracking
}

// NewBoundedMailbox creates a new bounded mailbox
// Hides channel creation from callers
func NewBoundedMailbox(capacity int) Mailbox {
	// Fail-fast: capacity must be positive
	if capacity <= 0 {
		failFastIf(true, "mailbox capacity must be positive")
	}

	return &boundedMailbox{
		ch:       make(chan interface{}, capacity), // Hidden: channel creation
		capacity: capacity,
	}
}

// Send implements Mailbox interface
// Hides channel send and select statements
func (mb *boundedMailbox) Send(msg interface{}) error {
	if atomic.LoadInt32(&mb.closed) == 1 {
		return ErrMailboxClosed
	}

	// Try to send (non-blocking for backpressure)
	select {
	case mb.ch <- msg: // Hidden: channel send
		atomic.AddInt64(&mb.size, 1)
		return nil
	default:
		// Mailbox full - backpressure
		return ErrMailboxFull
	}
}

// Receive implements Mailbox interface
// Hides channel receive and select statements
func (mb *boundedMailbox) Receive(ctx context.Context) (interface{}, error) {
	// Fail-fast: context cannot be nil
	if ctx == nil {
		failFastIf(true, "context cannot be nil")
	}
	if atomic.LoadInt32(&mb.closed) == 1 {
		return nil, ErrMailboxClosed
	}

	// Receive with context cancellation
	select {
	case msg, ok := <-mb.ch: // Hidden: channel receive
		if !ok {
			return nil, ErrMailboxClosed
		}
		atomic.AddInt64(&mb.size, -1)
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// TryReceive implements Mailbox interface
// Hides channel receive and select statements
func (mb *boundedMailbox) TryReceive() (interface{}, bool, error) {
	if atomic.LoadInt32(&mb.closed) == 1 {
		return nil, false, ErrMailboxClosed
	}

	// Try to receive (non-blocking)
	select {
	case msg, ok := <-mb.ch: // Hidden: channel receive
		if !ok {
			return nil, false, ErrMailboxClosed
		}
		atomic.AddInt64(&mb.size, -1)
		return msg, true, nil
	default:
		// Mailbox empty
		return nil, false, nil
	}
}

// Close implements Mailbox interface
// Hides channel close operation
func (mb *boundedMailbox) Close() {
	if atomic.CompareAndSwapInt32(&mb.closed, 0, 1) {
		close(mb.ch) // Hidden: channel close
	}
}

// Capacity implements Mailbox interface
func (mb *boundedMailbox) Capacity() int {
	return mb.capacity
}

// Size implements Mailbox interface
// Optimized: Uses atomic counter for accurate size tracking under concurrency
func (mb *boundedMailbox) Size() int {
	return int(atomic.LoadInt64(&mb.size))
}

// IsClosed implements Mailbox interface
func (mb *boundedMailbox) IsClosed() bool {
	return atomic.LoadInt32(&mb.closed) == 1
}
