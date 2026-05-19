# TLS/HTTPS Performance Optimization

## Vấn Đề: TLS Chậm Vì Không Chuyển Task Cho CPU

### Nguyên Nhân

TLS/HTTPS operations là **CPU-bound** nhưng đang chạy trong **IO-bound goroutines**, gây ra bottleneck:

```
┌─────────────────────────────────────────┐
│  IO-Bound Worker (fasthttp goroutine)   │
│                                          │
│  1. Accept connection (network I/O)      │
│  2. TLS Handshake (CPU-bound!) ⚠️       │
│  3. TLS Encryption (CPU-bound!) ⚠️      │
│  4. Process HTTP request (IO-bound)      │
│  5. TLS Decryption (CPU-bound!) ⚠️      │
│  6. Send response (network I/O)          │
└─────────────────────────────────────────┘
```

**Vấn đề:**
- TLS handshake: **CPU-intensive** (ECDHE key exchange, RSA operations)
- TLS encryption/decryption: **CPU-intensive** (AES-GCM, ChaCha20-Poly1305)
- Nhưng chạy trong **IO-bound workers** → Block workers, giảm throughput

### Tại Sao Chậm?

1. **TLS Handshake (CPU-bound):**
   ```
   Client → Server: ClientHello
   Server → Client: ServerHello + Certificate + ServerKeyExchange
   Client → Server: ClientKeyExchange + ChangeCipherSpec
   Server → Client: ChangeCipherSpec + Finished
   ```
   - ECDHE key exchange: **CPU-intensive**
   - RSA signature verification: **CPU-intensive**
   - Chạy trong IO-bound goroutine → Block worker

2. **TLS Encryption/Decryption (CPU-bound):**
   ```
   Every request/response:
   - Encrypt data (AES-GCM, ChaCha20-Poly1305)
   - Decrypt data
   - MAC verification
   ```
   - CPU-intensive operations
   - Chạy trong IO-bound goroutine → Block worker

3. **Impact:**
   - IO-bound workers bị block bởi CPU-bound TLS operations
   - CPU không được tận dụng tốt (1-2 cores busy với TLS)
   - RPS thấp vì workers bị block

## Giải Pháp

### 1. Tối Ưu TLS Configuration (Immediate)

#### A. Enable TLS Session Resumption

Go's `crypto/tls` đã hỗ trợ session resumption, nhưng cần đảm bảo được enable:

```go
tlsConfig := &tls.Config{
    // Session tickets (TLS 1.2+)
    SessionTicketsDisabled: false,  // ✅ Enable session tickets
    
    // Session resumption
    ClientSessionCache: tls.NewLRUClientSessionCache(1000),  // Cache 1000 sessions
}
```

**Lợi ích:**
- Giảm TLS handshake overhead (resume session thay vì full handshake)
- Giảm CPU usage cho handshake

#### B. Use Efficient Cipher Suites

```go
// Prefer AES-GCM (hardware accelerated on modern CPUs)
CipherSuites: []uint16{
    tls.TLS_AES_128_GCM_SHA256,      // TLS 1.3 (fastest)
    tls.TLS_AES_256_GCM_SHA384,      // TLS 1.3
    tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,  // TLS 1.2 (AES-NI)
    tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,  // TLS 1.2 (AES-NI)
}
```

**Lợi ích:**
- AES-GCM có hardware acceleration (AES-NI) trên modern CPUs
- Nhanh hơn software encryption

### 2. Tăng Workers Cho TLS Overhead

Vì TLS operations block IO-bound workers, cần tăng số workers:

```go
// For TLS/HTTPS, increase workers to handle TLS overhead
config := web.HighThroughputIOBoundConfigWithTargetRPS(":8443", 100000, 20)
// ↑ Increase latency estimate (10ms → 20ms) to account for TLS overhead
// This calculates: Workers = (100k RPS * 20ms) / 1000 = 2000 workers
```

**Lý do:**
- TLS handshake: +5-10ms latency
- TLS encryption/decryption: +1-2ms per request
- Total: ~20ms latency (thay vì 10ms cho HTTP)

### 3. Hardware Acceleration

#### A. AES-NI (Automatic)

Modern CPUs có AES-NI instruction set → Go's crypto/tls tự động sử dụng:
- AES-GCM encryption/decryption nhanh hơn 10x
- Không cần config gì thêm

#### B. Check AES-NI Support

```bash
# Linux
grep -m1 -o aes /proc/cpuinfo

# macOS
sysctl machdep.cpu.features | grep -i aes
```

### 4. TLS Offloading (Advanced)

**Option A: Reverse Proxy với TLS Termination**

```
Client → Nginx/Traefik (TLS termination) → Go Server (HTTP)
```

**Lợi ích:**
- TLS xử lý bởi reverse proxy (optimized C code)
- Go server chỉ xử lý HTTP (pure IO-bound)
- Dễ scale và optimize

**Option B: TLS Offload Hardware**

- Hardware SSL/TLS accelerator cards
- Offload TLS operations sang dedicated hardware
- Đắt tiền, chỉ cho high-end deployments

### 5. Connection Pooling & Keep-Alive

```go
// Enable HTTP/1.1 keep-alive
serverCfg := &web.FastHTTPServerConfig{
    // ... other config
    ReadTimeout:  30 * time.Second,  // Longer timeout for keep-alive
    WriteTimeout: 30 * time.Second,
}
```

**Lợi ích:**
- Reuse TLS connections (không cần handshake mỗi request)
- Giảm TLS handshake overhead

## Best Practices

### 1. Configuration cho TLS/HTTPS

```go
// High-throughput config với TLS overhead
config := web.HighThroughputIOBoundConfigWithTargetRPS(":8443", 100000, 20)
// ↑ 20ms latency (10ms HTTP + 10ms TLS overhead)

// Enable TLS với session resumption
tlsCfg, _ := web.NewTLSConfigFromFiles("cert.crt", "key.key")
tlsCfg.MinVersion = tls.VersionTLS12
tlsCfg.MaxVersion = tls.VersionTLS13
// Session tickets enabled by default in Go

config.TLSConfig = tlsCfg
```

### 2. Monitoring TLS Performance

```go
// Add TLS metrics
metrics := server.Metrics()
tlsInfo := server.TLSInfo()

// Monitor:
// - TLS handshake time
// - TLS encryption/decryption overhead
// - Session resumption rate
```

### 3. Load Testing với TLS

```bash
# Test với TLS
wrk -t12 -c400 -d30s https://localhost:8443/ping

# So sánh với HTTP
wrk -t12 -c400 -d30s http://localhost:8080/ping

# Expected: HTTPS ~50-70% throughput của HTTP
```

## So Sánh Performance

| Operation | HTTP | HTTPS | Overhead |
|-----------|------|-------|----------|
| **Handshake** | 0ms | 5-10ms | +5-10ms |
| **Encryption** | 0ms | 1-2ms | +1-2ms |
| **Decryption** | 0ms | 1-2ms | +1-2ms |
| **Total per request** | ~10ms | ~20ms | +10ms |

**Impact:**
- HTTPS throughput: ~50-70% của HTTP
- Cần tăng workers: 2x để compensate TLS overhead

## Recommendations

### For High RPS với TLS:

1. **Use High-Throughput Config:**
   ```go
   config := web.HighThroughputIOBoundConfigWithTargetRPS(":8443", 100000, 20)
   ```

2. **Enable Session Resumption:**
   - Go's crypto/tls đã enable by default
   - Đảm bảo không disable: `SessionTicketsDisabled: false`

3. **Use AES-GCM Cipher Suites:**
   - Hardware accelerated (AES-NI)
   - Fastest option

4. **Increase Workers:**
   - TLS overhead: +10ms latency
   - Workers = (RPS * 20ms) / 1000 (thay vì 10ms)

5. **Consider Reverse Proxy:**
   - Nginx/Traefik cho TLS termination
   - Go server chỉ xử lý HTTP

## Example: Optimized TLS Config

```go
package main

import (
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/web"
)

func main() {
    gocmd := core.NewGoCMD(context.Background())
    
    // High-throughput config với TLS overhead
    config := web.HighThroughputIOBoundConfigWithTargetRPS(":8443", 100000, 20)
    
    // TLS config với session resumption
    tlsCfg, _ := web.NewTLSConfigFromFiles("cert.crt", "key.key")
    config.TLSConfig = tlsCfg
    
    server := web.NewFastHTTPServer(gocmd, config)
    server.RegisterMetricsEndpoint()
    
    // ... register routes
    
    server.Start()
}
```

## Tài Liệu Liên Quan

- [IO-Bound Optimization Guide](./IO_BOUND_OPTIMIZATION.md)
- [Thread Management Guide](../core/compute/THREAD_MANAGEMENT.md)
- [Go crypto/tls Documentation](https://pkg.go.dev/crypto/tls)

