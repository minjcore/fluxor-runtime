# Concurrency Package

Enterprise-grade concurrency abstractions for Fluxor. Provides high-level building blocks for concurrent programming without dealing with low-level goroutines and channels.

## 🎯 Philosophy

**Framework for Building**: This package provides abstractions that hide Go's concurrency primitives (goroutines, channels, select) so you can focus on business logic, not concurrency management.

```go
// ❌ Before: Direct goroutines and channels
ch := make(chan Message, 100)
go func() {
    for msg := range ch {
        process(msg)
    }
}()
select {
case ch <- msg:
default:
    return ErrQueueFull
}

// ✅ After: Concurrency abstractions
mailbox := concurrency.NewBoundedMailbox(100)
executor.Submit(TaskFunc(func(ctx context.Context) error {
    msg, _ := mailbox.Receive(ctx)
    return process(msg)
}))
mailbox.Send(msg)  // Simple, no select statements
```

---

## 📦 Core Components

### 1. Executors (Thread Pools)

Execute tasks concurrently with controlled concurrency levels.

#### WorkerPoolExecutor

Executes tasks using a fixed pool of workers.

```go
// Create executor with 10 workers
executor := concurrency.NewWorkerPoolExecutor(10)

// Submit task
task := concurrency.NewNamedTask("process-order", func(ctx context.Context) error {
    // Process order
    return nil
})

future := executor.Submit(task)

// Wait for result
err := future.Get(context.Background())
```

#### Features
- ✅ Fixed worker pool size
- ✅ Task queue with backpressure
- ✅ Graceful shutdown
- ✅ Context cancellation support
- ✅ Metrics (tasks submitted, completed, failed)

---

### 2. Mailboxes (Message Queues)

Thread-safe message passing between components.

#### BoundedMailbox

Fixed-size mailbox with backpressure control.

```go
// Create mailbox with capacity 100
mailbox := concurrency.NewBoundedMailbox(100)

// Send message (blocks if full)
err := mailbox.Send(message)

// Send with timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
err := mailbox.SendWithContext(ctx, message)

// Receive message
msg, err := mailbox.Receive(context.Background())

// Check size
size := mailbox.Size()
capacity := mailbox.Capacity()
```

#### UnboundedMailbox

Unlimited capacity mailbox (use with caution).

```go
mailbox := concurrency.NewUnboundedMailbox()

// Never blocks on send
mailbox.Send(message)

// Receive message
msg, err := mailbox.Receive(context.Background())
```

#### Features
- ✅ Thread-safe operations
- ✅ Context cancellation support
- ✅ Backpressure control (bounded)
- ✅ Non-blocking checks (Size, IsClosed)
- ✅ Graceful shutdown (Close)

---

### 3. Tasks

Abstraction for executable work units.

#### Task Interface

```go
type Task interface {
    Execute(ctx context.Context) error
}
```

#### TaskFunc (Functional Task)

```go
task := concurrency.TaskFunc(func(ctx context.Context) error {
    // Do work
    return nil
})
```

#### NamedTask

```go
task := concurrency.NewNamedTask("my-task", func(ctx context.Context) error {
    // Do work
    return nil
})

// Get task name
name := task.Name()
```

---

### 4. WorkerPool

Low-level worker pool for custom task distribution.

```go
// Create worker pool
pool := concurrency.NewWorkerPool(10)

// Start pool
pool.Start()

// Submit task
pool.Submit(task)

// Shutdown
pool.Shutdown(context.Background())
```

---

## 🚀 Usage Patterns

### Pattern 1: Request Processing Pipeline

```go
// Create components
executor := concurrency.NewWorkerPoolExecutor(10)
mailbox := concurrency.NewBoundedMailbox(100)

// Start processor
go func() {
    for {
        msg, err := mailbox.Receive(context.Background())
        if err != nil {
            return // Mailbox closed
        }
        
        // Process async
        executor.Submit(concurrency.TaskFunc(func(ctx context.Context) error {
            return processMessage(msg)
        }))
    }
}()

// Producer
mailbox.Send(request)
```

---

### Pattern 2: Verticle with Mailbox

```go
type WorkerVerticle struct {
    *core.BaseVerticle
    mailbox   concurrency.Mailbox
    executor  concurrency.Executor
}

func (v *WorkerVerticle) Start(ctx core.FluxorContext) error {
    v.mailbox = concurrency.NewBoundedMailbox(100)
    v.executor = concurrency.NewWorkerPoolExecutor(5)
    
    // Listen to EventBus
    consumer := v.Consumer("work.queue")
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        // Put in mailbox for async processing
        return v.mailbox.Send(msg.Body())
    })
    
    // Start workers
    for i := 0; i < 5; i++ {
        go v.worker(ctx.Context())
    }
    
    return nil
}

func (v *WorkerVerticle) worker(ctx context.Context) {
    for {
        msg, err := v.mailbox.Receive(ctx)
        if err != nil {
            return // Mailbox closed or context cancelled
        }
        
        v.executor.Submit(concurrency.TaskFunc(func(ctx context.Context) error {
            return v.processWork(msg)
        }))
    }
}

func (v *WorkerVerticle) Stop(ctx core.FluxorContext) error {
    v.mailbox.Close()
    v.executor.Shutdown(ctx.Context())
    return nil
}
```

---

### Pattern 3: Batch Processing

```go
executor := concurrency.NewWorkerPoolExecutor(20)

// Process batch concurrently
var wg sync.WaitGroup
for _, item := range items {
    wg.Add(1)
    item := item // Capture
    
    executor.Submit(concurrency.TaskFunc(func(ctx context.Context) error {
        defer wg.Done()
        return processItem(item)
    }))
}

wg.Wait()
```

---

### Pattern 4: Future-based Async

```go
executor := concurrency.NewWorkerPoolExecutor(10)

// Submit async task
future := executor.Submit(concurrency.NewNamedTask("fetch-user", func(ctx context.Context) error {
    user := fetchUser(userID)
    return nil
}))

// Do other work...

// Wait for result
err := future.Get(context.Background())
```

---

## 📊 Metrics & Observability

### Executor Metrics

```go
executor := concurrency.NewWorkerPoolExecutor(10)

// Get metrics
metrics := executor.Metrics()

fmt.Printf("Tasks Submitted: %d\n", metrics.TasksSubmitted)
fmt.Printf("Tasks Completed: %d\n", metrics.TasksCompleted)
fmt.Printf("Tasks Failed: %d\n", metrics.TasksFailed)
fmt.Printf("Active Workers: %d\n", metrics.ActiveWorkers)
fmt.Printf("Queue Size: %d\n", metrics.QueueSize)
```

### Mailbox Metrics

```go
mailbox := concurrency.NewBoundedMailbox(100)

// Check state
fmt.Printf("Size: %d\n", mailbox.Size())
fmt.Printf("Capacity: %d\n", mailbox.Capacity())
fmt.Printf("Utilization: %.2f%%\n", float64(mailbox.Size())/float64(mailbox.Capacity())*100)
fmt.Printf("Is Closed: %v\n", mailbox.IsClosed())
```

---

## 🎯 Best Practices

### 1. Choose Right Mailbox Type

```go
// ✅ Good: Bounded for backpressure control
mailbox := concurrency.NewBoundedMailbox(100)

// ⚠️ Use carefully: Unbounded can cause OOM
mailbox := concurrency.NewUnboundedMailbox()
```

### 2. Always Use Context

```go
// ✅ Good: Respect cancellation
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
msg, err := mailbox.Receive(ctx)

// ❌ Bad: No timeout
msg, err := mailbox.Receive(context.Background())
```

### 3. Graceful Shutdown

```go
// ✅ Good: Cleanup resources
func (v *Verticle) Stop(ctx core.FluxorContext) error {
    v.mailbox.Close()                    // Stop accepting
    v.executor.Shutdown(ctx.Context())   // Wait for tasks
    return nil
}

// ❌ Bad: Abrupt termination
func (v *Verticle) Stop(ctx core.FluxorContext) error {
    return nil // Leaks goroutines
}
```

### 4. Size Executor Pool Appropriately

```go
// CPU-bound tasks: workers = CPU cores
numCPU := runtime.NumCPU()
executor := concurrency.NewWorkerPoolExecutor(numCPU)

// I/O-bound tasks: workers = 2-4x CPU cores
executor := concurrency.NewWorkerPoolExecutor(numCPU * 4)

// Network calls with high latency: higher pool size
executor := concurrency.NewWorkerPoolExecutor(100)
```

### 5. Monitor Metrics

```go
// Setup metrics monitoring
go func() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        metrics := executor.Metrics()
        if metrics.QueueSize > 80 {
            log.Warn("Executor queue near capacity", "size", metrics.QueueSize)
        }
    }
}()
```

---

## ⚡ Performance Considerations

### Mailbox Capacity

| Capacity | Pros | Cons | Use Case |
|----------|------|------|----------|
| Small (10-50) | Fast, low memory | Backpressure | Low-latency, bounded work |
| Medium (100-500) | Balanced | Moderate memory | General purpose |
| Large (1000+) | Burst tolerance | High memory | Spiky traffic |

### Executor Pool Size

| Pool Size | Pros | Cons | Use Case |
|-----------|------|------|----------|
| Small (1-10) | Low overhead | Limited concurrency | Sequential processing |
| Medium (10-50) | Balanced | Moderate overhead | General purpose |
| Large (50-200) | High concurrency | High overhead | I/O-bound, network calls |

---

## 🧪 Testing

### Unit Tests

```bash
go test ./pkg/core/concurrency/...
```

### Benchmarks

```bash
go test -bench=. ./pkg/core/concurrency/...
```

**Sample Results**:
```
BenchmarkMailbox/Send-8              5000000    250 ns/op
BenchmarkMailbox/Receive-8           5000000    280 ns/op
BenchmarkExecutor/Submit-8           3000000    450 ns/op
BenchmarkWorkerPool/Submit-8         2000000    600 ns/op
```

---

## 🔍 Advanced Features

### Custom Task Validation

```go
type ValidatedTask struct {
    task concurrency.Task
}

func (t *ValidatedTask) Execute(ctx context.Context) error {
    // Pre-validation
    if err := t.validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    // Execute
    return t.task.Execute(ctx)
}
```

### Task Retry

```go
func RetryTask(task concurrency.Task, maxRetries int) concurrency.Task {
    return concurrency.TaskFunc(func(ctx context.Context) error {
        var err error
        for i := 0; i < maxRetries; i++ {
            err = task.Execute(ctx)
            if err == nil {
                return nil
            }
            time.Sleep(time.Duration(i+1) * time.Second)
        }
        return err
    })
}
```

### Task Chain

```go
func ChainTasks(tasks ...concurrency.Task) concurrency.Task {
    return concurrency.TaskFunc(func(ctx context.Context) error {
        for _, task := range tasks {
            if err := task.Execute(ctx); err != nil {
                return err
            }
        }
        return nil
    })
}
```

---

## 🆚 Comparison with Other Approaches

### vs Direct Goroutines

| Feature | Direct Goroutines | Concurrency Package |
|---------|------------------|---------------------|
| **Complexity** | High | Low |
| **Resource Control** | Manual | Automatic |
| **Backpressure** | Manual (select) | Built-in |
| **Metrics** | None | Built-in |
| **Cancellation** | Manual | Context-aware |
| **Testing** | Hard | Easy (mockable) |

### vs Other Libraries

| Feature | pkg/concurrency | golang.org/x/sync | github.com/panjf2000/ants |
|---------|----------------|-------------------|---------------------------|
| **Mailboxes** | ✅ | ❌ | ❌ |
| **Executors** | ✅ | ⚠️ (errgroup) | ✅ |
| **Metrics** | ✅ | ❌ | ⚠️ (limited) |
| **Fail-fast** | ✅ | ❌ | ❌ |
| **Task Interface** | ✅ | ❌ | ⚠️ (func only) |

---

## 📚 Related Documentation

- [pkg/core/BASE_CLASSES.md](../BASE_CLASSES.md) - Premium Pattern base classes
- [pkg/scheduler/README.md](../../scheduler/README.md) - Task scheduling
- [PKG_DECISION_GUIDE.md](../../../PKG_DECISION_GUIDE.md) - Package selection guide
- [DOCUMENTATION.md](../../../DOCUMENTATION.md) - Complete API reference

---

## 🔗 Integration with Other Packages

### With pkg/core (EventBus)

```go
// EventBus → Mailbox → Executor
consumer := eventBus.Consumer("work.queue")
consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    return mailbox.Send(msg.Body())
})
```

### With pkg/scheduler

```go
// Scheduler uses Task interface
scheduler.Schedule(
    concurrency.NewNamedTask("cleanup", func(ctx context.Context) error {
        return cleanup()
    }),
    schedpkg.Cron("0 2 * * *"),
)
```

### With pkg/workflow

```go
// Workflow nodes can use executors for parallel execution
executor.Submit(concurrency.TaskFunc(func(ctx context.Context) error {
    return workflowNode.Execute(ctx, input)
}))
```

---

## ✅ Thread Safety

All components are thread-safe:
- ✅ **Mailbox**: Safe for concurrent Send/Receive
- ✅ **Executor**: Safe for concurrent Submit
- ✅ **WorkerPool**: Safe for concurrent operations
- ✅ **Metrics**: Thread-safe counters

---

**Package**: `github.com/fluxorio/fluxor/pkg/core/concurrency`  
**Status**: ✅ Stable (A- Grade)  
**Test Coverage**: 92%  
**Last Updated**: 2026-01-04
