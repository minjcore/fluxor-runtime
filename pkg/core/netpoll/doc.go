// Package netpoll provides OS-level I/O multiplexing for the Fluxor framework:
// epoll on Linux, kqueue on BSD/macOS. It does not use CGO.
//
// # Build tags
//
// The implementation is selected at compile time by build tags:
//   - Linux: poller_epoll.go (//go:build linux)
//   - Darwin, FreeBSD, NetBSD, OpenBSD: poller_kqueue.go (//go:build darwin || freebsd || netbsd || openbsd)
//   - Other platforms (e.g. Windows): poller_unsupported.go; NewPoller returns ErrNotSupported
//
// # When to use
//
// Use netpoll when you need:
//   - High-throughput TCP listeners (accept + read/write) with a single goroutine instead of one per connection
//   - Integration with pkg/core/eventloop: socket readiness events can be dispatched as events to the loop
//   - Lower syscall overhead than blocking net.Conn per goroutine under heavy load
//
// Typical flow: create a Poller with NewPoller(), add the listener fd with Add(fd, Read),
// then loop: Wait(ctx, timeout) → for each Event with ReadReady on the listener fd, call Accept();
// for connection fds, Add(fd, Read) or Add(fd, Write) and handle read/write in your handler.
// Use Wake() from another goroutine to unblock Wait (e.g. on shutdown).
//
// # Dependencies
//
// Uses golang.org/x/sys/unix for epoll and kqueue syscalls. No CGO required.
package netpoll
