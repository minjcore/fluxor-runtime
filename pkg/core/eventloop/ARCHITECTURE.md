# EventLoop Architecture

## Overview

The EventLoop architecture provides CPU-based event routing for optimal performance and cache locality.

## Architecture Diagram

```
                ┌───────────┐
gRPC Request ──▶│ Dispatcher│
                └─────┬─────┘
                      │ hash(key) % N
        ┌─────────────┼─────────────┐
        ▼             ▼             ▼
  EventLoop0     EventLoop1     EventLoop2
    (CPU0)         (CPU1)         (CPU2)
       │               │               │
 Internal Bus     Internal Bus     Internal Bus
       │               │               │
     NATS           NATS           NATS
```

## Flow Description

### 1. Ingress Layer (gRPC/NATS/HTTP)

**gRPC Handler Example:**
```go
package main

import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core"
    pb "your/proto/package"
)

func (s *Service) HandleRequest(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    // Extract routing key from request
    key := req.GetUserId()
    
    // Set routing key in context/headers
    // EventBus will extract from headers automatically
    ctx = core.WithRequestID(ctx, req.GetRequestId())
    
    // Publish to EventBus - automatically routes via EventLoopGroup
    // Key extraction: X-Route-Key > X-User-ID > X-Session-ID > X-Request-ID
    err := eb.Publish("user.events", map[string]interface{}{
        "userId":  key,
        "action":  req.GetAction(),
        "payload": req.GetPayload(),
    })
    
    if err != nil {
        return nil, err
    }
    
    return &pb.Response{Status: "ok"}, nil
}
```

**NATS Callback (Automatic):**
```go
// When NATS message arrives:
// 1. NATS callback (onMsg) receives message
// 2. Extracts headers: X-Route-Key, X-User-ID, X-Session-ID, X-Request-ID
// 3. Creates Event with extracted key
// 4. Dispatches to EventLoopGroup via Dispatcher
// 5. EventLoop processes and calls handler

// This is handled automatically by ClusterNATS/ClusterJetStream EventBus
// when UseEventLoopGroup: true is set in config
```

### 2. Dispatcher Layer

**Key Extraction:**
- Priority: `X-Route-Key` > `X-Flox-ID` > `X-User-ID` > `X-Session-ID` > `X-Request-ID`
- FloxID is a universal routing identifier (aggregate ID, stream ID, entity ID, etc.)
- If no key found → round-robin

**Hash Routing:**
```go
hash := FNV-1a(key)
loopIndex := hash % numLoops
```

**Properties:**
- Same key → same loop (consistent hashing)
- Thread-safe concurrent dispatch
- Low latency (< 1ms p99)

### 3. EventLoop Layer

**Per-CPU Event Loops:**
- N loops = `GOMAXPROCS(0)` (default)
- Each loop runs on single goroutine
- Bounded channel queue (default: 4096)
- Optional CPU affinity (pin to core)

**Processing:**
```go
for {
    select {
    case event := <-loop.queue:
        event.Handler(ctx, event)  // Sequential processing
    case <-ctx.Done():
        return
    }
}
```

**Benefits:**
- Single-writer guarantee (no locks needed)
- Cache locality (same CPU core)
- Predictable latency

### 4. Internal Bus Layer

**EventBus Integration:**
- Each EventLoop forwards to Internal EventBus
- EventBus routes to registered handlers
- Executor executes handlers (backward compatible)

**Flow:**
```
EventLoop → Internal Bus → Consumer Mailbox → Executor → Handler
```

### 5. NATS Integration

**NATS → EventLoop:**
```
NATS Message → Extract Key → Dispatcher → EventLoop → Internal Bus → Handler
```

**Properties:**
- NATS callback extracts key from headers
- Dispatches to EventLoopGroup
- Handler executes on assigned EventLoop
- Preserves Executor for handler execution

## Key Concepts

### Consistent Hashing

Same routing key always routes to same EventLoop:
- `hash("user-123") % 4` → always EventLoop 2 (example)
- Ensures stateful operations stay on same CPU
- Enables single-writer pattern

### Single-Writer Guarantee

Each EventLoop processes events sequentially:
- No concurrent access to loop state
- No locks needed in handlers
- Predictable execution order

### Backpressure

When queue is full:
- **Block**: Wait for space (default)
- **Drop**: Drop message (configurable)

### Metrics

Per-loop metrics:
- Queue length
- Dropped messages
- Processed messages
- Latency (p50, p95, p99)
- Throughput

## Configuration

```go
config := EventLoopConfig{
    NumLoops:     0,  // Auto: GOMAXPROCS
    QueueSize:    4096,
    CPUAffinity:  false,  // Optional: pin to CPU
    Backpressure: BackpressureBlock,
    Metrics:      true,
}
```

## Performance Characteristics

- **Latency**: < 1ms p99 for dispatch
- **Throughput**: > 100k messages/sec per loop
- **Memory**: < 1MB overhead per loop
- **CPU**: Better utilization with CPU affinity

## Use Cases

### Stateful Operations
```go
// Same user always routes to same loop
headers := map[string]string{"X-User-ID": userId}
eb.Publish("user.update", data)
```

### Stateless Operations
```go
// No key → round-robin distribution
eb.Publish("metrics.collect", data)
```

### High Throughput
```go
// Multiple loops distribute load
config.NumLoops = runtime.GOMAXPROCS(0)
```

## Best Practices

1. **Always provide routing keys** for stateful operations
2. **Use consistent keys** (e.g., user ID, session ID)
3. **Monitor queue lengths** to detect backpressure
4. **Enable metrics** for observability
5. **Start with defaults** and tune based on workload

## Migration Path

1. **Phase 1**: Add EventLoopGroup (disabled by default)
2. **Phase 2**: Enable for new instances (`UseEventLoopGroup: true`)
3. **Phase 3**: Make default after validation

## Thread Safety

- ✅ EventLoopGroup: Thread-safe for concurrent dispatch
- ✅ Dispatcher: Thread-safe routing
- ✅ EventLoop: Single goroutine (no locks needed)
- ✅ Handlers: No concurrent access (single-writer)

