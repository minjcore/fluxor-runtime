# EventLoop Package

EventLoop package provides CPU-based event routing for the Fluxor framework, enabling optimal cache locality and reduced lock contention.

## Overview

The EventLoop package implements an EventLoopGroup pattern where:
- N event loops are created (N = GOMAXPROCS by default)
- Each loop runs on a single goroutine
- Events are routed to loops based on routing key hash
- Same key always routes to same loop (consistent hashing)

## Architecture

```
Ingress (gRPC/NATS/HTTP)
    ↓
Dispatcher (hash routing)
    ↓
EventLoopGroup (N loops = CPU cores)
    ↓
Executor (handler execution)
```

## Usage

### Basic Usage

```go
import "github.com/fluxorio/fluxor/pkg/core/eventloop"

ctx := context.Background()
config := eventloop.DefaultEventLoopConfig()
group, err := eventloop.NewEventLoopGroup(ctx, config)
if err != nil {
    // handle error
}
defer group.Close()

// Dispatch event
event := &eventloop.Event{
    Key:     "user-123",
    Address: "user.events",
    Body:    data,
    Headers: map[string]string{"X-Route-Key": "user-123"},
    Handler: func(ctx context.Context, ev *eventloop.Event) error {
        // Process event
        return nil
    },
}

err = group.Dispatch(ctx, event)
```

### With EventBus

```go
import "github.com/fluxorio/fluxor/pkg/core"

config := core.EventBusConfig{
    UseEventLoopGroup: true,
    EventLoopConfig: eventloop.EventLoopConfig{
        NumLoops:     0, // Auto: GOMAXPROCS
        QueueSize:    4096,
        CPUAffinity:  false,
        Backpressure: eventloop.BackpressureBlock,
        Metrics:      true,
    },
}

eb := core.NewEventBusWithConfig(ctx, gocmd, config)
```

### With NATS/JetStream

```go
natsConfig := clusterbus.ClusterNATSConfig{
    URL: "nats://localhost:4222",
    UseEventLoopGroup: true,
    EventLoopConfig: eventloop.DefaultEventLoopConfig(),
}

eb, err := clusterbus.NewClusterEventBusNATS(ctx, gocmd, natsConfig)
```

## Configuration

### EventLoopConfig

- `NumLoops`: Number of event loops (0 = GOMAXPROCS)
- `QueueSize`: Maximum queue size per loop (default: 4096)
- `CPUAffinity`: Pin goroutines to CPU cores (default: false)
- `Backpressure`: Policy when queue is full:
  - `BackpressureBlock`: Wait (default)
  - `BackpressureDrop`: Drop message
- `Metrics`: Enable per-loop metrics (default: true)

## Key Extraction

Routing key is extracted from message headers in priority order:
1. `X-Route-Key`
2. `X-User-ID`
3. `X-Session-ID`
4. `X-Request-ID`

If no key is found, events are routed using round-robin.

## Metrics

Per-loop metrics are available:
- Queue length
- Dropped messages
- Processed messages
- Average latency
- Throughput

Metrics can be exposed via Prometheus using `RegisterMetrics()`.

## Performance

- **Latency**: < 1ms p99 for dispatch
- **Throughput**: > 100k messages/sec per loop
- **Memory**: < 1MB overhead per loop

## Thread Safety

- EventLoopGroup is thread-safe for concurrent dispatch
- Each EventLoop processes events sequentially (single goroutine)
- No locks needed within event handlers (single-writer guarantee)

## Best Practices

1. **Always provide routing keys** for stateful operations
2. **Use consistent keys** (e.g., user ID) to ensure same-loop routing
3. **Monitor queue lengths** to detect backpressure
4. **Enable metrics** in production for observability
5. **Start with default config** and tune based on workload

## Examples

See test files for complete examples:
- `group_test.go`: Basic usage
- `dispatcher_test.go`: Routing examples
- `integration_test.go`: Concurrent dispatch patterns

