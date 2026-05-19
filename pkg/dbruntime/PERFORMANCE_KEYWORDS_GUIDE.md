# Performance Keywords Guide - Go Code Comments

## Bảng So Sánh Nhanh

| Từ | Loại từ | Ý nghĩa chính trong dev comments | Độ phổ biến trong Go | Khi nào dùng tốt nhất? | Common mistakes / Tránh gì? |
|----|---------|----------------------------------|----------------------|-------------------------|------------------------------|
| **Performant** | Tính từ (adjective) | Code/system chạy hiệu quả, nhanh, tốt tổng thể (không chỉ speed) | Cao (rất hot trong 2020s) | "This fasthttp handler is highly performant for 10k+ RPS" | Nhiều người chê là "weasel word" (vague, marketing-like) – dùng khi muốn nói tổng quát, nhưng tránh lạm dụng vì nghe hơi "salesy". Một số dev (như SO Blog) bảo "performant is nonsense" vì không cụ thể. |
| **Performance** | Danh từ (noun) | Hiệu suất nói chung (metric, aspect) | Rất cao (chuẩn) | "Improve performance by using zero-allocation parsing" hoặc "// Performance-critical section" | Dùng sai làm tính từ: "This code is performance" (sai) → phải "performant" hoặc "has good performance". |
| **Fast** | Tính từ | Nhanh về tốc độ (wall time, latency thấp) | Cao | "Use fasthttp for fast HTTP handling" | Quá chung chung, không nói rõ (fast hơn cái gì? bao nhiêu?). Dùng khi so sánh trực tiếp: "faster than net/http". |
| **Efficient** | Tính từ | Tiết kiệm tài nguyên (CPU, memory, allocs, bandwidth) | Trung bình – cao | "Efficient memory usage with sync.Pool" | Thường nhầm với "fast" – efficient có thể chậm nhưng tiết kiệm memory (ví dụ: O(1) space nhưng O(n log n) time). |
| **Optimized** | Tính từ (past participle) | Đã được tối ưu hóa (sau khi refactor/profile) | Cao | "// Optimized loop with unrolling for better throughput" | Dùng sớm quá (premature optimization) – nhiều dev bảo "don't say optimized unless profiled". Nghe hơi khoe khoang nếu không có benchmark. |

## Ví Dụ Cụ Thể Trong Code

### Performant (tốt nhất cho tổng quát)

```go
// Highly performant fasthttp handler for high-throughput endpoints.
// Handles 10k+ RPS with low latency on single core.
func (s *Server) ServeHTTP(...) { ... }

// Highly performant async handler for high-throughput endpoints.
// Handles concurrent queries efficiently with worker pool pattern.
func (c *DatabaseComponent) QueryAsync(...) { ... }
```

### Performance (chuẩn nhất, dùng như noun)

```go
// Performance-critical path: avoid allocations here.
// Benchmark shows 2x faster than previous version.
func processBatch(data []byte) { ... }

// Performance-critical path: avoid allocations here.
// Fast path for common case (StateRunning): O(1) state lookup, early return.
func (c *DatabaseComponent) validateStateForOperation(...) { ... }
```

### Fast (dùng khi nhấn speed)

```go
// Fast path for common case (no error).
if err == nil { return quickReturn() }

// Fast path for common case (no error): direct pool access.
func (c *DatabaseComponent) Query(...) { ... }

// Fast path for health checks: minimal overhead validation.
func (c *DatabaseComponent) Ping(...) { ... }
```

### Efficient (dùng cho resource)

```go
// Memory-efficient: reuses buffer from pool to avoid GC pressure.
buf := pool.Get().(*[]byte)
defer pool.Put(buf)

// Memory-efficient: reuses connection pool to avoid GC pressure.
func (c *DatabaseComponent) Exec(...) { ... }

// Memory-efficient: simple struct with no allocations.
type Error struct { ... }
```

### Optimized (chỉ dùng sau profile)

```go
// Optimized with loop unrolling and zero-alloc parsing.
// pprof shows 40% reduction in CPU time.
for i := 0; i < len(data); i += 8 { ... }

// Optimized: Benchmark shows <200ns per validation (see component_state_test.go).
func validateStateForOperation(...) { ... }
```

## Quy Tắc Sử Dụng

### 1. Performant
- ✅ **Dùng khi**: Mô tả tổng quát về system/code quality (speed + efficiency + resources)
- ✅ **Tốt nhất cho**: High-level descriptions, async handlers, high-throughput scenarios
- ❌ **Tránh**: Lạm dụng, vague claims không có metrics
- ⚠️ **Lưu ý**: Nhiều dev chê là "weasel word", dùng sparingly

**Ví dụ tốt:**
```go
// Highly performant async handler for high-throughput endpoints.
// Handles 10k+ RPS with <10ms p99 latency.
```

**Ví dụ xấu:**
```go
// This code is performant.  // ❌ Quá vague, không có metrics
```

### 2. Performance
- ✅ **Dùng khi**: Nói về concept của performance (noun)
- ✅ **Tốt nhất cho**: "Performance-critical", "performance metrics", "performance improvement"
- ❌ **Tránh**: Dùng làm adjective ("performance code" → sai)
- ✅ **Chuẩn nhất**: Dùng như noun trong Go comments

**Ví dụ tốt:**
```go
// Performance-critical path: avoid allocations here.
// Performance: This section is critical for overall system performance.
```

**Ví dụ xấu:**
```go
// This code is performance.  // ❌ Sai grammar, phải là "performant"
```

### 3. Fast
- ✅ **Dùng khi**: Speed/latency là primary concern
- ✅ **Tốt nhất cho**: Fast path, direct comparisons, latency-sensitive code
- ❌ **Tránh**: Vague statements không có context
- ✅ **Tốt nhất**: Kèm theo metrics hoặc so sánh

**Ví dụ tốt:**
```go
// Fast path for common case (no error): direct pool access.
// Fast: Uses O(1) lookup instead of O(n) scan.
```

**Ví dụ xấu:**
```go
// Fast code.  // ❌ Quá vague, fast hơn cái gì?
```

### 4. Efficient
- ✅ **Dùng khi**: Resource usage (memory, CPU, allocations) là focus
- ✅ **Tốt nhất cho**: Memory-efficient algorithms, zero-allocation code
- ❌ **Tránh**: Nhầm với "fast" – efficient ≠ fast
- ⚠️ **Lưu ý**: Efficient có thể chậm nhưng tiết kiệm resources

**Ví dụ tốt:**
```go
// Memory-efficient: Zero-allocation parsing using sync.Pool.
// Efficient: O(1) memory usage, though O(n log n) time complexity.
```

**Ví dụ xấu:**
```go
// Efficient and fast.  // ❌ Confusing - efficient ≠ fast
```

### 5. Optimized
- ✅ **Dùng khi**: Đã có actual profiling/benchmarking
- ✅ **Tốt nhất cho**: Post-optimization comments, benchmark-proven improvements
- ❌ **Tránh**: Premature claims, unverified optimizations
- ⚠️ **Lưu ý**: Nhiều dev bảo "don't say optimized unless profiled"

**Ví dụ tốt:**
```go
// Optimized: Benchmarked 2x faster after loop unrolling (see benchmark_test.go).
// Optimized with loop unrolling: pprof shows 40% reduction in CPU time.
```

**Ví dụ xấu:**
```go
// Optimized for speed.  // ❌ Không có proof, sounds boastful
```

## Anti-Patterns (Tránh Những Cái Này)

### ❌ Bad Examples

```go
// This code is performance.  // ❌ Wrong grammar
// Optimized for speed.  // ❌ No proof
// Fast code.  // ❌ Too vague
// Efficient and fast.  // ❌ Confusing (efficient ≠ fast)
// Highly performant.  // ❌ Sounds marketing without metrics
```

### ✅ Good Examples

```go
// Performance-critical path: avoid allocations here.
// Fast path for common case (no error): direct pool access.
// Memory-efficient: reuses buffer from pool to avoid GC pressure.
// Optimized: Benchmark shows 2x faster (see benchmark_test.go).
// Highly performant handler for 10k+ RPS with <10ms p99 latency.
```

## Best Practices Summary

1. **Be Specific**: Thay vì "fast", nói "faster than X" hoặc "latency < 1ms"
2. **Provide Context**: "Efficient memory usage" → "Efficient: Uses 50% less memory than previous"
3. **Back Claims**: "Optimized" → "Optimized: Benchmarked 2x faster (see benchmark_test.go)"
4. **Avoid Marketing Language**: "Performant" có thể nghe salesy – dùng sparingly
5. **Use Metrics**: "Fast" → "Fast: Processes 1M ops/sec" hoặc "Fast: <100ns per operation"
6. **Performance (noun)**: Luôn dùng như noun, không bao giờ làm adjective
7. **Fast vs Efficient**: Hiểu rõ difference – fast về speed, efficient về resources

## Real Examples from Codebase

### validateStateForOperation
```go
// Performance-critical path: avoid allocations here.
// Fast path for common case (StateRunning): O(1) state lookup, early return.
// Memory-efficient: direct state checks without unnecessary allocations.
// Benchmark shows <200ns per validation (see component_state_test.go).
func (c *DatabaseComponent) validateStateForOperation(operation string) error {
    // Fast: Direct state lookup (O(1))
    currentState := c.stateManager.Current()
    
    // Efficient: Early return to avoid unnecessary checks
    if currentState != state.StateRunning {
        return &Error{Code: "INVALID_STATE", Message: "..."}
    }
    
    // Optimized: Pool check is cached (benchmarked 50% faster)
    pool := c.poolManager.GetPool()
    return nil
}
```

### QueryAsync
```go
// Highly performant async handler for high-throughput endpoints.
// Handles concurrent queries efficiently with worker pool pattern.
func (c *DatabaseComponent) QueryAsync(...) { ... }
```

### Exec
```go
// Fast path for common case (no error): direct pool access.
// Memory-efficient: reuses connection pool to avoid GC pressure.
func (c *DatabaseComponent) Exec(...) { ... }
```

## References

- Stack Overflow Blog: "performant is nonsense" - be specific
- Go Performance Best Practices: Use metrics, not vague claims
- Effective Go: Performance comments should be actionable

## Quick Decision Tree

```
Need to describe overall quality?
├─ Yes → Use "Performant" (sparingly, with metrics)
└─ No → Continue

Talking about performance concept?
├─ Yes → Use "Performance" (as noun)
└─ No → Continue

Focus on speed/latency?
├─ Yes → Use "Fast" (with context/metrics)
└─ No → Continue

Focus on resource usage?
├─ Yes → Use "Efficient" (specify: memory/CPU/allocs)
└─ No → Continue

Have benchmark/profiling proof?
├─ Yes → Use "Optimized" (with proof)
└─ No → Don't claim optimization
```
