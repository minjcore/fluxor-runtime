package script

import (
	"context"
	"fmt"

	"github.com/fluxorio/fluxor/pkg/core/concurrency"
)

// Ensure PHPTask implements concurrency.Task interface
var _ concurrency.Task = (*PHPTask)(nil)

// PHPTask is a Task implementation that executes a PHP script.
// It can be submitted directly to a WorkerPool for concurrent execution.
type PHPTask struct {
	executor *PHPExecutor
	script   string
	args     []string
	resultCh chan *ExecutionResult // Optional result channel for async result retrieval
	name     string                // Task name for logging
}

// NewPHPTask creates a new PHP task.
// The task will execute the specified PHP script with the given arguments.
func NewPHPTask(executor *PHPExecutor, script string, args []string) *PHPTask {
	return &PHPTask{
		executor: executor,
		script:   script,
		args:     args,
		name:     fmt.Sprintf("php-task:%s", script),
	}
}

// NewPHPTaskWithResult creates a new PHP task with a result channel.
// The result will be sent to the channel when execution completes.
func NewPHPTaskWithResult(executor *PHPExecutor, script string, args []string, resultCh chan *ExecutionResult) *PHPTask {
	return &PHPTask{
		executor: executor,
		script:   script,
		args:     args,
		resultCh: resultCh,
		name:     fmt.Sprintf("php-task:%s", script),
	}
}

// SetName sets a custom name for the task (for logging/debugging).
func (t *PHPTask) SetName(name string) {
	t.name = name
}

// Execute implements the concurrency.Task interface.
// It executes the PHP script and optionally sends the result to the result channel.
func (t *PHPTask) Execute(ctx context.Context) error {
	result, err := t.executor.ExecuteScript(ctx, t.script, t.args)

	// Send result to channel if provided (non-blocking)
	if t.resultCh != nil {
		select {
		case t.resultCh <- result:
		default:
			// Channel is full or closed, skip sending
		}
	}

	// Return error if execution failed
	if err != nil {
		return fmt.Errorf("PHP script execution failed: %w", err)
	}

	// Optionally treat non-zero exit codes as errors
	if result.ExitCode != 0 {
		return fmt.Errorf("PHP script exited with code %d: %s", result.ExitCode, result.Stderr)
	}

	return nil
}

// Name implements the concurrency.Task interface.
func (t *PHPTask) Name() string {
	return t.name
}

// GetResultChannel returns the result channel (if set).
func (t *PHPTask) GetResultChannel() chan *ExecutionResult {
	return t.resultCh
}

// Script returns the script path.
func (t *PHPTask) Script() string {
	return t.script
}

// Args returns the script arguments.
func (t *PHPTask) Args() []string {
	return t.args
}
