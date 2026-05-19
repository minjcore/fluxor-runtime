package actor

import (
	"context"
	"errors"
	"sync"
	"time"
)

// RestartStrategy defines how the supervisor reacts when a child stops with an error.
type RestartStrategy int

const (
	// OneForOne restarts only the child that failed.
	OneForOne RestartStrategy = iota
	// AllForOne restarts all children when any child fails.
	AllForOne
)

// Runner is a supervised process. When Run returns, the child has stopped.
// If Run returns a non-nil error, the supervisor may restart it according to its strategy.
type Runner interface {
	Run(ctx context.Context) error
}

// ChildSpec defines a supervised child: a name and a Start function that creates a Runner.
type ChildSpec struct {
	Name  string
	Start func(ctx context.Context) (Runner, error)
}

// SupervisorOption configures a Supervisor.
type SupervisorOption func(*Supervisor)

// WithStrategy sets the restart strategy (default: OneForOne).
func WithStrategy(s RestartStrategy) SupervisorOption {
	return func(sv *Supervisor) { sv.strategy = s }
}

// WithMaxRestarts sets the maximum number of restarts allowed in the restart window (default: 5).
// If exceeded, the supervisor stops restarting and leaves failed children down.
func WithMaxRestarts(n int) SupervisorOption {
	return func(sv *Supervisor) { sv.maxRestarts = n }
}

// WithRestartWindow sets the time window for counting restarts (default: 10s).
func WithRestartWindow(d time.Duration) SupervisorOption {
	return func(sv *Supervisor) { sv.restartWindow = d }
}

// Supervisor monitors child Runners and restarts them on failure according to its strategy.
type Supervisor struct {
	name          string
	children      []ChildSpec
	strategy      RestartStrategy
	maxRestarts   int
	restartWindow time.Duration

	mu            sync.Mutex
	childCtx      []context.Context
	childCan      []context.CancelFunc
	restarts      []time.Time
	running       bool
	done          chan struct{}
	stopRequested chan struct{}
}

// NewSupervisor creates a supervisor with the given name and child specs.
func NewSupervisor(name string, children []ChildSpec, opts ...SupervisorOption) *Supervisor {
	sv := &Supervisor{
		name:          name,
		children:      children,
		strategy:      OneForOne,
		maxRestarts:   5,
		restartWindow: 10 * time.Second,
		done:          make(chan struct{}),
		stopRequested: make(chan struct{}),
	}
	for _, o := range opts {
		o(sv)
	}
	return sv
}

// Start starts all children and begins monitoring. When a child's Run returns with an error,
// the supervisor restarts it (or all, for AllForOne) subject to maxRestarts/restartWindow.
// Start returns when ctx is cancelled or after stopping (e.g. after giving up on restarts).
func (sv *Supervisor) Start(ctx context.Context) {
	sv.mu.Lock()
	if sv.running {
		sv.mu.Unlock()
		return
	}
	sv.running = true
	sv.childCtx = make([]context.Context, len(sv.children))
	sv.childCan = make([]context.CancelFunc, len(sv.children))
	sv.restarts = nil
	sv.mu.Unlock()

	defer func() {
		sv.mu.Lock()
		sv.running = false
		close(sv.done)
		sv.done = make(chan struct{})
		sv.stopRequested = make(chan struct{})
		sv.mu.Unlock()
	}()

	type childResult struct {
		index int
		err   error
	}
	resultCh := make(chan childResult, len(sv.children))

	startOne := func(i int) {
		spec := sv.children[i]
		childCtx, cancel := context.WithCancel(ctx)
		sv.mu.Lock()
		sv.childCtx[i] = childCtx
		sv.childCan[i] = cancel
		sv.mu.Unlock()

		runner, err := spec.Start(childCtx)
		if err != nil {
			resultCh <- childResult{index: i, err: err}
			return
		}
		go func() {
			err := runner.Run(childCtx)
			resultCh <- childResult{index: i, err: err}
		}()
	}

	for i := range sv.children {
		startOne(i)
	}

	for {
		select {
		case <-ctx.Done():
			sv.stopChildren()
			return
		case <-sv.stopRequested:
			sv.stopChildren()
			return
		case res := <-resultCh:
			if res.err == nil || errors.Is(res.err, context.Canceled) {
				continue
			}
			if !sv.allowRestart() {
				sv.stopChildren()
				return
			}
			sv.recordRestart()

			switch sv.strategy {
			case OneForOne:
				sv.mu.Lock()
				if sv.childCan[res.index] != nil {
					sv.childCan[res.index]()
					sv.childCan[res.index] = nil
				}
				sv.mu.Unlock()
				startOne(res.index)
			case AllForOne:
				sv.stopChildren()
				for i := range sv.children {
					childCtx, cancel := context.WithCancel(ctx)
					sv.mu.Lock()
					sv.childCtx[i] = childCtx
					sv.childCan[i] = cancel
					sv.mu.Unlock()
					startOne(i)
				}
			}
		}
	}
}

func (sv *Supervisor) allowRestart() bool {
	sv.mu.Lock()
	defer sv.mu.Unlock()
	cutoff := time.Now().Add(-sv.restartWindow)
	n := 0
	for _, t := range sv.restarts {
		if t.After(cutoff) {
			n++
		}
	}
	return n < sv.maxRestarts
}

func (sv *Supervisor) recordRestart() {
	sv.mu.Lock()
	defer sv.mu.Unlock()
	sv.restarts = append(sv.restarts, time.Now())
}

func (sv *Supervisor) stopChildren() {
	sv.mu.Lock()
	defer sv.mu.Unlock()
	for _, cancel := range sv.childCan {
		if cancel != nil {
			cancel()
		}
	}
}

// Stop signals the supervisor to stop, cancels all child contexts, and waits for Start to return.
// It is a no-op if the supervisor is not running.
func (sv *Supervisor) Stop() {
	sv.mu.Lock()
	if !sv.running {
		sv.mu.Unlock()
		return
	}
	select {
	case <-sv.stopRequested:
	default:
		close(sv.stopRequested)
	}
	for _, cancel := range sv.childCan {
		if cancel != nil {
			cancel()
		}
	}
	done := sv.done
	sv.mu.Unlock()
	<-done
}
