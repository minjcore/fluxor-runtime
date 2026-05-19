# Scheduler Package
# Build
go build ./pkg/scheduler

# Test
go test ./pkg/scheduler

# Test với coverage
go test -cover ./pkg/scheduler

# Install (copy binary to $GOPATH/bin)
go install ./pkg/scheduler

# Check dependencies
go mod verify

# Format code
go fmt ./pkg/scheduler
A generic, high-performance task scheduling package for the Fluxor framework with support for cron-style scheduling, periodic execution, and delay-based scheduling.

## Features

- **Cron Expression Support**: Parse and execute tasks using standard cron expressions (5-field format)
- **Periodic Scheduling**: Execute tasks at fixed intervals (every N seconds/minutes/hours)
- **Delay Scheduling**: Execute tasks once after a specified delay
- **Thread-Safe**: All operations are safe for concurrent use
- **Context Support**: All operations accept `context.Context` for cancellation
- **Fail-Fast Validation**: All inputs are validated immediately with clear error messages
- **Interface-First Design**: Clean interfaces that hide implementation details

## Quick Start

### Basic Usage

```go
import (
    "context"
    "time"
    schedpkg "github.com/fluxorio/fluxor/pkg/scheduler"
    "github.com/fluxorio/fluxor/pkg/core/concurrency"
)

// Create a scheduler
ctx := context.Background()
scheduler := schedpkg.NewScheduler(ctx)

// Define a task
task := concurrency.NewNamedTask("my-task", func(ctx context.Context) error {
    // Your task logic here
    return nil
})

// Schedule with cron expression (daily at midnight)
id, err := scheduler.Schedule(task, schedpkg.Cron("0 0 * * *"))
if err != nil {
    log.Fatal(err)
}

// Start the scheduler
err = scheduler.Start(ctx)
if err != nil {
    log.Fatal(err)
}

// Later, stop the scheduler
defer scheduler.Stop()
```

### Cron Expressions

Cron expressions use the standard 5-field format: `minute hour day month weekday`

```go
// Every minute
scheduler.Cron("* * * * *")

// Daily at midnight
scheduler.Cron("0 0 * * *")

// Every 5 minutes
scheduler.Cron("*/5 * * * *")

// Every hour at minute 30
scheduler.Cron("30 * * * *")

// Every day at 9 AM
scheduler.Cron("0 9 * * *")

// Every Monday at 9 AM
scheduler.Cron("0 9 * * 1")

// Range: Every hour from 9 AM to 5 PM
scheduler.Cron("0 9-17 * * *")

// List: At 9 AM and 5 PM
scheduler.Cron("0 9,17 * * *")

// Step: Every 10 minutes
scheduler.Cron("*/10 * * * *")
```

**Field Ranges:**
- `minute`: 0-59
- `hour`: 0-23
- `day`: 1-31
- `month`: 1-12
- `weekday`: 0-7 (0 and 7 are Sunday)

**Special Characters:**
- `*` - Any value
- `,` - Value list separator
- `-` - Range
- `/` - Step values

### Periodic Scheduling

Execute tasks at fixed intervals:

```go
// Every 5 minutes
spec := scheduler.Interval(5 * time.Minute)

// Every hour
spec := scheduler.Interval(1 * time.Hour)

// Every 30 seconds
spec := scheduler.Interval(30 * time.Second)

id, err := scheduler.Schedule(task, spec)
```

### Delay Scheduling

Execute a task once after a delay:

```go
// Execute after 1 hour
spec := scheduler.Delay(1 * time.Hour)

// Execute after 30 minutes
spec := scheduler.Delay(30 * time.Minute)

id, err := scheduler.Schedule(task, spec)
```

### Managing Scheduled Tasks

```go
// List all scheduled tasks
tasks := scheduler.List()
for _, task := range tasks {
    fmt.Printf("Task %s: next run at %v\n", task.ID, task.NextRun)
}

// Unschedule a task
err := scheduler.Unschedule(id)
if err != nil {
    log.Printf("Failed to unschedule: %v", err)
}
```

## API Reference

### Scheduler Interface

```go
type Scheduler interface {
    // Schedule schedules a task with the given schedule specification.
    // Returns a unique ID for the scheduled task.
    Schedule(task concurrency.Task, spec ScheduleSpec) (string, error)

    // Unschedule removes a scheduled task by its ID.
    Unschedule(id string) error

    // Start starts the scheduler and begins executing scheduled tasks.
    Start(ctx context.Context) error

    // Stop stops the scheduler and cancels all scheduled tasks.
    Stop() error

    // List returns all currently scheduled tasks.
    List() []ScheduledTask
}
```

### ScheduleSpec Types

#### CronSpec

```go
// Cron creates a cron-style schedule specification
func Cron(expression string) ScheduleSpec
```

#### IntervalSpec

```go
// Interval creates a fixed interval schedule specification
func Interval(interval time.Duration) ScheduleSpec
```

#### DelaySpec

```go
// Delay creates a one-time delay schedule specification
func Delay(delay time.Duration) ScheduleSpec
```

### ScheduledTask

```go
type ScheduledTask struct {
    ID       string              // Unique identifier
    Task     concurrency.Task    // The task to execute
    Spec     ScheduleSpec        // Schedule specification
    NextRun  time.Time           // Next scheduled execution time
    LastRun  time.Time           // Last execution time (zero if not executed)
    RunCount int64               // Number of executions
}
```

## Fail-Fast Principles

All operations follow fail-fast principles:

- **Input Validation**: All inputs are validated immediately
- **Nil Checks**: Context and task instances are checked for nil
- **Clear Errors**: Error messages include context about what failed

Example:

```go
import schedpkg "github.com/fluxorio/fluxor/pkg/scheduler"

// This will panic with a clear error
spec := schedpkg.Cron("")  // Error: "fail-fast: cron expression cannot be empty"

// This will return an error
scheduler := schedpkg.NewScheduler(ctx)
_, err := scheduler.Schedule(nil, spec)  // Error: "task cannot be nil"
```

## Best Practices

### 1. Use Context for Cancellation

```go
import schedpkg "github.com/fluxorio/fluxor/pkg/scheduler"

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

scheduler := schedpkg.NewScheduler(ctx)
// ... schedule tasks ...

// Graceful shutdown
scheduler.Stop()
```

### 2. Handle Errors

```go
id, err := scheduler.Schedule(task, spec)
if err != nil {
    log.Printf("Failed to schedule task: %v", err)
    return
}
```

### 3. Use Named Tasks

```go
task := concurrency.NewNamedTask("cleanup-task", func(ctx context.Context) error {
    // Task logic
    return nil
})
```

### 4. Monitor Task Execution

```go
tasks := scheduler.List()
for _, task := range tasks {
    if task.RunCount > 0 {
        fmt.Printf("Task %s executed %d times, last run: %v\n",
            task.ID, task.RunCount, task.LastRun)
    }
}
```

### 5. Clean Up One-Time Tasks

One-time tasks (DelaySpec) are automatically removed after execution. Recurring tasks (CronSpec, IntervalSpec) continue until explicitly unscheduled.

## Thread Safety

All scheduler operations are thread-safe and can be used concurrently from multiple goroutines.

## Performance Considerations

- The scheduler checks for due tasks every second
- Tasks are executed in separate goroutines to avoid blocking
- Task execution has a default timeout of 5 minutes
- The scheduler uses efficient data structures for task lookup

## Integration with Fluxor

The scheduler package integrates seamlessly with Fluxor's core components:

- Uses `pkg/core/concurrency.Task` interface (no circular dependencies)
- Can be used by `pkg/workflow` for schedule node execution
- Can be used by `pkg/entrypoint` runtime for scheduled tasks
- Follows acyclic dependency pattern (depends on core, not vice versa)

## Examples

### Scheduled Cleanup Task

```go
import schedpkg "github.com/fluxorio/fluxor/pkg/scheduler"

ctx := context.Background()
scheduler := schedpkg.NewScheduler(ctx)

cleanupTask := concurrency.NewNamedTask("cleanup", func(ctx context.Context) error {
    // Perform cleanup operations
    return nil
})

// Run daily at 2 AM
id, err := scheduler.Schedule(cleanupTask, schedpkg.Cron("0 2 * * *"))
if err != nil {
    log.Fatal(err)
}

scheduler.Start(ctx)
defer scheduler.Stop()
```

### Periodic Health Check

```go
healthCheckTask := concurrency.NewNamedTask("health-check", func(ctx context.Context) error {
    // Perform health check
    return checkHealth()
})

// Check every 30 seconds
id, err := scheduler.Schedule(healthCheckTask, schedpkg.Interval(30 * time.Second))
```

### Deferred Task Execution

```go
deferredTask := concurrency.NewNamedTask("deferred", func(ctx context.Context) error {
    // This will run once after 1 hour
    return processDeferred()
})

// Execute after 1 hour
id, err := scheduler.Schedule(deferredTask, schedpkg.Delay(1 * time.Hour))
```

## Testing

The package includes comprehensive unit tests:

```bash
go test ./pkg/scheduler/...
```

Test coverage includes:
- Cron expression parsing and validation
- Scheduler operations (schedule, unschedule, start, stop)
- Task execution
- Edge cases (invalid expressions, nil tasks, etc.)

## Limitations

- Cron expressions use 5-field format (minute, hour, day, month, weekday)
- Second-level precision is not supported in cron expressions
- Task execution timeout is fixed at 5 minutes (configurable in implementation)
- The scheduler checks for due tasks every second (not sub-second precision)

## Future Enhancements

Potential future improvements:
- Support for second-level cron expressions (6-field format)
- Configurable task execution timeout
- Task execution history and statistics
- Distributed scheduling support
- Timezone support for cron expressions
- More flexible scheduling patterns

