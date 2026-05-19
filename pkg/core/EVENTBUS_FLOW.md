# EventBus Flow Architecture

## Overview

EventBus trong Fluxor sử dụng kiến trúc EventLoop để route messages dựa trên CPU cores, đảm bảo cache locality và performance tối ưu.

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

**HTTP Request với FastRequestContext (Web Integration):**
```go
// Example: Auto-routing với UserID và FloxID
router.POSTFast("/orders/:orderId/events", func(ctx *web.FastRequestContext) error {
    // FloxID automatically extracted from path param :orderId
    // UserID automatically extracted from JWT/Context/Header
    
    // Auto-routing: same stream → same EventLoop
    err := ctx.PublishWithRouting("order.events", eventData)
    if err != nil {
        return err
    }
    
    return ctx.JSON(200, map[string]string{"status": "ok"})
})

// Example: Manual routing với UserID
router.GETFast("/users/:userId/profile", func(ctx *web.FastRequestContext) error {
    // UserID automatically extracted (JWT > Context > Header priority)
    userID := ctx.UserID()
    
    // Get all routing headers
    headers := ctx.GetRoutingHeaders()
    // Returns: X-User-ID, X-Request-ID, X-Flox-ID (if available)
    
    // Publish with automatic routing
    err := ctx.PublishWithRouting("user.profile.requested", map[string]interface{}{
        "user_id": userID,
    })
    
    return ctx.JSON(200, map[string]string{"user_id": userID})
})
```

**gRPC Request:**
```go
func (s *Service) HandleRequest(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    // Extract routing key from request
    streamID := req.GetStreamId()
    
    // Set FloxID in context (highest priority for routing)
    ctx = core.WithFloxID(ctx, streamID)
    ctx = core.WithRequestID(ctx, req.GetRequestId())
    
    // Publish to EventBus - FloxID automatically extracted from context
    err := eb.PublishWithContext(ctx, "stream.events", req.GetPayload())
    
    return &pb.Response{Status: "ok"}, err
}
```

**NATS Message:**
- NATS callback tự động extract key từ headers
- Dispatch qua EventLoopGroup
- Handler chạy trên EventLoop được assign

### 2. Dispatcher Layer

**Key Extraction Priority:**
1. **FloxID from context** (highest priority - checked first)
2. `X-Route-Key` (explicit routing key in headers)
3. `X-Flox-ID` (universal routing ID in headers - aggregate ID, stream ID, entity ID, etc.)
4. `X-User-ID` (user-based routing)
5. `X-Session-ID` (session-based routing)
6. `X-Request-ID` (request-based routing - lowest priority)
7. Round-robin (nếu không có key)

**Hash Routing:**
```go
hash := FNV-1a(key)
loopIndex := hash % numLoops
```

**Properties:**
- ✅ Same key → same loop (consistent hashing)
- ✅ Thread-safe concurrent dispatch
- ✅ Low latency (< 1ms p99)

### 3. EventLoop Layer

**Per-CPU Event Loops:**
- N loops = `GOMAXPROCS(0)` (default)
- Mỗi loop chạy trên 1 goroutine
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
- ✅ Single-writer guarantee (no locks needed)
- ✅ Cache locality (same CPU core)
- ✅ Predictable latency

### 4. Internal Bus Layer

**EventBus Integration:**
- Mỗi EventLoop forward đến Internal EventBus
- EventBus route đến registered handlers
- Executor execute handlers (backward compatible)

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
- NATS callback extracts key từ headers
- Dispatches to EventLoopGroup
- Handler executes trên assigned EventLoop
- Preserves Executor cho handler execution

## Implementation Details

### EventBus Configuration

```go
config := core.EventBusConfig{
    UseEventLoopGroup: true,  // Enable CPU-based routing
    EventLoopConfig: eventloop.EventLoopConfig{
        NumLoops:  0,  // Auto: GOMAXPROCS
        QueueSize: 4096,
        CPUAffinity: false,  // Optional: pin to CPU core
    },
}

eb := core.NewEventBusWithConfig(ctx, gocmd, config)
```

### Key Extraction

```go
// Priority order trong Dispatcher
keyOrder := []string{
    "X-Route-Key",    // Highest priority
    "X-User-ID",
    "X-Session-ID",
    "X-Request-ID",    // Lowest priority
}
```

### Routing Algorithm

```go
// Hash-based routing (consistent hashing)
func RouteKey(key string, numLoops int) int {
    hash := fnv1a(key)
    return int(hash % uint64(numLoops))
}
```

## Key Concepts

### Consistent Hashing

Same routing key luôn route đến cùng EventLoop:
- `hash("user-123") % 4` → luôn EventLoop 2 (ví dụ)
- Đảm bảo stateful operations stay trên same CPU
- Enables single-writer pattern

### Single-Writer Guarantee

Mỗi EventLoop process events sequentially:
- ✅ No concurrent access to loop state
- ✅ No locks needed in handlers
- ✅ Predictable execution order

### Backpressure

Khi queue đầy:
- EventLoop queue full → return error
- Handler mailbox full → skip message
- NATS queue full → drop message

## Usage Examples

### Basic Publish

```go
// Publish với FloxID (universal routing ID)
ctx := core.WithFloxID(ctx, "aggregate-123")
eb.Publish("user.events", payload)
// FloxID automatically extracted from context and used for routing

// Hoặc với explicit routing key
headers := map[string]string{
    "X-Route-Key": "custom-key",
}
eb.Publish("user.events", payload)

// Hoặc với UserID
headers := map[string]string{
    "X-User-ID": "user-123",
}
eb.Publish("user.events", payload)
```

### Send (Request-Reply)

```go
// Send với routing key
reply, err := eb.Send("user.service", payload, 5*time.Second)
```

### Subscribe

```go
// Subscribe handler
consumer := eb.Consumer("user.events")
consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    // Handler chạy trên EventLoop được assign
    // Sequential processing - no locks needed
    return processMessage(msg)
})
```

## Performance Characteristics

### Latency

- **Dispatcher routing**: < 1ms p99
- **EventLoop dispatch**: < 100μs p99
- **Handler execution**: depends on handler

### Throughput

- **Per EventLoop**: ~100k events/sec
- **Total (N loops)**: N × 100k events/sec
- **Bottleneck**: Handler execution time

### Scalability

- **Horizontal**: Add more EventLoops (up to CPU cores)
- **Vertical**: Optimize handler execution
- **Backpressure**: Bounded queues prevent OOM

## Migration Guide

### Enable EventLoopGroup

```go
// Old way (Executor-only)
eb := core.NewEventBus(ctx, gocmd)

// New way (EventLoopGroup)
config := core.EventBusConfig{
    UseEventLoopGroup: true,
}
eb := core.NewEventBusWithConfig(ctx, gocmd, config)
```

### Set Routing Key

**Web Integration (FastRequestContext):**
```go
// Automatic routing - recommended for web handlers
router.POSTFast("/orders/:orderId/events", func(ctx *web.FastRequestContext) error {
    // FloxID automatically extracted from :orderId path param
    // UserID automatically extracted from JWT/Context/Header
    // RequestID automatically generated
    
    // Publish with automatic routing
    return ctx.PublishWithRouting("order.events", payload)
})

// Manual extraction
router.GETFast("/users/:userId", func(ctx *web.FastRequestContext) error {
    userID := ctx.UserID()        // JWT > Context > Header priority
    floxid := ctx.FloxID()        // Header > Path params > Context priority
    headers := ctx.GetRoutingHeaders()  // All routing headers
    
    // Use context-aware methods
    ctxWithRouting := ctx.Context()  // Includes FloxID and RequestID
    return eb.PublishWithContext(ctxWithRouting, "user.events", payload)
})
```

**Direct EventBus Usage:**
```go
// Option 1: Via FloxID in context (HIGHEST PRIORITY - recommended)
ctx := core.WithFloxID(ctx, "aggregate-123")
eb.PublishWithContext(ctx, "user.events", payload)
// FloxID from context has highest priority - automatically extracted and set in Event.Key

// Option 2: Via headers (lower priority than context)
headers := map[string]string{
    "X-Route-Key": "custom-key",   // Explicit routing key
    "X-Flox-ID": "aggregate-123",  // Universal routing ID
    "X-User-ID": "user-123",       // User-based routing
}
msg := core.NewMessage(payload, headers, "", eb)
eb.Publish("user.events", payload)  // Headers extracted from message

// Option 3: Via context (RequestID - lowest priority)
ctx := core.WithRequestID(ctx, "request-456")
eb.PublishWithContext(ctx, "user.events", payload)
```

**Routing Priority Order:**
1. **FloxID from context** (highest - set via `core.WithFloxID()`)
2. `X-Route-Key` (explicit routing key in headers)
3. `X-Flox-ID` (in headers)
4. `X-User-ID` (user-based routing)
5. `X-Session-ID` (session-based routing)
6. `X-Request-ID` (request-based routing - lowest priority)

## Troubleshooting

### Messages not routing correctly

1. Check routing key extraction:
   ```go
   // Debug key extraction
   key := extractor(headers, address, body)
   fmt.Printf("Extracted key: %s\n", key)
   ```

2. Verify EventLoopGroup is enabled:
   ```go
   if eb.loopGroup == nil {
       // EventLoopGroup not enabled
   }
   ```

### High latency

1. Check EventLoop queue length:
   ```go
   stats := eb.loopGroup.Stats()
   for _, stat := range stats {
       fmt.Printf("Queue length: %d\n", stat.QueueLength)
   }
   ```

2. Check handler execution time:
   ```go
   // Profile handler
   start := time.Now()
   handler(ctx, msg)
   duration := time.Since(start)
   ```

### Backpressure issues

1. Increase queue size:
   ```go
   config.EventLoopConfig.QueueSize = 8192
   ```

2. Optimize handler:
   - Reduce execution time
   - Use async processing
   - Batch operations

## Related Documentation

- [EventLoop Architecture](./eventloop/ARCHITECTURE.md)
- [Envelope Architecture](./ENVELOPE_ARCHITECTURE.md)
- [EventBus API](./eventbus.go)
