package entrypoint

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/core"
)

// Runtime provides a reactive runtime abstraction over gostacks
// It manages the execution of reactive workflows and tasks
type Runtime interface {
	// Start starts the runtime
	Start(ctx context.Context) error

	// Stop stops the runtime
	Stop() error

	// Execute executes a task/workflow
	Execute(task Task) error

	// Deploy deploys a verticle
	Deploy(verticle core.Verticle) (string, error)

	// GoCMD returns the underlying GoCMD instance (kept as GoCMD for backward compatibility)
	GoCMD() core.GoCMD
}

// Task represents a unit of work that can be executed
type Task interface {
	// Execute executes the task
	Execute(ctx context.Context) error

	// Name returns the task name
	Name() string
}

// runtime implements Runtime
type runtime struct {
	gocmd  core.GoCMD
	tasks  []Task
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	stack  StackManager
}

// StackManager manages execution stacks (abstraction over gostacks)
type StackManager interface {
	// Push pushes a task onto the stack
	Push(task Task) error

	// Pop pops a task from the stack
	Pop() (Task, error)

	// Execute executes tasks in the stack
	Execute(ctx context.Context) error

	// Size returns the current stack size
	Size() int
}

// NewRuntime creates a new runtime instance
func NewRuntime(ctx context.Context) Runtime {
	ctx, cancel := context.WithCancel(ctx)
	gocmd := core.NewGoCMD(ctx)

	return &runtime{
		gocmd:  gocmd,
		tasks:  make([]Task, 0),
		ctx:    ctx,
		cancel: cancel,
		stack:  newStackManager(),
	}
}

// NewVerticleRuntime creates a new runtime instance with background context
// This is a convenience function for simple verticle deployments
func NewVerticleRuntime() Runtime {
	return NewRuntime(context.Background())
}

func (r *runtime) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Start processing tasks
	go r.processTasks()

	return nil
}

func (r *runtime) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cancel()
	return r.gocmd.Close()
}

func (r *runtime) Execute(task Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.stack.Push(task)
}

func (r *runtime) Deploy(verticle core.Verticle) (string, error) {
	return r.gocmd.DeployVerticle(verticle)
}

func (r *runtime) GoCMD() core.GoCMD {
	return r.gocmd
}

func (r *runtime) processTasks() {
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
			if err := r.stack.Execute(r.ctx); err != nil {
				// Log error in production
				_ = err
			}
		}
	}
}
