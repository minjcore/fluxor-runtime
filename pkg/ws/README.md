# WebSocket Server Package

The `ws` package provides a general-purpose WebSocket server with optional EventBus integration for Fluxor applications.

## Features

- **General-purpose WebSocket server** for custom WebSocket applications
- **EventBus integration** using `core.WebSocketEventBusBridge` for EventBus-based messaging
- **Connection management** with automatic cleanup
- **Configurable** timeouts, buffer sizes, and connection limits
- **Ping/pong support** for connection health checks
- **BaseServer pattern** for lifecycle management

## Quick Start

### Basic WebSocket Server

```go
package main

import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/ws"
    "github.com/fluxorio/fluxor/pkg/entrypoint"
)

func main() {
    ctx := context.Background()
    gocmd := core.NewGoCMD(ctx)
    defer gocmd.Close()

    // Create WebSocket server
    config := ws.DefaultServerConfig(":8080", "/ws")
    server := ws.NewServer(gocmd, config)

    // Set connection handler
    server.SetHandler(func(conn *ws.Connection) error {
        // Send welcome message
        conn.WriteJSON(map[string]string{"message": "connected"})

        // Echo messages
        for {
            messageType, data, err := conn.ReadMessage()
            if err != nil {
                return err // Connection closed
            }
            // Echo back
            conn.WriteMessage(messageType, data)
        }
    })

    // Start server
    server.Start()
}
```

### WebSocket Server with EventBus Integration

```go
package main

import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/ws"
    "github.com/fluxorio/fluxor/pkg/entrypoint"
)

func main() {
    ctx := context.Background()
    gocmd := core.NewGoCMD(ctx)
    defer gocmd.Close()

    eventBus := gocmd.EventBus()

    // Create WebSocket server with EventBus integration
    config := ws.DefaultServerConfig(":8080", "/eventbus/ws")
    server := ws.NewServerWithEventBus(gocmd, config, eventBus)

    // Start server
    server.Start()
}
```

### Using as a Verticle

```go
type WebSocketVerticle struct {
    *core.BaseVerticle
    server *ws.WebSocketServer
}

func NewWebSocketVerticle() *WebSocketVerticle {
    return &WebSocketVerticle{
        BaseVerticle: core.NewBaseVerticle("websocket-verticle"),
    }
}

func (v *WebSocketVerticle) Start(ctx core.FluxorContext) error {
    if err := v.BaseVerticle.Start(ctx); err != nil {
        return err
    }

    config := ws.DefaultServerConfig(":8080", "/ws")
    v.server = ws.NewServer(ctx.GoCMD(), config)
    v.server.SetHandler(func(conn *ws.Connection) error {
        // Handle connection
        return nil
    })

    // Start server in goroutine (non-blocking)
    go v.server.Start()
    return nil
}

func (v *WebSocketVerticle) Stop(ctx core.FluxorContext) error {
    if v.server != nil {
        v.server.Stop()
    }
    return v.BaseVerticle.Stop(ctx)
}
```

## Configuration

### ServerConfig

```go
config := &ws.ServerConfig{
    Addr:            ":8080",              // Server address
    Path:            "/ws",                // WebSocket endpoint path
    ReadBufferSize:  4096,                 // Read buffer size
    WriteBufferSize: 4096,                 // Write buffer size
    ReadDeadline:    60 * time.Second,     // Read deadline
    WriteDeadline:   10 * time.Second,     // Write deadline
    PongWait:        60 * time.Second,     // Pong wait timeout
    PingPeriod:      54 * time.Second,     // Ping interval (90% of PongWait)
    MaxConnections:  100,                  // Max concurrent connections (0 = unlimited)
    CheckOrigin:     nil,                  // Origin check function (nil = allow all)
}
```

### Default Configuration

Use `DefaultServerConfig(addr, path)` for sensible defaults:

```go
config := ws.DefaultServerConfig(":8080", "/ws")
```

## Connection API

### Connection Methods

```go
// Write text/binary message
conn.WriteMessage(messageType int, data []byte) error

// Write JSON message
conn.WriteJSON(v interface{}) error

// Read message
messageType, data, err := conn.ReadMessage()

// Close connection
conn.Close() error

// Check if closed
isClosed := conn.IsClosed()

// Get connection ID
id := conn.ID()

// Get connection context
ctx := conn.Context()

// Set deadlines
conn.SetReadDeadline(t time.Time) error
conn.SetWriteDeadline(t time.Time) error
```

## EventBus Integration

When using `NewServerWithEventBus`, the server automatically integrates with Fluxor's EventBus using the `WebSocketEventBusBridge`. Clients can:

- **Publish** messages to EventBus addresses
- **Send** point-to-point messages
- **Request** messages with replies
- **Subscribe** to EventBus addresses to receive messages
- **Unsubscribe** from addresses

### EventBus Message Format

```json
{
  "op": "publish|send|request|subscribe|unsubscribe",
  "address": "my.address",
  "body": {...},
  "id": "request-id",
  "timeout": 5000
}
```

### Operations

- **publish**: Publish message to all consumers on address
- **send**: Send message to one consumer on address
- **request**: Send message and wait for reply (request-reply pattern)
- **subscribe**: Subscribe to address to receive messages
- **unsubscribe**: Unsubscribe from address

## Architecture

The WebSocket server follows Fluxor's architectural patterns:

- **BaseServer** for lifecycle management
- **HTTP server** as the underlying transport
- **Connection management** with automatic cleanup
- **EventBus integration** via `core.WebSocketEventBusBridge`

## Best Practices

1. **Connection Cleanup**: Connections are automatically cleaned up when closed. Implement proper error handling in your handler.

2. **Non-blocking Handlers**: Use goroutines for long-running operations within connection handlers.

3. **EventBus Integration**: Use `NewServerWithEventBus` when you need EventBus functionality, otherwise use `NewServer` for simpler use cases.

4. **MaxConnections**: Set `MaxConnections` to prevent resource exhaustion in production.

5. **CheckOrigin**: Implement proper origin checking in production:

```go
config.CheckOrigin = func(r interface{}) bool {
    req := r.(*http.Request)
    origin := req.Header.Get("Origin")
    return origin == "https://example.com"
}
```

## See Also

- `pkg/core/eventbus_ws.go` - WebSocketEventBusBridge implementation
- `pkg/web/` - HTTP server package
- `pkg/tcp/` - TCP server package
- `examples/wasm-eventbus/` - WebSocket EventBus example

