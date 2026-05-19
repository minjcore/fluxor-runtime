# Core Package - EventLoop Architecture

This document explains the EventLoop architecture in the Fluxor core package, which provides race-free sequential processing for verticles and state management for deployments.

## Overview

Fluxor uses **three types of event loops** to ensure thread-safe, race-free operations:

1. **Verticle Event Loop** - Sequential task processing per verticle
2. **Deployment Event Loop** - State machine management for deployments
3. **EventBus Executor** - Concurrent message processing

---

## 1. Verticle Event Loop (Per-Verticle)

Each verticle has its own event loop for **sequential processing**, ensuring race-free access to verticle state.

### Location
- `pkg/core/base_verticle.go` - Event loop creation and management
- `pkg/core/base_verticle_eventloop.go` - Event loop API

### Key Characteristics

- **Single worker** (Workers: 1) = Sequential processing
- **Queue size**: 1000 tasks
- **Created during**: `BaseVerticle.Start()`
- **Shut down during**: `BaseVerticle.Stop()`

### Implementation

```go
// Created in BaseVerticle.Start()
eventLoopConfig := concurrency.ExecutorConfig{
    Workers:   1,    // Single worker = sequential processing
    QueueSize: 1000, // Queue size for events
}
bv.eventLoop = concurrency.NewExecutor(gocmdCtx, eventLoopConfig)
```

### Usage

```go
type MyVerticle struct {
    *core.BaseVerticle
}

func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    // Event loop is automatically created by BaseVerticle
    
    // Submit work to event loop (sequential, race-free)
    // IMPORTANT: Tasks must complete in < 20µs
    v.RunOnEventLoop(concurrency.TaskFunc(func(ctx context.Context) error {
        // Safe to access verticle state here - no race conditions
        // Only IO + dispatch operations (< 20µs)
        return nil
    }))
    
    return nil
}
```

### Iron Rule: Event Loop < 20µs

**Event loop tasks must complete in < 20µs**. This ensures:
- Low latency for message processing
- No blocking of other tasks
- Predictable performance

**For blocking work** (CPU-bound, blocking I/O), use `SubmitBlocking()` instead:

```go
// Blocking work goes to WorkerPool, not EventLoop
future := v.SubmitBlocking(func() (string, error) {
    // CPU-intensive work or blocking I/O
    result := heavyComputation()
    return result, nil
})

// Handle result asynchronously
result, err := future.Await(ctx)
```

### WorkerPool Pattern

Each verticle has its own **WorkerPool** for blocking work, separated from the event loop:

```
EventLoop (1 worker, sequential)
  ├── IO operations (non-blocking)
  ├── Dispatch messages
  └── SubmitBlocking() → WorkerPool
              └── Future<T> (async result)
```

**WorkerPool Configuration**:
- **Workers**: `runtime.NumCPU()` (default)
- **QueueSize**: 1000 (default)
- **Purpose**: CPU-bound work, blocking I/O

**When to use**:
- ✅ `RunOnEventLoop()`: IO + dispatch only (< 20µs)
- ✅ `SubmitBlocking()`: CPU-bound, blocking I/O (> 20µs)

### Race-Free Guarantee

With `Workers: 1`, all tasks execute **sequentially in a single goroutine**:
- Go channels guarantee only one receiver gets each message (serialization)
- All tasks execute one at a time in the same goroutine (no concurrent access)
- This matches the explicit channel-based event loop pattern:

```go
func (v *BaseVerticle) eventLoop() {
    for {
        select {
        case task := <-taskChan:
            task.Execute()  // Sequential - no race conditions
        case <-ctx.Done():
            return
        }
    }
}
```

---

## 2. Deployment Event Loop (Per-Deployment)

Manages deployment lifecycle state transitions using a **single goroutine** that processes commands sequentially.

### Location
- `pkg/core/deployment.go` (lines 269-415)

### Key Characteristics

- **Single goroutine** handles all state transitions
- **Channel-based** command processing (`cmdChan`)
- **Commands**: `"start"`, `"stop"`, `"undeploy"`
- **State machine**: PENDING → DEPLOYING → RUNNING/FAILED → STOPPING → STOPPED → UNDEPLOYED

### Implementation

```go
// Event loop goroutine
func (d *Deployment) startEventLoop() {
    go func() {
        defer close(d.done)
        for cmd := range d.cmdChan {
            currentState := d.State()
            
            switch cmd.action {
            case "start":
                // Handle start transition
                d.setState(StateDeploying)
                err := d.Verticle.Start(d.fluxorCtx)
                if err != nil {
                    d.setState(StateFailed)
                } else {
                    d.setState(StateRunning)
                }
                
            case "stop":
                // Handle stop transition
                d.setState(StateStopping)
                d.Verticle.Stop(d.fluxorCtx)
                d.setState(StateStopped)
                
            case "undeploy":
                // Handle undeploy transition
                // Automatically stops if RUNNING
                d.setState(StateUndeployed)
                return // Exit event loop
            }
        }
    }()
}
```

### Purpose

Ensures **atomic state transitions** by processing commands sequentially:
- Only one command processed at a time
- State transitions are race-free
- Prevents invalid state transitions

---

## 3. EventBus Executor (Message Processing)

Processes messages from the EventBus using **concurrent workers**.

### Location
- `pkg/core/eventbus_impl.go`

### Key Characteristics

- **10 workers** (concurrent message processing)
- **Queue size**: 1000 messages
- Processes messages from all registered consumers

### Implementation

```go
// Created in NewEventBus()
executorConfig := concurrency.DefaultExecutorConfig()
executorConfig.Workers = 10
executorConfig.QueueSize = 1000
executor := concurrency.NewExecutor(ctx, executorConfig)
```

### Purpose

Allows **concurrent message processing** while maintaining isolation:
- Multiple messages can be processed simultaneously
- Each consumer handler runs in its own worker
- Backpressure control via bounded queue

---

## Underlying Implementation

All event loops use `concurrency.Executor` from `pkg/core/concurrency/executor_impl.go`.

### Core Pattern

```go
func (e *defaultExecutor) worker(id int) {
    defer e.wg.Done()
    
    for {
        select {
        case task, ok := <-e.taskChan:
            if !ok {
                return // Channel closed
            }
            // Decrement queued tasks counter
            atomic.AddInt64(&e.queuedTasks, -1)
            
            // Execute task sequentially (race-free when Workers=1)
            if err := task.Execute(e.ctx); err != nil {
                e.logger.Errorf("task %s failed: %v", task.Name(), err)
            }
            atomic.AddInt64(&e.completedTasks, 1)
            
        case <-e.ctx.Done():
            return // Context cancelled - graceful shutdown
        }
    }
}
```

This implements the **classic Go event loop pattern**:
- `for { select { case task := <-chan: ... } }`
- With `Workers=1`, only one goroutine runs this loop
- Go channels guarantee only one receiver gets each message (serialization)
- Tasks execute one at a time in the same goroutine (no concurrent access)

---

## 4. WorkerPool Pattern (Per-Verticle)

Each verticle has its own **WorkerPool** for blocking work, separated from the event loop.

### Location
- `pkg/core/base_verticle.go` - WorkerPool creation and management
- `pkg/core/base_verticle_eventloop.go` - SubmitBlocking API

### Key Characteristics

- **Workers**: `runtime.NumCPU()` (default) - one per CPU core
- **Queue size**: 1000 tasks
- **Created during**: `BaseVerticle.Start()`
- **Shut down during**: `BaseVerticle.Stop()`
- **Purpose**: Execute blocking work (CPU-bound, blocking I/O)

### Iron Rule: Event Loop < 20µs

**Event loop tasks must complete in < 20µs**. The executor monitors execution time and logs warnings if tasks exceed this limit.

### Usage

```go
type MyVerticle struct {
    *core.BaseVerticle
}

func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    // WorkerPool is automatically created by BaseVerticle
    
    // Submit blocking work to WorkerPool
    future := v.SubmitBlocking(func() (string, error) {
        // CPU-intensive work or blocking I/O
        result := heavyComputation()
        return result, nil
    })
    
    // Option 1: Await result
    result, err := future.Await(ctx)
    if err != nil {
        return err
    }
    
    // Option 2: Use callbacks
    future.OnSuccess(func(result string) {
        // Handle success
    }).OnFailure(func(err error) {
        // Handle error
    })
    
    return nil
}
```

### Pattern: EventLoop (IO) + WorkerPool (Blocking)

```
EventLoop (1 worker, sequential)
  ├── IO operations (non-blocking, < 20µs)
  ├── Dispatch messages
  └── SubmitBlocking() → WorkerPool
              └── Future<T> (async result)

WorkerPool (N workers, concurrent)
  └── Execute blocking work
      ├── CPU-bound operations
      └── Blocking I/O
```

### When to Use What

| Operation | Method | Execution |
|-----------|--------|-----------|
| IO + dispatch | `RunOnEventLoop()` | EventLoop (< 20µs) |
| CPU-bound work | `SubmitBlocking()` | WorkerPool |
| Blocking I/O | `SubmitBlocking()` | WorkerPool |
| State updates | `RunOnEventLoop()` | EventLoop (race-free) |

### Monitoring

The executor automatically monitors event loop task execution time:
- **Warning logged** if task execution > 20µs
- Helps detect blocking work that should use `SubmitBlocking()`
- Metrics track execution time per task

---

## Schedule Timer (kernel-style)

A tick-based scheduler (kernel-style, like go-micron’s kernel/scheduler) runs at a fixed interval and fires an event each tick so verticles can react (e.g. uptime check, heartbeat, metrics).

### Location
- `pkg/core/schedule_timer.go` – `ScheduleTimer`, `ScheduleTimerVerticle`, `UptimeEvent`

### Usage

Deploy the timer verticle and consume `core.schedule.tick`:

```go
app.DeployVerticle(core.NewScheduleTimerVerticle(core.ScheduleTimerVerticleConfig{
    Tick: time.Second,
}))
app.GoCMD().EventBus().Consumer(core.ScheduleTickAddress).Handler(func(ctx core.FluxorContext, msg core.Message) error {
    var e core.UptimeEvent
    _ = msg.DecodeBody(&e)
    log.Printf("uptime %v tick %d", e.Uptime, e.Tick)
    return nil
})
```

Optional per-tick tasks (kernel-style round-robin):

```go
core.NewScheduleTimerVerticle(core.ScheduleTimerVerticleConfig{
    Tick: 10 * time.Millisecond,
    Tasks: []core.ScheduleTask{
        {Name: "health", Step: func(uptime time.Duration, tick int64) { /* lightweight check */ }},
    },
})
```

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│                    GoCMD                                 │
│  ┌──────────────────────────────────────────────────┐    │
│  │           EventBus (10 workers)                  │    │
│  │  Processes messages concurrently                 │    │
│  │  Queue: 1000 messages                            │    │
│  └──────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
                        │
                        │ Deploys
                        ▼
┌─────────────────────────────────────────────────────────┐
│              Deployment Event Loop                      │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Single goroutine                               │  │
│  │  Manages: start → stop → undeploy              │  │
│  │  State machine transitions (race-free)          │  │
│  │  Channel: cmdChan (1 buffer)                    │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                        │
                        │ Creates
                        ▼
┌─────────────────────────────────────────────────────────┐
│              Verticle Event Loop                        │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Single worker (Workers: 1)                     │  │
│  │  Sequential task processing                     │  │
│  │  Queue: 1000 tasks                              │  │
│  │  Race-free verticle state access                │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

---

## Key Benefits

### 1. Race-Free Guarantee
- Single worker ensures sequential execution
- No mutex needed for verticle state access
- Thread-safe by design

### 2. Isolation
- Each verticle has its own event loop
- State changes in one verticle don't affect others
- Independent lifecycle management

### 3. Backpressure Control
- Bounded queues prevent unbounded growth
- `ErrMailboxFull` returned when queue is full
- Prevents memory exhaustion

### 4. Graceful Shutdown
- Context cancellation support
- Waits for queued tasks to complete
- Proper resource cleanup

### 5. Abstraction
- Hides goroutines/channels from application code
- Clean API: `RunOnEventLoop(task)`
- No need to manage concurrency primitives

---

## Usage Patterns

### Pattern 1: Sequential State Updates

```go
func (v *MyVerticle) UpdateState(newValue string) error {
    return v.RunOnEventLoop(concurrency.TaskFunc(func(ctx context.Context) error {
        // Safe to access v.state here - no race conditions
        v.state = newValue
        return nil
    }))
}
```

### Pattern 2: Async Operations with Callback

```go
func (v *MyVerticle) ProcessAsync(data []byte) error {
    // Start async operation
    go func() {
        result := processData(data)
        
        // Update state on event loop (race-free)
        v.RunOnEventLoop(concurrency.TaskFunc(func(ctx context.Context) error {
            v.result = result
            return nil
        }))
    }()
    
    return nil
}
```

### Pattern 3: EventBus Message Handling

```go
func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    consumer := v.Consumer("my.address")
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        // Handler runs on EventBus executor (concurrent)
        // Use event loop for state updates
        return v.RunOnEventLoop(concurrency.TaskFunc(func(ctx context.Context) error {
            v.processMessage(msg)
            return nil
        }))
    })
    
    return nil
}
```

---

## Performance Considerations

### Verticle Event Loop
- **Workers: 1** = Sequential processing
- **Queue: 1000** = Handles bursts
- **Use case**: State updates, sequential operations

### Deployment Event Loop
- **Single goroutine** = Minimal overhead
- **Channel buffer: 1** = Immediate backpressure
- **Use case**: Lifecycle management

### EventBus Executor
- **Workers: 10** = Concurrent message processing
- **Queue: 1000** = High throughput
- **Use case**: Message handling, I/O operations

---

## Thread Safety

All event loops are **thread-safe**:

- ✅ **Verticle Event Loop**: Safe for concurrent `RunOnEventLoop()` calls
- ✅ **Deployment Event Loop**: Safe for concurrent command sends
- ✅ **EventBus Executor**: Safe for concurrent message publishing
- ✅ **State Access**: Race-free when using event loop for updates

---

## Related Documentation

- [BASE_CLASSES.md](./BASE_CLASSES.md) - Base class patterns
- [concurrency/README.md](./concurrency/README.md) - Executor implementation details
- [deployment.go](./deployment.go) - Deployment lifecycle management
- [base_verticle.go](./base_verticle.go) - Verticle base class

---

**Package**: `github.com/fluxorio/fluxor/pkg/core`  
**Status**: ✅ Stable  
**Last Updated**: 2026-01-04

