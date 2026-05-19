# TCP Server: CPU-Bound vs IO-Bound Configuration Examples

This guide demonstrates how to configure TCP servers for CPU-bound and IO-bound workloads, following the patterns from `IO_BOUND_OPTIMIZATION.md`, `TLS_OPTIMIZATION_SUMMARY.md`, `TLS_PERFORMANCE.md`, and `THREAD_MANAGEMENT.md`.

## Overview

### CPU-Bound vs IO-Bound

| Characteristic | CPU-Bound | IO-Bound |
|----------------|-----------|----------|
| **Workers** | ≤ CPU cores | Many workers OK |
| **CPU Utilization** | 80-100% | 10-30% |
| **Blocking Operations** | Few | Many (network I/O) |
| **Thread Pinning** | ✅ Use `runtime.LockOSThread()` | ❌ Not needed |
| **Example** | Crypto, data processing | Network I/O, file I/O |

## Configuration Functions

### 1. IO-Bound Configuration

#### HighThroughputIOBoundConfig

For high connection rate scenarios with low CPU utilization:

```go
package main

import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/tcp"
)

func main() {
    gocmd := core.NewGoCMD(context.Background())
    
    // IO-bound config: 2000 workers, 50k queue, 100k max connections
    config := tcp.HighThroughputIOBoundConfig(":9000")
    
    server := tcp.NewTCPServer(gocmd, config)
    
    // IO-bound handler: Network I/O operations
    server.SetHandler(func(ctx *tcp.ConnContext) error {
        // Read from connection (network I/O - CPU idle)
        buf := make([]byte, 1024)
        n, err := ctx.Conn.Read(buf)
        if err != nil {
            return err
        }
        
        // Echo back (network I/O - CPU idle)
        _, err = ctx.Conn.Write(buf[:n])
        return err
    })
    
    // Start server (non-blocking in verticle)
    go func() {
        if err := server.Start(); err != nil {
            panic(err)
        }
    }()
    
    <-gocmd.Context().Done()
}
```

**Characteristics:**
- **Workers**: 2000 (many goroutines OK for IO-bound)
- **Queue**: 50,000 (large buffer for connection bursts)
- **MaxConns**: 100,000 (high connection limit)
- **Use Case**: Echo server, proxy, file transfer

#### HighThroughputIOBoundConfigWithTargetRPS

For specific connection rate targets:

```go
// Target: 100k connections/sec with 5ms latency (plain TCP)
config := tcp.HighThroughputIOBoundConfigWithTargetRPS(":9000", 100000, 5)
// Workers = (100k * 5ms) / 1000 = 500 workers
// Queue = 100k * 0.5 = 50k

server := tcp.NewTCPServer(gocmd, config)
```

**Formula:**
- `Workers = (targetRPS × avgLatencyMs) / 1000`
- `QueueSize = targetRPS × 0.5` (buffer for 500ms)

### 2. CPU-Bound Configuration

#### CPUBoundConfig

For CPU-intensive connection handling:

```go
package main

import (
    "context"
    "crypto/sha256"
    "runtime"
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/tcp"
)

func main() {
    gocmd := core.NewGoCMD(context.Background())
    
    // CPU-bound config: Workers = CPU cores, moderate queue
    config := tcp.CPUBoundConfig(":9000")
    // Workers = runtime.NumCPU() (e.g., 8 for 8-core CPU)
    // Queue = 1000
    // MaxConns = workers * 10
    
    server := tcp.NewTCPServer(gocmd, config)
    
    // CPU-bound handler: CPU-intensive operations
    server.SetHandler(func(ctx *tcp.ConnContext) error {
        // Pin OS thread for CPU-bound work (optional, for native code)
        runtime.LockOSThread()
        defer runtime.UnlockOSThread()
        
        // Read data
        buf := make([]byte, 1024)
        n, err := ctx.Conn.Read(buf)
        if err != nil {
            return err
        }
        
        // CPU-intensive: Hash computation
        hash := sha256.Sum256(buf[:n])
        
        // Write hash back
        _, err = ctx.Conn.Write(hash[:])
        return err
    })
    
    go func() {
        if err := server.Start(); err != nil {
            panic(err)
        }
    }()
    
    <-gocmd.Context().Done()
}
```

**Characteristics:**
- **Workers**: `runtime.NumCPU()` (e.g., 8 for 8-core CPU)
- **Queue**: 1,000 (moderate, CPU-bound work is slower)
- **MaxConns**: `workers × 10` (limited by CPU capacity)
- **Use Case**: Crypto operations, data processing, computation

**⚠️ Important:** 
- Do NOT spawn too many goroutines for CPU-bound work
- Use `runtime.LockOSThread()` for native code (CGO, llama.cpp)
- Workers should be ≤ CPU cores

### 3. TLS/Encrypted TCP Configuration

#### HighThroughputIOBoundTLSConfig

For TLS/encrypted TCP with IO-bound characteristics:

```go
package main

import (
    "context"
    "crypto/tls"
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/tcp"
)

func main() {
    gocmd := core.NewGoCMD(context.Background())
    
    // TLS IO-bound config: Increased workers for TLS overhead
    config := tcp.HighThroughputIOBoundTLSConfig(":9443")
    
    // Configure TLS with session resumption
    cert, _ := tls.LoadX509KeyPair("cert.pem", "key.pem")
    config.TLSConfig = &tls.Config{
        Certificates: []tls.Certificate{cert},
        SessionTicketsDisabled: false, // ✅ Enable session resumption
        CipherSuites: []uint16{
            tls.TLS_AES_128_GCM_SHA256,      // TLS 1.3 (fastest, hardware accelerated)
            tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, // TLS 1.2 (AES-NI)
        },
        MinVersion: tls.VersionTLS12,
        MaxVersion: tls.VersionTLS13,
    }
    
    server := tcp.NewTCPServer(gocmd, config)
    
    // IO-bound handler (TLS overhead handled by Go's crypto/tls)
    server.SetHandler(func(ctx *tcp.ConnContext) error {
        buf := make([]byte, 1024)
        n, err := ctx.Conn.Read(buf)
        if err != nil {
            return err
        }
        _, err = ctx.Conn.Write(buf[:n])
        return err
    })
    
    go func() {
        if err := server.Start(); err != nil {
            panic(err)
        }
    }()
    
    <-gocmd.Context().Done()
}
```

**Characteristics:**
- **Workers**: 2000 (higher than plain TCP to account for TLS overhead)
- **Queue**: 50,000 (large buffer for TLS connection bursts)
- **MaxConns**: 100,000 (TLS can handle many connections with session resumption)
- **TLS Overhead**: +10ms latency (5ms IO + 10ms TLS)

**TLS Performance Notes:**
- TLS handshake: +5-10ms (CPU-bound, blocks IO-bound workers)
- TLS encryption/decryption: +1-2ms per connection (CPU-bound)
- Session resumption: Reduces handshake overhead
- AES-GCM: Hardware accelerated (AES-NI) on modern CPUs

#### HighThroughputIOBoundTLSConfigWithTargetRPS

For specific TLS connection rate targets:

```go
// Target: 100k connections/sec with 15ms latency (TLS overhead)
config := tcp.HighThroughputIOBoundTLSConfigWithTargetRPS(":9443", 100000, 15)
// Workers = (100k * 15ms) / 1000 = 1500 workers
// Queue = 100k * 0.5 = 50k

cert, _ := tls.LoadX509KeyPair("cert.pem", "key.pem")
config.TLSConfig = &tls.Config{
    Certificates: []tls.Certificate{cert},
    SessionTicketsDisabled: false, // Enable session resumption
}

server := tcp.NewTCPServer(gocmd, config)
```

**TLS Latency Estimates:**
- Plain TCP: 5ms
- TLS TCP: 15ms (5ms IO + 10ms TLS overhead)
- TLS handshake: +5-10ms (first connection)
- TLS encryption/decryption: +1-2ms per connection

## Comparison Table

| Config | Workers | Queue | MaxConns | Use Case |
|--------|---------|-------|----------|----------|
| `DefaultTCPServerConfig` | 50 | 1,000 | Unlimited | General purpose |
| `HighThroughputIOBoundConfig` | 2,000 | 50,000 | 100,000 | High RPS IO-bound |
| `HighThroughputIOBoundConfigWithTargetRPS(100k, 5ms)` | 500 | 50,000 | 100,000 | Specific RPS target |
| `CPUBoundConfig` | CPU cores | 1,000 | workers × 10 | CPU-intensive |
| `HighThroughputIOBoundTLSConfig` | 2,000 | 50,000 | 100,000 | TLS IO-bound |
| `HighThroughputIOBoundTLSConfigWithTargetRPS(100k, 15ms)` | 1,500 | 50,000 | 100,000 | TLS with RPS target |

## Best Practices

### 1. Choose the Right Config

**IO-Bound (Network I/O, File I/O):**
```go
// Many workers OK, large queue
config := tcp.HighThroughputIOBoundConfig(":9000")
```

**CPU-Bound (Crypto, Computation):**
```go
// Limited workers = CPU cores, moderate queue
config := tcp.CPUBoundConfig(":9000")
// In handler: runtime.LockOSThread() for native code
```

**TLS/Encrypted:**
```go
// Increased workers for TLS overhead
config := tcp.HighThroughputIOBoundTLSConfig(":9443")
// Enable session resumption
config.TLSConfig.SessionTicketsDisabled = false
```

### 2. Thread Management

**CPU-Bound Handler:**
```go
server.SetHandler(func(ctx *tcp.ConnContext) error {
    // Pin OS thread for CPU-bound native code
    runtime.LockOSThread()
    defer runtime.UnlockOSThread()
    
    // CPU-intensive work (llama.cpp, crypto, etc.)
    result := cpuIntensiveOperation(data)
    return nil
})
```

**IO-Bound Handler:**
```go
server.SetHandler(func(ctx *tcp.ConnContext) error {
    // No thread pinning needed - Go scheduler handles I/O
    // CPU idle during network I/O
    data, err := readFromConnection(ctx.Conn)
    return err
})
```

### 3. TLS Optimization

```go
config.TLSConfig = &tls.Config{
    // ✅ Enable session resumption (reduces handshake overhead)
    SessionTicketsDisabled: false,
    
    // ✅ Use hardware-accelerated cipher suites
    CipherSuites: []uint16{
        tls.TLS_AES_128_GCM_SHA256, // TLS 1.3 (AES-NI)
    },
    
    // ✅ Enable TLS 1.3 (fastest)
    MinVersion: tls.VersionTLS12,
    MaxVersion: tls.VersionTLS13,
}
```

### 4. Monitoring

```go
// Check metrics
metrics := server.Metrics()

// Queue utilization
if metrics.QueueUtilization > 90.0 {
    // Queue nearly full → increase QueueSize
}

// Rejected connections
if metrics.RejectedConnections > 0 {
    // Backpressure rejecting → increase capacity
}

// CCU utilization
if metrics.CCUUtilization > 90.0 {
    // High capacity utilization → increase workers/queue
}
```

## Example: Mixed Workload

For handlers that mix IO-bound and CPU-bound operations:

```go
config := tcp.HighThroughputIOBoundConfig(":9000")
server := tcp.NewTCPServer(gocmd, config)

server.SetHandler(func(ctx *tcp.ConnContext) error {
    // 1. IO-bound: Read from connection
    data, err := readFromConnection(ctx.Conn)
    if err != nil {
        return err
    }
    
    // 2. CPU-bound: Process data
    // Option A: Process in same goroutine (blocks IO-bound worker)
    result := processData(data) // CPU-intensive
    
    // Option B: Offload to CPU-bound worker pool (better)
    // future := cpuPool.Submit(ctx, key, data)
    // result, _ := future.Get(ctx)
    
    // 3. IO-bound: Write response
    _, err = ctx.Conn.Write(result)
    return err
})
```

**Recommendation:** For mixed workloads, use IO-bound config and offload CPU-bound work to a separate worker pool (see `pkg/core/compute`).

## References

- [IO-Bound Optimization Guide](../web/IO_BOUND_OPTIMIZATION.md)
- [TLS Optimization Summary](../web/TLS_OPTIMIZATION_SUMMARY.md)
- [TLS Performance Guide](../web/TLS_PERFORMANCE.md)
- [Thread Management Guide](../core/compute/THREAD_MANAGEMENT.md)
- [TCP Server README](./README.md)

