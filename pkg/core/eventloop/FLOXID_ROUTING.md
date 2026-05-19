# FloxID Routing với EventLoop

## Overview

FloxID routing đảm bảo messages với cùng FloxID luôn route đến cùng EventLoop, đảm bảo cache locality và sequential processing.

## Architecture

```
gRPC Request
    │
    │ Extract FloxID
    ▼
Dispatcher
    │
    │ hash(FloxID) % numLoops
    ▼
EventLoop[hash % N]
    │
    │ Sequential processing
    ▼
Handler (same CPU core)
```

## Auto Assignment theo CPU

### 1. Số lượng EventLoop = số CPU

```go
numCPU := runtime.GOMAXPROCS(0)
// EventLoopGroup tự động tạo N loops = numCPU
```

### 2. EventLoop Structure

```go
type EventLoop struct {
    id      int           // Loop ID (0 to N-1)
    queue   chan *Event   // Bounded queue (default: 4096)
    ctx     context.Context
    cancel  context.CancelFunc
    config  EventLoopConfig
    metrics *LoopMetrics
}
```

### 3. Khởi tạo EventLoop Pool

```go
// Tự động trong NewEventLoopGroup
numLoops := runtime.GOMAXPROCS(0)  // Auto: số CPU cores
loops := make([]*EventLoop, numLoops)

for i := 0; i < numLoops; i++ {
    loops[i] = NewEventLoop(i, ctx, config)
    // Mỗi loop chạy trên 1 goroutine riêng
}
```

### 4. Pin Goroutine vào CPU (Optional)

```go
func (l *EventLoop) run() {
    // Optional: Pin goroutine to CPU core
    if l.config.CPUAffinity {
        runtime.LockOSThread()
        defer runtime.UnlockOSThread()
    }

    for {
        select {
        case event := <-l.queue:
            event.Handler(l.ctx, event)
        case <-l.ctx.Done():
            return
        }
    }
}
```

**Enable CPU Affinity:**
```go
config := eventloop.EventLoopConfig{
    CPUAffinity: true,  // Pin goroutine to CPU core
}
```

## Dispatcher - Hash Routing

### Hash Routing với FloxID

```go
type Dispatcher struct {
    loops []*EventLoop
}

func (d *Dispatcher) Dispatch(key string, ev *Event) {
    // Hash routing: same key → same loop
    idx := RouteKey(key, len(d.loops))
    d.loops[idx].queue <- ev
}
```

### Key Extraction Priority

```go
// Priority order:
1. X-Route-Key    (explicit - highest)
2. X-Flox-ID      (universal routing ID)
3. X-User-ID      (user-based)
4. X-Session-ID   (session-based)
5. X-Request-ID   (request-based - lowest)
```

### Hash Function

```go
// FNV-1a hash for fast, good distribution
func RouteKey(key string, numLoops int) int {
    hash := HashKey(key)  // FNV-1a
    return int(hash % uint32(numLoops))
}
```

## gRPC → EventBus Mapping với FloxID

### Example: gRPC Handler

```go
func (s *OrderService) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
    // Extract FloxID (aggregate ID)
    aggregateID := req.GetOrderId()
    
    // Set FloxID in context
    ctx = core.WithFloxID(ctx, aggregateID)
    ctx = core.WithRequestID(ctx, req.GetRequestId())
    
    // Publish - automatically routes by FloxID
    err := eb.Publish("order.created", orderData)
    if err != nil {
        return nil, err
    }
    
    return &pb.CreateOrderResponse{OrderId: aggregateID}, nil
}
```

### Flow

```
1. gRPC Request arrives
   ↓
2. Extract FloxID from request
   ↓
3. Set FloxID in context: core.WithFloxID(ctx, "aggregate-123")
   ↓
4. EventBus.Publish() extracts FloxID from context
   ↓
5. Set header: X-Flox-ID = "aggregate-123"
   ↓
6. Dispatcher extracts key: X-Flox-ID
   ↓
7. Hash routing: hash("aggregate-123") % numLoops
   ↓
8. Route to EventLoop[hash % N]
   ↓
9. Sequential processing on same CPU core
```

## Benefits

### 1. Cache Locality

Same FloxID → same EventLoop → same CPU core:
- CPU cache stays hot
- Lower latency
- Better performance

### 2. Consistent Hashing

Same FloxID always routes to same EventLoop:
- Predictable routing
- Stateful operations stay on same CPU
- Single-writer guarantee

### 3. Sequential Processing

Each EventLoop processes events sequentially:
- No locks needed
- Race-free access
- Predictable execution order

## Configuration

### Default Configuration

```go
config := eventloop.DefaultEventLoopConfig()
// NumLoops: 0 (auto: GOMAXPROCS)
// QueueSize: 4096
// CPUAffinity: false
// Backpressure: BackpressureBlock
// Metrics: true
```

### With CPU Affinity

```go
config := eventloop.EventLoopConfig{
    NumLoops:     0,  // Auto: GOMAXPROCS
    QueueSize:    4096,
    CPUAffinity:  true,  // Pin goroutine to CPU core
    Backpressure: eventloop.BackpressureBlock,
    Metrics:      true,
}
```

## Usage Examples

### DDD Aggregate Routing

```go
// Aggregate ID = FloxID
func (s *OrderService) ProcessOrder(ctx context.Context, orderID string) error {
    ctx = core.WithFloxID(ctx, orderID)
    
    // All events for this aggregate route to same EventLoop
    return eb.Publish("order.processed", orderData)
}
```

### Event Sourcing Stream Routing

```go
// Stream ID = FloxID
func (s *EventStore) AppendEvent(ctx context.Context, streamID string, event Event) error {
    ctx = core.WithFloxID(ctx, streamID)
    
    // All events for this stream route to same EventLoop
    return eb.Publish("stream.events", event)
}
```

### Entity-Based Routing

```go
// Entity ID = FloxID
func (s *UserService) UpdateUser(ctx context.Context, userID string, data UserData) error {
    ctx = core.WithFloxID(ctx, userID)
    
    // All events for this entity route to same EventLoop
    return eb.Publish("user.updated", data)
}
```

## Performance Characteristics

### Latency

- **Dispatcher routing**: < 1ms p99
- **EventLoop dispatch**: < 100μs p99
- **Hash calculation**: < 1μs

### Throughput

- **Per EventLoop**: ~100k events/sec
- **Total (N loops)**: N × 100k events/sec
- **Bottleneck**: Handler execution time

### Scalability

- **Horizontal**: Add more EventLoops (up to CPU cores)
- **Vertical**: Optimize handler execution
- **Backpressure**: Bounded queues prevent OOM

## Best Practices

### 1. Use FloxID for Domain Identifiers

✅ **Good:**
```go
ctx = core.WithFloxID(ctx, aggregateID)  // Aggregate ID
ctx = core.WithFloxID(ctx, streamID)     // Stream ID
ctx = core.WithFloxID(ctx, entityID)     // Entity ID
```

❌ **Bad:**
```go
ctx = core.WithFloxID(ctx, userID)  // Use X-User-ID instead
ctx = core.WithFloxID(ctx, sessionID)  // Use X-Session-ID instead
```

### 2. Consistent FloxID Format

Use consistent format for FloxID:
```go
// Good: Consistent format
aggregateID := fmt.Sprintf("order-%s", orderID)
streamID := fmt.Sprintf("stream-%s", streamName)
```

### 3. CPU Affinity for Low Latency

Enable CPU affinity for low-latency requirements:
```go
config.CPUAffinity = true  // Pin goroutine to CPU core
```

## Troubleshooting

### Messages not routing correctly

1. Check FloxID extraction:
   ```go
   floxid := core.GetFloxID(ctx)
   fmt.Printf("FloxID: %s\n", floxid)
   ```

2. Verify EventLoopGroup is enabled:
   ```go
   if eb.loopGroup == nil {
       // EventLoopGroup not enabled
   }
   ```

3. Check routing key:
   ```go
   key := extractor(headers, address, body)
   fmt.Printf("Routing key: %s\n", key)
   ```

### High latency

1. Enable CPU affinity:
   ```go
   config.CPUAffinity = true
   ```

2. Check EventLoop queue length:
   ```go
   stats := eb.loopGroup.Stats()
   for _, stat := range stats {
       fmt.Printf("Queue length: %d\n", stat.QueueLength)
   }
   ```

## Related Documentation

- [FloxID Concept](../FLOXID.md)
- [EventBus Flow](../EVENTBUS_FLOW.md)
- [EventLoop Architecture](./ARCHITECTURE.md)
