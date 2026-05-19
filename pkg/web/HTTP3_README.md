# HTTP/3 Server Implementation

High-performance HTTP/3 (QUIC) server implementation for Fluxor web package.

## 🎯 Overview

HTTP/3 is the latest version of the HTTP protocol, built on top of QUIC (Quick UDP Internet Connections). It provides:

- ✅ **Improved Performance** - Lower latency, especially on high-latency connections
- ✅ **Better Multiplexing** - No head-of-line blocking
- ✅ **Connection Migration** - Seamless connection migration when switching networks
- ✅ **Built-in Encryption** - TLS 1.3 is mandatory
- ✅ **HTTP/1.1 and HTTP/2 Fallback** - Automatic fallback for compatibility

## 🚀 Quick Start

### Basic HTTP/3 Server

```go
package main

import (
    "context"
    "crypto/tls"
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/web"
)

func main() {
    // Create runtime
    gocmd := core.NewGoCMD(context.Background())
    
    // Create TLS configuration (required for HTTP/3)
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert}, // Your TLS certificate
        MinVersion:   tls.VersionTLS12,
        MaxVersion:   tls.VersionTLS13,
        NextProtos:   []string{"h3", "h2", "http/1.1"},
    }
    
    // Create HTTP/3 server configuration
    config := web.DefaultHTTP3ServerConfig(":443", tlsConfig)
    
    // Create HTTP/3 server
    server, err := web.NewHTTP3Server(gocmd, config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Set up routes
    router := server.Router()
    router.GET("/", func(ctx *web.RequestContext) error {
        return ctx.JSON(200, map[string]string{"message": "Hello HTTP/3!"})
    })
    
    // Start server
    if err := server.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### HTTP/3 with UDP Package Integration

Reuse UDP package infrastructure for better resource sharing:

```go
package main

import (
    "context"
    "crypto/tls"
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/udp"
    "github.com/fluxorio/fluxor/pkg/web"
)

func main() {
    gocmd := core.NewGoCMD(context.Background())
    
    // Create UDP server first
    udpConfig := udp.DefaultUDPServerConfig(":9001")
    udpServer := udp.NewUDPServer(gocmd, udpConfig)
    
    // Start UDP server
    go udpServer.Start()
    
    // Create TLS config
    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        NextProtos:   []string{"h3", "h2", "http/1.1"},
    }
    
    // Create HTTP/3 server reusing UDP server's PacketConn
    config := web.DefaultHTTP3ServerConfig(":443", tlsConfig)
    config.UDPServer = udpServer  // Reuse UDP server's PacketConn
    config.NormalCapacity = 1000 // Enable backpressure
    
    server, err := web.NewHTTP3Server(gocmd, config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Use router as normal
    router := server.Router()
    router.GET("/", handler)
    
    server.Start()
}
```

### Using TLS Certificates from Files

```go
import (
    "crypto/tls"
    "github.com/fluxorio/fluxor/pkg/web"
)

// Load TLS certificate
cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
if err != nil {
    log.Fatal(err)
}

tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{cert},
    MinVersion:   tls.VersionTLS12,
    MaxVersion:   tls.VersionTLS13,
    NextProtos:   []string{"h3", "h2", "http/1.1"},
}

config := web.DefaultHTTP3ServerConfig(":443", tlsConfig)
server, err := web.NewHTTP3Server(gocmd, config)
```

## 📋 Configuration

### HTTP3ServerConfig

```go
type HTTP3ServerConfig struct {
    Addr            string        // Listen address (e.g., ":443")
    TLSConfig       *tls.Config   // TLS configuration (required)
    EnableFallback  bool          // Enable HTTP/1.1 and HTTP/2 fallback
    FallbackAddr    string        // Address for fallback server (if different)
    ReadTimeout     time.Duration // Read timeout
    WriteTimeout    time.Duration // Write timeout
    IdleTimeout     time.Duration // Idle connection timeout
    MaxHeaderBytes  int           // Maximum header size
    ReadBufferSize  int           // QUIC read buffer size
    WriteBufferSize int           // QUIC write buffer size
    
    // UDP Package Integration
    PacketConn      net.PacketConn // Optional: reuse existing UDP PacketConn
    UDPServer       *udp.UDPServer // Optional: reuse UDP server's PacketConn
    NormalCapacity  int            // Backpressure: target capacity (0 = disabled)
    MaxRequestsPerSecond int       // Rate limit: max requests/second (0 = unlimited)
}
```

### Default Configuration

```go
config := web.DefaultHTTP3ServerConfig(":443", tlsConfig)
// Addr: ":443"
// EnableFallback: true
// ReadTimeout: 30s
// WriteTimeout: 30s
// IdleTimeout: 120s
// MaxHeaderBytes: 1MB
// ReadBufferSize: 64KB
// WriteBufferSize: 64KB
```

## 🏗️ Architecture

### Components

```
HTTP/3 Server Architecture
│
├─ HTTP/3 Server (QUIC)
│   └─ Handles HTTP/3 connections
│
├─ HTTP Fallback Server (HTTP/1.1 & HTTP/2)
│   └─ Handles non-HTTP/3 connections
│
├─ Router
│   └─ Routes requests to handlers
│
└─ Metrics Collector
    ├─ HTTP/3 request tracking
    ├─ Fallback request tracking
    └─ Performance metrics
```

### Request Flow

```
1. Client Connects
   ↓
2. Protocol Negotiation
   → HTTP/3 (QUIC) if supported
   → HTTP/2 or HTTP/1.1 fallback
   ↓
3. Request Routing
   ↓
4. Handler Execution
   ↓
5. Response
```

## 📊 Metrics

### HTTP3Metrics

```go
type HTTP3Metrics struct {
    // Request metrics
    TotalRequests      int64   // Total requests
    HTTP3Requests      int64   // HTTP/3 requests
    FallbackRequests   int64   // HTTP/1.1/2 fallback requests
    SuccessfulRequests int64   // Successful requests
    ErrorRequests      int64   // Error requests
    RejectedRequests   int64   // Rejected requests (backpressure)
    
    // Traffic metrics
    BytesSent     int64 // Bytes sent
    BytesReceived int64 // Bytes received
    
    // Backpressure metrics (aligned with UDP package)
    BackpressureMetrics *udp.BackpressureMetrics // Backpressure statistics
    QueueUtilization    float64                 // Queue utilization (0-100%)
    NormalCapacity      int64                   // Target capacity
    CurrentLoad         int64                   // Current load
    Utilization         float64                 // Utilization percentage
}
```

### Accessing Metrics

```go
http3Server := server.(*web.HTTP3Server)
metrics := http3Server.Metrics()

fmt.Printf("HTTP/3 Requests: %d\n", metrics.HTTP3Requests)
fmt.Printf("Fallback Requests: %d\n", metrics.FallbackRequests)
fmt.Printf("Total Requests: %d\n", metrics.TotalRequests)
fmt.Printf("Rejected Requests: %d\n", metrics.RejectedRequests)

// Backpressure metrics
if metrics.BackpressureMetrics != nil {
    fmt.Printf("Utilization: %.2f%%\n", metrics.Utilization)
    fmt.Printf("Current Load: %d / %d\n", metrics.CurrentLoad, metrics.NormalCapacity)
}
```

## 🔧 TLS Configuration

### Requirements

- **TLS 1.2+ Required** - HTTP/3 requires TLS
- **ALPN Support** - Must include "h3" in NextProtos
- **Valid Certificates** - Real certificates required for production

### Example TLS Config

```go
tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{cert},
    MinVersion:   tls.VersionTLS12,
    MaxVersion:   tls.VersionTLS13,
    NextProtos:   []string{"h3", "h2", "http/1.1"}, // Important: h3 first
    CipherSuites: []uint16{
        tls.TLS_AES_128_GCM_SHA256,
        tls.TLS_AES_256_GCM_SHA384,
        tls.TLS_CHACHA20_POLY1305_SHA256,
    },
}
```

## 🎨 Usage Patterns

### Pattern 1: HTTP/3 Only

```go
config := &web.HTTP3ServerConfig{
    Addr:           ":443",
    TLSConfig:      tlsConfig,
    EnableFallback: false, // Disable fallback
}
```

### Pattern 2: HTTP/3 with Fallback

```go
config := web.DefaultHTTP3ServerConfig(":443", tlsConfig)
// EnableFallback is true by default
```

### Pattern 3: Custom Timeouts

```go
config := &web.HTTP3ServerConfig{
    Addr:           ":443",
    TLSConfig:      tlsConfig,
    EnableFallback: true,
    ReadTimeout:    60 * time.Second,
    WriteTimeout:   60 * time.Second,
    IdleTimeout:    300 * time.Second,
}
```

### Pattern 4: Reuse UDP Package PacketConn

```go
// Option 1: Provide PacketConn directly
packetConn, _ := net.ListenPacket("udp", ":443")
config := &web.HTTP3ServerConfig{
    Addr:       ":443",
    TLSConfig:  tlsConfig,
    PacketConn: packetConn, // Reuse existing PacketConn
}

// Option 2: Reuse from UDP server
udpServer := udp.NewUDPServer(gocmd, udp.DefaultUDPServerConfig(":9001"))
go udpServer.Start()

config := &web.HTTP3ServerConfig{
    Addr:       ":443",
    TLSConfig:  tlsConfig,
    UDPServer:  udpServer, // Automatically gets PacketConn from UDP server
}
```

### Pattern 5: HTTP/3 with Backpressure

```go
config := &web.HTTP3ServerConfig{
    Addr:           ":443",
    TLSConfig:      tlsConfig,
    NormalCapacity: 1000, // Target capacity for backpressure
    // When capacity exceeded, requests return 503 Service Unavailable
}
```

### Pattern 6: HTTP/3 with Rate Limiting

```go
config := &web.HTTP3ServerConfig{
    Addr:                ":443",
    TLSConfig:           tlsConfig,
    MaxRequestsPerSecond: 1000, // Limit to 1000 req/s
}
```

## 🆚 Comparison

### vs HTTP/2

| Feature | HTTP/3 | HTTP/2 |
|---------|--------|--------|
| **Transport** | QUIC (UDP) | TCP |
| **Head-of-Line Blocking** | ❌ No | ⚠️ Yes |
| **Connection Migration** | ✅ Yes | ❌ No |
| **Latency** | ✅ Lower | ⚠️ Higher |
| **Multiplexing** | ✅ Better | ⚠️ Good |

### vs FastHTTPServer

| Feature | HTTP3Server | FastHTTPServer |
|---------|-------------|----------------|
| **Protocol** | HTTP/3 (QUIC) | HTTP/1.1 |
| **TLS Required** | ✅ Yes | ⚠️ Optional |
| **Performance** | ✅ Excellent | ✅ Excellent |
| **Compatibility** | ⚠️ Modern clients | ✅ All clients |

## 🧪 Testing

### Unit Tests

```bash
go test ./pkg/web/... -v
```

### Integration Test

```go
func TestHTTP3ServerIntegration(t *testing.T) {
    // Requires real TLS certificates
    // Requires HTTP/3 client (quic-go client)
    // See http3_server_test.go for basic tests
}
```

## ⚠️ Limitations

- **TLS Required** - HTTP/3 requires TLS, cannot run without encryption
- **Client Support** - Requires HTTP/3 capable clients (modern browsers, curl with quic support)
- **Network Requirements** - Some networks/firewalls may block UDP (QUIC uses UDP)
- **Certificate Management** - Requires proper TLS certificate management

## 🔗 Dependencies

- `github.com/quic-go/quic-go/http3` - QUIC/HTTP3 implementation
- `crypto/tls` - TLS support (standard library)

## 📚 Related Documentation

- [pkg/web/README.md](./README.md) - Web package overview
- [pkg/web/fast_server.go](./fast_server.go) - FastHTTP server
- [pkg/web/http_server.go](./http_server.go) - Standard HTTP server

---

## 🔗 UDP Package Integration

HTTP/3 can reuse UDP package infrastructure for better resource sharing and consistency.

### Benefits

1. **Resource Efficiency**: Share UDP socket between HTTP/3 and raw UDP services
2. **Unified Backpressure**: Use UDP package's backpressure controller
3. **Consistent Metrics**: Aligned metrics structure with UDP package
4. **Better Monitoring**: Single metrics endpoint for UDP-based services

### Usage Examples

#### Reuse UDP Server's PacketConn

```go
// Create and start UDP server
udpServer := udp.NewUDPServer(gocmd, udp.DefaultUDPServerConfig(":9001"))
go udpServer.Start()

// Create HTTP/3 server reusing UDP server's PacketConn
config := web.DefaultHTTP3ServerConfig(":443", tlsConfig)
config.UDPServer = udpServer // Automatically gets PacketConn

server, _ := web.NewHTTP3Server(gocmd, config)
server.Start()
```

#### Use Custom PacketConn

```go
// Create custom UDP listener
packetConn, _ := net.ListenPacket("udp", ":443")

// Use with HTTP/3
config := web.DefaultHTTP3ServerConfig(":443", tlsConfig)
config.PacketConn = packetConn

server, _ := web.NewHTTP3Server(gocmd, config)
server.Start()
```

#### Enable Backpressure

```go
config := web.DefaultHTTP3ServerConfig(":443", tlsConfig)
config.NormalCapacity = 1000 // Enable backpressure with target capacity of 1000

// When capacity exceeded, requests return 503 Service Unavailable
server, _ := web.NewHTTP3Server(gocmd, config)
```

### Backpressure Behavior

- **NormalCapacity**: Target capacity for normal operations (e.g., 80% of max)
- **TryAcquire()**: Checks if capacity available before processing request
- **Release()**: Releases capacity when request completes
- **Rejection**: Returns `503 Service Unavailable` when capacity exceeded

### Metrics Alignment

HTTP/3 metrics are aligned with UDP package's `ServerMetrics`:
- Similar structure for consistency
- Backpressure metrics included
- Utilization tracking
- Queue metrics (where applicable)

---

**Package**: `github.com/fluxorio/fluxor/pkg/web`  
**Status**: ✅ Implemented  
**Protocol**: HTTP/3 (QUIC)  
**UDP Integration**: ✅ Supported  
**Last Updated**: 2026-01-16
