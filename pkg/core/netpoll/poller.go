package netpoll

import (
	"context"
	"errors"
	"time"
)

// ErrNotSupported is returned by NewPoller on unsupported platforms (e.g. Windows).
var ErrNotSupported = errors.New("netpoll: epoll/kqueue not supported on this platform")

// ErrClosed is returned by Add, Remove, Wait, Wake when the poller is already closed.
var ErrClosed = errors.New("netpoll: poller is closed")

// errFdClosed is set on Event.Err when the fd has error or hangup (EPOLLERR/EPOLLHUP or EV_EOF).
var errFdClosed = errors.New("fd error or hangup")

// Mode specifies which I/O events to watch for a file descriptor.
type Mode uint8

const (
	Read  Mode = 1 << 0
	Write Mode = 1 << 1
)

// Event represents a readiness notification for a file descriptor.
type Event struct {
	FD          int
	ReadReady   bool
	WriteReady  bool
	Err         error // non-nil if fd has error or EOF (e.g. connection closed)
}

// Poller is the interface for OS-level I/O multiplexing.
// Implementations use epoll on Linux and kqueue on BSD/macOS.
type Poller interface {
	// Add registers fd for the given mode (Read, Write, or both).
	// The fd should be non-blocking. Add is idempotent for the same fd;
	// calling Add again updates the mode.
	Add(fd int, mode Mode) error

	// Remove unregisters fd from the poller. Safe to call if fd was not added.
	Remove(fd int) error

	// Wait blocks until at least one fd is ready, the context is cancelled, or timeout elapses.
	// Returns a slice of events (read/write ready, or error). The slice may be reused by the next Wait.
	// If ctx is cancelled, Wait returns ctx.Err().
	Wait(ctx context.Context, timeout time.Duration) ([]Event, error)

	// Wake unblocks a concurrent Wait (e.g. for shutdown or to add more fds from another goroutine).
	// Safe to call from any goroutine.
	Wake() error

	// Close releases resources. After Close, Add/Remove/Wait/Wake must not be called.
	Close() error
}
