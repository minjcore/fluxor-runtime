package concurrency

import (
	"context"
)

// Task represents a unit of work that can be executed
// This abstraction hides goroutine creation and channel operations
type Task interface {
	// Execute performs the task work
	// ctx provides cancellation and timeout support
	Execute(ctx context.Context) error

	// Name returns a human-readable name for the task (for logging/debugging)
	Name() string
}

// TaskFunc is a function type that implements Task
// Allows functions to be used as tasks without creating a struct
type TaskFunc func(ctx context.Context) error

// Execute implements Task interface for TaskFunc
func (f TaskFunc) Execute(ctx context.Context) error {
	return f(ctx)
}

// Name returns a default name for TaskFunc
func (f TaskFunc) Name() string {
	return "TaskFunc"
}

// NamedTask wraps a TaskFunc with a custom name
type NamedTask struct {
	name string
	task TaskFunc
}

// NewNamedTask creates a new NamedTask
func NewNamedTask(name string, task TaskFunc) *NamedTask {
	// Fail-fast: name cannot be empty
	if name == "" {
		failFastIf(true, "task name cannot be empty")
	}
	// Fail-fast: task cannot be nil
	if task == nil {
		failFastIf(true, "task cannot be nil")
	}
	return &NamedTask{
		name: name,
		task: task,
	}
}

// Execute implements Task interface
func (nt *NamedTask) Execute(ctx context.Context) error {
	return nt.task(ctx)
}

// Name returns the task name
func (nt *NamedTask) Name() string {
	return nt.name
}
