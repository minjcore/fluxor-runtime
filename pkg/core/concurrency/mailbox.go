package concurrency

import (
	"context"
	"errors"
)

var (
	// ErrMailboxClosed is returned when trying to send/receive on a closed mailbox
	ErrMailboxClosed = errors.New("mailbox is closed")

	// ErrMailboxFull is returned when trying to send to a full mailbox (backpressure)
	ErrMailboxFull = errors.New("mailbox is full")

	// ErrMailboxEmpty is returned when trying to receive from an empty mailbox (non-blocking)
	ErrMailboxEmpty = errors.New("mailbox is empty")
)

// Mailbox abstracts channel operations behind a message passing API
// Hides chan type and select statements from application code
type Mailbox interface {
	// Send sends a message to the mailbox
	// Returns ErrMailboxFull if mailbox is full (backpressure)
	// Returns ErrMailboxClosed if mailbox is closed
	Send(msg interface{}) error

	// Receive receives a message from the mailbox
	// Blocks until a message is available or ctx is cancelled
	// Returns ErrMailboxClosed if mailbox is closed
	Receive(ctx context.Context) (interface{}, error)

	// TryReceive attempts to receive a message without blocking
	// Returns (msg, true) if message available, (nil, false) if empty
	// Returns ErrMailboxClosed if mailbox is closed
	TryReceive() (interface{}, bool, error)

	// Close closes the mailbox
	// After closing, Send/Receive operations will return ErrMailboxClosed
	Close()

	// Capacity returns the maximum capacity of the mailbox
	Capacity() int

	// Size returns the current number of messages in the mailbox
	Size() int

	// IsClosed returns true if the mailbox is closed
	IsClosed() bool
}
