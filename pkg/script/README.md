# Script Package

The `script` package provides utilities for executing external scripts (currently PHP) with proper context cancellation, output capture, and timeout handling.

## Features

- **Context Cancellation**: Supports Go context for cancellation and timeouts
- **Output Capture**: Captures both stdout and stderr
- **Security**: Validates script paths to prevent directory traversal attacks
- **WorkerPool Integration**: Provides Task implementation for concurrent execution
- **Workflow Integration**: Can be used as a workflow function node

## PHP Executor

### Basic Usage

```go
import "github.com/fluxorio/fluxor/pkg/script"

// Create executor with defaults
executor := script.NewPHPExecutor()

// Configure options
executor.SetTimeout(30 * time.Second)
executor.SetWorkingDir("/tmp")
executor.SetPHPPath("/usr/bin/php")
executor.SetEnvironment([]string{"ENV_VAR=value"})

// Execute script
ctx := context.Background()
result, err := executor.ExecuteScript(ctx, "script.php", []string{"arg1", "arg2"})

if err != nil {
    log.Printf("Error: %v", err)
    return
}

fmt.Printf("Exit Code: %d\n", result.ExitCode)
fmt.Printf("Stdout: %s\n", result.Stdout)
fmt.Printf("Stderr: %s\n", result.Stderr)
fmt.Printf("Duration: %v\n", result.Duration)
```

### Configuration Options

- **PHPPath**: Path to PHP executable (default: "php")
- **Timeout**: Execution timeout (default: 30 seconds, 0 = no timeout)
- **WorkingDir**: Working directory for script execution
- **Environment**: Environment variables in "KEY=VALUE" format

## PHP Task (WorkerPool Integration)

### Basic Usage

```go
import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core/concurrency"
    "github.com/fluxorio/fluxor/pkg/script"
)

// Create context and worker pool
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

config := concurrency.DefaultWorkerPoolConfig()
pool := concurrency.NewWorkerPool(ctx, config)
pool.Start()
defer pool.Stop(context.Background())

// Create executor
executor := script.NewPHPExecutor()

// Create and submit task
task := script.NewPHPTask(executor, "script.php", []string{"arg1", "arg2"})
task.SetName("my-php-task")

if err := pool.Submit(task); err != nil {
    log.Printf("Failed to submit task: %v", err)
}
```

### Task with Result Channel

```go
// Create result channel
resultCh := make(chan *script.ExecutionResult, 1)

// Create task with result channel
task := script.NewPHPTaskWithResult(executor, "script.php", []string{"arg1"}, resultCh)

// Submit task
pool.Submit(task)

// Wait for result
select {
case result := <-resultCh:
    fmt.Printf("Exit Code: %d\n", result.ExitCode)
    fmt.Printf("Stdout: %s\n", result.Stdout)
case <-time.After(5 * time.Second):
    fmt.Println("Timeout")
}
```

## Workflow Integration

The PHP executor can be used in workflows by registering the `executePHPScript` function:

```go
// Register function
wfVerticle.RegisterFunction("executePHP", executePHPScript)
```

### Workflow Node Configuration

```json
{
  "id": "php-script-node",
  "type": "function",
  "config": {
    "function": "executePHP"
  },
  "next": ["next-node"]
}
```

### Input Data Format

```json
{
  "script": "/path/to/script.php",
  "args": ["arg1", "arg2"],
  "timeout": "30s",
  "workingDir": "/optional/path",
  "phpPath": "/usr/bin/php",
  "environment": ["KEY=VALUE"]
}
```

### Output Format

```json
{
  "exitCode": 0,
  "stdout": "script output",
  "stderr": "",
  "duration": "1.234s",
  "scriptPath": "/path/to/script.php",
  "args": ["arg1", "arg2"],
  "success": true
}
```

## Security Considerations

1. **Path Validation**: Script paths are validated to prevent directory traversal attacks
2. **Argument Sanitization**: Arguments are passed directly to the PHP process - ensure they are properly sanitized
3. **Sandboxing**: For untrusted scripts, consider running in a sandboxed environment
4. **Timeout**: Always set appropriate timeouts to prevent hanging scripts
5. **Working Directory**: Set working directory to limit script access to specific paths

## Error Handling

The executor handles various error conditions:

- **Script Not Found**: Returns error if script file doesn't exist
- **Invalid Path**: Returns error for directory traversal attempts
- **Execution Timeout**: Returns error if execution exceeds timeout
- **Non-Zero Exit Code**: Returns error if script exits with non-zero code (configurable in Task)

## Examples

See `examples/workflow-demo/php_example.go` for complete usage examples including:
- Direct execution
- WorkerPool submission
- Workflow integration

