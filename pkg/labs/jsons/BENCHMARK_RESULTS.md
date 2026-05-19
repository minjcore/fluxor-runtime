# JSON Library Benchmark Results

## Overview

This document contains benchmark results comparing different JSON libraries for Go. The benchmarks test both **speed** (performance) and **purpose** (unique strengths) of each library.

## Libraries Benchmarked

1. **encoding/json** - Standard library (baseline)
2. **jsoniter** - High-performance drop-in replacement
3. **gjson** - Fast read-only parser (zero allocation for path queries)

**Note:** sonic and easyjson are planned but not included due to:
- **sonic**: Go 1.24+ compatibility issues (documented in plan)
- **easyjson**: Requires code generation step (see plan for details)

## Running Benchmarks

### Run All Benchmarks

```bash
go test ./pkg/labs/jsons -bench=BenchmarkJSON -benchmem
```

### Run Specific Library

```bash
# Standard library only
go test ./pkg/labs/jsons -bench=Benchmark.*_Stdlib -benchmem

# Jsoniter only
go test ./pkg/labs/jsons -bench=Benchmark.*_Jsoniter -benchmem

# GJSON only
go test ./pkg/labs/jsons -bench=BenchmarkGjson -benchmem
```

### Compare Libraries

```bash
# Compare encoding performance
go test ./pkg/labs/jsons -bench=BenchmarkEncode_Compare -benchmem

# Compare decoding performance
go test ./pkg/labs/jsons -bench=BenchmarkDecode_Compare -benchmem

# Memory allocation comparison
go test ./pkg/labs/jsons -bench=BenchmarkAlloc -benchmem
```

### Memory Profiling

```bash
# Generate memory profile
go test ./pkg/labs/jsons -bench=BenchmarkAlloc -benchmem -memprofile=mem.prof

# Analyze profile
go tool pprof mem.prof
```

## Benchmark Categories

### 1. Encoding (Marshal) Performance

Tests the performance of encoding Go structs to JSON bytes.

**Benchmarks:**
- `BenchmarkEncode_Small_*` - Small objects (< 1KB)
- `BenchmarkEncode_Medium_*` - Medium objects (1KB-100KB)
- `BenchmarkEncode_Large_*` - Large objects (100KB-10MB)
- `BenchmarkEncode_Structured_*` - Complex nested structures
- `BenchmarkEncode_Parallel_*` - Concurrent encoding

**Expected Results:**
- jsoniter: 1.5-2x faster than stdlib
- stdlib: Baseline performance

### 2. Decoding (Unmarshal) Performance

Tests the performance of decoding JSON bytes to Go structs.

**Benchmarks:**
- `BenchmarkDecode_Small_*` - Small JSON strings
- `BenchmarkDecode_Medium_*` - Medium JSON strings
- `BenchmarkDecode_Large_*` - Large JSON strings
- `BenchmarkDecode_Structured_*` - Complex nested structures
- `BenchmarkDecode_Parallel_*` - Concurrent decoding

**Expected Results:**
- jsoniter: 1.5-2x faster than stdlib
- stdlib: Baseline performance

### 3. Throughput Tests

Tests operations per second (ops/sec) for encode/decode operations.

**Benchmarks:**
- `BenchmarkThroughput_Encode_*` - Encoding throughput
- `BenchmarkThroughput_Decode_*` - Decoding throughput
- `BenchmarkThroughput_RoundTrip_*` - Full encode/decode cycle

### 4. Memory Allocation Benchmarks

Tests memory allocation per operation. Critical for high-throughput scenarios.

**Benchmarks:**
- `BenchmarkAlloc_Encode_*` - Allocation per encode operation
- `BenchmarkAlloc_Decode_*` - Allocation per decode operation

**Expected Results:**
- jsoniter: Similar or slightly lower allocation than stdlib
- stdlib: Baseline allocation

### 5. Use Case Specific Benchmarks

#### EventBus Usage (Small Messages)

**Benchmarks:**
- `BenchmarkEventBus_JSON_*` - EventBus-like small messages

**Use Case:** Many small JSON messages in event-driven systems.

#### REST API Usage (Medium Messages)

**Benchmarks:**
- `BenchmarkAPI_JSON_*` - REST API-like medium messages

**Use Case:** HTTP API responses with structured data.

#### Config Loading (Structured Data)

**Benchmarks:**
- `BenchmarkConfig_JSON_*` - Configuration file structures

**Use Case:** Loading and parsing configuration files.

#### High-Throughput Scenarios

**Benchmarks:**
- `BenchmarkHighThroughput_SmallPayloads_*` - Many small messages concurrently

**Use Case:** EventBus scenarios with 200k+ RPS targets.

### 6. GJSON Benchmarks (Read-Only Parsing)

GJSON is a read-only parser optimized for path-based queries without full unmarshaling.

**Benchmarks:**
- `BenchmarkGjson_PartialParse` - Path-based queries
- `BenchmarkGjson_LargeFiles` - Large file parsing

**Use Case:** When you only need to extract specific fields from JSON without full unmarshaling.

**Expected Results:**
- GJSON: Very fast for path queries, zero allocation for simple queries
- Much faster than full unmarshaling when you only need a few fields

## Test Functions

### Compatibility Tests

- `TestStdlib_Compatibility` - RFC 7159 compliance verification
- `TestStdlib_ErrorHandling` - Robust error handling verification

### JSONITER Tests

- `TestJsoniter_DropInReplacement` - API compatibility with stdlib
- `BenchmarkJsoniter_VsStdlib` - Direct performance comparison

### GJSON Tests

- `TestGjson_PathQueries` - Path query functionality verification

## Performance Comparison Table

*Note: Run benchmarks on your system to get actual results. Results vary by hardware, Go version, and workload.*

| Library | Encode Speed | Decode Speed | Memory Alloc | Use Case |
|---------|--------------|--------------|--------------|----------|
| **encoding/json** | Baseline | Baseline | Baseline | General purpose, compatibility |
| **jsoniter** | 1.5-2x faster | 1.5-2x faster | Similar | Drop-in replacement, performance |
| **gjson** | N/A (read-only) | Fast (partial) | Very Low | Path queries, large files |

## Memory Allocation Analysis

Memory allocation is critical for high-throughput scenarios:

- **Small allocations** reduce GC pressure
- **Zero-allocation** operations are ideal for hot paths
- **Predictable allocation** helps with memory planning

**Run with `-benchmem` flag to see allocation metrics:**

```bash
go test ./pkg/labs/jsons -bench=BenchmarkAlloc -benchmem
```

## Use Case Recommendations

### General Purpose / Default Choice

**Recommended: encoding/json**

- Standard library, zero dependencies
- Full RFC 7159 compliance
- Good enough for most use cases
- Maximum compatibility

### High-Performance Upgrade

**Recommended: jsoniter**

- Drop-in replacement for encoding/json
- 1.5-2x performance improvement
- No code generation required
- Good balance of features and performance

### Read-Only Parsing

**Recommended: gjson**

- Fastest for path-based queries
- Zero-allocation for simple queries
- Perfect for large files
- Only when you don't need full unmarshaling

### Future: Maximum Performance

**Planned: sonic or easyjson**

- **sonic**: When Go 1.24+ compatibility is resolved
- **easyjson**: When code generation is acceptable
- 3-4x performance improvement over stdlib

## Interpreting Results

### Throughput (ops/sec)

Higher is better. Indicates how many operations per second the library can handle.

### Allocations

Lower is better. Fewer allocations mean:
- Less GC pressure
- Better performance under load
- More predictable latency

### Latency (ns/op)

Lower is better. Indicates how long each operation takes on average.

### B/op (Bytes per operation)

Lower is better. Indicates memory allocated per operation.

### allocs/op (Allocations per operation)

Lower is better. Indicates number of allocations per operation.

## Example Output

```
BenchmarkEncode_Small_Stdlib-8          1000000    1200 ns/op     512 B/op    3 allocs/op
BenchmarkEncode_Small_Jsoniter-8        2000000     800 ns/op     480 B/op    3 allocs/op
```

This shows:
- jsoniter is ~1.5x faster (1200ns vs 800ns)
- Similar memory allocation (512B vs 480B)
- Same number of allocations (3)

## Notes

- **Warm-up runs**: First few iterations may be slower due to cache effects
- **Fair comparison**: All benchmarks use the same test data
- **Reproducibility**: Run multiple times and average for stable results
- **System variance**: Results vary by CPU, memory, and Go version

## Future Enhancements

1. Add sonic benchmarks when Go 1.24+ compatibility is resolved
2. Add easyjson benchmarks with code generation setup
3. Add streaming benchmarks for large documents
4. Add more real-world scenario benchmarks
5. Add CPU profiling support
6. Add automated result collection and comparison

## References

- [encoding/json](https://pkg.go.dev/encoding/json) - Go standard library
- [jsoniter](https://github.com/json-iterator/go) - High-performance JSON library
- [gjson](https://github.com/tidwall/gjson) - Fast JSON parser
- [JSON Library Comparison](./json_libs_comparison.md) - Detailed library comparison

---

**Last Updated:** 2026-01-04  
**Maintained by:** Fluxor Labs
