//go:build darwin || freebsd || netbsd || openbsd

package netpoll

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

type kqueuePoller struct {
	kq     int
	wakeR  int
	wakeW  int
	mu     sync.Mutex
	fdMode map[int]Mode
	events []unix.Kevent_t
	closed bool
}

// NewPoller creates a new poller using kqueue (BSD/macOS).
func NewPoller() (Poller, error) {
	kq, err := unix.Kqueue()
	if err != nil {
		return nil, err
	}

	fds := make([]int, 2)
	if err := unix.Pipe(fds); err != nil {
		_ = unix.Close(kq)
		return nil, err
	}
	if err := unix.SetNonblock(fds[0], true); err != nil {
		_ = unix.Close(kq)
		_ = unix.Close(fds[0])
		_ = unix.Close(fds[1])
		return nil, err
	}
	if err := unix.SetNonblock(fds[1], true); err != nil {
		_ = unix.Close(kq)
		_ = unix.Close(fds[0])
		_ = unix.Close(fds[1])
		return nil, err
	}

	ev := setKevent(uintptr(fds[0]), unix.EVFILT_READ, unix.EV_ADD|unix.EV_ENABLE)
	_, err = unix.Kevent(kq, []unix.Kevent_t{ev}, nil, nil)
	if err != nil {
		_ = unix.Close(kq)
		_ = unix.Close(fds[0])
		_ = unix.Close(fds[1])
		return nil, err
	}

	p := &kqueuePoller{
		kq:     kq,
		wakeR:  fds[0],
		wakeW:  fds[1],
		fdMode: make(map[int]Mode),
		events: make([]unix.Kevent_t, 128),
	}
	return p, nil
}

func (p *kqueuePoller) Add(fd int, mode Mode) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrClosed
	}

	var changes []unix.Kevent_t
	if mode&Read != 0 {
		changes = append(changes, setKevent(uintptr(fd), unix.EVFILT_READ, unix.EV_ADD|unix.EV_ENABLE))
	}
	if mode&Write != 0 {
		changes = append(changes, setKevent(uintptr(fd), unix.EVFILT_WRITE, unix.EV_ADD|unix.EV_ENABLE))
	}
	if len(changes) == 0 {
		return nil
	}

	_, err := unix.Kevent(p.kq, changes, nil, nil)
	if err != nil {
		return err
	}
	p.fdMode[fd] = mode
	return nil
}

func (p *kqueuePoller) Remove(fd int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.fdMode, fd)
	changes := []unix.Kevent_t{
		setKevent(uintptr(fd), unix.EVFILT_READ, unix.EV_DELETE),
		setKevent(uintptr(fd), unix.EVFILT_WRITE, unix.EV_DELETE),
	}
	_, _ = unix.Kevent(p.kq, changes, nil, nil)
	return nil
}

func (p *kqueuePoller) Wait(ctx context.Context, timeout time.Duration) ([]Event, error) {
	var tsp *unix.Timespec
	if timeout > 0 {
		ts := unix.NsecToTimespec(timeout.Nanoseconds())
		tsp = &ts
	}

	for {
		n, err := unix.Kevent(p.kq, nil, p.events, tsp)
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
		seen := make(map[int]*Event)
		for i := 0; i < n; i++ {
			ev := &p.events[i]
			fd := int(ev.Ident)

			if fd == p.wakeR {
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
			fdMode := p.fdMode[fd]
			p.mu.Unlock()

			e, ok := seen[fd]
			if !ok {
				e = &Event{FD: fd}
				seen[fd] = e
			}
			if ev.Flags&unix.EV_EOF != 0 {
				e.Err = errFdClosed
			}
			switch ev.Filter {
			case unix.EVFILT_READ:
				e.ReadReady = (fdMode&Read != 0)
			case unix.EVFILT_WRITE:
				e.WriteReady = (fdMode&Write != 0)
			}
		}

		for _, e := range seen {
			if e.ReadReady || e.WriteReady || e.Err != nil {
				out = append(out, *e)
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

func (p *kqueuePoller) Wake() error {
	_, err := unix.Write(p.wakeW, []byte{1})
	if err == unix.EAGAIN {
		return nil
	}
	return err
}

func (p *kqueuePoller) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	_ = p.Wake()
	_ = unix.Close(p.wakeR)
	_ = unix.Close(p.wakeW)
	return unix.Close(p.kq)
}

// setKevent builds a Kevent_t for (ident, filter, flags). Portable across Darwin/BSD.
func setKevent(ident uintptr, filter, flags int) unix.Kevent_t {
	return unix.Kevent_t{
		Ident:  uint64(ident),
		Filter: int16(filter),
		Flags:  uint16(flags),
	}
}
