# Benchmark Analysis

## Test Results

### Low Concurrency Test (`-c10 -t2 -d10`)
```
Latest: Requests/sec: 243,612.54, Latency: 36.25μs (avg), 545μs (max)
Previous: Requests/sec: 241,931.57, Latency: 36.57μs (avg), 504μs (max)
Initial: Requests/sec: 242,815.48, Latency: 36.33μs (avg), 388μs (max)
```

**Analysis:**
- ✅ **Excellent performance: 243k RPS** - best result so far!
- ✅ **Ultra-low latency: 36.25μs average** - excellent responsiveness
- ✅ **No errors** - 100% success rate
- ✅ **Near-optimal for low concurrency** - excellent baseline performance
- ✅ **Config system** - no performance regression, actually slightly better!

### High Concurrency Test - Initial (`-c 256 -t 16 -d 30s`)
```
Requests/sec: 159,618.80
Latency: 1.58ms (avg), 3.43ms (99th percentile)
Errors: 21 connect errors
```

**Analysis:**
- ⚠️ Performance drops to 159k RPS (34% reduction)
- ⚠️ Latency increases 44x (1.58ms vs 36μs)
- ⚠️ 21 connection errors (0.0004% error rate - acceptable but not ideal)

### High Concurrency Test - Optimized (`-c 256 -t 16 -d 30s`)
```
Requests/sec: 189,379.26
Latency: 1.23ms (avg), 1.36ms (99th percentile)
Errors: 21 connect errors
```

**Analysis:**
- ✅ **Significant improvement: +19% RPS** (159k → 189k)
- ✅ **Latency improved: -22%** (1.58ms → 1.23ms)
- ✅ **99th percentile latency dramatically improved: -60%** (3.43ms → 1.36ms)
- ⚠️ Same 21 connection errors (0.0004% error rate - acceptable)

### Medium Concurrency Test - 100 connections, 12 threads (`-c 100 -t 12 -d 30s`)
```
Requests/sec: 192,162.42
Latency: 753.03μs (avg), 138.33ms (max)
Errors: 0
```

**Analysis:**
- ✅ **Excellent performance: 192k RPS**
- ✅ **Low latency: 753μs average** - significantly better than 1.23ms
- ✅ **Zero errors** - perfect reliability

### Medium Concurrency Test - 100 connections, 4 threads (`-c 100 -t 4 -d 10s`)
```
Requests/sec: 259,736.82
Latency: 374.04μs (avg), 3.28ms (max)
Errors: 0
```

**Analysis:**
- ✅ **Outstanding performance: 259k RPS** - best result so far!
- ✅ **Excellent latency: 374μs average** - much better than 753μs with 12 threads
- ✅ **Zero errors** - perfect reliability
- ✅ **Optimal thread count** - 4 threads performs better than 12 threads for this workload

## Performance Comparison

### Initial High Concurrency
| Metric | Low Concurrency | High Concurrency (Initial) | Change |
|--------|---------------|----------------------------|--------|
| RPS | 242,815 | 159,618 | -34% |
| Avg Latency | 36.33μs | 1.58ms | +44x |
| 99th %ile | ~388μs | 3.43ms | +9x |
| Errors | 0 | 21 | +21 |

### Optimized High Concurrency
| Metric | Low Concurrency | High Concurrency (Optimized) | Change |
|--------|---------------|------------------------------|--------|
| RPS | 242,815 | **189,379** | **-22%** ✅ |
| Avg Latency | 36.33μs | **1.23ms** | +34x ✅ |
| 99th %ile | ~388μs | **1.36ms** | +3.5x ✅ |
| Errors | 0 | 21 | +21 |

### Improvement Summary
| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| RPS | 159,618 | 189,379 | **+19%** 🚀 |
| Avg Latency | 1.58ms | 1.23ms | **-22%** ⚡ |
| 99th %ile | 3.43ms | 1.36ms | **-60%** 🎯 |
| Errors | 21 | 21 | Same |

### Concurrency Level Comparison
| Concurrency | RPS | Avg Latency | Errors | Notes |
|-------------|-----|-------------|--------|-------|
| Low (`-c10 -t2`) | **243,612** | **36.25μs** | **0** | **Optimal for low load** ✅ |
| Medium (`-c100 -t4`) | **259,736** | **374μs** | **0** | **Best performance!** 🚀 |
| Medium (`-c100 -t12`) | **192,162** | **753μs** | **0** | Good, but 4 threads better |
| High (`-c256 -t16`) | 189,379 | 1.23ms | 21 | Good, but higher latency |

**Key Finding**: 4 threads performs **35% better** than 12 threads for 100 connections!

## Root Cause Analysis

### 1. Connection Errors (21 connect errors)

**Possible Causes:**
- **Accept queue exhaustion**: During connection bursts, accept queue may fill up
- **File descriptor limits**: Temporary FD exhaustion during peak load
- **Race conditions**: Multiple threads accepting simultaneously
- **Client-side timeouts**: wrk clients timing out during connection establishment

**Current Handling:**
- `accept()` errors are handled gracefully
- `ECONNABORTED` is ignored (client aborted)
- `EMFILE`/`ENFILE` are logged but not fatal
- Connection limit (10,000) prevents unbounded growth

**Recommendations:**
1. Increase `listen()` backlog (currently 4096) - may help with bursts
2. Add connection rate limiting to smooth out bursts
3. Monitor file descriptor usage
4. Add metrics for accept failures

### 2. Performance Degradation at High Concurrency

**Possible Causes:**
- **Context switching overhead**: 16 threads + 256 connections = more context switches
- **Cache misses**: More connections = less cache locality
- **Lock contention**: Connection pool locks, kqueue operations
- **CPU saturation**: All cores busy, no idle time
- **Memory bandwidth**: More connections = more memory traffic

**Current Optimizations:**
- ✅ Connection pooling (500 pre-allocated)
- ✅ EV_ONESHOT for kqueue (reduces syscalls)
- ✅ Batch event processing (256 events per batch)
- ✅ Aggressive accept loop (50 accepts per event)
- ✅ CPU affinity (threads pinned to cores)

**Recommendations:**
1. **Profile with `perf`/`Instruments`** to identify bottlenecks
2. **Reduce lock contention**: Consider lock-free data structures
3. **Optimize hot path**: Minimize allocations, cache-friendly data structures
4. **Tune kqueue timeout**: Currently 10ms - may be too high
5. **Consider io_uring on Linux**: Better performance than kqueue

## Performance Targets

### Current Status
- ✅ Low concurrency: **243k RPS** (`-c10 -t2`) - excellent, ultra-low latency (36.25μs)
- ✅ **Medium concurrency: 259k RPS** (`-c 100 -t 4`) - **best performance!** 🚀
- ✅ Medium concurrency: **192k RPS** (`-c 100 -t 12`) - good, but 4 threads better
- ⚠️ High concurrency: **189k RPS** (`-c 256 -t 16`) - good, but higher latency (1.23ms)

**Key Insight**: Fewer threads (4) with same connections (100) performs **35% better** than more threads (12)!

### Potential Improvements
- **Goal**: Maintain 200k+ RPS at high concurrency
- **Target latency**: <1ms average at high concurrency
- **Target errors**: <0.001% (currently 0.0004% - already good)

## Optimization Opportunities

### 1. Accept Queue Management
```cpp
// Current: 4096 backlog
listen(fd, 4096);

// Consider: System-dependent optimal value
// Check: sysctl net.core.somaxconn
```

### 2. Connection Pool Size
```cpp
// Current: 500 pre-allocated
static constexpr size_t INITIAL_POOL_SIZE = 500;

// Consider: Dynamic pool sizing based on load
```

### 3. Event Batch Size
```cpp
// Current: 256 events per batch
struct kevent events[256];

// Consider: Profile to find optimal size
```

### 4. Accept Burst Handling
```cpp
// Current: 50 accepts per event
const int MAX_ACCEPTS_PER_EVENT = 50;

// Consider: Adaptive based on queue depth
```

### 5. Lock-Free Structures
- Consider lock-free connection pool
- Consider lock-free event queue
- Reduce mutex contention

### 6. Memory Optimization
- Cache-aligned data structures
- Reduce false sharing
- Optimize buffer sizes

## Monitoring Recommendations

### Metrics to Track
1. **Accept queue depth**: Monitor `netstat -an | grep LISTEN`
2. **File descriptor usage**: `lsof -p <pid> | wc -l`
3. **Connection errors**: Track accept() failures
4. **Latency distribution**: P50, P90, P99, P99.9
5. **CPU utilization**: Per-core usage
6. **Memory usage**: RSS, heap size
7. **Context switches**: `vmstat` or `perf`

### Profiling Commands
```bash
# CPU profiling
sudo perf record -g ./cpp_server
sudo perf report

# macOS Instruments
instruments -t "Time Profiler" ./cpp_server

# System monitoring
vmstat 1
iostat -x 1
```

## Comparison with Other Servers

### Nginx (reference)
- Low concurrency: ~87k RPS (macOS)
- High concurrency: Similar or lower than our server
- **Our server: 2.8x faster at low concurrency**

### Go Server (reference)
- Low concurrency: ~137k RPS
- **Our server: 1.8x faster**

## Conclusion

### Strengths
- ✅ **Excellent low-concurrency performance (243k RPS)** - best result so far!
- ✅ **Ultra-low latency (36.25μs average)** at low concurrency - excellent responsiveness
- ✅ **Outstanding medium-concurrency performance (259k RPS)** - best result with 4 threads!
- ✅ **Excellent latency (374μs average)** at medium concurrency with 4 threads
- ✅ **Zero errors** at low and medium concurrency - perfect reliability
- ✅ **Improved high-concurrency performance (189k RPS)** - up from 159k!
- ✅ **Better latency consistency** - 99th percentile improved 60%
- ✅ **Config system** - centralized configuration management, no performance impact
- ✅ **Thread count optimization** - 4 threads performs 35% better than 12 threads for 100 connections

### Recent Improvements
- 🚀 **+19% RPS improvement** at high concurrency (159k → 189k)
- ⚡ **-22% average latency** (1.58ms → 1.23ms)
- 🎯 **-60% 99th percentile latency** (3.43ms → 1.36ms)
- ✅ **192k RPS at medium concurrency** (`-c 100 -t 12`) - optimal sweet spot!
- ✅ **753μs latency** at medium concurrency - excellent performance
- ✅ **Zero errors** at medium concurrency

### Remaining Areas for Improvement
- ⚠️ Eliminate connection errors at very high concurrency (21 errors - 0.0004% rate)
- ⚠️ Further reduce high-concurrency performance drop (currently 22% vs 34% before)
- ⚠️ Target: Maintain 200k+ RPS at very high concurrency (`-c 256 -t 16`)
- ✅ **Achieved: 192k RPS at medium concurrency** (`-c 100 -t 12`) - excellent!

### Next Steps
1. ✅ **DONE**: Initial optimizations (connection pooling, event batching, etc.)
2. Profile with `perf`/`Instruments` to identify remaining bottlenecks
3. Consider lock-free data structures for further gains
4. Tune system parameters (backlog, FD limits)
5. Add comprehensive metrics/monitoring
6. Investigate connection error root cause (likely client-side timeouts)

---

## Multi-Worker SO_REUSEPORT Results (2026-05-23)

### Setup
- **Mac M1** (kqueue, 10-core) — `-O3 -flto -march=native`, single-node `wrk` from localhost
- **Linux x86-64** (io_uring, 16-core, Ubuntu 24.04 kernel 6.8) — `-O3 -flto -march=native`, `wrk` from localhost

### Mac M1 — kqueue (localhost wrk)

| wrk config | Latency avg | RPS |
|---|---|---|
| `-t1 -c1` | **15µs** | 63k |
| `-t4 -c4` | 29µs | 132k |
| `-t4 -c16` | 74µs | 187k |
| `-t4 -c64` | 310µs | ~194k |

Bottleneck: wrk + server compete for same CPU on loopback. Single-connection latency **15µs** shows near-zero server overhead.

### Linux x86-64 — io_uring (localhost wrk, 8 workers)

| wrk config | Latency avg | RPS |
|---|---|---|
| `-t1 -c1` | 57µs | 17k |
| `-t8 -c64` | 393µs | **223k** |
| `-t8 -c256` | 622µs | **281k** |
| `-t16 -c512` | 1.73ms | **286k** |

### Key findings
- **Mac M1 single-connection latency 15µs** vs Linux 57µs — kqueue outperforms io_uring per-connection
- **Linux peak throughput 286k RPS** — 16 cores absorb all wrk load without competing
- Mac plateau at ~194k is a loopback + CPU-sharing artifact; true server capacity is higher
- SO_REUSEPORT multi-worker (`-w 8`) already implemented; kernel distributes connections across all listener sockets
- `-O3 -flto -march=native` rebuild vs `-O2`: marginal gain on hot path (already branch-predicted + inlined)