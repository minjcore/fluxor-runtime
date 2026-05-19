# Queue Package: RabbitMQ Integration

Thư viện kết nối RabbitMQ cho Fluxor, cung cấp publisher/subscriber pattern với fail-fast validation và connection pooling.

## Tổng quan

Package `pkg/queue` cung cấp:
- ✅ RabbitMQ connection management
- ✅ Publisher/Consumer interfaces
- ✅ Fluxor Component integration
- ✅ Fail-fast validation
- ✅ Context support
- ✅ Automatic JSON encoding/decoding
- ✅ Message acknowledgment handling

## Quick Start

### 1. Tạo Queue Component

```go
package main

import (
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/queue"
)

func main() {
    // Tạo config
    config := queue.DefaultConfig()
    config.Host = "localhost"
    config.Port = 5672
    config.Username = "guest"
    config.Password = "guest"

    // Tạo component
    queueComp := queue.NewQueueComponent(config)

    // Deploy vào Vertx
    vertx := core.NewVertx()
    vertx.Deploy(queueComp)
}
```

### 2. Publish Messages

```go
// Lấy publisher
publisher, err := queueComp.Publisher()
if err != nil {
    log.Fatal(err)
}
defer publisher.Close()

// Publish message
msg := queue.Message{
    Body: map[string]interface{}{
        "user_id": 123,
        "action": "login",
    },
    Headers: map[string]string{
        "X-Request-ID": "req-123",
    },
}

ctx := context.Background()
err = publisher.Publish(ctx, "events", "user.login", msg)
if err != nil {
    log.Fatal(err)
}
```

### 3. Consume Messages

```go
// Lấy consumer
consumer, err := queueComp.Consumer()
if err != nil {
    log.Fatal(err)
}
defer consumer.Close()

// Consume messages
ctx := context.Background()
err = consumer.Consume(ctx, "user.events", func(ctx context.Context, delivery *queue.Delivery) error {
    // Decode message
    var data map[string]interface{}
    if err := delivery.DecodeBody(&data); err != nil {
        return err
    }

    // Process message
    log.Printf("Received: %+v", data)
    return nil
})
if err != nil {
    log.Fatal(err)
}
```

## Configuration

### Basic Config

```go
config := queue.Config{
    Host:     "localhost",
    Port:     5672,
    Username: "guest",
    Password: "guest",
    VHost:    "/",
}
```

### Advanced Config

```go
config := queue.Config{
    URL: "amqp://user:pass@localhost:5672/vhost",
    ConnectionTimeout: 30 * time.Second,
    Heartbeat: 10 * time.Second,
    Retry: &queue.RetryConfig{
        MaxRetries:     3,
        InitialInterval: 1 * time.Second,
        MaxInterval:    30 * time.Second,
        Multiplier:     2.0,
    },
}
```

### TLS Config

```go
config := queue.Config{
    Host: "rabbitmq.example.com",
    Port: 5671,
    TLS: &queue.TLSConfig{
        Enabled:            true,
        InsecureSkipVerify: false,
        CAFile:            "/path/to/ca.crt",
        CertFile:          "/path/to/client.crt",
        KeyFile:           "/path/to/client.key",
    },
}
```

## Consumer Configuration

```go
consumerConfig := queue.ConsumerConfig{
    Queue:        "user.events",
    ConsumerTag:  "my-consumer",
    AutoAck:      false,
    Exclusive:    false,
    PrefetchCount: 10,
    PrefetchSize:  0,
    Global:       false,
}

consumer, _ := queueComp.Consumer()
consumerImpl := consumer.(*queue.ConsumerImpl)
err := consumerImpl.ConsumeWithConfig(ctx, consumerConfig, handler)
```

## Message Types

### Publishing

```go
msg := queue.Message{
    Body:         data,                    // Will be JSON encoded
    Headers:      map[string]string{},     // Custom headers
    ContentType:  "application/json",      // Default
    DeliveryMode: 2,                       // 2 = persistent
    Priority:     0,                       // 0-255
    Expiration:   "60000",                 // TTL in milliseconds
    MessageID:    "msg-123",
    Timestamp:    time.Now(),
    ReplyTo:      "reply.queue",
    CorrelationID: "corr-123",
}
```

### Delivery

```go
func handler(ctx context.Context, delivery *queue.Delivery) error {
    // Access raw body
    body := delivery.Body

    // Decode JSON
    var data MyStruct
    err := delivery.DecodeBody(&data)

    // Access metadata
    exchange := delivery.Exchange
    routingKey := delivery.RoutingKey
    messageID := delivery.MessageID
    headers := delivery.Headers

    // Acknowledge (if AutoAck = false)
    // Automatic if handler returns nil

    return nil
}
```

## Error Handling

Package sử dụng fail-fast pattern:

```go
// Invalid config → error on NewQueueComponent
config := queue.Config{} // Missing required fields
comp := queue.NewQueueComponent(config) // Panic!

// Invalid state → error on operations
publisher.Publish(ctx, "", "", msg) // Error: exchange name cannot be empty

// Connection closed → error
conn.Close()
publisher.Publish(ctx, "exchange", "key", msg) // Error: connection is closed
```

## Integration với Fluxor

### Trong Verticle

```go
type MyVerticle struct {
    *core.BaseVerticle
    queue *queue.QueueComponent
}

func (v *MyVerticle) doStart(ctx core.FluxorContext) error {
    // Queue component đã được deploy và started
    publisher, err := v.queue.Publisher()
    if err != nil {
        return err
    }

    // Publish message
    msg := queue.Message{Body: "Hello"}
    return publisher.Publish(ctx, "exchange", "routing.key", msg)
}
```

## Best Practices

1. **Always close publishers/consumers**: Use `defer publisher.Close()`
2. **Handle errors**: Check errors from Publish/Consume
3. **Use context**: Pass context for cancellation/timeout
4. **Acknowledge messages**: Return nil from handler to ack, error to nack
5. **Set prefetch**: Use ConsumerConfig to limit unacked messages
6. **Use persistent messages**: Set DeliveryMode = 2 for durability

## Dependencies

Package sử dụng `github.com/rabbitmq/amqp091-go` cho RabbitMQ client.

Thêm vào `go.mod`:
```bash
go get github.com/rabbitmq/amqp091-go
```

