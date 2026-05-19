# Proxy Server Package

High-performance proxy server with load balancing, health checking, and fail-fast design principles.

## 🎯 Overview

The proxy server package provides a production-ready proxy implementation with:

- ✅ **HTTP and TCP proxy support** - Forward HTTP requests and TCP connections
- ✅ **Load balancing** - Multiple strategies (round-robin, least-connections, weighted, random)
- ✅ **Health checking** - Automatic backend health monitoring
- ✅ **Fail-fast design** - Immediate validation and clear error messages
- ✅ **Rate limiting** - Configurable request rate limits
- ✅ **Graceful shutdown** - Clean connection termination
- ✅ **Rich metrics** - Performance monitoring and observability
- ✅ **EventBus integration** - Reactive patterns with Fluxor EventBus

---

## 🚀 Quick Start

### Basic HTTP Proxy

```go
package main

import (
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/proxy"
)

func main() {
    // Create runtime
    gocmd := core.NewGoCMD(context.Background())
    
    // Create proxy configuration
    config := proxy.DefaultConfig()
    config.ListenAddr = ":8080"
    config.Backends = []proxy.Backend{
        {URL: "http://localhost:3000", Weight: 1},
        {URL: "http://localhost:3001", Weight: 1},
    }
    
    // Create proxy server
    proxyServer := proxy.NewProxyServer(gocmd, config)
    
    // Start proxy server (blocking)
    if err := proxyServer.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### Using Proxy Component

```go
type MyVerticle struct {
    *core.BaseVerticle
    proxy *proxy.ProxyComponent
}

func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    // Create proxy component
    config := proxy.DefaultConfig()
    config.ListenAddr = ":8080"
    config.Backends = []proxy.Backend{
        {URL: "http://localhost:3000"},
        {URL: "http://localhost:3001"},
    }
    v.proxy = proxy.NewProxyComponent(config)
    
    // Start component
    return v.proxy.Start(ctx)
}

func (v *MyVerticle) Stop(ctx core.FluxorContext) error {
    if v.proxy != nil {
        return v.proxy.Stop(ctx)
    }
    return nil
}
```

---

## 📋 Configuration

### Config Structure

```go
type Config struct {
    ListenAddr            string        // Listen address (e.g., ":8080")
    Protocol              string        // "http", "tcp", or "both"
    Backends              []Backend     // Backend servers
    LoadBalancingStrategy string        // "round-robin", "least-connections", "weighted", "random"
    HealthCheckInterval   time.Duration // Health check interval
    HealthCheckTimeout    time.Duration // Health check timeout
    MaxConnections        int           // Max concurrent connections (0 = unlimited)
    ConnectionTimeout     time.Duration // Backend connection timeout
    ReadTimeout           time.Duration // Read timeout
    WriteTimeout          time.Duration // Write timeout
    IdleTimeout           time.Duration // Idle connection timeout
    EnableMetrics         bool          // Enable metrics collection
    RateLimit             int           // Requests per second (0 = unlimited)
}
```

### Default Configuration

```go
config := proxy.DefaultConfig()
// ListenAddr: ":8080"
// Protocol: "http"
// LoadBalancingStrategy: "round-robin"
// HealthCheckInterval: 30s
// HealthCheckTimeout: 5s
// MaxConnections: 0 (unlimited)
// ConnectionTimeout: 30s
// ReadTimeout: 30s
// WriteTimeout: 30s
// IdleTimeout: 90s
// EnableMetrics: true
// RateLimit: 0 (unlimited)
```

### Custom Configuration

```go
config := proxy.Config{
    ListenAddr:            ":8080",
    Protocol:              "http",
    Backends: []proxy.Backend{
        {URL: "http://localhost:3000", Weight: 1},
        {URL: "http://localhost:3001", Weight: 2}, // Higher weight = more traffic
    },
    LoadBalancingStrategy: "weighted",
    HealthCheckInterval:   10 * time.Second,
    MaxConnections:        1000,
    RateLimit:             100, // 100 requests per second
}
```

### Environment Variables

```bash
export PROXY_LISTEN_ADDR=":8080"
export PROXY_PROTOCOL="http"
export PROXY_LB_STRATEGY="round-robin"
export PROXY_HEALTH_CHECK_INTERVAL="30s"
export PROXY_MAX_CONNECTIONS="1000"
export PROXY_RATE_LIMIT="100"
```

---

## 🏗️ Architecture

### Components

```
Proxy Server Architecture
│
├─ Listener (HTTP/TCP)
│   └─ Accept connections/requests
│
├─ Load Balancer
│   ├─ Round-robin
│   ├─ Least-connections
│   ├─ Weighted
│   └─ Random
│
├─ Health Checker
│   ├─ Periodic health checks
│   ├─ Mark backends as healthy/unhealthy
│   └─ Automatic failover
│
├─ Connection Pool
│   ├─ Track active connections
│   └─ Enforce max connections
│
└─ Metrics Collector
    ├─ Connection metrics
    ├─ Request metrics
    └─ Backend metrics
```

### Request Flow

```
1. Client Request
   ↓
2. Rate Limiting Check
   → Reject if exceeded
   ↓
3. Select Backend (Load Balancer)
   → Fail if no healthy backends
   ↓
4. Forward Request/Connection
   ↓
5. Update Metrics
   ↓
6. Return Response
```

---

## 🎨 Usage Patterns

### Pattern 1: HTTP Proxy with Multiple Backends

```go
config := proxy.DefaultConfig()
config.ListenAddr = ":8080"
config.Backends = []proxy.Backend{
    {URL: "http://api1.example.com:3000", Weight: 1},
    {URL: "http://api2.example.com:3000", Weight: 1},
    {URL: "http://api3.example.com:3000", Weight: 1},
}
config.LoadBalancingStrategy = "round-robin"

proxyServer := proxy.NewProxyServer(gocmd, config)
proxyServer.Start()
```

### Pattern 2: TCP Proxy

```go
config := proxy.DefaultConfig()
config.ListenAddr = ":9000"
config.Protocol = "tcp"
config.Backends = []proxy.Backend{
    {URL: "tcp://backend1.example.com:9000"},
    {URL: "tcp://backend2.example.com:9000"},
}
config.LoadBalancingStrategy = "least-connections"

proxyServer := proxy.NewProxyServer(gocmd, config)
proxyServer.Start()
```

### Pattern 3: Weighted Load Balancing

```go
config := proxy.DefaultConfig()
config.ListenAddr = ":8080"
config.Backends = []proxy.Backend{
    {URL: "http://primary.example.com:3000", Weight: 3},  // 60% traffic
    {URL: "http://secondary.example.com:3000", Weight: 2}, // 40% traffic
}
config.LoadBalancingStrategy = "weighted"

proxyServer := proxy.NewProxyServer(gocmd, config)
proxyServer.Start()
```

### Pattern 4: Dynamic Backend Management

```go
proxyServer := proxy.NewProxyServer(gocmd, config)
proxyServer.Start()

// Add backend at runtime
newBackend := proxy.Backend{
    URL:    "http://new-backend.example.com:3000",
    Weight: 1,
}
proxyServer.AddBackend(newBackend)

// Remove backend at runtime
proxyServer.RemoveBackend("http://old-backend.example.com:3000")
```

### Pattern 5: EventBus Integration

```go
// Listen for proxy events
eventBus := gocmd.EventBus()

eventBus.Consumer("proxy.ready").Handler(func(ctx core.FluxorContext, msg core.Message) error {
    var data map[string]interface{}
    if err := msg.Decode(&data); err != nil {
        return err
    }
    
    listenAddr := data["listenAddr"].(string)
    log.Printf("Proxy ready on %s", listenAddr)
    return nil
})
```

---

## 📊 Metrics & Monitoring

### Server Metrics

```go
metrics := proxyServer.Metrics()

fmt.Printf("Total Connections: %d\n", metrics.TotalConnections)
fmt.Printf("Active Connections: %d\n", metrics.ActiveConnections)
fmt.Printf("Total Requests: %d\n", metrics.TotalRequests)
fmt.Printf("Successful Requests: %d\n", metrics.SuccessfulRequests)
fmt.Printf("Failed Requests: %d\n", metrics.FailedRequests)
fmt.Printf("Healthy Backends: %d\n", metrics.HealthyBackends)
fmt.Printf("Requests/Second: %.2f\n", metrics.RequestsPerSecond)
```

### Metrics Structure

```go
type ServerMetrics struct {
    TotalConnections    int64   // Total connections accepted
    ActiveConnections   int64   // Current active connections
    RejectedConnections int64   // Connections rejected (rate limit, max conns)
    FailedConnections   int64   // Failed connections
    TotalRequests       int64   // Total HTTP requests
    SuccessfulRequests  int64   // Successful requests
    FailedRequests      int64   // Failed requests
    AverageResponseTime float64 // Average response time (ms)
    BackendCount        int     // Total backends
    HealthyBackends     int     // Healthy backends
    UnhealthyBackends   int     // Unhealthy backends
    RequestsPerSecond   float64 // Current requests per second
    BytesTransferred    int64   // Total bytes transferred
}
```

### Health Check Endpoint

```go
// HTTP health endpoint
router.GET("/proxy/metrics", func(ctx *web.RequestContext) error {
    metrics := proxyServer.Metrics()
    return ctx.JSON(200, metrics)
})

router.GET("/proxy/health", func(ctx *web.RequestContext) error {
    metrics := proxyServer.Metrics()
    
    healthy := metrics.HealthyBackends > 0
    statusCode := 200
    if !healthy {
        statusCode = 503
    }
    
    return ctx.JSON(statusCode, map[string]interface{}{
        "healthy":          healthy,
        "healthy_backends": metrics.HealthyBackends,
        "total_backends":   metrics.BackendCount,
    })
})
```

---

## ⚖️ Load Balancing Strategies

### Round-Robin

Distributes requests evenly across all healthy backends in sequence.

```go
config.LoadBalancingStrategy = "round-robin"
```

**Use when**: All backends have similar capacity and you want even distribution.

### Least-Connections

Routes to the backend with the fewest active connections.

```go
config.LoadBalancingStrategy = "least-connections"
```

**Use when**: Backends have different processing speeds or connection handling capacity.

### Weighted

Distributes traffic based on backend weights (higher weight = more traffic).

```go
config.LoadBalancingStrategy = "weighted"
config.Backends = []proxy.Backend{
    {URL: "http://backend1:3000", Weight: 3}, // 60% traffic
    {URL: "http://backend2:3000", Weight: 2}, // 40% traffic
}
```

**Use when**: Backends have different capacities and you want proportional distribution.

### Random

Randomly selects a backend from healthy backends.

```go
config.LoadBalancingStrategy = "random"
```

**Use when**: You want simple random distribution (less common).

---

## 🛡️ Health Checking

### Automatic Health Checks

The proxy automatically performs health checks on configured intervals:

```go
config.HealthCheckInterval = 30 * time.Second
config.HealthCheckTimeout = 5 * time.Second
```

### Health Check Behavior

- **HTTP backends**: TCP connection check to backend URL
- **TCP backends**: TCP connection check to backend address
- **Unhealthy backends**: Automatically excluded from load balancing
- **Recovery**: Unhealthy backends are re-checked on next interval

### Custom Health Check URLs

```go
config.Backends = []proxy.Backend{
    {
        URL:             "http://api.example.com:3000",
        HealthCheckURL:  "http://api.example.com:3000/health", // Optional
        HealthCheckInterval: 10 * time.Second,
    },
}
```

---

## 🔧 Rate Limiting

### Enable Rate Limiting

```go
config.RateLimit = 100 // 100 requests per second
```

### Rate Limiting Behavior

- **HTTP requests**: Rate limited per second
- **TCP connections**: Not rate limited (connection-based)
- **Rejection**: Returns `429 Too Many Requests` for HTTP
- **Automatic**: Uses token bucket algorithm

---

## 🎯 Best Practices

### 1. Configure Appropriate Timeouts

```go
config.ConnectionTimeout = 30 * time.Second  // Backend connection
config.ReadTimeout = 30 * time.Second        // Read from backend
config.WriteTimeout = 30 * time.Second       // Write to backend
config.IdleTimeout = 90 * time.Second        // Idle connections
```

### 2. Set Max Connections for Resource Protection

```go
config.MaxConnections = 1000 // Prevent resource exhaustion
```

### 3. Use Health Checks for Reliability

```go
config.HealthCheckInterval = 10 * time.Second // Frequent checks
config.HealthCheckTimeout = 5 * time.Second   // Fast timeout
```

### 4. Monitor Metrics

```go
go func() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        metrics := proxyServer.Metrics()
        
        if metrics.HealthyBackends == 0 {
            logger.Warn("No healthy backends available")
        }
        
        if metrics.FailedRequests > 0 {
            logger.Warn("Failed requests detected", "count", metrics.FailedRequests)
        }
    }
}()
```

### 5. Use Weighted Load Balancing for Different Capacities

```go
// Primary server gets 70% traffic
// Secondary server gets 30% traffic
config.Backends = []proxy.Backend{
    {URL: "http://primary:3000", Weight: 7},
    {URL: "http://secondary:3000", Weight: 3},
}
config.LoadBalancingStrategy = "weighted"
```

---

## 🧪 Testing

### Unit Tests

```bash
go test ./pkg/proxy/...
```

### Integration Test

```go
func TestProxyServer(t *testing.T) {
    gocmd := core.NewGoCMD(context.Background())
    
    // Start backend server
    backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
        w.Write([]byte("OK"))
    }))
    defer backend.Close()
    
    // Create proxy
    config := proxy.DefaultConfig()
    config.ListenAddr = ":0" // Random port
    config.Backends = []proxy.Backend{
        {URL: backend.URL},
    }
    
    proxyServer := proxy.NewProxyServer(gocmd, config)
    
    go proxyServer.Start()
    defer proxyServer.Stop()
    
    // Test proxy
    resp, err := http.Get("http://" + proxyServer.Addr() + "/test")
    require.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
}
```

---

## 🆚 Comparison

### vs nginx/haproxy

| Feature | pkg/proxy | nginx/haproxy |
|---------|-----------|---------------|
| **Configuration** | Go code/YAML | Config files |
| **Dynamic Updates** | ✅ Runtime API | ⚠️ Reload required |
| **EventBus Integration** | ✅ Yes | ❌ No |
| **Metrics** | ✅ Built-in | ⚠️ External tools |
| **Health Checks** | ✅ Built-in | ✅ Built-in |
| **Load Balancing** | ✅ Multiple strategies | ✅ Multiple strategies |

### vs pkg/web (HTTP Server)

| Feature | pkg/proxy | pkg/web |
|---------|-----------|---------|
| **Purpose** | Forward requests | Handle requests |
| **Protocols** | HTTP + TCP | HTTP only |
| **Load Balancing** | ✅ Yes | ❌ No |
| **Health Checks** | ✅ Yes | ❌ No |
| **Use Case** | Reverse proxy | Application server |

---

## 📚 Related Documentation

- [pkg/tcp/README.md](../tcp/README.md) - TCP server implementation
- [pkg/web/README.md](../web/README.md) - HTTP server implementation
- [pkg/core/README.md](../core/README.md) - Core components

---

## 🔗 Integration Examples

### With pkg/web (Hybrid Setup)

```go
// HTTP application server
httpServer := web.NewFastHTTPServer(gocmd, web.DefaultConfig(":3000"))
go httpServer.Start()

// Proxy in front
proxyConfig := proxy.DefaultConfig()
proxyConfig.ListenAddr = ":8080"
proxyConfig.Backends = []proxy.Backend{
    {URL: "http://localhost:3000"},
}
proxyServer := proxy.NewProxyServer(gocmd, proxyConfig)
go proxyServer.Start()
```

### With EventBus

```go
// Proxy publishes events
eventBus.Consumer("proxy.request").Handler(func(ctx core.FluxorContext, msg core.Message) error {
    // Log requests, update metrics, etc.
    return nil
})
```

---

**Package**: `github.com/fluxorio/fluxor/pkg/proxy`  
**Status**: ✅ Stable  
**Test Coverage**: TBD  
**Last Updated**: 2026-01-16
