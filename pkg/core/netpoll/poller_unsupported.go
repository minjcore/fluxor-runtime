//go:build !linux && !darwin && !freebsd && !netbsd && !openbsd

package netpoll

// NewPoller returns (nil, ErrNotSupported) on platforms without epoll/kqueue (e.g. Windows).
func NewPoller() (Poller, error) {
	return nil, ErrNotSupported
}
