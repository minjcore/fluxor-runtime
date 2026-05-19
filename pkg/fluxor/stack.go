package entrypoint

import (
	"context"
	"sync"
)

// stackManager implements StackManager
// This is an abstraction over gostacks package
type stackManager struct {
	stack []Task
	mu    sync.Mutex
}

// newStackManager creates a new stack manager
func newStackManager() StackManager {
	return &stackManager{
		stack: make([]Task, 0),
	}
}

func (sm *stackManager) Push(task Task) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.stack = append(sm.stack, task)
	return nil
}

func (sm *stackManager) Pop() (Task, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if len(sm.stack) == 0 {
		return nil, ErrEmptyStack
	}

	task := sm.stack[len(sm.stack)-1]
	sm.stack = sm.stack[:len(sm.stack)-1]
	return task, nil
}

func (sm *stackManager) Execute(ctx context.Context) error {
	sm.mu.Lock()
	if len(sm.stack) == 0 {
		sm.mu.Unlock()
		return nil
	}

	// Pop task
	task := sm.stack[len(sm.stack)-1]
	sm.stack = sm.stack[:len(sm.stack)-1]
	sm.mu.Unlock()

	// Execute task asynchronously
	go func() {
		if err := task.Execute(ctx); err != nil {
			// Handle error - could publish to event bus
			_ = err
		}
	}()

	return nil
}

func (sm *stackManager) Size() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return len(sm.stack)
}

// ErrEmptyStack is returned when trying to pop from an empty stack
var ErrEmptyStack = &StackError{Message: "stack is empty"}

// StackError represents a stack operation error
type StackError struct {
	Message string
}

func (e *StackError) Error() string {
	return e.Message
}
