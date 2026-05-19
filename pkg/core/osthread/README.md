# OS Thread Management Package

The `osthread` package provides utilities for managing **OS threads** (system-level threads) concurrently in Fluxor. It supports thread pinning, thread pools, and thread-local storage for CPU-bound workloads.

## Important Distinction: OS Threads vs Goroutines

**OS Threads** (what this package manages):
- System-level threads managed by the operating system
- Heavyweight (1-2MB stack per thread)
- Limited by GOMAXPROCS (typically = CPU cores)
- Pinned using `runtime.LockOSThread()`
- Used for CPU-bound native code (CGO, llama.cpp)

**Goroutines** (Go's concurrency primitives):
- Lightweight threads managed by Go runtime
- Small stack (2KB initially, grows as needed)
- Can have millions of goroutines
- Scheduled by Go runtime onto OS threads
- Used for general concurrent work (IO-bound, event loops)

This package manages **OS threads**, not goroutines. It pins goroutines to OS threads for CPU-bound work.

## Overview

This package is designed for:
- **CPU-bound native code** (CGO, llama.cpp, FFmpeg)
- **Work requiring CPU affinity** (NUMA, cache locality)
- **Real-time constraints** (low latency)
- **Thread-local state management on OS threads**

## Key Features

1. **Thread Pool**: Managed pool of pinned OS threads
2. **Thread Pinning**: Utilities for pinning goroutines to OS threads
3. **Thread-Local Storage**: Isolated storage per OS thread
4. **Backpressure**: Bounded queues prevent unbounded memory growth
5. **Graceful Shutdown**: Context-based cancellation and cleanup

## Usage

### Thread Pool

```go
import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core/osthread"
)

// Create thread pool
ctx := context.Background()
pool := osthread.NewThreadPool(ctx, osthread.DefaultConfig())

// Start pool
if err := pool.Start(); err != nil {
    log.Fatal(err)
}
defer pool.Stop(context.Background())

// Submit task
task := osthread.NewNamedTask("my-task", func(ctx context.Context) error {
    // CPU-bound work on pinned OS thread
    return doCPUWork()
})

if err := pool.Submit(ctx, task); err != nil {
    log.Fatal(err)
}

// Execute function directly
result, err := pool.Execute(ctx, func() (interface{}, error) {
    // CPU-bound work
    return compute(), nil
})
```

### Thread Pinning

```go
import "github.com/fluxorio/fluxor/pkg/core/osthread"

// Pin current thread
unpin := osthread.PinCurrentThread()
defer unpin()
// CPU-bound work on pinned thread

// Execute on pinned thread
result, err := osthread.WithPinnedThread(func() (interface{}, error) {
    // CPU-bound work on pinned thread
    return nativeFunction(), nil
})
```

### Thread-Local Storage

```go
import "github.com/fluxorio/fluxor/pkg/core/osthread"

// Create thread-local storage
var cache = osthread.NewThreadLocal[*MyCache]()

// In worker thread (pinned)
cache.Set(&MyCache{data: make(map[string]interface{})})

// Later in same thread
myCache := cache.Get()
myCache.data["key"] = "value"

// Clear when done
cache.Clear()
```

## Architecture

### Thread Pool Architecture

```
┌─────────────────────────────────────────┐
│         ThreadPool (OS Thread Pool)     │
│                                          │
│  Task Submission (Round-Robin)          │
│         │                                │
│    ┌────┼────┐                           │
│    │    │    │                           │
│  ┌─▼─┐ ┌▼─┐ ┌▼─┐                        │
│  │ W1│ │W2│ │WN│  (Each has own channel)│
│  │OS │ │OS│ │OS│  (No global shared)    │
│  │T1 │ │T2│ │TN│                        │
│  │(G)│ │(G)│ │(G)│                       │
│  └───┘ └───┘ └───┘                       │
└─────────────────────────────────────────┘
  W = Worker (goroutine)
  OS T = OS Thread (system thread)
  (G) = Goroutine pinned to OS thread
```

**Key Design: No Global Shared Access**
- Each worker has its **own dedicated channel** (not shared)
- Tasks are **routed to specific workers** using round-robin
- No global queue that all workers compete for
- Better cache locality (tasks stay on same OS thread)

Each worker:
- Is a **goroutine** pinned to an **OS thread** (`runtime.LockOSThread()`)
- Runs on a dedicated OS thread (not scheduled by Go runtime)
- Has its own isolated task queue (no global shared state)
- Processes tasks sequentially from its own queue
- Maintains thread-local state (no locking needed)
- Limited by GOMAXPROCS (typically = CPU cores)

### Thread Pinning

When a goroutine calls `runtime.LockOSThread()`:
- The goroutine is **pinned** to the current OS thread
- Go scheduler **cannot** move it to another thread
- Critical for native code that spawns threads (CGO, llama.cpp)

### Thread-Local Storage

Thread-local storage provides:
- **Isolated state** per OS thread
- **No locking** needed (single-threaded access)
- **Cache locality** (state stays in CPU cache)

## Best Practices

### When to Use OS Thread Pinning

✅ **DO** use for:
- CPU-bound native code (CGO, llama.cpp, FFmpeg)
- Work requiring CPU affinity (NUMA, cache locality)
- Real-time constraints (low latency)

❌ **DON'T** use for:
- IO-bound work (HTTP, database, file I/O)
- Work that benefits from Go scheduler preemption
- General-purpose goroutines

### Thread Pool Configuration

```go
// Default: Auto-scale to GOMAXPROCS
config := osthread.DefaultConfig()

// Custom configuration
config := osthread.Config{
    Workers:   8,              // Number of worker threads
    QueueSize: 1000,           // Bounded queue size
    Timeout:  30 * time.Second, // Task timeout
}

pool := osthread.NewThreadPool(ctx, config)
```

### Thread-Local State

```go
// Each worker has its own state (no locking)
type WorkerState struct {
    model *Model       // Thread-local model
    cache *Cache       // Thread-local cache
}

// Access pattern: Only from worker's goroutine
func (w *worker) processTask(task Task) {
    state := w.getState() // Thread-local, no lock
    result := state.model.Infer(task.Input)
    state.cache.Put(task.Key, result)
}
```

## Performance Considerations

1. **Worker Count**: Should match CPU cores (GOMAXPROCS)
2. **Queue Size**: Bounded to prevent unbounded memory growth
3. **Thread Pinning**: Overhead is minimal, but don't overuse
4. **Thread-Local Storage**: Fast access, no contention

## Examples

See `examples/` directory for complete examples:
- CPU-bound computation
- Native code integration (CGO)
- Thread-local caching
- Real-time processing

## Related Packages

- `pkg/core/compute`: Compute pool for CPU-bound work (uses OS thread pinning)
- `pkg/core/concurrency`: General concurrency primitives (goroutines, channels)
- `pkg/core/eventloop`: Event loops (cooperative scheduling)
