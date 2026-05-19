# Stream-Based Routing với FloxID

## Overview

Trong Event Sourcing, **FloxID = Stream ID**. Tất cả events của cùng một stream phải route đến cùng một EventLoop để đảm bảo:
- Sequential processing (tuần tự)
- Event ordering (thứ tự events được giữ nguyên)
- Cache locality (cùng CPU core)
- No race conditions (không có race condition)

## Architecture

```
Stream Event
    │
    │ Extract Stream ID = FloxID
    ▼
EventBus.Publish()
    │
    │ Set FloxID in context → Event.Key
    ▼
Dispatcher
    │
    │ hash(Stream ID) % numLoops
    ▼
EventLoop[hash % N]
    │
    │ Sequential processing (same stream = same loop)
    ▼
Handler (preserves event order)
```

## Stream ID = FloxID

### Concept

```go
// FloxID = Stream ID trong Event Sourcing
streamID := "order-123"
ctx = core.WithFloxID(ctx, streamID)

// Tất cả events của stream này route đến cùng EventLoop
eb.Publish("order.created", event1)
eb.Publish("order.updated", event2)
eb.Publish("order.completed", event3)
// → Tất cả route đến cùng EventLoop, đảm bảo thứ tự
```

## Auto Assignment theo CPU

### 1. Số lượng EventLoop = số CPU

```go
// Tự động tạo N EventLoops = số CPU cores
numLoops := runtime.GOMAXPROCS(0)  // Ví dụ: 8 CPU cores = 8 loops
```

### 2. EventLoop Structure

```go
type EventLoop struct {
    id      int           // Loop ID (0 to N-1)
    queue   chan *Event   // Bounded queue (default: 4096)
    ctx     context.Context
    cancel  context.CancelFunc
    config  EventLoopConfig
}
```

### 3. Khởi tạo EventLoop Pool

```go
// Tự động trong NewEventLoopGroup
loops := make([]*EventLoop, numLoops)
for i := 0; i < numLoops; i++ {
    loops[i] = NewEventLoop(i, ctx, config)
    // Mỗi loop chạy trên 1 goroutine riêng
}
```

### 4. Pin Goroutine vào CPU (Optional nhưng mạnh)

```go
func (l *EventLoop) run() {
    // Pin goroutine to CPU core
    if l.config.CPUAffinity {
        runtime.LockOSThread()
        defer runtime.UnlockOSThread()
    }

    for {
        select {
        case event := <-l.queue:
            event.Handler(l.ctx, event)  // Sequential processing
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

## Dispatcher - Hash Routing với Stream ID

### Hash Routing

```go
// Dispatcher routes bằng hash của Stream ID
func (d *Dispatcher) Dispatch(ctx context.Context, event *Event) error {
    // Event.Key chứa Stream ID (FloxID từ context)
    key := event.Key  // Stream ID
    
    // Hash routing: same stream → same loop
    idx := RouteKey(key, len(d.loops))
    d.loops[idx].queue <- event
}
```

### Hash Function

```go
// FNV-1a hash cho distribution tốt
func RouteKey(streamID string, numLoops int) int {
    hash := HashKey(streamID)  // FNV-1a
    return int(hash % uint32(numLoops))
}
```

**Properties:**
- Same stream ID → same hash → same EventLoop
- Consistent hashing
- Good distribution across loops

## gRPC → EventBus Mapping với Stream ID

### Example: gRPC Handler với Stream ID

```go
func (s *EventStoreService) AppendEvent(ctx context.Context, req *pb.AppendEventRequest) (*pb.AppendEventResponse, error) {
    // Extract Stream ID từ request
    streamID := req.GetStreamId()  // FloxID = Stream ID
    
    // Set Stream ID (FloxID) in context - HIGHEST PRIORITY
    ctx = core.WithFloxID(ctx, streamID)
    ctx = core.WithRequestID(ctx, req.GetRequestId())
    
    // Publish event - automatically routes by Stream ID
    err := eb.Publish("stream.events", req.GetEvent())
    if err != nil {
        return nil, err
    }
    
    return &pb.AppendEventResponse{Success: true}, nil
}
```

### Flow với Stream ID

```
1. gRPC Request arrives
   streamID = "order-123"
   ↓
2. Set FloxID in context: core.WithFloxID(ctx, "order-123")
   ↓
3. EventBus.Publish() extracts FloxID from context
   ↓
4. Set Event.Key = "order-123" (highest priority)
   ↓
5. Dispatcher routes: hash("order-123") % 8 → EventLoop[3]
   ↓
6. Event processed sequentially on EventLoop[3]
   ↓
7. Next event for "order-123" → same EventLoop[3]
   (Event ordering preserved)
```

## Event Ordering Guarantee

### Sequential Processing

Cùng stream ID → cùng EventLoop → sequential processing:

```go
// Event 1
ctx = core.WithFloxID(ctx, "order-123")
eb.Publish("order.created", event1)
// Routes to EventLoop[hash("order-123") % 8]

// Event 2 (same stream)
ctx = core.WithFloxID(ctx, "order-123")
eb.Publish("order.updated", event2)
// Routes to SAME EventLoop[hash("order-123") % 8]
// Processed AFTER event1 (sequential)

// Event 3 (same stream)
ctx = core.WithFloxID(ctx, "order-123")
eb.Publish("order.completed", event3)
// Routes to SAME EventLoop
// Processed AFTER event2 (sequential)
```

**Guarantee:**
- Events của cùng stream được process tuần tự
- Event ordering được preserve
- No race conditions

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

### Stream-Optimized Configuration

```go
config := eventloop.EventLoopConfig{
    NumLoops:     0,  // Auto: GOMAXPROCS (1 loop per CPU core)
    QueueSize:    8192,  // Larger queue for high-throughput streams
    CPUAffinity:  true,  // Pin goroutine to CPU core (low latency)
    Backpressure: eventloop.BackpressureBlock,  // Preserve ordering
    Metrics:      true,  // Monitor stream processing
}
```

## Usage Examples

### Event Store với Stream Routing

```go
type EventStore struct {
    eventBus core.EventBus
}

func (s *EventStore) AppendToStream(ctx context.Context, streamID string, events []Event) error {
    // Set Stream ID (FloxID) in context
    ctx = core.WithFloxID(ctx, streamID)
    
    // Append events - all route to same EventLoop
    for _, event := range events {
        if err := s.eventBus.Publish("stream.append", event); err != nil {
            return err
        }
    }
    
    // Events processed sequentially, order preserved
    return nil
}
```

### Order Processing với Stream ID

```go
func (s *OrderService) ProcessOrder(ctx context.Context, orderID string) error {
    // Stream ID = Order ID
    streamID := fmt.Sprintf("order-%s", orderID)
    ctx = core.WithFloxID(ctx, streamID)
    
    // All order events route to same EventLoop
    events := []OrderEvent{
        {Type: "created", Data: orderData},
        {Type: "validated", Data: validationData},
        {Type: "processed", Data: processData},
    }
    
    for _, event := range events {
        if err := s.eventBus.Publish("order.events", event); err != nil {
            return err
        }
    }
    
    // Events processed in order on same EventLoop
    return nil
}
```

## Benefits

### 1. Event Ordering

Same stream → same EventLoop → sequential processing:
- Events processed in order
- No out-of-order events
- Deterministic behavior

### 2. Cache Locality

Same stream → same EventLoop → same CPU core:
- CPU cache stays hot
- Lower latency
- Better performance

### 3. Consistent Hashing

Same stream ID always routes to same EventLoop:
- Predictable routing
- Easy to debug
- Stateless routing

## Best Practices

### 1. Use Stream ID as FloxID

```go
// ✅ Good: Stream ID = FloxID
streamID := fmt.Sprintf("order-%s", orderID)
ctx = core.WithFloxID(ctx, streamID)

// ❌ Bad: Random or request-based FloxID
ctx = core.WithFloxID(ctx, uuid.New().String())  // Breaks ordering
```

### 2. Consistent Stream ID Format

```go
// ✅ Good: Consistent format
streamID := fmt.Sprintf("order-%s", orderID)
streamID := fmt.Sprintf("user-%s", userID)
streamID := fmt.Sprintf("product-%s", productID)

// ❌ Bad: Inconsistent format
streamID := orderID  // No prefix
streamID := fmt.Sprintf("ORDER-%s", orderID)  // Different case
```

### 3. CPU Affinity for Low Latency

```go
// Enable CPU affinity for low-latency stream processing
config.CPUAffinity = true  // Pin goroutine to CPU core
```

## Troubleshooting

### Events out of order

1. Check Stream ID is consistent:
   ```go
   streamID := core.GetFloxID(ctx)
   fmt.Printf("Stream ID: %s\n", streamID)
   ```

2. Verify routing:
   ```go
   key := event.Key  // Should be Stream ID
   idx := RouteKey(key, numLoops)
   fmt.Printf("Routes to EventLoop[%d]\n", idx)
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
       fmt.Printf("EventLoop[%d] queue: %d\n", stat.LoopID, stat.QueueLength)
   }
   ```

## Related Documentation

- [FloxID Concept](../FLOXID.md)
- [EventBus Flow](../EVENTBUS_FLOW.md)
- [EventLoop Architecture](./ARCHITECTURE.md)
