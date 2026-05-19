# TLS Performance: Tại Sao Chậm và Cách Tối Ưu

## Vấn Đề Chính

**TLS/HTTPS chậm vì TLS operations (CPU-bound) chạy trong IO-bound goroutines**

### Phân Tích

```
IO-Bound Worker Goroutine:
├── Accept connection (network I/O) ✅
├── TLS Handshake (CPU-bound!) ⚠️  ← Block worker
│   ├── ECDHE key exchange (CPU-intensive)
│   ├── RSA signature verification (CPU-intensive)
│   └── Certificate validation (CPU-intensive)
├── TLS Encryption (CPU-bound!) ⚠️  ← Block worker
├── Process HTTP request (IO-bound) ✅
├── TLS Decryption (CPU-bound!) ⚠️  ← Block worker
└── Send response (network I/O) ✅
```

**Impact:**
- IO-bound workers bị block bởi CPU-bound TLS operations
- CPU không được tận dụng tốt (1-2 cores busy với TLS)
- RPS thấp vì workers bị block

## Giải Pháp

### 1. Tăng Workers (Immediate Fix)

Vì TLS overhead, cần tăng workers:

```go
// HTTP: 10ms latency
config := web.HighThroughputIOBoundConfigWithTargetRPS(":8080", 100000, 10)
// Workers = (100k * 10ms) / 1000 = 1000 workers

// HTTPS: 20ms latency (10ms IO + 10ms TLS)
config := web.HighThroughputIOBoundConfigWithTargetRPS(":8443", 100000, 20)
// Workers = (100k * 20ms) / 1000 = 2000 workers
```

### 2. Enable Session Resumption

Go's crypto/tls đã enable session tickets by default, nhưng đảm bảo:

```go
tlsConfig := &tls.Config{
    SessionTicketsDisabled: false,  // ✅ Enable (default)
    // Session resumption giảm handshake overhead
}
```

### 3. Use Efficient Cipher Suites

```go
// Prefer AES-GCM (hardware accelerated)
CipherSuites: []uint16{
    tls.TLS_AES_128_GCM_SHA256,      // TLS 1.3 (fastest)
    tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,  // TLS 1.2 (AES-NI)
}
```

### 4. Consider Reverse Proxy

```
Client → Nginx/Traefik (TLS termination) → Go Server (HTTP)
```

**Lợi ích:**
- TLS xử lý bởi reverse proxy (optimized)
- Go server chỉ xử lý HTTP (pure IO-bound)

## Quick Reference

| Scenario | Latency | Workers Formula |
|----------|---------|-----------------|
| HTTP | 10ms | (RPS * 10) / 1000 |
| HTTPS | 20ms | (RPS * 20) / 1000 |
| HTTPS + DB | 30ms | (RPS * 30) / 1000 |

## Example

```go
// High-throughput HTTPS server
config := web.HighThroughputIOBoundConfigWithTargetRPS(":8443", 100000, 20)
tlsCfg, _ := web.NewTLSConfigFromFiles("cert.crt", "key.key")
config.TLSConfig = tlsCfg

server := web.NewFastHTTPServer(gocmd, config)
```

Xem [TLS_PERFORMANCE.md](./TLS_PERFORMANCE.md) để biết chi tiết.

