# EventBus Package

The `EventBus` package provides publish-subscribe and point-to-point messaging for Fluxor applications. It supports both JSON and protobuf message formats, CPU-based routing via EventLoopGroup, and integrates seamlessly with the Fluxor reactive runtime.

## Features

- **Publish-Subscribe**: Broadcast messages to all registered handlers
- **Point-to-Point**: Send messages to a single handler (load-balanced)
- **Request-Reply**: Synchronous request-response pattern with timeout support
- **Context-Aware Routing**: CPU-based routing using FloxID, UserID, and other routing keys
- **EventLoop Integration**: Optional EventLoopGroup for CPU-based message routing
- **Auto-Encoding**: Automatic protobuf or JSON encoding based on message type
- **Thread-Safe**: All operations are safe for concurrent use
- **Backpressure Handling**: Built-in queue management and mailbox overflow handling

## Installation

```go
import "github.com/fluxorio/fluxor/pkg/core"
```

EventBus is automatically created when you create a `GoCMD` instance:

```go
ctx := context.Background()
gocmd := core.NewGoCMD(ctx)
eventBus := gocmd.EventBus()
```

## Basic Usage

### Publish (Broadcast)

Publish sends a message to **all** handlers registered for an address:

```go
import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core"
)

func main() {
    ctx := context.Background()
    gocmd := core.NewGoCMD(ctx)
    defer gocmd.Close()
    
    eventBus := gocmd.EventBus()
    
    // Publish a message - all handlers receive it
    err := eventBus.Publish("user.events", map[string]interface{}{
        "userID": "123",
        "action": "created",
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

### Send (Point-to-Point)

Send delivers a message to **one** handler (load-balanced if multiple handlers exist):

```go
// Send to a single handler
err := eventBus.Send("user.service", map[string]interface{}{
    "userID": "123",
    "action": "get",
})
if err != nil {
    log.Printf("Send failed: %v", err)
}
```

### Request (Request-Reply)

Request sends a message and waits for a reply:

```go
import "time"

// Request with 5 second timeout
reply, err := eventBus.Request("user.service", map[string]interface{}{
    "userID": "123",
}, 5*time.Second)

if err != nil {
    log.Printf("Request failed: %v", err)
    return
}

// Decode reply
var user map[string]interface{}
if err := reply.DecodeBody(&user); err != nil {
    log.Printf("Decode failed: %v", err)
    return
}

log.Printf("User: %v", user)
```

## Consumer Patterns

### Basic Consumer

```go
import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core"
)

func setupConsumer(gocmd core.GoCMD) {
    eventBus := gocmd.EventBus()
    
    // Create consumer for an address
    consumer := eventBus.Consumer("user.events")
    
    // Set handler
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        // Decode message body
        var event map[string]interface{}
        if err := msg.DecodeBody(&event); err != nil {
            return err
        }
        
        log.Printf("Received event: %v", event)
        
        // Process event
        processEvent(event)
        
        return nil
    })
    
    // Cleanup when done
    defer consumer.Unregister()
}
```

### Multiple Consumers (Pub-Sub)

Multiple consumers can subscribe to the same address:

```go
// Consumer 1
consumer1 := eventBus.Consumer("user.events")
consumer1.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    log.Println("Consumer 1 received:", msg.Body())
    return nil
})

// Consumer 2
consumer2 := eventBus.Consumer("user.events")
consumer2.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    log.Println("Consumer 2 received:", msg.Body())
    return nil
})

// Publish - both consumers receive the message
eventBus.Publish("user.events", "test message")
```

### Request-Reply Handler

```go
// Service handler that replies
consumer := eventBus.Consumer("user.service")
consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    // Decode request
    var request map[string]interface{}
    if err := msg.DecodeBody(&request); err != nil {
        return msg.Fail(400, "Invalid request format")
    }
    
    userID := request["userID"].(string)
    
    // Get user data
    user, err := getUser(userID)
    if err != nil {
        return msg.Fail(404, "User not found")
    }
    
    // Reply with user data
    return msg.Reply(user)
})
```

## Message Handling

### Message Interface

```go
type Message interface {
    // Body returns the message body (usually []byte after encoding)
    Body() interface{}
    
    // Headers returns message headers
    Headers() map[string]string
    
    // ReplyAddress returns the reply address for request messages
    ReplyAddress() string
    
    // Reply sends a reply to this message
    Reply(body interface{}) error
    
    // DecodeBody decodes the message body into v
    DecodeBody(v interface{}) error
    
    // Fail indicates that processing failed
    Fail(failureCode int, message string) error
}
```

### Decoding Messages

```go
consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    // Decode as JSON
    var data map[string]interface{}
    if err := msg.DecodeBody(&data); err != nil {
        return err
    }
    
    // Or decode as protobuf (if message is protobuf)
    var user pb.User
    if err := msg.DecodeBody(&user); err != nil {
        return err
    }
    
    return nil
})
```

### Headers and Metadata

```go
consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    // Get headers
    headers := msg.Headers()
    
    // Access specific headers
    requestID := headers["X-Request-ID"]
    userID := headers["X-User-ID"]
    floxID := headers["X-Flox-ID"]
    
    // Use headers for routing or logging
    log.Printf("Request ID: %s, User ID: %s", requestID, userID)
    
    return nil
})
```

## Context-Aware Routing

### FloxID Routing (Highest Priority)

FloxID is a universal routing identifier that ensures messages with the same FloxID are routed to the same EventLoop:

```go
import "github.com/fluxorio/fluxor/pkg/core"

// Set FloxID in context
ctx := core.WithFloxID(context.Background(), "aggregate-123")

// Publish with context - FloxID automatically extracted
err := eventBus.PublishWithContext(ctx, "user.events", payload)

// Send with context
reply, err := eventBus.SendWithContext(ctx, "user.service", payload, 5*time.Second)
```

### Routing Priority Order

1. **FloxID from context** (highest priority - set via `core.WithFloxID()`)
2. `X-Route-Key` (explicit routing key in headers)
3. `X-Flox-ID` (in headers)
4. `X-User-ID` (user-based routing)
5. `X-Session-ID` (session-based routing)
6. `X-Request-ID` (request-based routing - lowest priority)
7. Round-robin (if no routing key available)

### Example: User-Based Routing

```go
// Messages for the same user always route to the same EventLoop
ctx := context.Background()

// Option 1: Via FloxID (recommended)
ctx = core.WithFloxID(ctx, "user-123")
eventBus.PublishWithContext(ctx, "user.events", payload)

// Option 2: Via headers
headers := map[string]string{
    "X-User-ID": "user-123",
}
// Headers are automatically extracted from context or message
```

## EventLoop Integration

### Enable EventLoopGroup

EventLoopGroup provides CPU-based routing for better cache locality and performance:

```go
import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/core/eventloop"
)

ctx := context.Background()
gocmd := core.NewGoCMD(ctx)

// Configure EventBus with EventLoopGroup
config := core.EventBusConfig{
    UseEventLoopGroup: true,
    EventLoopConfig: eventloop.EventLoopConfig{
        NumLoops:   0,    // Auto: GOMAXPROCS
        QueueSize:  4096, // Queue size per loop
        CPUAffinity: false, // Optional: pin to CPU core
    },
}

eventBus := core.NewEventBusWithConfig(ctx, gocmd, config)
```

### Benefits of EventLoopGroup

- **Cache Locality**: Same routing key → same CPU core
- **Single-Writer Guarantee**: No locks needed in handlers
- **Predictable Latency**: Sequential processing per loop
- **Scalability**: Scales with CPU cores

## Message Encoding

### Automatic Encoding

EventBus automatically encodes messages based on type:

```go
// Protobuf message (if body implements proto.Message)
user := &pb.User{
    Id:   "123",
    Name: "John",
}
eventBus.Publish("user.events", user) // Automatically encoded as protobuf

// JSON message (for other types)
eventBus.Publish("user.events", map[string]interface{}{
    "id":   "123",
    "name": "John",
}) // Automatically encoded as JSON

// Already encoded (pass through)
data := []byte(`{"id":"123"}`)
eventBus.Publish("user.events", data) // Passed through as-is
```

### Protobuf Messages

```go
import (
    "github.com/fluxorio/fluxor/proto/fluxor/common"
    "google.golang.org/protobuf/proto"
)

// Send protobuf message
user := &common.User{
    Id:        "123",
    Name:      "John Doe",
    Email:     "john@example.com",
    CreatedAt: time.Now().Unix(),
    Active:    true,
}

err := eventBus.Publish("user.created", user)

// Receive and decode protobuf
consumer := eventBus.Consumer("user.created")
consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    var user common.User
    if err := msg.DecodeBody(&user); err != nil {
        return err
    }
    
    log.Printf("User: %s", user.GetName())
    return nil
})
```

## Error Handling

### EventBus Errors

```go
import "github.com/fluxorio/fluxor/pkg/core"

reply, err := eventBus.Request("user.service", payload, 5*time.Second)
if err != nil {
    if ebErr, ok := err.(*core.EventBusError); ok {
        switch ebErr.Code {
        case "NO_HANDLERS":
            log.Println("No handlers registered for address")
        case "TIMEOUT":
            log.Println("Request timed out")
        default:
            log.Printf("EventBus error: %v", ebErr)
        }
    } else {
        log.Printf("Error: %v", err)
    }
}
```

### Handler Errors

```go
consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    // Return error to signal processing failure
    if err := processMessage(msg); err != nil {
        return err // Error is logged but doesn't crash the handler
    }
    
    return nil // Success
})
```

### Fail Response

```go
consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    // Fail with error code and message
    if invalid {
        return msg.Fail(400, "Invalid request")
    }
    
    // Success
    return msg.Reply(result)
})
```

## Advanced Patterns

### Service Pattern (BaseService)

Use `BaseService` for request-reply services:

```go
import "github.com/fluxorio/fluxor/pkg/core"

type UserService struct {
    *core.BaseService
    users map[string]map[string]interface{}
}

func (s *UserService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    var request map[string]interface{}
    if err := msg.DecodeBody(&request); err != nil {
        return s.Fail(msg, 400, "Invalid request")
    }
    
    userID := request["userID"].(string)
    user, exists := s.users[userID]
    if !exists {
        return s.Fail(msg, 404, "User not found")
    }
    
    return s.Reply(msg, user)
}

func (s *UserService) doStart(ctx core.FluxorContext) error {
    s.users = make(map[string]map[string]interface{})
    return nil
}

// Deploy service
service := &UserService{
    BaseService: core.NewBaseService("user-service", "user.service"),
}
gocmd.DeployVerticle(service)
```

### Verticle Pattern (BaseVerticle)

Use `BaseVerticle` for event-driven components:

```go
type OrderVerticle struct {
    *core.BaseVerticle
    userServiceAddr string
}

func (v *OrderVerticle) doStart(ctx core.FluxorContext) error {
    v.userServiceAddr = "user.service"
    
    // Subscribe to order events
    consumer := v.Consumer("order.create")
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        var order map[string]interface{}
        msg.DecodeBody(&order)
        
        // Call service via EventBus
        reply, err := v.Request(v.userServiceAddr, map[string]interface{}{
            "userID": order["userID"],
        }, 5*time.Second)
        
        if err != nil {
            return msg.Fail(502, "Service unavailable")
        }
        
        var user map[string]interface{}
        reply.DecodeBody(&user)
        
        return msg.Reply(map[string]interface{}{
            "orderID": order["orderID"],
            "user":    user,
        })
    })
    
    return nil
}
```

### CPU-Bound Service Pattern

For CPU-intensive work, use WorkerPool:

```go
type ImageProcessingService struct {
    *core.BaseService
}

func (s *ImageProcessingService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    var request map[string]interface{}
    msg.DecodeBody(&request)
    
    imageData := request["image"].([]byte)
    
    // CPU-bound work: Use WorkerPool
    result, err := ctx.GoCMD().ExecuteBlocking(func() (interface{}, error) {
        // CPU-intensive processing (runs on worker thread)
        return processImage(imageData), nil
    }, 30*time.Second)
    
    if err != nil {
        return s.Fail(msg, 500, "Processing failed")
    }
    
    return s.Reply(msg, map[string]interface{}{
        "processedImage": result,
    })
}
```

## Best Practices

### ✅ DO

1. **Use EventBus for Service Communication**: Prefer EventBus over direct injection
2. **Handle Timeouts**: Always specify timeout for `Request()` calls
3. **Handle Errors**: Check errors from `Publish()`, `Send()`, and `Request()`
4. **Use Context-Aware Routing**: Set FloxID/UserID for consistent routing
5. **Decode Messages**: Always decode message bodies before use
6. **Clean Up Consumers**: Call `Unregister()` when done
7. **Use BaseService**: For request-reply services, use `BaseService`
8. **Use WorkerPool for CPU Work**: Don't block event loops with CPU-intensive work

### ❌ DON'T

1. **Don't Block Handlers**: Keep handler execution < 20µs for event loop tasks
2. **Don't Ignore Errors**: Always handle errors from EventBus operations
3. **Don't Share State**: Communicate via immutable messages, not shared state
4. **Don't Use Empty Addresses**: Addresses must be non-empty (validated)
5. **Don't Forget Timeouts**: Always specify timeout for `Request()` calls
6. **Don't Block Event Loops**: Use `ExecuteBlocking()` for CPU-bound work

## Performance Considerations

### Throughput

- **Per EventLoop**: ~100k events/sec
- **Total (N loops)**: N × 100k events/sec
- **Bottleneck**: Handler execution time

### Latency

- **Dispatcher routing**: < 1ms p99
- **EventLoop dispatch**: < 100µs p99
- **Handler execution**: Depends on handler

### Queue Sizes

- **EventLoop queue**: Default 4096 (configurable)
- **Consumer mailbox**: Default 100 (configurable)
- **Backpressure**: Messages dropped when queues are full

## Thread Safety

All EventBus methods are thread-safe and can be called concurrently from multiple goroutines:

- `Publish()`: Thread-safe (uses RLock for reads)
- `Send()`: Thread-safe (uses RLock for reads)
- `Request()`: Thread-safe (uses RLock for reads)
- `Consumer()`: Thread-safe (uses Lock for writes)
- Handler execution: Each handler runs in isolation

## Configuration

### Default Configuration

```go
// Default EventBus (Executor-only, no EventLoopGroup)
eventBus := core.NewEventBus(ctx, gocmd)
```

### With EventLoopGroup

```go
config := core.EventBusConfig{
    UseEventLoopGroup: true,
    EventLoopConfig: eventloop.EventLoopConfig{
        NumLoops:   0,     // Auto: GOMAXPROCS
        QueueSize:  4096,  // Queue size per loop
        CPUAffinity: false, // Optional CPU pinning
        Metrics:    true,   // Enable metrics
    },
}

eventBus := core.NewEventBusWithConfig(ctx, gocmd, config)
```

## Examples

### Example 1: Simple Pub-Sub

```go
// Publisher
eventBus.Publish("news.article", map[string]interface{}{
    "title": "Breaking News",
    "body":  "Content here",
})

// Subscriber 1
consumer1 := eventBus.Consumer("news.article")
consumer1.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    log.Println("Subscriber 1 received article")
    return nil
})

// Subscriber 2
consumer2 := eventBus.Consumer("news.article")
consumer2.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    log.Println("Subscriber 2 received article")
    return nil
})
```

### Example 2: Request-Reply Service

```go
// Service
type CalculatorService struct {
    *core.BaseService
}

func (s *CalculatorService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    var req map[string]interface{}
    msg.DecodeBody(&req)
    
    a := req["a"].(float64)
    b := req["b"].(float64)
    op := req["op"].(string)
    
    var result float64
    switch op {
    case "+":
        result = a + b
    case "-":
        result = a - b
    case "*":
        result = a * b
    case "/":
        if b == 0 {
            return s.Fail(msg, 400, "Division by zero")
        }
        result = a / b
    default:
        return s.Fail(msg, 400, "Unknown operator")
    }
    
    return s.Reply(msg, map[string]interface{}{"result": result})
}

// Deploy
service := &CalculatorService{
    BaseService: core.NewBaseService("calculator", "calculator.service"),
}
gocmd.DeployVerticle(service)

// Use
reply, err := eventBus.Request("calculator.service", map[string]interface{}{
    "a":  10.0,
    "b":  5.0,
    "op": "+",
}, 5*time.Second)
```

### Example 3: Orchestrating Multiple Services

```go
type OrderService struct {
    *core.BaseService
    userServiceAddr    string
    paymentServiceAddr string
}

func (s *OrderService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    var order map[string]interface{}
    msg.DecodeBody(&order)
    
    userID := order["userID"].(string)
    amount := order["amount"].(float64)
    
    // Step 1: Validate user
    _, err := s.Request(s.userServiceAddr, map[string]interface{}{
        "userID": userID,
    }, 5*time.Second)
    if err != nil {
        return s.Fail(msg, 400, "Invalid user")
    }
    
    // Step 2: Process payment
    _, err = s.Request(s.paymentServiceAddr, map[string]interface{}{
        "userID": userID,
        "amount": amount,
    }, 10*time.Second)
    if err != nil {
        return s.Fail(msg, 402, "Payment failed")
    }
    
    return s.Reply(msg, map[string]interface{}{
        "orderID": "order-123",
        "status":  "completed",
    })
}
```

## Troubleshooting

### Messages Not Received

1. **Check Address**: Ensure address matches exactly (case-sensitive)
2. **Check Handler Registration**: Verify handler is registered before publishing
3. **Check Context**: Ensure context is not cancelled
4. **Check Mailbox**: Mailbox might be full (messages dropped)

### High Latency

1. **Check Handler Execution Time**: Profile handler performance
2. **Check Queue Length**: Monitor EventLoop queue lengths
3. **Check Backpressure**: Messages might be queued
4. **Enable Metrics**: Use EventLoop metrics to identify bottlenecks

### Handler Not Executing

1. **Check Error Logs**: Handler errors are logged but don't crash
2. **Check Mailbox**: Mailbox might be closed
3. **Check Context**: Context cancellation stops handlers
4. **Check Unregister**: Ensure consumer is not unregistered

## Related Documentation

- [EVENTBUS_FLOW.md](./EVENTBUS_FLOW.md) - EventBus flow architecture
- [EVENTBUS_SERVICE_PATTERN.md](./EVENTBUS_SERVICE_PATTERN.md) - Service patterns
- [eventloop/README.md](./eventloop/README.md) - EventLoop architecture
- [ENVELOPE_ARCHITECTURE.md](./ENVELOPE_ARCHITECTURE.md) - Envelope pattern
