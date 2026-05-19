package kernel

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// State is kernel lifecycle.
type State int

const (
	// StateStopped — initial or after successful Stop(); Start() is allowed.
	StateStopped State = iota
	StateStarting
	StateRunning
	StateStopping
)

// Kernel coordinates components under one root context (in-process “platform”).
type Kernel interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Register(c Component) error
	Context() context.Context
	State() State
	Errors() <-chan error
	// EmitError sends a non-fatal runtime error to the async channel (best-effort; needs errBuf > 0 in NewKernel).
	EmitError(err error)
}

type kernel struct {
	mu         sync.Mutex
	state      State
	components []Component
	ctx        context.Context
	cancel     context.CancelFunc

	errCh   chan error
	errOnce sync.Once
}

// NewKernel creates a kernel. errBuf is async error channel buffer (0 = unbuffered, disables non-blocking send).
func NewKernel(errBuf int) Kernel {
	k := &kernel{state: StateStopped}
	if errBuf > 0 {
		k.errCh = make(chan error, errBuf)
	}
	return k
}

func (k *kernel) Register(c Component) error {
	if c == nil {
		return errors.New("kernel: nil component")
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.state != StateStopped {
		return fmt.Errorf("kernel: Register only when stopped (state=%v)", k.state)
	}
	k.components = append(k.components, c)
	return nil
}

func (k *kernel) Start(ctx context.Context) error {
	k.mu.Lock()
	if k.state != StateStopped {
		k.mu.Unlock()
		return fmt.Errorf("kernel: already active (state=%v)", k.state)
	}
	k.state = StateStarting
	parent := ctx
	if parent == nil {
		parent = context.Background()
	}
	k.ctx, k.cancel = context.WithCancel(parent)
	comps := append([]Component(nil), k.components...)
	k.mu.Unlock()

	for i, c := range comps {
		if err := c.Start(k.ctx); err != nil {
			for j := i - 1; j >= 0; j-- {
				_ = comps[j].Stop(context.Background())
			}
			k.mu.Lock()
			if k.cancel != nil {
				k.cancel()
			}
			k.ctx = nil
			k.cancel = nil
			k.state = StateStopped
			k.mu.Unlock()
			return fmt.Errorf("kernel: start %s: %w", c.Name(), err)
		}
	}

	k.mu.Lock()
	k.state = StateRunning
	k.mu.Unlock()
	return nil
}

func (k *kernel) Stop(ctx context.Context) error {
	k.mu.Lock()
	if k.state != StateRunning {
		k.mu.Unlock()
		return nil
	}
	k.state = StateStopping
	comps := append([]Component(nil), k.components...)
	cancel := k.cancel
	k.mu.Unlock()

	stopCtx := ctx
	if stopCtx == nil {
		stopCtx = context.Background()
	}
	for i := len(comps) - 1; i >= 0; i-- {
		_ = comps[i].Stop(stopCtx)
	}
	if cancel != nil {
		cancel()
	}

	k.mu.Lock()
	k.ctx = nil
	k.cancel = nil
	k.state = StateStopped
	k.mu.Unlock()

	k.errOnce.Do(func() {
		if k.errCh != nil {
			close(k.errCh)
		}
	})
	return nil
}

func (k *kernel) Context() context.Context {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.ctx != nil {
		return k.ctx
	}
	return context.Background()
}

func (k *kernel) State() State {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.state
}

// Errors returns the async error channel, or nil if NewKernel(0).
func (k *kernel) Errors() <-chan error {
	return k.errCh
}

func (k *kernel) EmitError(err error) {
	if err == nil || k.errCh == nil {
		return
	}
	k.mu.Lock()
	ok := k.state == StateRunning
	k.mu.Unlock()
	if !ok {
		return
	}
	select {
	case k.errCh <- err:
	default:
	}
}
