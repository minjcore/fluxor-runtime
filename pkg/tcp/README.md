# TCP Server Package

High-performance TCP server with CCU-based backpressure control, worker pool architecture, and fail-fast design principles.

## 🎯 Overview

The TCP server package provides a production-ready TCP server implementation with:

- ✅ **CCU-based backpressure**: Automatic connection rejection when capacity exceeded
- ✅ **Worker pool architecture**: Configurable concurrency with bounded queue
- ✅ **Fail-fast design**: Immediate validation and clear error messages
- ✅ **Middleware support**: Composable connection handlers
- ✅ **Graceful shutdown**: Clean connection termination
- ✅ **Rich metrics**: Performance monitoring and observability
- ✅ **Panic isolation**: Per-connection panic recovery

---

## 🚀 Quick Start

### Basic TCP Server

```go
package main

import (
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/tcp"
)

func main() {
    // Create runtime
    gocmd := core.NewGoCMD(context.Background())
    
    // Create TCP server
    config := tcp.DefaultTCPServerConfig(":9000")
    server := tcp.NewTCPServer(gocmd, config)
    
    // Set connection handler
    server.SetHandler(func(ctx *tcp.ConnContext) error {
        // Read data
        buf := make([]byte, 1024)
        n, err := ctx.Conn.Read(buf)
        if err != nil {
            return err
        }
        
        // Process and respond
        response := []byte("Echo: " + string(buf[:n]))
        _, err = ctx.Conn.Write(response)
        return err
    })
    
    // Start server (blocking)
    if err := server.Start(); err != nil {
        log.Fatal(err)
    }
}
```

---

## 📋 Configuration

### TCPServerConfig

```go
type TCPServerConfig struct {
    Addr         string        // Listen address (e.g., ":9000")
    MaxQueue     int           // Bounded queue size (default: 1000)
    Workers      int           // Worker pool size (default: 50)
    MaxConns     int           // Max concurrent connections (0 = unlimited)
    TLSConfig    *tls.Config   // TLS configuration (optional)
    ReadTimeout  time.Duration // Per-connection read timeout
    WriteTimeout time.Duration // Per-connection write timeout
}
```

### Default Configuration

```go
config := tcp.DefaultTCPServerConfig(":9000")
// MaxQueue: 1000
// Workers: 50
// MaxConns: 0 (unlimited)
// ReadTimeout: 5 seconds
// WriteTimeout: 5 seconds
```

### Custom Configuration

```go
config := &tcp.TCPServerConfig{
    Addr:         ":9000",
    MaxQueue:     2000,        // Larger queue for burst traffic
    Workers:      100,         // More workers for higher concurrency
    MaxConns:     5000,        // Hard limit on connections
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 10 * time.Second,
}

server := tcp.NewTCPServer(gocmd, config)
```

---

## 🏗️ Architecture

### Components

```
TCP Server Architecture
│
├─ Listener (net.Listener)
│   └─ Accept() → new connections
│
├─ Backpressure Controller
│   ├─ Normal Capacity = MaxQueue + Workers
│   ├─ Reject if capacity exceeded
│   └─ Periodic reset (60s)
│
├─ Connection Mailbox (Bounded Queue)
│   ├─ Size: MaxQueue (default: 1000)
│   └─ Blocks when full (backpressure)
│
├─ Worker Pool
│   ├─ Size: Workers (default: 50)
│   ├─ Each worker: Receive → Handle → Close
│   └─ Panic isolation per-connection
│
└─ Metrics (Atomic Counters)
    ├─ Accepted, Queued, Rejected
    ├─ Handled, Errors, Active
    └─ Queue/CCU utilization
```

### Connection Flow

```
1. Accept Connection
   ↓
2. Check MaxConns (if configured)
   → Reject if exceeded
   ↓
3. Check Backpressure (Normal Capacity)
   → Reject if exceeded
   ↓
4. Send to Mailbox (Bounded Queue)
   → Reject if queue full
   ↓
5. Worker Receives from Mailbox
   ↓
6. Execute Handler (with panic isolation)
   ↓
7. Close Connection
   ↓
8. Release Backpressure & Slot
```

---

## 🎨 Usage Patterns

### Pattern 1: Echo Server

```go
server.SetHandler(func(ctx *tcp.ConnContext) error {
    buf := make([]byte, 1024)
    n, err := ctx.Conn.Read(buf)
    if err != nil {
        return err
    }
    
    _, err = ctx.Conn.Write(buf[:n])
    return err
})
```

### Pattern 2: Protocol Buffers Server

```go
import (
    "github.com/fluxorio/fluxor/pkg/protobuf"
    "google.golang.org/protobuf/proto"
)

server.SetHandler(func(ctx *tcp.ConnContext) error {
    // Read length-prefixed protobuf message
    msg := &MyProtoMessage{}
    if err := protobuf.ReadMessage(ctx.Conn, msg); err != nil {
        return err
    }
    
    // Process message
    response := processMessage(msg)
    
    // Write response
    return protobuf.WriteMessage(ctx.Conn, response)
})
```

### Pattern 3: EventBus Integration

```go
server.SetHandler(func(ctx *tcp.ConnContext) error {
    // Read request
    buf := make([]byte, 1024)
    n, err := ctx.Conn.Read(buf)
    if err != nil {
        return err
    }
    
    // Publish to EventBus
    ctx.EventBus.Publish("tcp.data.received", map[string]interface{}{
        "data": string(buf[:n]),
        "from": ctx.RemoteAddr.String(),
    })
    
    // Acknowledge
    _, err = ctx.Conn.Write([]byte("ACK\n"))
    return err
})
```

### Pattern 4: Request-Reply via EventBus

```go
server.SetHandler(func(ctx *tcp.ConnContext) error {
    // Read request
    buf := make([]byte, 1024)
    n, err := ctx.Conn.Read(buf)
    if err != nil {
        return err
    }
    
    // Send to backend service via EventBus (request-reply)
    reply, err := ctx.EventBus.Request(
        "backend.process",
        buf[:n],
        5*time.Second,
    )
    if err != nil {
        return err
    }
    
    // Write response
    _, err = ctx.Conn.Write(reply.Body().([]byte))
    return err
})
```

### Pattern 5: Connection Context Storage

```go
server.SetHandler(func(ctx *tcp.ConnContext) error {
    // Store connection metadata (BaseRequestContext)
    ctx.Set("client_id", extractClientID(ctx.Conn))
    ctx.Set("session_start", time.Now())
    
    // Process request
    // ...
    
    // Access stored data
    clientID := ctx.Get("client_id").(string)
    
    return nil
})
```

---

## 🔧 Middleware

### Middleware Pattern

```go
type Middleware func(next ConnectionHandler) ConnectionHandler
```

### Built-in Middleware Examples

#### Logging Middleware

```go
func LoggingMiddleware(logger core.Logger) tcp.Middleware {
    return func(next tcp.ConnectionHandler) tcp.ConnectionHandler {
        return func(ctx *tcp.ConnContext) error {
            start := time.Now()
            
            logger.Info("Connection received",
                "remote", ctx.RemoteAddr.String(),
            )
            
            err := next(ctx)
            
            logger.Info("Connection handled",
                "remote", ctx.RemoteAddr.String(),
                "duration", time.Since(start),
                "error", err,
            )
            
            return err
        }
    }
}

// Usage
server.Use(LoggingMiddleware(logger))
```

#### Metrics Middleware

```go
func MetricsMiddleware(metrics *prometheus.Metrics) tcp.Middleware {
    return func(next tcp.ConnectionHandler) tcp.ConnectionHandler {
        return func(ctx *tcp.ConnContext) error {
            start := time.Now()
            metrics.TCPConnectionsTotal.Inc()
            
            err := next(ctx)
            
            duration := time.Since(start).Seconds()
            metrics.TCPConnectionDuration.Observe(duration)
            
            if err != nil {
                metrics.TCPConnectionErrors.Inc()
            }
            
            return err
        }
    }
}
```

#### Auth Middleware

```go
func AuthMiddleware(validTokens map[string]bool) tcp.Middleware {
    return func(next tcp.ConnectionHandler) tcp.ConnectionHandler {
        return func(ctx *tcp.ConnContext) error {
            // Read auth token
            buf := make([]byte, 64)
            n, err := ctx.Conn.Read(buf)
            if err != nil {
                return err
            }
            
            token := string(buf[:n])
            if !validTokens[token] {
                ctx.Conn.Write([]byte("UNAUTHORIZED\n"))
                return fmt.Errorf("invalid token")
            }
            
            // Store authenticated user
            ctx.Set("token", token)
            ctx.Set("authenticated", true)
            
            return next(ctx)
        }
    }
}
```

### Chaining Middleware

```go
server.Use(
    LoggingMiddleware(logger),
    MetricsMiddleware(metrics),
    AuthMiddleware(tokens),
)
```

**Note**: Last added middleware runs **outermost** (consistent with HTTP middleware pattern).

---

## 📊 Metrics & Monitoring

### Server Metrics

```go
metrics := server.Metrics()

fmt.Printf("Queue Utilization: %.2f%%\n", metrics.QueueUtilization)
fmt.Printf("CCU Utilization: %.2f%%\n", metrics.CCUUtilization)
fmt.Printf("Total Accepted: %d\n", metrics.TotalAccepted)
fmt.Printf("Rejected: %d\n", metrics.RejectedConnections)
fmt.Printf("Handled: %d\n", metrics.HandledConnections)
fmt.Printf("Errors: %d\n", metrics.ErrorConnections)
fmt.Printf("Active: %d\n", metrics.ActiveConnections)
```

### Metrics Structure

```go
type ServerMetrics struct {
    QueuedConnections   int64   // Current queued
    RejectedConnections int64   // Total rejected (backpressure)
    QueueCapacity       int     // Max queue size
    Workers             int     // Worker pool size
    QueueUtilization    float64 // Queue %
    NormalCCU           int     // Normal capacity (queue + workers)
    CurrentCCU          int     // Current load
    CCUUtilization      float64 // CCU %
    TotalAccepted       int64   // Total accepted
    HandledConnections  int64   // Total handled
    ErrorConnections    int64   // Total errors
    ActiveConnections   int64   // Current in-flight
    MaxConns            int     // Configured max (0 = unlimited)
}
```

### Health Check Endpoint

```go
// HTTP health endpoint
router.GETFast("/tcp/metrics", func(ctx *web.FastRequestContext) error {
    metrics := tcpServer.Metrics()
    return ctx.JSON(200, metrics)
})

// TCP health check
router.GETFast("/tcp/health", func(ctx *web.FastRequestContext) error {
    metrics := tcpServer.Metrics()
    
    healthy := metrics.QueueUtilization < 90.0 && metrics.CCUUtilization < 90.0
    statusCode := 200
    if !healthy {
        statusCode = 503
    }
    
    return ctx.JSON(statusCode, map[string]interface{}{
        "healthy": healthy,
        "queue_utilization": metrics.QueueUtilization,
        "ccu_utilization": metrics.CCUUtilization,
    })
})
```

---

## ⚡ Backpressure Control

### Multi-Layer Protection

The TCP server implements **3 layers** of backpressure protection:

#### Layer 1: MaxConns Limit

```go
config := &tcp.TCPServerConfig{
    MaxConns: 5000,  // Hard limit
}
```

Rejects connections when `ActiveConnections >= MaxConns`.

#### Layer 2: Backpressure Controller

```go
// Normal Capacity = MaxQueue + Workers
normalCapacity := 1000 + 50  // 1050

// Reject when current load exceeds normal capacity
if currentLoad >= normalCapacity {
    // Reject connection
}
```

Automatically resets every 60 seconds to handle traffic spikes.

#### Layer 3: Bounded Mailbox

```go
// Mailbox blocks when full
if mailbox.Size() >= mailbox.Capacity() {
    // Connection rejected (queue full)
}
```

### Backpressure Metrics

```go
metrics := server.Metrics()

// Queue utilization
fmt.Printf("Queue: %d/%d (%.2f%%)\n",
    metrics.QueuedConnections,
    metrics.QueueCapacity,
    metrics.QueueUtilization)

// CCU utilization (relative to normal capacity)
fmt.Printf("CCU: %d/%d (%.2f%%)\n",
    metrics.CurrentCCU,
    metrics.NormalCCU,
    metrics.CCUUtilization)

// Rejection rate
fmt.Printf("Rejected: %d (%.2f%% of accepted)\n",
    metrics.RejectedConnections,
    float64(metrics.RejectedConnections)/float64(metrics.TotalAccepted)*100)
```

---

## 🛡️ Error Handling & Resilience

### Panic Isolation

Each connection handler runs with **panic recovery**:

```go
// Panic in one connection does NOT terminate worker
defer func() {
    if r := recover(); r != nil {
        logger.Error("panic in tcp handler (isolated)", "error", r)
        // Worker continues processing other connections
    }
}()
```

### Timeout Protection

```go
config := &tcp.TCPServerConfig{
    ReadTimeout:  5 * time.Second,
    WriteTimeout: 5 * time.Second,
}

// Applied per-connection (best-effort)
conn.SetReadDeadline(time.Now().Add(config.ReadTimeout))
conn.SetWriteDeadline(time.Now().Add(config.WriteTimeout))
```

### Graceful Shutdown

```go
// Stop server
server.Stop()

// Behavior:
// 1. Stop accepting new connections
// 2. Close listener
// 3. Wait for in-flight connections to complete
// 4. Shutdown worker pool
```

---

## 🎯 Best Practices

### 1. Size Worker Pool Appropriately

```go
// CPU-bound: workers = CPU cores
config.Workers = runtime.NumCPU()

// I/O-bound: workers = 2-4x CPU cores
config.Workers = runtime.NumCPU() * 4

// Network with high latency: higher pool size
config.Workers = 100
```

### 2. Configure Queue Size for Traffic Patterns

```go
// Low-latency, predictable traffic
config.MaxQueue = 100

// Burst traffic tolerance
config.MaxQueue = 2000

// Balance: Default 1000
config.MaxQueue = 1000
```

### 3. Use MaxConns for Resource Protection

```go
// Production: Set hard limit
config.MaxConns = 10000

// Development: Unlimited
config.MaxConns = 0
```

### 4. Always Close Connections

```go
// ✅ Server automatically closes after handler returns
server.SetHandler(func(ctx *tcp.ConnContext) error {
    // Process connection
    return nil
    // Connection auto-closed by server
})
```

### 5. Monitor Metrics

```go
go func() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        metrics := server.Metrics()
        
        if metrics.CCUUtilization > 80.0 {
            logger.Warn("High CCU utilization", "pct", metrics.CCUUtilization)
        }
        
        if metrics.RejectedConnections > 0 {
            logger.Warn("Connections rejected", "count", metrics.RejectedConnections)
        }
    }
}()
```

### 6. Use Middleware for Cross-Cutting Concerns

```go
// ✅ Good: Reusable middleware
server.Use(
    LoggingMiddleware(logger),
    MetricsMiddleware(metrics),
)

// ❌ Bad: Duplicate logic in handler
server.SetHandler(func(ctx *tcp.ConnContext) error {
    logger.Info("Connection")
    metrics.Inc()
    // ... handler logic
})
```

---

## 🧪 Testing

### Unit Tests

```bash
go test ./pkg/tcp/...
```

### Integration Tests

```go
func TestTCPServer(t *testing.T) {
    gocmd := core.NewGoCMD(context.Background())
    server := tcp.NewTCPServer(gocmd, tcp.DefaultTCPServerConfig(":0"))
    
    server.SetHandler(func(ctx *tcp.ConnContext) error {
        _, err := ctx.Conn.Write([]byte("PONG\n"))
        return err
    })
    
    go server.Start()
    defer server.Stop()
    
    // Connect and test
    conn, err := net.Dial("tcp", server.Addr())
    require.NoError(t, err)
    defer conn.Close()
    
    buf := make([]byte, 1024)
    n, err := conn.Read(buf)
    require.NoError(t, err)
    assert.Equal(t, "PONG\n", string(buf[:n]))
}
```

---

## 🆚 Comparison

### vs net.Listener (Raw Go)

| Feature | pkg/tcp | net.Listener |
|---------|---------|--------------|
| **Backpressure** | ✅ Built-in | ❌ Manual |
| **Worker Pool** | ✅ Managed | ❌ Manual goroutines |
| **Metrics** | ✅ Built-in | ❌ Manual |
| **Middleware** | ✅ Yes | ❌ No |
| **Panic Isolation** | ✅ Yes | ❌ Manual |
| **Graceful Shutdown** | ✅ Yes | ⚠️ Manual |

### vs pkg/web (HTTP Server)

| Feature | pkg/tcp | pkg/web |
|---------|---------|---------|
| **Protocol** | TCP | HTTP |
| **Backpressure** | CCU-based | CCU-based |
| **Worker Pool** | ✅ Yes | ✅ Yes |
| **Middleware** | ✅ Yes | ✅ Yes |
| **Metrics** | ✅ Yes | ✅ Yes |
| **Use Case** | Custom protocols | REST APIs |

---

## 📚 Related Documentation

- [BEST_PRACTICES.md](BEST_PRACTICES.md) - Detailed best practices
- [pkg/core/concurrency/README.md](../core/concurrency/README.md) - Worker pools and executors
- [pkg/protobuf/README.md](../protobuf/README.md) - Protocol Buffers integration
- [examples/protobuf-tcp](../../examples/protobuf-tcp) - Complete example

---

## 🔗 Integration Examples

### With pkg/web (Hybrid Server)

```go
// HTTP + TCP in same application
func main() {
    gocmd := core.NewGoCMD(context.Background())
    
    // HTTP server
    httpServer := web.NewFastHTTPServer(gocmd, web.DefaultConfig(":8080"))
    go httpServer.Start()
    
    // TCP server
    tcpServer := tcp.NewTCPServer(gocmd, tcp.DefaultConfig(":9000"))
    go tcpServer.Start()
    
    // Both share same EventBus via gocmd
}
```

### With pkg/protobuf

```go
import "github.com/fluxorio/fluxor/pkg/protobuf"

server.SetHandler(func(ctx *tcp.ConnContext) error {
    msg := &pb.MyMessage{}
    if err := protobuf.ReadMessage(ctx.Conn, msg); err != nil {
        return err
    }
    
    response := &pb.MyResponse{Status: "OK"}
    return protobuf.WriteMessage(ctx.Conn, response)
})
```

---

**Package**: `github.com/fluxorio/fluxor/pkg/tcp`  
**Status**: ✅ Stable (B+ Grade)  
**Test Coverage**: 75%  
**Last Updated**: 2026-01-04
