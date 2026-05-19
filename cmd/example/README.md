# Fluxor Example Application

A comprehensive example demonstrating Fluxor framework patterns and best practices.

## Overview

This example demonstrates:

1. **BaseVerticle Pattern** - Using `BaseVerticle` for lifecycle management
2. **EventBus** - Publish/subscribe messaging between verticles
3. **HTTP Server** - FastHTTPServer with routing
4. **Worker Pool** - Using `ExecuteBlocking` for blocking operations
5. **MainVerticle Pattern** - Application entry point with graceful shutdown
6. **Non-blocking Start** - Proper async server startup

## Building

```bash
go build -o example ./cmd/example
```

Or from the project root:

```bash
go build -o bin/example ./cmd/example
```

## Running

```bash
./example
```

The application will:
- Start PingVerticle (publishes ping messages every 2 seconds)
- Start 2 PongVerticle instances (subscribe to ping messages)
- Start HTTPExampleVerticle (HTTP server on :8080)

## Components

### PingVerticle

- **Purpose**: Publishes ping messages to EventBus
- **Pattern**: BaseVerticle with background goroutine
- **Key Points**:
  - Uses `BaseVerticle.Start()` for lifecycle
  - Background goroutine for periodic tasks
  - `Start()` returns immediately (non-blocking)
  - Respects context cancellation for graceful shutdown

### PongVerticle

- **Purpose**: Subscribes to ping messages and logs them
- **Pattern**: BaseVerticle with EventBus consumer
- **Key Points**:
  - Uses `BaseVerticle.Consumer()` convenience method
  - Message handler runs on reactor (non-blocking)
  - Can be deployed multiple times for scaling

### HTTPExampleVerticle

- **Purpose**: HTTP server demonstrating various patterns
- **Pattern**: BaseVerticle with FastHTTPServer
- **Key Points**:
  - Server started in goroutine (non-blocking)
  - HTTP handlers can publish to EventBus
  - Demonstrates worker pool usage
  - Proper cleanup in `Stop()` method

## API Endpoints

### GET /

Returns a hello message.

```bash
curl http://localhost:8080/
```

Response:
```json
{
  "message": "Hello from Fluxor Example!",
  "version": "1.0.0"
}
```

### GET /api/ping

Health check endpoint.

```bash
curl http://localhost:8080/api/ping
```

Response:
```json
{
  "status": "ok",
  "time": 1234567890
}
```

### GET /api/eventbus/publish

Publishes a message to EventBus from HTTP handler.

```bash
curl http://localhost:8080/api/eventbus/publish
```

Response:
```json
{
  "message": "Published to EventBus",
  "payload": "HTTP triggered message at 2026-01-13T..."
}
```

### GET /api/info

Returns system information and features.

```bash
curl http://localhost:8080/api/info
```

Response:
```json
{
  "message": "Fluxor Example Application",
  "features": [
    "BaseVerticle Pattern",
    "EventBus Publish/Subscribe",
    "HTTP Server",
    "Non-blocking Start",
    "Graceful Shutdown"
  ],
  "timestamp": 1234567890
}
```

## Key Patterns Demonstrated

### 1. BaseVerticle Pattern

```go
type MyVerticle struct {
    *core.BaseVerticle
}

func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    // Always call BaseVerticle.Start first
    if err := v.BaseVerticle.Start(ctx); err != nil {
        return err
    }
    // Custom initialization
    return nil
}

func (v *MyVerticle) Stop(ctx core.FluxorContext) error {
    // Cleanup
    return v.BaseVerticle.Stop(ctx)
}
```

### 2. Non-blocking Start

```go
// ❌ BAD: Blocking in Start()
func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    return server.Start() // Blocks forever!
}

// ✅ GOOD: Start in goroutine
func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    go func() {
        if err := server.Start(); err != nil {
            logger.Error(fmt.Sprintf("Server error: %v", err))
        }
    }()
    return nil
}
```

### 3. EventBus Publish/Subscribe

```go
// Publish
v.Publish("my-topic", "message")

// Subscribe
consumer := v.Consumer("my-topic")
consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    logger.Info(fmt.Sprintf("Received: %v", msg.Body()))
    return nil
})
```

### 4. Background Goroutines for Long-Running Tasks

```go
// Start long-running task in goroutine (non-blocking)
go func() {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Context().Done():
            return // Graceful shutdown
        case <-ticker.C:
            // Periodic work
        }
    }
}()
```

### 5. MainVerticle Pattern

```go
app, err := entrypoint.NewMainVerticle("")
if err != nil {
    panic(err)
}

// Deploy verticles
app.DeployVerticle(NewMyVerticle())

// Start (blocks until SIGINT/SIGTERM)
app.Start()
```

## Best Practices

1. **Always call `BaseVerticle.Start()` first** in your `Start()` method
2. **Never block in `Start()`** - use goroutines for blocking operations
3. **Respect context cancellation** - check `ctx.Context().Done()` in goroutines
4. **Clean up resources in `Stop()`** - stop servers, close connections, etc.
5. **Use structured logging** - use `core.NewDefaultLogger()` for consistent logging
6. **Handle errors properly** - return errors, don't panic (except in main)
7. **Use convenience methods** - `v.Publish()`, `v.Consumer()`, etc. from BaseVerticle

## Architecture

```
┌─────────────────┐
│  MainVerticle   │
│  (Entry Point)  │
└────────┬────────┘
         │
         ├──► PingVerticle ──► EventBus ──► PongVerticle (x2)
         │
         └──► HTTPExampleVerticle
                  │
                  └──► FastHTTPServer (:8080)
```

## Graceful Shutdown

The application handles graceful shutdown automatically:

1. On SIGINT/SIGTERM, MainVerticle stops all verticles
2. Each verticle's `Stop()` method is called
3. Resources are cleaned up (servers stopped, goroutines cancelled)
4. Application exits cleanly

## See Also

- [BaseVerticle Documentation](../../pkg/core/BASE_CLASSES.md)
- [Fluxor Project Rules](../../.cursorrules)
- [SSR Example](../ssr-example/README.md)
