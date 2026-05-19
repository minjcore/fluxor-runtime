# JSON Library Benchmarks

This package contains comprehensive benchmark tests for comparing different JSON libraries in Go. The benchmarks test both **speed** (performance) and **purpose** (unique strengths) of each library.

## Quick Start

### Run All Benchmarks

```bash
go test ./pkg/labs/jsons -bench=. -benchmem
```

### Run Specific Categories

```bash
# Encoding benchmarks
go test ./pkg/labs/jsons -bench=BenchmarkEncode -benchmem

# Decoding benchmarks
go test ./pkg/labs/jsons -bench=BenchmarkDecode -benchmem

# Memory allocation benchmarks
go test ./pkg/labs/jsons -bench=BenchmarkAlloc -benchmem

# Use case specific benchmarks
go test ./pkg/labs/jsons -bench=BenchmarkEventBus -benchmem
go test ./pkg/labs/jsons -bench=BenchmarkAPI -benchmem
```

### Compare Libraries

```bash
# Compare stdlib vs jsoniter
go test ./pkg/labs/jsons -bench=BenchmarkJsoniter_VsStdlib -benchmem
```

## Libraries Benchmarked

1. **encoding/json** - Standard library (baseline)
2. **jsoniter** - High-performance drop-in replacement
3. **gjson** - Fast read-only parser (zero allocation for path queries)

**Note:** sonic and easyjson are planned but not included due to:
- **sonic**: Go 1.24+ compatibility issues
- **easyjson**: Requires code generation step

## Package Structure

```
pkg/labs/jsons/
├── README.md              # This file
├── benchmark_types.go     # Shared types and test data
├── helpers.go             # Library helper functions
├── json_bench_test.go     # Main benchmark tests
└── BENCHMARK_RESULTS.md   # Benchmark results documentation
```

## Benchmark Categories

### 1. Speed Benchmarks (Performance)

- **Encoding**: Small, medium, large, and structured payloads
- **Decoding**: Small, medium, large, and structured payloads
- **Throughput**: Operations per second measurements
- **Parallel**: Concurrent encode/decode operations

### 2. Purpose Benchmarks (Strengths)

- **encoding/json**: Compatibility and safety demonstrations
- **jsoniter**: Drop-in replacement ease and performance
- **gjson**: Partial parsing and zero-allocation for read-only scenarios

### 3. Memory Allocation

- Allocation per encode/decode operation
- Critical for high-throughput scenarios (200k+ RPS)

### 4. Use Case Specific

- **EventBus**: Small message scenarios
- **REST API**: Medium message scenarios
- **Config**: Structured configuration loading
- **High-Throughput**: Concurrent processing scenarios

## Expected Results

*Run benchmarks on your system for actual results. Results vary by hardware, Go version, and workload.*

| Library | Performance | Use Case |
|---------|-------------|----------|
| **encoding/json** | Baseline | General purpose, compatibility |
| **jsoniter** | 1.5-2x faster | Drop-in replacement, performance |
| **gjson** | Very fast (partial) | Path queries, large files |

## Helper Functions

The `helpers.go` file provides helper functions for each library:

- `encodeWithStdlib(data)` / `decodeWithStdlib(data, v)`
- `encodeWithJsoniter(data)` / `decodeWithJsoniter(data, v)`
- `encodeWithSonic(data)` / `decodeWithSonic(data, v)` (placeholder)
- `encodeWithEasyjson(data)` / `decodeWithEasyjson(data, v)` (placeholder)
- `parseWithGjson(jsonStr, path)` - Path-based parsing

## Running Tests

```bash
# Run all tests (not benchmarks)
go test ./pkg/labs/jsons -v

# Run specific test
go test ./pkg/labs/jsons -v -run TestJsoniter
```

## Memory Profiling

```bash
# Generate memory profile
go test ./pkg/labs/jsons -bench=BenchmarkAlloc -benchmem -memprofile=mem.prof

# Analyze profile
go tool pprof mem.prof
```

## Documentation

For detailed benchmark results and recommendations, see:
- [Benchmark Results](./BENCHMARK_RESULTS.md) - Detailed benchmark documentation
- [JSON Library Comparison](../json_libs_comparison.md) - Comprehensive library comparison

## Contributing

When adding new benchmarks:

1. Use shared types from `benchmark_types.go`
2. Follow naming convention: `BenchmarkCategory_Library`
3. Use `b.ResetTimer()` before the measured loop
4. Use `b.ReportAllocs()` for allocation benchmarks
5. Document what the benchmark tests and expected results

## Notes

- Benchmarks use the same test data for fair comparison
- First few iterations may be slower due to cache effects
- Run multiple times and average for stable results
- Results vary by CPU, memory, and Go version

---

**Part of:** [Fluxor Labs](../README.md)  
**Maintained by:** Fluxor Labs
