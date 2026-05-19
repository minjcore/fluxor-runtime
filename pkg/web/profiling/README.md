# Advanced Profiling System

Hệ thống profiling cao cấp để phát hiện bottleneck và anti-patterns trong IO-Bound và CPU-Bound workloads.

## Tính năng

1. **Work Classification**: Phân loại tự động IO-Bound vs CPU-Bound work dựa trên stack traces
2. **Goroutine Profiling**: Track goroutine states và work types
3. **Bottleneck Detection**: Tự động phát hiện bottlenecks (queue full, IO-bound, CPU-bound, mixed work)
4. **Anti-Pattern Detection**: Phát hiện khi IO workers đang xử lý CPU work (anti-pattern)

## Cách sử dụng

### Trong Quadgate_io

Hệ thống profiling đã được tích hợp sẵn vào app `apps/quadgate-io`. Các endpoints sau đã có sẵn:

#### 1. `/metrics/profiling` - Profiling metrics tổng hợp

```bash
curl http://localhost:8080/metrics/profiling | jq
```

Response:
```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "workClassification": {
    "ioBound": {
      "activeWorkers": 150,
      "queuedTasks": 500,
      "utilization": 75.5
    },
    "cpuBound": {
      "activeWorkers": 0,
      "queuedTasks": 0,
      "utilization": 0.0
    },
    "mixed": {
      "activeWorkers": 5,
      "queuedTasks": 0,
      "utilization": 2.5,
      "warning": "Anti-pattern detected: 5 workers doing both IO and CPU work"
    }
  },
  "bottlenecks": [
    {
      "type": "io_bound",
      "severity": "high",
      "description": "IO-bound workers saturated",
      "recommendation": "Increase IO-bound workers or optimize IO operations"
    },
    {
      "type": "mixed_work",
      "severity": "warning",
      "description": "5 workers detected doing both IO and CPU work",
      "recommendation": "Offload CPU work to dedicated CPU-bound worker pool"
    }
  ],
  "goroutines": {
    "total": 200,
    "byState": {
      "running": 150,
      "waiting": 30,
      "blocked": 20
    },
    "byWorkType": {
      "ioBound": 145,
      "cpuBound": 0,
      "mixed": 5
    },
    "mixedWork": 5
  }
}
```

#### 2. `/metrics/goroutines` - Chi tiết goroutines

```bash
curl http://localhost:8080/metrics/goroutines | jq
```

Response bao gồm:
- Goroutine ID
- State (running, waiting, blocked)
- Work type (io-bound, cpu-bound, mixed)
- Stack trace
- Last seen timestamp

#### 3. `/metrics/bottlenecks` - Phân tích bottlenecks

```bash
curl http://localhost:8080/metrics/bottlenecks | jq
```

Response:
```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "bottlenecks": [
    {
      "type": "queue_full",
      "severity": "high",
      "description": "Queue utilization at 95.00%, queue is nearly full",
      "recommendation": "Increase queue size or reduce incoming request rate"
    }
  ],
  "summary": {
    "total": 1,
    "critical": 0,
    "high": 1,
    "medium": 0,
    "low": 0
  }
}
```

## Monitoring trong Load Test

### Real-time monitoring

```bash
# Monitor profiling metrics trong khi load test
watch -n 1 'curl -s http://localhost:8080/metrics/profiling | jq .bottlenecks'

# Check for mixed work (anti-pattern)
watch -n 1 'curl -s http://localhost:8080/metrics/profiling | jq ".workClassification.mixed"'

# Monitor goroutine counts by work type
watch -n 1 'curl -s http://localhost:8080/metrics/profiling | jq ".goroutines.byWorkType"'
```

### Load test script example

```bash
#!/bin/bash
# Run load test và monitor profiling

# Start server (from repo root: cd apps/quadgate-io && go run . config.json &)
cd apps/quadgate-io && go run . config.json &
SERVER_PID=$!

# Wait for server to start
sleep 2

# Run load test in background
k6 run loadtest/load_test.js &
K6_PID=$!

# Monitor profiling
echo "Monitoring profiling metrics..."
for i in {1..60}; do
  echo "=== Sample $i ==="
  curl -s http://localhost:8080/metrics/profiling | jq '{
    bottlenecks: .bottlenecks,
    mixedWork: .workClassification.mixed,
    goroutines: .goroutines.byWorkType
  }'
  sleep 2
done

# Cleanup
kill $K6_PID
kill $SERVER_PID
```

## Bottleneck Types

### 1. `queue_full`
- **Mô tả**: Queue đang gần đầy (>90%)
- **Severity**: Medium → High → Critical
- **Recommendation**: Tăng queue size hoặc giảm request rate

### 2. `io_bound`
- **Mô tả**: IO-bound workers đang saturated
- **Severity**: Medium → High
- **Recommendation**: Tăng IO-bound workers hoặc optimize IO operations

### 3. `cpu_bound`
- **Mô tả**: CPU-bound workers đang saturated
- **Severity**: Medium → High
- **Recommendation**: Offload CPU work sang dedicated CPU-bound worker pool

### 4. `mixed_work` (Anti-pattern)
- **Mô tả**: Workers đang làm cả IO và CPU work
- **Severity**: Low → Medium → High
- **Recommendation**: Tách IO và CPU work thành separate worker pools

## Work Classification

Hệ thống tự động phân loại work type dựa trên stack traces:

### IO-Bound Patterns
- `net.*`, `syscall.*`, `io.*`
- `Read`, `Write`, `Accept`, `Listen`
- `fasthttp.*`, `http.*`
- `database/sql.*`, `redis.*`
- `context.*`, `time.Sleep`
- Channel operations (`chan`, `<-`)

### CPU-Bound Patterns
- `crypto/*`, `encoding/*`, `compress/*`
- `hash.*`, `sha256.*`, `md5.*`, `aes.*`
- `json.Marshal`, `json.Unmarshal`
- `image.*`, `math.*`, `sort.*`
- `strings.*`, `bytes.*`, `regexp.*`

### Mixed Work
- Khi stack trace chứa cả IO và CPU patterns
- Đây là anti-pattern cần tránh

## Performance Considerations

1. **Stack trace capture**: Có overhead, profiling chạy mỗi 5 giây (có thể config)
2. **Memory usage**: Store stack traces có thể tốn memory
3. **CPU usage**: Stack analysis có overhead nhỏ

## API Reference

### WorkClassifier
- `Classify(goroutineID int, stackTrace []string) WorkType`
- `GetWorkType(goroutineID int) WorkType`
- `GetWorkTypeStats() map[WorkType]int`

### GoroutineProfiler
- `Update(goroutineID int, stackTrace []string, workType WorkType)`
- `GetProfile(goroutineID int) *GoroutineProfile`
- `GetAllProfiles() map[int]*GoroutineProfile`
- `GetStats() GoroutineStats`

### BottleneckDetector
- `Detect(metrics *ServerMetricsForProfiling, goroutineStats *GoroutineStats) []Bottleneck`

### RuntimeProfiler
- `Start(ctx context.Context)`
- `Stop()`
- `Profile() error`
- `GetProfilingData() *ProfilingData`

## Best Practices

1. **Monitor trong load test**: Sử dụng profiling để identify bottlenecks
2. **Check mixed work**: Thường xuyên check anti-pattern
3. **Tune based on bottlenecks**: Adjust workers/queue dựa trên bottleneck detection
4. **Separate IO and CPU**: Tránh mixed work bằng cách tách IO và CPU work pools

## Troubleshooting

### Không thấy profiling data
- Kiểm tra server đã start chưa
- Kiểm tra profiling system đã được initialize chưa
- Đợi 5 giây để profiling chạy lần đầu

### Mixed work được detect nhưng không đúng
- Kiểm tra stack traces trong `/metrics/goroutines`
- Verify work classification patterns
- Có thể cần adjust patterns trong `StackAnalyzer`

### Performance impact
- Tăng profiling interval nếu overhead quá cao
- Disable profiling trong production nếu không cần
- Chỉ enable profiling khi debugging/tuning
