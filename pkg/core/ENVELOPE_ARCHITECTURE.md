# Envelope Architecture

## Overview

Envelope pattern provides unified message format across Internal Bus and NATS, enabling consistent routing through EventLoopGroup.

## Architecture

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

## Envelope Structure

```go
type Envelope struct {
    Topic string              // Event bus address/topic
    Key   string              // Routing key (extracted from Meta if empty)
    Data  []byte              // Raw payload (protobuf/JSON/msgpack)
    Meta  map[string]string   // Metadata (headers, tracing, etc.)
}
```

## Key Extraction Priority

Routing key is extracted in this order:
1. `Envelope.Key` field (explicit - highest priority)
2. `Meta["X-Route-Key"]` (explicit routing key)
3. `Meta["X-Flox-ID"]` (universal routing ID - aggregate ID, stream ID, entity ID, etc.)
4. `Meta["X-User-ID"]` (user-based routing)
5. `Meta["X-Session-ID"]` (session-based routing)
6. `Meta["X-Request-ID"]` (request-based routing - lowest priority)

If no key found → round-robin routing.

## Bus Interface

```go
type Bus interface {
    Publish(ctx context.Context, env *Envelope) error
    Subscribe(topic string, fn func(ctx context.Context, env *Envelope) error) (unsubscribe func() error, err error)
    Send(ctx context.Context, env *Envelope) error
    Request(ctx context.Context, env *Envelope, timeout time.Duration) (*Envelope, error)
    Close() error
}
```

## Dispatcher Interface

```go
type DispatcherInterface interface {
    DispatchEnvelope(ctx context.Context, env *Envelope) error
}
```

## Implementation Strategy

### Internal Bus
- `Publish(env)` → Extract key → Dispatcher → EventLoop → Handler
- Uses EventLoopGroup for routing

### NATS Bus
- `Publish(env)` → NATS.Publish(topic, data)
- Subscribe callback → Extract key → Dispatcher → EventLoop → Handler

## Usage Example

```go
// Create envelope
env := core.NewEnvelope(
    "user.events",
    "user-123",  // Routing key
    data,        // []byte payload
    map[string]string{
        "X-Request-ID": "req-456",
    },
)

// Publish via Bus
bus.Publish(ctx, env)

// Or via EventBus (with adapter)
eb.PublishEnvelope(ctx, env)
```

## Migration Path

1. **Phase 1**: Add Envelope and Bus interface (current)
2. **Phase 2**: EventBus implements Bus interface with Envelope
3. **Phase 3**: NATS/JetStream implement Bus interface
4. **Phase 4**: Deprecate old EventBus methods (future)

## Benefits

- ✅ Unified message format (Internal + NATS)
- ✅ Consistent routing semantics
- ✅ Type-safe envelope structure
- ✅ Easy to add new bus implementations
- ✅ Backward compatible (EventBus still works)

