package entrypoint

import (
	"context"
	"fmt"
	"sync"
)

// Workflow represents a reactive workflow
// Inspired by Vert.x patterns, workflows are composed of reactive steps
type Workflow interface {
	// Name returns the workflow name
	Name() string

	// Execute executes the workflow
	Execute(ctx context.Context) error

	// Steps returns all workflow steps
	Steps() []Step
}

// Step represents a single step in a workflow
type Step interface {
	// Execute executes the step
	Execute(ctx context.Context, data interface{}) (interface{}, error)

	// Name returns the step name
	Name() string
}

// workflow implements Workflow
type workflow struct {
	name  string
	steps []Step
	mu    sync.RWMutex
}

// NewWorkflow creates a new workflow
func NewWorkflow(name string, steps ...Step) Workflow {
	return &workflow{
		name:  name,
		steps: steps,
	}
}

func (w *workflow) Name() string {
	return w.name
}

func (w *workflow) Steps() []Step {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.steps
}

func (w *workflow) Execute(ctx context.Context) error {
	var data interface{} = nil

	for _, step := range w.steps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		result, err := step.Execute(ctx, data)
		if err != nil {
			return fmt.Errorf("step %s failed: %w", step.Name(), err)
		}

		data = result

		// After each step, we could publish events to the event bus
		// This enables reactive composition
	}

	return nil
}

// FuncStep is a step implemented as a function
type FuncStep struct {
	name string
	fn   func(ctx context.Context, data interface{}) (interface{}, error)
}

// NewStep creates a new function-based step
func NewStep(name string, fn func(ctx context.Context, data interface{}) (interface{}, error)) Step {
	return &FuncStep{
		name: name,
		fn:   fn,
	}
}

func (s *FuncStep) Name() string {
	return s.name
}

func (s *FuncStep) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	return s.fn(ctx, data)
}
