# Performance Keywords Guide for Go Code Comments

## Quick Reference

| Keyword | Type | Usage Pattern | Example |
|---------|------|---------------|---------|
| **Performant** | Adjective | Tổng quát, high-throughput | `// Highly performant handler for 10k+ RPS` |
| **Performance** | Noun | Performance-critical path | `// Performance-critical: avoid allocations here` |
| **Fast** | Adjective | Nhấn speed, fast path | `// Fast path for common case (no error)` |
| **Efficient** | Adjective | Resource usage, memory | `// Memory-efficient: reuses buffer from pool` |
| **Optimized** | Adjective | Sau profile, có benchmark | `// Optimized: pprof shows 40% reduction` |

## Usage Examples from Codebase

### Performant (tốt nhất cho tổng quát)

```go
// Highly performant async handler for high-throughput endpoints.
// Handles concurrent queries efficiently with worker pool pattern.
func (c *DatabaseComponent) QueryAsync(...) { ... }
```

### Performance (chuẩn nhất, dùng như noun)

```go
// Performance-critical path: avoid allocations here.
// Fast path for common case (StateRunning): O(1) state lookup, early return.
func (c *DatabaseComponent) validateStateForOperation(...) { ... }
```

### Fast (dùng khi nhấn speed)

```go
// Fast path for common case (no error): direct pool access.
if err := c.validateStateForOperation("Query"); err != nil {
    return nil, err
}
```

### Efficient (dùng cho resource)

```go
// Memory-efficient: reuses connection pool to avoid GC pressure.
func (c *DatabaseComponent) Exec(...) { ... }

// Memory-efficient: simple struct with no allocations.
type Error struct { ... }
```

### Optimized (chỉ dùng sau profile)

```go
// Optimized with loop unrolling and zero-alloc parsing.
// Benchmark shows <200ns per validation (see component_state_test.go).
func validateStateForOperation(...) { ... }
```

## Best Practices

1. **Performant**: Dùng cho tổng quát, high-level descriptions
   - ✅ "Highly performant handler for 10k+ RPS"
   - ❌ Tránh lạm dụng (sounds marketing)

2. **Performance**: Dùng như noun, performance-critical sections
   - ✅ "Performance-critical path: avoid allocations"
   - ❌ Không dùng làm adjective ("performance code" → sai)

3. **Fast**: Nhấn speed, fast path
   - ✅ "Fast path for common case (no error)"
   - ❌ Tránh vague ("fast code" → thiếu context)

4. **Efficient**: Resource usage, memory/CPU
   - ✅ "Memory-efficient: reuses buffer from pool"
   - ❌ ≠ Fast (efficient có thể chậm nhưng tiết kiệm)

5. **Optimized**: Chỉ sau profile/benchmark
   - ✅ "Optimized: Benchmark shows 2x faster"
   - ❌ Không claim không có proof

## Common Mistakes to Avoid

❌ **Bad**: `// This code is performance` (wrong grammar)  
✅ **Good**: `// Performance-critical: this code path`

❌ **Bad**: `// Optimized for speed` (no proof)  
✅ **Good**: `// Optimized: Benchmark shows 2x faster (see benchmark_test.go)`

❌ **Bad**: `// Fast code` (too vague)  
✅ **Good**: `// Fast path for common case (no error)`

❌ **Bad**: `// Efficient and fast` (confusing - efficient ≠ fast)  
✅ **Good**: `// Memory-efficient: O(1) space, Fast: <100ns latency`

❌ **Bad**: `// Highly performant` (sounds marketing without metrics)  
✅ **Good**: `// Highly performant handler for 10k+ RPS with <10ms p99 latency`
