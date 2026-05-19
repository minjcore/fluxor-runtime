# FloxID - Universal Routing Identifier (Stream-Based)

## Overview

**FloxID** is a universal routing identifier used for EventLoop routing in Fluxor. In Event Sourcing patterns, FloxID typically represents a **Stream ID**, ensuring all events for the same stream route to the same EventLoop for sequential processing and cache locality.

## Concept

FloxID is designed to be a **universal routing identifier** that can represent:
- **Stream ID** (Event Sourcing - primary use case)
- **Aggregate ID** (DDD - Domain-Driven Design)
- **Entity ID** (Entity-based routing)
- **Partition ID** (Sharding)
- **Any domain-specific identifier** for routing

**In Event Sourcing**: FloxID = Stream ID. All events for the same stream must route to the same EventLoop to maintain order and enable sequential processing.

## Priority in Key Extraction

FloxID from **context** has **highest priority** in the routing key extraction chain:

```
Priority Order:
1. FloxID from context    (highest priority - checked first)
2. X-Route-Key            (explicit routing key in headers)
3. X-Flox-ID              (universal routing ID in headers)
4. X-User-ID              (user-based routing)
5. X-Session-ID           (session-based routing)
6. X-Request-ID           (request-based routing - lowest priority)
```

**Important**: FloxID from context is checked at EventBus level and set in `Event.Key` before dispatch, ensuring it has highest priority over all headers.

## Usage

### Setting FloxID in Context (Stream ID)

```go
import "github.com/fluxorio/fluxor/pkg/core"

// Set FloxID = Stream ID in context (HIGHEST PRIORITY)
streamID := "order-123"
ctx := core.WithFloxID(ctx, streamID)

// Publish - FloxID automatically extracted from context and used for routing
// All events for this stream route to same EventLoop (sequential processing)
err := eb.Publish("stream.events", payload)
```

### Setting FloxID in Headers

```go
headers := map[string]string{
    "X-Flox-ID": "aggregate-123",
}
msg := newMessage(jsonBody, headers, "", eb)
```

### Getting FloxID from Context

```go
floxid := core.GetFloxID(ctx)
if floxid != "" {
    // Use FloxID for routing
}
```

## Examples

### Event Sourcing Stream Routing (Primary Use Case)

```go
// Stream ID routing - FloxID = Stream ID
func (s *EventStore) AppendEvent(ctx context.Context, streamID string, event Event) error {
    // Set FloxID = Stream ID for stream-based routing
    // All events for the same stream route to same EventLoop (sequential processing)
    ctx = core.WithFloxID(ctx, streamID)
    
    // Publish event - automatically routes to EventLoop based on stream ID
    return eb.Publish("stream.events", event)
}

// Example: Order stream
func (s *OrderService) ProcessOrder(ctx context.Context, orderID string) error {
    // Stream ID = Order ID
    streamID := fmt.Sprintf("order-%s", orderID)
    ctx = core.WithFloxID(ctx, streamID)
    
    // All events for this order stream route to same EventLoop
    // Ensures sequential processing and event ordering
    return eb.Publish("order.events", orderEvent)
}
```

### DDD Aggregate Routing

```go
// Aggregate ID routing (can also use FloxID)
func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) error {
    aggregateID := req.GetOrderId()
    
    // Set FloxID = Aggregate ID for aggregate-based routing
    ctx = core.WithFloxID(ctx, aggregateID)
    
    // All events for this aggregate route to same EventLoop
    return eb.Publish("order.created", orderData)
}
```

### Entity-Based Routing

```go
// Entity ID routing
func (s *UserService) UpdateUser(ctx context.Context, userID string, data UserData) error {
    // Set FloxID for entity-based routing
    ctx = core.WithFloxID(ctx, userID)
    
    // All events for this entity route to same EventLoop
    return eb.Publish("user.updated", data)
}
```

## Benefits

1. **Cache Locality**: Same FloxID → same EventLoop → same CPU core
2. **Consistent Hashing**: Same FloxID always routes to same EventLoop
3. **Domain-Driven**: Aligns with DDD, Event Sourcing, and domain concepts
4. **Flexible**: Can represent any domain identifier
5. **High Priority**: Takes precedence over UserID/SessionID for routing

## Implementation Details

### Context Integration

FloxID is stored in Go's `context.Context` using a typed key:

```go
var FloxIDKey = struct{ name string }{"floxid"}

func WithFloxID(ctx context.Context, floxid string) context.Context {
    return context.WithValue(ctx, FloxIDKey, floxid)
}
```

### EventBus Integration

EventBus automatically extracts FloxID from context and sets it in message headers:

```go
// In eventbus_impl.go
if floxid := GetFloxID(eb.ctx); floxid != "" {
    headers["X-Flox-ID"] = floxid
}
```

### Dispatcher Integration

Dispatcher extracts FloxID from headers with high priority:

```go
// In eventloop/key.go
keyOrder := []string{
    "X-Route-Key",  // Highest priority
    "X-Flox-ID",    // Universal routing ID
    "X-User-ID",     // User-based
    // ...
}
```

## When to Use FloxID

✅ **Use FloxID when (primary use cases):**
- **Stream ID routing** (Event Sourcing - main use case)
- Routing by aggregate ID (DDD)
- Routing by entity ID
- Routing by partition ID
- Any domain-specific identifier that needs consistent routing

❌ **Don't use FloxID when:**
- Routing by user (use X-User-ID)
- Routing by session (use X-Session-ID)
- Routing by request (use X-Request-ID)
- Need explicit routing key (use X-Route-Key)

## Stream-Based Routing (Event Sourcing)

**FloxID = Stream ID** ensures:
- Same stream → same EventLoop
- Sequential processing of events
- Event ordering preserved
- Cache locality (same CPU core)
- No race conditions

## Migration from UserID

If you're currently using UserID for routing but want to use FloxID:

```go
// Before
headers := map[string]string{
    "X-User-ID": userID,
}

// After (if userID is actually an aggregate/entity ID)
ctx := core.WithFloxID(ctx, userID)
// Or
headers := map[string]string{
    "X-Flox-ID": userID,  // More semantic
}
```

## Related Documentation

- [EventBus Flow](./EVENTBUS_FLOW.md)
- [EventLoop Architecture](./eventloop/ARCHITECTURE.md)
- [Envelope Architecture](./ENVELOPE_ARCHITECTURE.md)
