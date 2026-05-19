# Achieving 200k RPS with EventBus

This guide shows how to configure EventBus to achieve **200,000 requests per second (RPS)**.

## Performance Targets

- **Goal**: 200,000 RPS
- **Per Loop**: > 100k messages/sec (EventLoopGroup)
- **Required Loops**: 2+ (typically 4-8 for safety margin)

## Configuration

### 1. Enable EventLoopGroup (Required)

EventLoopGroup provides CPU-based routing and can achieve > 100k RPS per loop.

```go
package main

import (
    "context"
    "runtime"
    
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/core/eventloop"
)

func main() {
    ctx := context.Background()
    gocmd := core.NewGoCMD(ctx)
    defer gocmd.Close()
    
    // High-throughput EventBus configuration
    config := core.EventBusConfig{
        UseEventLoopGroup: true, // REQUIRED for 200k RPS
        EventLoopConfig: eventloop.EventLoopConfig{
            // Number of event loops (one per CPU core)
            // For 200k RPS: Need at least 2 loops, recommend 4-8
            NumLoops: runtime.GOMAXPROCS(0), // Auto: use all CPU cores
            
            // Queue size per loop (larger = more buffering)
            // For 200k RPS: 8192-16384 recommended
            QueueSize: 16384,
            
            // CPU affinity (optional, may improve cache locality)
            CPUAffinity: false, // Let Go scheduler manage
            
            // Backpressure policy
            Backpressure: eventloop.BackpressureBlock, // Wait when queue full
            
            // Enable metrics for monitoring
            Metrics: true,
        },
    }
    
    eb := core.NewEventBusWithConfig(ctx, gocmd, config)
    defer eb.Close()
    
    // Use EventBus...
}
```

### 2. Optimize Executor Configuration

The default executor (10 workers, 1000 queue) may be insufficient. You can't directly configure it, but EventLoopGroup handles routing efficiently.

**Note**: The executor is used for handler execution. With EventLoopGroup, routing is handled by loops, and handlers execute in the executor pool.

### 3. Use Protobuf (Not JSON)

Protobuf encoding/decoding is **2-3x faster** than JSON.

```go
// ❌ SLOW: JSON
err := eb.Send("payments.authorize", map[string]interface{}{
    "paymentID": "pay_123",
    "amount": 100.0,
})

// ✅ FAST: Protobuf
paymentReq := &PaymentRequest{
    PaymentId: "pay_123",
    Amount:    100.0,
}
err := eb.Send("payments.authorize", paymentReq)
```

### 4. Optimize Handler Execution Time

**Critical**: Handlers must complete in **< 20µs** to maintain throughput.

```go
consumer := eb.Consumer("payments.authorize")
consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    // ✅ FAST: Type assertion + simple logic (< 20µs)
    body, ok := msg.Body().([]byte)
    if !ok {
        return msg.Reply(&PaymentReply{OK: false})
    }
    
    // ✅ FAST: Protobuf decode (faster than JSON)
    var req PaymentRequest
    if err := msg.DecodeBody(&req); err != nil {
        return msg.Reply(&PaymentReply{OK: false})
    }
    
    // ✅ FAST: Simple processing
    authID := "auth_" + req.PaymentId
    
    // ✅ FAST: Reply (non-blocking)
    return msg.Reply(&PaymentReply{OK: true, AuthId: authID})
    
    // ❌ SLOW: Blocking operations (> 20µs)
    // - Database queries
    // - Network calls
    // - Heavy computation
    // Use SubmitBlocking() for these!
})
```

### 5. Use Multiple Consumers (Load Distribution)

For high throughput, register multiple consumers to distribute load:

```go
// Register 4 consumers for load balancing
for i := 0; i < 4; i++ {
    consumer := eb.Consumer("payments.authorize")
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        // Fast handler (< 20µs)
        return processPayment(msg)
    })
}
```

### 6. Provide Routing Keys (For Stateful Operations)

For stateful operations, provide routing keys to ensure same-key routes to same loop:

```go
// Set routing key in context
ctx := core.WithRequestID(ctx, paymentID)

// Or in headers
headers := map[string]string{
    "X-Route-Key": paymentID, // Highest priority
    // Or: "X-User-ID": userID,
    // Or: "X-Session-ID": sessionID,
}

// Send with routing
err := eb.SendWithContext(ctx, "payments.authorize", req)
```

## Complete Example

```go
package main

import (
    "context"
    "runtime"
    "time"
    
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/core/eventloop"
)

func main() {
    ctx := context.Background()
    gocmd := core.NewGoCMD(ctx)
    defer gocmd.Close()
    
    // High-throughput EventBus configuration
    config := core.EventBusConfig{
        UseEventLoopGroup: true,
        EventLoopConfig: eventloop.EventLoopConfig{
            NumLoops:     runtime.GOMAXPROCS(0), // Use all CPU cores
            QueueSize:    16384,                 // Large queue for buffering
            CPUAffinity:  false,                 // Let Go scheduler manage
            Backpressure: eventloop.BackpressureBlock,
            Metrics:      true,
        },
    }
    
    eb := core.NewEventBusWithConfig(ctx, gocmd, config)
    defer eb.Close()
    
    // Register multiple consumers for load distribution
    for i := 0; i < 4; i++ {
        consumer := eb.Consumer("payments.authorize")
        consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
            // Fast handler: < 20µs
            body, ok := msg.Body().([]byte)
            if !ok {
                return msg.Reply(&PaymentReply{OK: false})
            }
            
            // Decode protobuf (fast)
            var req PaymentRequest
            if err := msg.DecodeBody(&req); err != nil {
                return msg.Reply(&PaymentReply{OK: false})
            }
            
            // Simple processing
            authID := "auth_" + req.PaymentId
            
            // Reply (non-blocking)
            return msg.Reply(&PaymentReply{OK: true, AuthId: authID})
        })
    }
    
    // Send messages with routing keys
    for i := 0; i < 200000; i++ {
        req := &PaymentRequest{
            PaymentId: fmt.Sprintf("pay_%d", i),
            Amount:    100.0,
        }
        
        // Set routing key for consistent routing
        ctx := core.WithRequestID(ctx, req.PaymentId)
        
        err := eb.SendWithContext(ctx, "payments.authorize", req)
        if err != nil {
            // Handle error
        }
    }
}
```

## Performance Tuning

### 1. Monitor Queue Lengths

```go
// Get EventLoopGroup stats
if ebWithLoop, ok := eb.(*eventBus); ok && ebWithLoop.loopGroup != nil {
    stats := ebWithLoop.loopGroup.Stats()
    for _, stat := range stats {
        fmt.Printf("Loop %d: Queue=%d, Throughput=%.0f msg/s\n",
            stat.LoopID, stat.QueueLength, stat.Throughput)
        
        // Alert if queue is getting full
        if stat.QueueLength > 10000 {
            fmt.Printf("WARNING: Loop %d queue is high: %d\n",
                stat.LoopID, stat.QueueLength)
        }
    }
}
```

### 2. Adjust Queue Size

If queues are consistently full:
- **Increase QueueSize**: 16384 → 32768
- **Trade-off**: More memory usage, but better buffering

### 3. Adjust Number of Loops

If single loop is bottleneck:
- **Increase NumLoops**: More loops = more parallel processing
- **Trade-off**: More CPU usage, but better distribution

### 4. Use CPU Affinity (Advanced)

For maximum performance on dedicated hardware:

```go
config := eventloop.EventLoopConfig{
    NumLoops:    runtime.GOMAXPROCS(0),
    QueueSize:   16384,
    CPUAffinity: true, // Pin loops to CPU cores
    Metrics:     true,
}
```

**Note**: CPU affinity may not help on virtualized environments.

## Benchmarking

### Run Benchmarks

```bash
# Benchmark EventBus Send (Protobuf)
go test -bench=BenchmarkEventBus_Send_Protobuf -benchmem ./pkg/core

# Benchmark EventBus Send (Parallel)
go test -bench=BenchmarkEventBus_Send_Protobuf_Parallel -benchmem ./pkg/core

# Benchmark EventLoopGroup Dispatch
go test -bench=BenchmarkEventLoopGroup_Dispatch -benchmem ./pkg/core/eventloop
```

### Expected Results

With EventLoopGroup enabled:
- **Single loop**: > 100k RPS
- **4 loops**: > 400k RPS (theoretical)
- **8 loops**: > 800k RPS (theoretical)

**Real-world**: Expect 60-80% of theoretical due to overhead.

## Troubleshooting

### Issue: Not Reaching 200k RPS

**Solutions**:
1. **Enable EventLoopGroup**: `UseEventLoopGroup: true` (REQUIRED)
2. **Increase NumLoops**: Use all CPU cores
3. **Increase QueueSize**: 16384 or higher
4. **Use Protobuf**: Not JSON
5. **Optimize handlers**: Keep < 20µs
6. **Multiple consumers**: Register 4+ consumers
7. **Check CPU usage**: Ensure CPU is not bottleneck

### Issue: High Latency

**Solutions**:
1. **Reduce QueueSize**: Smaller queues = lower latency
2. **Optimize handlers**: Faster execution = lower latency
3. **Use routing keys**: Better cache locality

### Issue: Queue Full Errors

**Solutions**:
1. **Increase QueueSize**: 16384 → 32768
2. **Add more consumers**: Distribute load
3. **Optimize handlers**: Faster processing
4. **Use BackpressureDrop**: Drop messages instead of blocking (if acceptable)

## Best Practices

1. ✅ **Always enable EventLoopGroup** for high throughput
2. ✅ **Use Protobuf** instead of JSON
3. ✅ **Keep handlers < 20µs** (use SubmitBlocking for heavy work)
4. ✅ **Register multiple consumers** for load distribution
5. ✅ **Provide routing keys** for stateful operations
6. ✅ **Monitor queue lengths** to detect backpressure
7. ✅ **Enable metrics** for observability
8. ✅ **Start with defaults** and tune based on workload

## Configuration Summary

| Setting | Default | 200k RPS Recommended |
|---------|---------|----------------------|
| `UseEventLoopGroup` | `false` | **`true`** (REQUIRED) |
| `NumLoops` | `GOMAXPROCS` | `GOMAXPROCS` (4-16) |
| `QueueSize` | `4096` | **`16384`** |
| `CPUAffinity` | `false` | `false` (or `true` on dedicated HW) |
| `Backpressure` | `Block` | `Block` |
| `Metrics` | `true` | `true` |
| Encoding | JSON | **Protobuf** |
| Handler Time | < 20µs | **< 20µs** |
| Consumers | 1 | **4+** |

## See Also

- `pkg/core/eventloop/README.md` - EventLoopGroup documentation
- `pkg/core/eventloop/ARCHITECTURE.md` - Architecture details
- `pkg/core/EVENTBUS_README.md` - EventBus usage guide
- `pkg/core/eventbus_bench_test.go` - Benchmark examples
