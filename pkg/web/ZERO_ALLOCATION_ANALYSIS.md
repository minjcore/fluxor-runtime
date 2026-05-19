# Zero-Allocation và Hot-Path Analysis

Phân tích chi tiết về zero-allocation patterns và hot-path optimization trong Fluxor web framework.

## Tổng quan

Hot-path là code được thực thi nhiều nhất trong mỗi request (100k+ RPS). Mỗi allocation trong hot-path sẽ được nhân lên 100,000 lần mỗi giây, gây ra:
- **GC pressure**: Tăng garbage collection overhead
- **Memory churn**: Tăng memory usage
- **Latency spikes**: GC pauses ảnh hưởng đến latency

## Hot-Path Code Flow

### Request Handling Flow

```
1. handleRequest() [fast_server.go:676]
   ↓
2. processRequest() [fast_server.go:842]
   ↓
3. router.ServeFastHTTP() [fast_router.go:42]
   ↓
4. matchPath() + extractParams() [fast_router.go:226, 246]
   ↓
5. Handler execution
```

### Hot-Path Functions

| Function | Location | Calls/sec (100k RPS) | Allocations |
|----------|----------|---------------------|-------------|
| `handleRequest()` | `fast_server.go:676` | 100,000 | ❌ Multiple |
| `processRequest()` | `fast_server.go:842` | 100,000 | ❌ Multiple |
| `ServeFastHTTP()` | `fast_router.go:42` | 100,000 | ❌ Multiple |
| `matchPath()` | `fast_router.go:226` | 100,000 | ❌ Yes |
| `extractParams()` | `fast_router.go:246` | ~50,000 | ❌ Yes |

## Allocations trong Hot-Path

### 1. String Conversions (Critical)

**Location**: `fast_server.go:677-678`, `fast_router.go:46-47`

```go
// ❌ BAD: Allocates new string every call
method := string(ctx.Method())
path := string(ctx.Path())
```

**Impact**: 
- 2 allocations per request
- 200,000 allocations/sec at 100k RPS
- ~16 bytes each = 3.2 MB/sec

**Solution**: Use byte slices directly
```go
// ✅ GOOD: Zero allocation
method := ctx.Method()  // Returns []byte
path := ctx.Path()      // Returns []byte
```

### 2. Map Allocations (Critical)

**Location**: `fast_server.go:870`, `fast_router.go:246`

```go
// ❌ BAD: Allocates map every request
Params: make(map[string]string),
```

**Impact**:
- 1 allocation per request with params
- ~50,000 allocations/sec
- ~48 bytes each = 2.4 MB/sec

**Solution**: Use sync.Pool (already implemented in lite/webfast)
```go
// ✅ GOOD: Reuse from pool
type FastRouter struct {
    paramPool sync.Pool // stores map[string]string
}

func (r *FastRouter) extractParams(...) {
    params := r.paramPool.Get().(map[string]string)
    defer func() {
        // Clear and return to pool
        for k := range params {
            delete(params, k)
        }
        r.paramPool.Put(params)
    }()
    // ... use params
}
```

### 3. fmt.Sprintf() trong Logging (High)

**Location**: Multiple locations in hot-path

```go
// ❌ BAD: Allocates string
s.Logger().Info(fmt.Sprintf("processing request: %s %s", method, path))
```

**Impact**:
- 1 allocation per log statement
- Can be 100,000+ allocations/sec if logging every request

**Solution**: 
- Use structured logging with byte slices
- Or disable logging in hot-path (use sampling)

```go
// ✅ GOOD: Zero allocation logging
logger.Info("processing request",
    "method", ctx.Method(),  // []byte
    "path", ctx.Path(),      // []byte
)
```

### 4. JSON Response Maps (High)

**Location**: `fast_server.go:701`, `verticle.go:213`

```go
// ❌ BAD: Allocates map
response := map[string]interface{}{
    "error":   "capacity_exceeded",
    "message": "Server at normal capacity",
}
```

**Impact**:
- 1 allocation per error response
- ~1,000-10,000 allocations/sec (depending on error rate)

**Solution**: Pre-allocate error responses or use byte slices
```go
// ✅ GOOD: Pre-allocated response
var backpressureResponse = []byte(`{"error":"capacity_exceeded","message":"Server at normal capacity"}`)

// In handler:
ctx.Write(backpressureResponse)
```

### 5. strings.Split() trong Path Matching (Medium)

**Location**: `fast_router.go:227-228`, `fast_router.go:247-248`

```go
// ❌ BAD: Allocates slices
patternParts := strings.Split(pattern, "/")
pathParts := strings.Split(path, "/")
```

**Impact**:
- 2 allocations per route match attempt
- ~200,000 allocations/sec (if matching many routes)

**Solution**: Use byte slice operations or pre-compile routes
```go
// ✅ GOOD: Zero-allocation path matching
func matchPathBytes(pattern, path []byte) bool {
    // Use byte slice operations instead of strings.Split
    // Or pre-compile routes into segments
}
```

## Zero-Allocation Patterns Hiện Có

### 1. sync.Pool trong lite/webfast/router.go ✅

```go
type Router struct {
    paramPool sync.Pool // stores []fx.Param
}

func matchAndFill(rt *route, path []byte, c *fx.FastContext, pool *sync.Pool) bool {
    // Get from pool
    params := pool.Get().([]fx.Param)
    defer func() {
        params = params[:0]  // Clear
        pool.Put(params)     // Return to pool
    }()
    // ... use params
}
```

**Status**: ✅ Implemented in lite package
**Recommendation**: Apply to main `FastRouter` in `pkg/web`

### 2. FastHTTP Byte Slices ✅

FastHTTP uses `[]byte` instead of `string` to avoid allocations:
- `ctx.Method()` returns `[]byte`
- `ctx.Path()` returns `[]byte`
- `ctx.QueryArgs().Peek(key)` returns `[]byte`

**Status**: ✅ Available but not always used
**Recommendation**: Use byte slices throughout hot-path

### 3. Atomic Operations ✅

```go
atomic.AddInt64(&s.queuedRequests, -1)
atomic.LoadInt64(&s.rejectedRequests)
```

**Status**: ✅ Already used
**Impact**: Zero allocations for counters

## Optimization Recommendations

### Priority 1: Critical (100k+ allocations/sec)

1. **Eliminate string() conversions in hot-path**
   - Use `[]byte` directly from FastHTTP
   - Update all hot-path code to use byte slices

2. **Implement sync.Pool for params map**
   - Copy pattern from `lite/webfast/router.go`
   - Apply to `FastRouter.extractParams()`

3. **Pre-allocate error responses**
   - Create static byte slices for common error responses
   - Use `ctx.Write()` instead of JSON encoding

### Priority 2: High (10k-100k allocations/sec)

4. **Optimize path matching**
   - Pre-compile routes into byte slice segments
   - Use byte slice comparison instead of `strings.Split()`

5. **Disable/sample logging in hot-path**
   - Use sampling (log 1 in 1000 requests)
   - Or use structured logging with byte slices

6. **Reuse JSON encoders**
   - Use `sync.Pool` for JSON encoders
   - Or pre-encode common responses

### Priority 3: Medium (1k-10k allocations/sec)

7. **Optimize middleware**
   - Avoid allocations in middleware chain
   - Use byte slices for header operations

8. **Cache route matching results**
   - For static routes, cache match results
   - Use radix tree or similar for O(1) matching

## Implementation Plan

### Phase 1: Zero-Allocation Hot-Path

```go
// Before (with allocations)
func (s *FastHTTPServer) handleRequest(ctx *fasthttp.RequestCtx) {
    method := string(ctx.Method())  // ❌ Allocation
    path := string(ctx.Path())      // ❌ Allocation
    // ...
}

// After (zero-allocation)
func (s *FastHTTPServer) handleRequest(ctx *fasthttp.RequestCtx) {
    method := ctx.Method()  // ✅ []byte, no allocation
    path := ctx.Path()      // ✅ []byte, no allocation
    // ...
}
```

### Phase 2: sync.Pool for Params

```go
type FastRouter struct {
    // ... existing fields
    paramPool sync.Pool // stores map[string]string
}

func NewFastRouter() *FastRouter {
    r := &FastRouter{
        // ... existing initialization
    }
    r.paramPool = sync.Pool{
        New: func() interface{} {
            return make(map[string]string, 8) // Pre-allocate capacity
        },
    }
    return r
}

func (r *FastRouter) extractParams(pattern, path string, params map[string]string) {
    // Get from pool if params is nil
    if params == nil {
        params = r.paramPool.Get().(map[string]string)
        defer func() {
            // Clear and return to pool
            for k := range params {
                delete(params, k)
            }
            r.paramPool.Put(params)
        }()
    }
    // ... extract params
}
```

### Phase 3: Pre-allocated Error Responses

```go
var (
    // Pre-allocated error responses (zero allocation)
    backpressureResponse = []byte(`{"error":"capacity_exceeded","message":"Server at normal capacity - backpressure applied","code":"BACKPRESSURE"}`)
    panicResponsePrefix  = []byte(`{"error":"handler_panic","message":"Request handler failed","request_id":"`)
    panicResponseSuffix  = []byte(`"}`)
)

func (s *FastHTTPServer) handleRequest(ctx *fasthttp.RequestCtx) {
    if !s.backpressure.TryAcquire() {
        ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
        ctx.SetContentType("application/json")
        ctx.Write(backpressureResponse) // ✅ Zero allocation
        return
    }
    // ...
}
```

### Phase 4: Optimized Path Matching

```go
// Pre-compile routes into segments
type fastRoute struct {
    method  string
    path    string
    handler FastRequestHandler
    // Pre-compiled segments (zero allocation matching)
    segments [][]byte  // Split path into segments at route registration
    isStatic bool      // Fast path for static routes
}

func (r *FastRouter) RouteFast(method, path string, handler FastRequestHandler) {
    segments := make([][]byte, 0, 8)
    // Split once at registration time
    for _, seg := range bytes.Split([]byte(path), []byte("/")) {
        segments = append(segments, seg)
    }
    
    r.routes = append(r.routes, &fastRoute{
        method:   method,
        path:     path,
        handler:  handler,
        segments: segments,
        isStatic: !strings.Contains(path, ":"),
    })
}

func (r *FastRouter) matchPathBytes(patternSegments [][]byte, path []byte) bool {
    pathSegments := bytes.Split(path, []byte("/"))
    // Compare pre-compiled segments (zero allocation)
    // ...
}
```

## Performance Impact Estimate

### Current State (with allocations)

| Operation | Allocations/Request | Allocations/sec (100k RPS) | Memory/sec |
|-----------|-------------------|--------------------------|------------|
| String conversions | 2 | 200,000 | 3.2 MB |
| Params map | 1 | 50,000 | 2.4 MB |
| fmt.Sprintf | 1-3 | 100,000-300,000 | 1.6-4.8 MB |
| JSON maps | 0.1 | 10,000 | 0.5 MB |
| **Total** | **4-7** | **360,000-560,000** | **~8-11 MB/sec** |

### After Optimization (zero-allocation)

| Operation | Allocations/Request | Allocations/sec (100k RPS) | Memory/sec |
|-----------|-------------------|--------------------------|------------|
| String conversions | 0 | 0 | 0 MB |
| Params map (pooled) | 0 | 0 | 0 MB |
| fmt.Sprintf | 0 (sampled) | 100-300 | 0.002 MB |
| JSON maps (pre-alloc) | 0 | 0 | 0 MB |
| **Total** | **0** | **100-300** | **~0.002 MB/sec** |

**Improvement**: 
- **99.9% reduction** in allocations
- **99.98% reduction** in memory churn
- **Lower GC pressure** → better latency
- **Higher throughput** → can handle more RPS

## Measurement và Validation

### Tools

1. **pprof** - Profile allocations
```bash
go test -bench=. -cpuprofile=cpu.prof -memprofile=mem.prof
go tool pprof -alloc_objects mem.prof
```

2. **Benchmark với allocations**
```go
func BenchmarkHandleRequest(b *testing.B) {
    // ... setup
    b.ReportAllocs()  // Report allocations
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        handleRequest(ctx)
    }
}
```

3. **Runtime stats**
```go
var m1, m2 runtime.MemStats
runtime.ReadMemStats(&m1)
// ... run workload
runtime.ReadMemStats(&m2)
allocations := m2.Mallocs - m1.Mallocs
```

### Validation Checklist

- [ ] Zero allocations in `handleRequest()` hot-path
- [ ] Zero allocations in `processRequest()` hot-path
- [ ] Zero allocations in `ServeFastHTTP()` hot-path
- [ ] Params map uses `sync.Pool`
- [ ] Error responses are pre-allocated
- [ ] Logging uses sampling or byte slices
- [ ] Path matching uses byte slices
- [ ] Benchmark shows < 10 allocations per request

## Best Practices

1. **Always use `[]byte` in hot-path**
   - FastHTTP provides byte slices, use them
   - Avoid `string()` conversions

2. **Use `sync.Pool` for frequently allocated objects**
   - Maps, slices, buffers
   - Clear before returning to pool

3. **Pre-allocate static responses**
   - Error messages
   - Common JSON responses
   - Status messages

4. **Sample logging in hot-path**
   - Log 1 in 1000 requests
   - Or use structured logging with byte slices

5. **Pre-compile routes**
   - Split paths at registration time
   - Cache match results for static routes

6. **Profile before optimizing**
   - Use `pprof` to find real bottlenecks
   - Don't optimize prematurely

## References

- [FastHTTP Zero-Allocation Guide](https://github.com/valyala/fasthttp#performance-optimization-tips-for-package-users)
- [Go sync.Pool Best Practices](https://github.com/golang/go/blob/master/src/sync/pool.go)
- [Zero-Allocation JSON](https://github.com/json-iterator/go)
