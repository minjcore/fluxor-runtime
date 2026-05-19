# IO-Bound Workload Optimization Guide

Hướng dẫn tối ưu cho IO-bound workloads (HTTP server, database queries, file I/O) khi CPU utilization thấp nhưng RPS không đạt target.

## Vấn Đề Thường Gặp

### Triệu Chứng:
- ✅ CPU utilization thấp (1-2 cores, 10-30%)
- ❌ RPS (Requests Per Second) thấp, không đạt target
- ❌ Network bị kẹt nhưng không biết ở đâu
- ❌ Loadtest với wrk cho thấy bottleneck

### Nguyên Nhân:
1. **Workers quá ít**: Default 100 workers có thể không đủ cho IO-bound
2. **Queue size quá nhỏ**: Queue đầy → reject requests sớm
3. **System limits**: File descriptors, network buffers quá thấp
4. **Backpressure quá sớm**: Normal capacity (67%) reject requests sớm

## Giải Pháp

### 1. Sử Dụng High-Throughput Config

```go
import "github.com/fluxorio/fluxor/pkg/web"

// Option 1: Default high-throughput config
config := web.HighThroughputIOBoundConfig(":8080")
// Workers: 2000, Queue: 50000, MaxConns: 100000

// Option 2: Config theo target RPS
config := web.HighThroughputIOBoundConfigWithTargetRPS(":8080", 100000, 10)
// 100k RPS với latency 10ms → Workers: 1000, Queue: 50000

server := web.NewFastHTTPServer(gocmd, config)
server.RegisterMetricsEndpoint()  // Thêm /metrics endpoint
```

### 2. Kiểm Tra System Limits

```bash
# Chạy script kiểm tra
./scripts/check_system_limits.sh

# Hoặc kiểm tra thủ công:
ulimit -n                    # File descriptors (cần 100000+)
sysctl net.core.somaxconn    # Network backlog (cần 4096+)
```

### 3. Tăng System Limits (Linux)

```bash
# File descriptors
ulimit -n 100000
# Hoặc thêm vào /etc/security/limits.conf:
# * soft nofile 100000
# * hard nofile 100000

# Network buffers
sudo sysctl -w net.core.somaxconn=4096
sudo sysctl -w net.ipv4.tcp_max_syn_backlog=4096
# Thêm vào /etc/sysctl.conf để persist
```

### 4. Tăng System Limits (macOS)

```bash
# File descriptors
ulimit -n 100000

# System-wide limits
sudo sysctl -w kern.maxfiles=100000
sudo sysctl -w kern.maxfilesperproc=100000
```

### 5. Monitor Metrics

```bash
# Kiểm tra metrics endpoint
curl http://localhost:8080/metrics | jq

# Các metrics quan trọng:
# - queue.utilization: Nếu >90% → queue quá nhỏ
# - requests.rejected: Nếu >0 → backpressure đang reject
# - ccu.utilization: Nếu >90% → capacity quá thấp
```

## Công Thức Tính Toán

### Workers cho IO-Bound:
```
Workers = (Target RPS × Avg Latency Ms) / 1000

Ví dụ:
- Target: 100k RPS
- Latency: 10ms
- Workers = (100000 × 10) / 1000 = 1000 workers
```

### Queue Size:
```
QueueSize = Target RPS × 0.5

Ví dụ:
- Target: 100k RPS
- QueueSize = 100000 × 0.5 = 50000
(Buffer cho 500ms traffic)
```

### Max Connections:
```
MaxConns = Target RPS (minimum 10000)

Ví dụ:
- Target: 100k RPS
- MaxConns = 100000
```

## Debugging Checklist

### 1. Queue có đầy không?
```go
metrics := server.Metrics()
if metrics.QueueUtilization > 90.0 {
    // Queue gần đầy → tăng QueueSize
    fmt.Printf("Queue utilization: %.2f%%\n", metrics.QueueUtilization)
}
```

### 2. Backpressure có reject không?
```go
if metrics.RejectedRequests > 0 {
    // Backpressure đang reject → tăng capacity
    fmt.Printf("Rejected requests: %d\n", metrics.RejectedRequests)
}
```

### 3. Workers có đủ không?
```go
// Nếu CPU thấp (<30%) nhưng RPS thấp
// → Có thể workers quá ít
// IO-bound: Nhiều workers OK (không giới hạn bởi CPU cores)
```

### 4. Network có bị giới hạn không?
```bash
# Kiểm tra active connections
netstat -an | grep ESTABLISHED | wc -l

# Kiểm tra file descriptors
lsof -p $(pgrep your-app) | wc -l
```

## So Sánh Config

| Config | Workers | Queue | MaxConns | Use Case |
|--------|---------|-------|----------|----------|
| `DefaultFastHTTPServerConfig` | 100 | 10000 | 100000 | General purpose |
| `HighThroughputIOBoundConfig` | 2000 | 50000 | 100000 | High RPS IO-bound |
| `HighThroughputIOBoundConfigWithTargetRPS(100k, 10ms)` | 1000 | 50000 | 100000 | Specific RPS target |

## Best Practices

1. **Bắt đầu với default**, monitor metrics
2. **Nếu CPU thấp + RPS thấp** → Tăng workers và queue
3. **Kiểm tra system limits** trước khi tăng config
4. **Monitor /metrics endpoint** để detect bottlenecks
5. **IO-bound: Workers không giới hạn bởi CPU cores**
   - CPU-bound: Workers ≤ CPU cores
   - IO-bound: Workers có thể >> CPU cores

## Ví Dụ Hoàn Chỉnh

```go
package main

import (
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/web"
)

func main() {
    gocmd := core.NewGoCMD(context.Background())
    
    // High-throughput config cho IO-bound
    config := web.HighThroughputIOBoundConfigWithTargetRPS(":8080", 100000, 10)
    
    server := web.NewFastHTTPServer(gocmd, config)
    
    // Register metrics endpoint
    server.RegisterMetricsEndpoint()
    
    // Register handlers
    router := server.FastRouter()
    router.GETFast("/ping", func(ctx *web.FastRequestContext) error {
        return ctx.String("pong")
    })
    
    // Start server
    if err := server.Start(); err != nil {
        panic(err)
    }
    
    // Monitor metrics
    go func() {
        for {
            metrics := server.Metrics()
            if metrics.QueueUtilization > 90.0 {
                log.Printf("⚠️  Queue utilization: %.2f%%", metrics.QueueUtilization)
            }
            if metrics.RejectedRequests > 0 {
                log.Printf("⚠️  Rejected requests: %d", metrics.RejectedRequests)
            }
            time.Sleep(5 * time.Second)
        }
    }()
    
    <-gocmd.Context().Done()
}
```

## Tài Liệu Liên Quan

- [Thread Management Guide](../core/compute/THREAD_MANAGEMENT.md) - CPU-Bound vs IO-Bound
- [FastHTTPServer API Documentation](./fast_server.go)

