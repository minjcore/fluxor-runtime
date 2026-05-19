# Testing EventBus (In-Memory)

This guide shows how to test the EventBus in-memory for unit tests and integration tests.

## Quick Start

The simplest way to get an in-memory EventBus is through `GoCMD`:

```go
package core_test

import (
    "context"
    "testing"
    "time"
    
    "github.com/fluxorio/fluxor/pkg/core"
)

func TestEventBus_Basic(t *testing.T) {
    // 1. Create context
    ctx := context.Background()
    
    // 2. Create GoCMD (automatically creates in-memory EventBus)
    gocmd := core.NewGoCMD(ctx)
    defer gocmd.Close()
    
    // 3. Get EventBus
    eb := gocmd.EventBus()
    defer eb.Close()
    
    // 4. Use EventBus for testing
    err := eb.Publish("test.address", "test message")
    if err != nil {
        t.Errorf("Publish() error = %v", err)
    }
}
```

## Test Patterns

### 1. Testing Publish (Pub-Sub)

```go
func TestEventBus_Publish(t *testing.T) {
    ctx := context.Background()
    gocmd := core.NewGoCMD(ctx)
    defer gocmd.Close()
    
    eb := gocmd.EventBus()
    defer eb.Close()
    
    // Register multiple consumers (pub-sub pattern)
    received1 := make(chan core.Message, 1)
    consumer1 := eb.Consumer("test.address")
    consumer1.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        received1 <- msg
        return nil
    })
    
    received2 := make(chan core.Message, 1)
    consumer2 := eb.Consumer("test.address")
    consumer2.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        received2 <- msg
        return nil
    })
    
    // Publish message (both consumers should receive it)
    err := eb.Publish("test.address", "test message")
    if err != nil {
        t.Fatalf("Publish() error = %v", err)
    }
    
    // Wait for both consumers to receive
    select {
    case msg1 := <-received1:
        if msg1.Body() != "test message" {
            t.Errorf("Consumer1 received = %v, want 'test message'", msg1.Body())
        }
    case <-time.After(1 * time.Second):
        t.Error("Consumer1 did not receive message")
    }
    
    select {
    case msg2 := <-received2:
        if msg2.Body() != "test message" {
            t.Errorf("Consumer2 received = %v, want 'test message'", msg2.Body())
        }
    case <-time.After(1 * time.Second):
        t.Error("Consumer2 did not receive message")
    }
}
```

### 2. Testing Send (Point-to-Point)

```go
func TestEventBus_Send(t *testing.T) {
    ctx := context.Background()
    gocmd := core.NewGoCMD(ctx)
    defer gocmd.Close()
    
    eb := gocmd.EventBus()
    defer eb.Close()
    
    // Register handler
    received := make(chan core.Message, 1)
    consumer := eb.Consumer("test.address")
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        received <- msg
        return nil
    })
    
    // Send message (only one consumer receives it - round-robin)
    err := eb.Send("test.address", "test message")
    if err != nil {
        t.Fatalf("Send() error = %v", err)
    }
    
    // Wait for message
    select {
    case msg := <-received:
        if msg.Body() != "test message" {
            t.Errorf("Received = %v, want 'test message'", msg.Body())
        }
    case <-time.After(1 * time.Second):
        t.Error("Message not received")
    }
}
```

### 3. Testing Request-Reply

```go
func TestEventBus_Request(t *testing.T) {
    ctx := context.Background()
    gocmd := core.NewGoCMD(ctx)
    defer gocmd.Close()
    
    eb := gocmd.EventBus()
    defer eb.Close()
    
    // Register handler that replies
    consumer := eb.Consumer("test.address")
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        // Reply to request
        return msg.Reply("reply message")
    })
    
    // Send request and wait for reply
    reply, err := eb.Request("test.address", "request", 1*time.Second)
    if err != nil {
        t.Fatalf("Request() error = %v", err)
    }
    
    if reply == nil {
        t.Fatal("Request() returned nil reply")
    }
    
    if reply.Body() != "reply message" {
        t.Errorf("Reply body = %v, want 'reply message'", reply.Body())
    }
}
```

### 4. Testing with Custom Types

```go
type PaymentRequest struct {
    PaymentID string
    Amount    float64
}

type PaymentReply struct {
    OK    bool
    AuthID string
}

func TestEventBus_CustomTypes(t *testing.T) {
    ctx := context.Background()
    gocmd := core.NewGoCMD(ctx)
    defer gocmd.Close()
    
    eb := gocmd.EventBus()
    defer eb.Close()
    
    // Register handler
    consumer := eb.Consumer("payments.authorize")
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        // Decode request
        body, ok := msg.Body().([]byte)
        if !ok {
            return msg.Reply(PaymentReply{OK: false})
        }
        
        var req PaymentRequest
        if err := core.JSONDecode(body, &req); err != nil {
            return msg.Reply(PaymentReply{OK: false})
        }
        
        // Process and reply
        return msg.Reply(PaymentReply{
            OK:     true,
            AuthID: "auth_" + req.PaymentID,
        })
    })
    
    // Send request
    req := PaymentRequest{
        PaymentID: "pay_123",
        Amount:    100.0,
    }
    
    reply, err := eb.Request("payments.authorize", req, 1*time.Second)
    if err != nil {
        t.Fatalf("Request() error = %v", err)
    }
    
    // Decode reply
    replyBody, ok := reply.Body().([]byte)
    if !ok {
        t.Fatal("Reply body is not []byte")
    }
    
    var paymentReply PaymentReply
    if err := core.JSONDecode(replyBody, &paymentReply); err != nil {
        t.Fatalf("JSONDecode() error = %v", err)
    }
    
    if !paymentReply.OK {
        t.Error("Payment reply OK = false, want true")
    }
    
    if paymentReply.AuthID != "auth_pay_123" {
        t.Errorf("AuthID = %v, want 'auth_pay_123'", paymentReply.AuthID)
    }
}
```

### 5. Testing Error Handling

```go
func TestEventBus_ErrorHandling(t *testing.T) {
    ctx := context.Background()
    gocmd := core.NewGoCMD(ctx)
    defer gocmd.Close()
    
    eb := gocmd.EventBus()
    defer eb.Close()
    
    // Test fail-fast: empty address
    err := eb.Publish("", "test")
    if err == nil {
        t.Error("Publish() with empty address should fail")
    }
    
    // Test fail-fast: nil body
    err = eb.Publish("test.address", nil)
    if err == nil {
        t.Error("Publish() with nil body should fail")
    }
    
    // Test fail-fast: no handlers for Send
    err = eb.Send("no.handlers", "test")
    if err == nil {
        t.Error("Send() with no handlers should fail")
    }
    if e, ok := err.(*core.EventBusError); ok {
        if e.Code != "NO_HANDLERS" {
            t.Errorf("Error code = %q, want 'NO_HANDLERS'", e.Code)
        }
    }
}
```

### 6. Testing with EventLoopGroup (CPU-based routing)

```go
func TestEventBus_WithEventLoopGroup(t *testing.T) {
    ctx := context.Background()
    gocmd := core.NewGoCMD(ctx)
    defer gocmd.Close()
    
    // Create EventBus with EventLoopGroup enabled
    config := core.EventBusConfig{
        UseEventLoopGroup: true,
        EventLoopConfig: eventloop.EventLoopConfig{
            Workers: 4, // 4 event loops
        },
    }
    
    eb := core.NewEventBusWithConfig(ctx, gocmd, config)
    defer eb.Close()
    
    // Register handler
    received := make(chan core.Message, 1)
    consumer := eb.Consumer("test.address")
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        received <- msg
        return nil
    })
    
    // Send message (routed via EventLoopGroup)
    err := eb.Send("test.address", "test message")
    if err != nil {
        t.Fatalf("Send() error = %v", err)
    }
    
    // Wait for message
    select {
    case msg := <-received:
        if msg.Body() != "test message" {
            t.Errorf("Received = %v, want 'test message'", msg.Body())
        }
    case <-time.After(1 * time.Second):
        t.Error("Message not received")
    }
}
```

### 7. Testing Consumer Completion

```go
func TestConsumer_Completion(t *testing.T) {
    ctx := context.Background()
    gocmd := core.NewGoCMD(ctx)
    defer gocmd.Close()
    
    eb := gocmd.EventBus()
    defer eb.Close()
    
    consumer := eb.Consumer("test.address")
    
    // Completion channel should be available
    done := consumer.Completion()
    if done == nil {
        t.Error("Completion() should not return nil")
    }
    
    // Set handler to start processing
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        return nil
    })
    
    // Unregister should close the mailbox and signal completion
    time.Sleep(100 * time.Millisecond) // Give time for executor to start
    
    err := consumer.Unregister()
    if err != nil {
        t.Errorf("Unregister() error = %v", err)
    }
    
    // Wait for completion (with timeout)
    select {
    case <-done:
        // Success - channel closed
    case <-time.After(2 * time.Second):
        t.Error("Completion channel not closed after unregister")
    }
}
```

### 8. Testing Multiple Consumers (Load Balancing)

```go
func TestConsumer_MultipleConsumers(t *testing.T) {
    ctx := context.Background()
    gocmd := core.NewGoCMD(ctx)
    defer gocmd.Close()
    
    eb := gocmd.EventBus()
    defer eb.Close()
    
    // Register multiple consumers for load balancing
    received1 := make(chan core.Message, 10)
    consumer1 := eb.Consumer("test.address")
    consumer1.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        received1 <- msg
        return nil
    })
    
    received2 := make(chan core.Message, 10)
    consumer2 := eb.Consumer("test.address")
    consumer2.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        received2 <- msg
        return nil
    })
    
    // Send 10 messages (round-robin distribution)
    for i := 0; i < 10; i++ {
        err := eb.Send("test.address", fmt.Sprintf("message-%d", i))
        if err != nil {
            t.Fatalf("Send() error = %v", err)
        }
    }
    
    // Wait for messages to be distributed
    time.Sleep(100 * time.Millisecond)
    
    // Check distribution (should be roughly equal)
    count1 := len(received1)
    count2 := len(received2)
    
    if count1+count2 != 10 {
        t.Errorf("Total messages received = %d, want 10", count1+count2)
    }
    
    // Messages should be distributed (not all to one consumer)
    if count1 == 0 || count2 == 0 {
        t.Error("Messages should be distributed to both consumers")
    }
}
```

## Direct EventBus Creation (Advanced)

If you need to create EventBus directly without GoCMD:

```go
func TestEventBus_DirectCreation(t *testing.T) {
    ctx := context.Background()
    
    // Create a minimal GoCMD for EventBus
    gocmd := core.NewGoCMD(ctx)
    defer gocmd.Close()
    
    // Create EventBus directly
    eb := core.NewEventBus(ctx, gocmd)
    defer eb.Close()
    
    // Or with configuration
    config := core.EventBusConfig{
        UseEventLoopGroup: true,
    }
    eb2 := core.NewEventBusWithConfig(ctx, gocmd, config)
    defer eb2.Close()
}
```

## Best Practices

1. **Always clean up**: Use `defer eb.Close()` or `defer gocmd.Close()`
2. **Use timeouts**: Always use timeouts when waiting for messages
3. **Test error cases**: Test fail-fast validation (empty address, nil body, etc.)
4. **Test async behavior**: Use channels to wait for async message delivery
5. **Test multiple consumers**: Verify pub-sub and load balancing behavior

## Common Patterns

### Helper Function for Test Setup

```go
func setupTestEventBus(t *testing.T) (core.GoCMD, core.EventBus) {
    ctx := context.Background()
    gocmd := core.NewGoCMD(ctx)
    eb := gocmd.EventBus()
    
    t.Cleanup(func() {
        eb.Close()
        gocmd.Close()
    })
    
    return gocmd, eb
}

func TestEventBus_WithHelper(t *testing.T) {
    _, eb := setupTestEventBus(t)
    
    // Use EventBus for testing
    err := eb.Publish("test.address", "test")
    if err != nil {
        t.Errorf("Publish() error = %v", err)
    }
}
```

### Testing Handler Errors

```go
func TestEventBus_HandlerError(t *testing.T) {
    ctx := context.Background()
    gocmd := core.NewGoCMD(ctx)
    defer gocmd.Close()
    
    eb := gocmd.EventBus()
    defer eb.Close()
    
    // Handler that returns error
    consumer := eb.Consumer("test.address")
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        return fmt.Errorf("handler error")
    })
    
    // Send message (error is logged but doesn't fail Send)
    err := eb.Send("test.address", "test")
    if err != nil {
        t.Errorf("Send() should succeed even if handler returns error: %v", err)
    }
}
```

## See Also

- `pkg/core/eventbus_test.go` - Comprehensive test examples
- `pkg/core/eventbus_consumer_test.go` - Consumer-specific tests
- `pkg/core/EVENTBUS_README.md` - EventBus usage guide
