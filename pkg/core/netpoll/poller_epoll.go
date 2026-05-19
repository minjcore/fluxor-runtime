//go:build linux

package netpoll

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

const (
	epollTagWakeup = 0
)

type epollPoller struct {
	epfd    int
	wakeR   int
	wakeW   int
	mu      sync.Mutex
	fdMode  map[int]Mode
	events  []unix.EpollEvent
	closed  bool
}

// NewPoller creates a new poller using epoll (Linux).
func NewPoller() (Poller, error) {
	epfd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, err
	}

	fds := make([]int, 2)
	if err := unix.Pipe(fds); err != nil {
		_ = unix.Close(epfd)
		return nil, err
	}
	if err := unix.SetNonblock(fds[0], true); err != nil {
		_ = unix.Close(epfd)
		_ = unix.Close(fds[0])
		_ = unix.Close(fds[1])
		return nil, err
	}
	if err := unix.SetNonblock(fds[1], true); err != nil {
		_ = unix.Close(epfd)
		_ = unix.Close(fds[0])
		_ = unix.Close(fds[1])
		return nil, err
	}

	ev := unix.EpollEvent{
		Events: unix.EPOLLIN,
		Fd:     int32(fds[0]),
	}
	if err := unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, fds[0], &ev); err != nil {
		_ = unix.Close(epfd)
		_ = unix.Close(fds[0])
		_ = unix.Close(fds[1])
		return nil, err
	}

	p := &epollPoller{
		epfd:   epfd,
		wakeR:  fds[0],
		wakeW:  fds[1],
		fdMode: make(map[int]Mode),
		events: make([]unix.EpollEvent, 128),
	}
	return p, nil
}

func (p *epollPoller) Add(fd int, mode Mode) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrClosed
	}

	var events uint32
	if mode&Read != 0 {
		events |= unix.EPOLLIN
	}
	if mode&Write != 0 {
		events |= unix.EPOLLOUT
	}
	events |= unix.EPOLLERR | unix.EPOLLHUP

	ev := unix.EpollEvent{
		Events: events,
		Fd:     int32(fd),
	}
	err := unix.EpollCtl(p.epfd, unix.EPOLL_CTL_MOD, fd, &ev)
	if err == unix.ENOENT {
		err = unix.EpollCtl(p.epfd, unix.EPOLL_CTL_ADD, fd, &ev)
	}
	if err != nil {
		return err
	}
	p.fdMode[fd] = mode
	return nil
}

func (p *epollPoller) Remove(fd int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.fdMode, fd)
	err := unix.EpollCtl(p.epfd, unix.EPOLL_CTL_DEL, fd, nil)
	if err == unix.ENOENT {
		return nil
	}
	return err
}

func (p *epollPoller) Wait(ctx context.Context, timeout time.Duration) ([]Event, error) {
	timeoutMs := int(timeout.Milliseconds())
	if timeoutMs < 0 {
		timeoutMs = 0
	}

	for {
		n, err := unix.EpollWait(p.epfd, p.events, timeoutMs)
		if err != nil {
			if err == unix.EINTR {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				default:
					continue
				}
			}
			return nil, err
		}

		var out []Event
		for i := 0; i < n; i++ {
			ev := &p.events[i]
			fd := int(ev.Fd)

			if fd == p.wakeR {
				// Drain wakeup pipe
				buf := make([]byte, 64)
				for {
					_, err := unix.Read(p.wakeR, buf)
					if err != nil && err != unix.EAGAIN {
						break
					}
					if err == unix.EAGAIN {
						break
					}
				}
				continue
			}

			p.mu.Lock()
			mode := p.fdMode[fd]
			p.mu.Unlock()

			var e Event
			e.FD = fd
			if ev.Events&(unix.EPOLLERR|unix.EPOLLHUP) != 0 {
				e.Err = errFdClosed
			}
			if mode&Read != 0 && (ev.Events&unix.EPOLLIN != 0 || e.Err != nil) {
				e.ReadReady = true
			}
			if mode&Write != 0 && (ev.Events&unix.EPOLLOUT != 0 || e.Err != nil) {
				e.WriteReady = true
			}
			if e.ReadReady || e.WriteReady {
				out = append(out, e)
			}
		}

		if len(out) > 0 {
			return out, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if n == 0 {
				return nil, nil
			}
		}
	}
}

func (p *epollPoller) Wake() error {
	_, err := unix.Write(p.wakeW, []byte{1})
	if err == unix.EAGAIN {
		return nil
	}
	return err
}

func (p *epollPoller) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	_ = p.Wake()
	_ = unix.Close(p.wakeR)
	_ = unix.Close(p.wakeW)
	return unix.Close(p.epfd)
}
