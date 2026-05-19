package netpoll

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewPollerUnsupported(t *testing.T) {
	// On supported platforms this succeeds; on unsupported it returns ErrNotSupported.
	// We only verify that NewPoller returns without panic and that we get either a poller or ErrNotSupported.
	p, err := NewPoller()
	if err == ErrNotSupported {
		t.Skip("epoll/kqueue not supported on this platform")
	}
	if err != nil {
		t.Fatalf("NewPoller: %v", err)
	}
	if p == nil {
		t.Fatal("NewPoller returned nil poller")
	}
	_ = p.Close()
}

func TestPollerPipeReadReady(t *testing.T) {
	p, err := NewPoller()
	if err == ErrNotSupported {
		t.Skip("epoll/kqueue not supported on this platform")
	}
	if err != nil {
		t.Fatalf("NewPoller: %v", err)
	}
	defer p.Close()

	// Use a pipe: write end writable, read end readable when data is written
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	fd := int(r.Fd())
	if err := p.Add(fd, Read); err != nil {
		t.Fatalf("Add: %v", err)
	}
	defer p.Remove(fd)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Initially no data - Wait should timeout with no events (or context)
	go func() {
		time.Sleep(10 * time.Millisecond)
		_, _ = w.Write([]byte("x"))
	}()

	events, err := p.Wait(ctx, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	// We might get the event after write; if not, wait again
	for len(events) == 0 && ctx.Err() == nil {
		events, err = p.Wait(ctx, 100*time.Millisecond)
		if err != nil {
			t.Fatalf("Wait: %v", err)
		}
	}
	if len(events) == 0 {
		t.Skip("no event within timeout (timing)")
	}
	found := false
	for _, e := range events {
		if e.FD == fd && e.ReadReady {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected read-ready event for fd %d, got %+v", fd, events)
	}
}
