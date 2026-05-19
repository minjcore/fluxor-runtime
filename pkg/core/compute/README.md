# Compute Framework Pattern

Framework-level pattern cho CPU-bound work (LLM, FFmpeg, ML, crypto, image processing) trong Fluxor event-driven architecture.

## Core Principles

1. **EventLoop ≠ Compute Loop**: EventLoop chỉ orchestrate, compute chạy trong worker pool
2. **Auto-scaling**: Tự động tính workers và threads dựa trên CPU
3. **Locality**: Route by key để giữ session affinity (quan trọng cho LLM KV cache)
4. **Backpressure**: Framework-level policies cho queue overflow
5. **Non-blocking**: EventLoop không bao giờ block

## Architecture

```
EventLoop (orchestration)
   |
   | Submit job
   v
ComputePool (CPU-bound workers)
   |
   | Route by key
   v
Workers (pinned OS threads)
   |
   v
Result → Future → EventLoop
```

## Quick Start

### Basic Usage

```go
import (
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/core/compute"
)

// Define your compute handler
handler := func(ctx context.Context, payload MyPayload) (interface{}, error) {
    // CPU-bound work here (LLM inference, image processing, etc.)
    result := doHeavyWork(payload)
    return result, nil
}

// Create compute component with auto-config
config := compute.DefaultConfig()
config.RouteByKey = true // Enable key-based routing for locality

component := compute.NewComputeComponent("my-compute", handler, config)

// Use in verticle
type MyVerticle struct {
    *core.BaseVerticle
    compute *compute.ComputeComponent[MyPayload]
}

func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    v.compute.SetParent(v.BaseVerticle)
    return v.compute.Start(ctx)
}

// Submit job (non-blocking)
future, err := v.compute.Submit(ctx, "session-123", MyPayload{...})
if err != nil {
    return err
}

// Get result asynchronously
go func() {
    result, err := future.Get(ctx.Context())
    // Handle result
}()
```

### LLM Integration Example

```go
import (
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/core/compute"
    "github.com/fluxorio/fluxor/pkg/private/llama"
)

type LLMVerticle struct {
    *core.BaseVerticle
    llm *compute.ComputeComponent[llama.ChatRequest]
}

func NewLLMVerticle(config llama.Config) *LLMVerticle {
    // Create LLM handler
    handler := func(ctx context.Context, req llama.ChatRequest) (interface{}, error) {
        client, _ := llama.NewClient(config)
        defer client.Close()
        return client.Chat(ctx, req)
    }

    // Auto-config compute pool
    computeConfig := compute.Config{
        Workers:           0, // Auto: CPU / 2
        ThreadsPerWorker: 0, // Auto: CPU / Workers
        QueueSize:        1000,
        RouteByKey:        true, // Session affinity for KV cache
        BackpressurePolicy: compute.Block,
    }

    component := compute.NewComputeComponent("llm", handler, computeConfig)

    return &LLMVerticle{
        BaseVerticle: core.NewBaseVerticle("llm-verticle"),
        llm:          component,
    }
}

func (v *LLMVerticle) Start(ctx core.FluxorContext) error {
    v.llm.SetParent(v.BaseVerticle)
    return v.llm.Start(ctx)
}

// Use in EventLoop handler
func (v *LLMVerticle) OnMessage(ctx core.FluxorContext, msg core.Message) error {
    var req llama.ChatRequest
    msg.DecodeBody(&req)
    
    // Extract session key for routing
    sessionKey := msg.Headers()["session-id"]
    
    // Submit (non-blocking)
    future, err := v.llm.Submit(ctx, sessionKey, req)
    if err != nil {
        return err
    }

    // Handle result asynchronously
    go func() {
        resp, err := future.Get(ctx.Context())
        if err != nil {
            // Handle error
            return
        }
        // Reply via EventBus
        _ = msg.Reply(resp)
    }()

    return nil
}
```

## Configuration

### Auto-scaling

```go
config := compute.Config{
    Workers:           0, // 0 = auto: GOMAXPROCS / 2
    ThreadsPerWorker:  0, // 0 = auto: GOMAXPROCS / Workers
    QueueSize:        1000,
}
```

**Auto-calculation rules:**
- `Workers = max(1, GOMAXPROCS / 2)`
- `ThreadsPerWorker = max(1, GOMAXPROCS / Workers)`

### Manual Configuration

```go
config := compute.Config{
    Workers:           4,
    ThreadsPerWorker: 2,
    QueueSize:        2000,
}
```

## Routing Policies

### RoundRobin (Default)

```go
config.RoutingPolicy = compute.RoundRobin
```

Distributes jobs evenly across workers.

### HashByKey (Locality)

```go
config.RouteByKey = true
// or
config.RoutingPolicy = compute.HashByKey
```

Routes jobs with same key to same worker. Critical for:
- LLM KV cache (session affinity)
- Stateful processing
- Cache locality

### LeastBusy

```go
config.RoutingPolicy = compute.LeastBusy
```

Routes to worker with shortest queue.

## Backpressure Policies

### Block (Default, Safe)

```go
config.BackpressurePolicy = compute.Block
```

Blocks caller until queue has space. Safe but can cause backpressure.

### DropNewest

```go
config.BackpressurePolicy = compute.DropNewest
```

Drops newest job when queue is full. Good for real-time systems.

### DropOldest

```go
config.BackpressurePolicy = compute.DropOldest
```

Drops oldest job when queue is full. Good for latest-state systems.

### CoalesceByKey (LLM/Streaming)

```go
config.BackpressurePolicy = compute.CoalesceByKey
```

Coalesces jobs with same key: keeps newest, drops older. Perfect for:
- LLM streaming (only latest prompt matters)
- Image processing (only latest frame)
- Real-time updates

## Future/Promise Pattern

```go
// Submit job (non-blocking)
future, err := component.Submit(ctx, "key", payload)

// Get result (blocking, but outside EventLoop)
result, err := future.Get(ctx.Context())

// With timeout
result, err := future.GetWithTimeout(5 * time.Second)

// Check if done
if future.IsDone() {
    // Result available
}
```

## Integration Patterns

### With EventBus

```go
// In EventLoop handler
func (v *MyVerticle) OnMessage(ctx core.FluxorContext, msg core.Message) error {
    var payload MyPayload
    msg.DecodeBody(&payload)
    
    key := msg.Headers()["session-id"]
    future, _ := v.compute.Submit(ctx, key, payload)
    
    go func() {
        result, _ := future.Get(ctx.Context())
        _ = msg.Reply(result)
    }()
    
    return nil
}
```

### With NATS/gRPC

```go
// NATS callback
func onNATSMessage(msg *nats.Msg) {
    var payload MyPayload
    json.Unmarshal(msg.Data, &payload)
    
    // Extract key from NATS subject or headers
    key := extractKey(msg)
    
    future, _ := computeComponent.Submit(ctx, key, payload)
    
    go func() {
        result, _ := future.Get(ctx.Context())
        // Reply via NATS
        msg.Respond(result)
    }()
}
```

## Best Practices

1. **Always use async**: `Submit()` not `SubmitSync()`
2. **Use keys for locality**: Session/user IDs for LLM, stream IDs for video
3. **Choose right backpressure**: CoalesceByKey for streaming, Block for batch
4. **Monitor queue length**: Prevent backpressure
5. **Auto-config**: Let framework calculate optimal settings

## Performance Considerations

### CPU Utilization

```
Optimal: Workers × ThreadsPerWorker ≈ GOMAXPROCS

Example (16 cores):
- 4 workers × 4 threads = 16 ✅
- 8 workers × 2 threads = 16 ✅
```

### Memory

```
Memory per Worker = Task Memory + Context

Total Memory = Workers × Memory per Worker
```

### Latency vs Throughput

```
Fewer Workers (more threads each):
- Lower latency per request
- Lower throughput

More Workers (fewer threads each):
- Higher latency per request
- Higher throughput
```

## Examples

See `pkg/private/llama/` for LLM integration example using this pattern.

