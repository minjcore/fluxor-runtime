# Thread & State Management in Go Runtime

Hướng dẫn chi tiết về quản lý threads và state trong Go runtime, đặc biệt cho CPU-bound workloads (LLM, crypto, ML).

## 1. Go Runtime Scheduler Overview

### Goroutines vs OS Threads

```
┌─────────────────────────────────────────┐
│  Go Runtime Scheduler (M:N Model)       │
│                                          │
│  ┌──────────┐  ┌──────────┐            │
│  │ Goroutine│  │ Goroutine│  ...        │
│  └────┬─────┘  └────┬─────┘            │
│       │             │                   │
│       └─────┬───────┘                   │
│             │                           │
│       ┌─────▼─────┐                     │
│       │ OS Thread │  (GOMAXPROCS)       │
│       └───────────┘                     │
└─────────────────────────────────────────┘
```

**Key Points:**
- **Goroutines**: Lightweight (2KB stack), managed by Go scheduler
- **OS Threads**: Heavyweight (1-2MB stack), managed by OS
- **M:N Model**: M goroutines mapped to N OS threads (N = GOMAXPROCS)
- **Preemption**: Go scheduler can preempt goroutines at safe points

### GOMAXPROCS

```go
// Get current value
numCPU := runtime.GOMAXPROCS(0)  // Returns current setting

// Set value (affects number of OS threads)
runtime.GOMAXPROCS(8)  // Use 8 OS threads

// Default: Number of CPU cores
```

**Rule of Thumb:**
- **IO-bound**: GOMAXPROCS = CPU cores (default)
- **CPU-bound**: GOMAXPROCS = CPU cores (default)
- **Mixed**: GOMAXPROCS = CPU cores (default)

## 2. CPU-Bound vs IO-Bound: Khái Niệm Cơ Bản

### 2.1 CPU-Bound (Ràng Buộc CPU)

**Định nghĩa:**
- **CPU-Bound** là các tác vụ mà thời gian xử lý chủ yếu phụ thuộc vào tốc độ CPU
- CPU làm việc liên tục, ít khi phải chờ đợi (waiting)
- Tác vụ sử dụng nhiều CPU cycles, ít I/O operations

**Đặc điểm:**
- ✅ CPU utilization cao (80-100%)
- ✅ Ít blocking operations
- ✅ Tính toán phức tạp, xử lý dữ liệu
- ✅ Native code (CGO, C/C++ libraries)

**Ví dụ CPU-Bound:**
```go
// 1. Machine Learning / AI Inference
func inferModel(input []float32) []float32 {
    // Matrix multiplication, neural network inference
    // CPU làm việc liên tục, không chờ I/O
    return model.Forward(input)
}

// 2. Cryptographic Operations
func hashPassword(password string) string {
    // bcrypt, scrypt - CPU-intensive
    hash, _ := bcrypt.GenerateFromPassword([]byte(password), 10)
    return string(hash)
}

// 3. Image/Video Processing
func processImage(img image.Image) image.Image {
    // Resize, filter, encode - CPU-intensive
    return resize.Resize(800, 600, img, resize.Lanczos3)
}

// 4. Data Compression
func compress(data []byte) []byte {
    // gzip, zstd - CPU-intensive
    var buf bytes.Buffer
    w := gzip.NewWriter(&buf)
    w.Write(data)
    w.Close()
    return buf.Bytes()
}

// 5. Scientific Computing
func calculatePi(n int) float64 {
    // Numerical computation - CPU-intensive
    sum := 0.0
    for i := 0; i < n; i++ {
        sum += 1.0 / float64(i*2+1)
    }
    return sum * 4
}
```

**Thread Management cho CPU-Bound:**
```go
// ✅ SỬ DỤNG: runtime.LockOSThread() cho CPU-bound native code
go func() {
    runtime.LockOSThread()  // Pin thread để tránh preemption
    defer runtime.UnlockOSThread()
    
    // CPU-bound work với native threads (llama.cpp)
    result := nativeLLMInference(request)
}()

// ✅ SỬ DỤNG: Worker pool với số lượng = CPU cores
pool := compute.NewComputePool(ctx, handler, compute.Config{
    Workers: runtime.NumCPU(),  // 1 worker per CPU core
})
```

**⚠️ QUAN TRỌNG: Không nên spawn quá nhiều goroutines cho CPU-Bound**

```go
// ❌ SAI: Spawn quá nhiều goroutines cho CPU-bound work
for i := 0; i < 1000; i++ {
    go func() {
        // CPU-bound computation
        calculatePi(1000000)  // Mỗi goroutine chiếm CPU
    }()
}
// Vấn đề:
// - 1000 goroutines cạnh tranh cho 8 CPU cores
// - Context switching overhead tăng cao
// - Cache thrashing (cache bị đẩy ra liên tục)
// - Preemption overhead (Go scheduler phải preempt liên tục)
// - Không cải thiện throughput (CPU đã busy 100%)
// - Thậm chí có thể làm chậm hơn!

// ✅ ĐÚNG: Giới hạn số goroutines = CPU cores
numCPU := runtime.NumCPU()
for i := 0; i < numCPU; i++ {
    go func() {
        // CPU-bound computation
        calculatePi(1000000)
    }()
}
// Hoặc tốt hơn: Dùng worker pool
pool := compute.NewComputePool(ctx, handler, compute.Config{
    Workers: runtime.NumCPU(),  // Chỉ tạo số workers = CPU cores
})
```

**Tại sao không nên quá nhiều goroutines cho CPU-Bound?**

1. **Context Switching Overhead:**
   ```
   CPU Core 0: [Goroutine 1] → [Goroutine 2] → [Goroutine 3] → ...
                ↑ Preempt      ↑ Preempt      ↑ Preempt
   ```
   - Go scheduler phải preempt liên tục
   - Mỗi lần switch tốn thời gian (microseconds)
   - Tích lũy lại = overhead đáng kể

2. **Cache Thrashing:**
   ```
   Goroutine 1: Load data vào CPU cache
   Goroutine 2: Preempt → Đẩy cache của G1 ra
   Goroutine 3: Preempt → Đẩy cache của G2 ra
   Goroutine 1: Resume → Cache miss, phải load lại
   ```
   - CPU cache bị đẩy ra liên tục
   - Cache miss rate tăng cao
   - Memory access chậm hơn (RAM vs L1/L2/L3 cache)

3. **Không cải thiện throughput:**
   ```
   8 CPU cores = Tối đa 8 tasks song song
   1000 goroutines ≠ 1000x nhanh hơn
   = Chỉ 8 tasks chạy, 992 tasks chờ
   = Overhead cao, không có lợi
   ```

4. **GOMAXPROCS giới hạn:**
   ```go
   runtime.GOMAXPROCS(0)  // Default: số CPU cores
   // Chỉ có N OS threads = N CPU cores
   // N goroutines CPU-bound cạnh tranh cho N threads
   // Nhiều hơn N = overhead, không có lợi
   ```

**Tối ưu cho CPU-Bound:**
- ✅ Số workers ≤ số CPU cores (thường = CPU cores)
- ✅ Sử dụng `runtime.LockOSThread()` cho native code
- ✅ Thread-local state để tránh contention
- ✅ Cache locality (NUMA awareness)
- ❌ KHÔNG spawn quá nhiều goroutines (≤ CPU cores)
- ❌ KHÔNG dùng pattern "1 goroutine per task" cho CPU-bound

### 2.2 IO-Bound (Ràng Buộc I/O)

**Định nghĩa:**
- **IO-Bound** là các tác vụ mà thời gian xử lý chủ yếu phụ thuộc vào tốc độ I/O
- CPU thường xuyên phải chờ đợi (waiting) cho I/O operations hoàn thành
- Tác vụ sử dụng ít CPU cycles, nhiều I/O operations

**Đặc điểm:**
- ✅ CPU utilization thấp (10-30%)
- ✅ Nhiều blocking operations (waiting)
- ✅ Network requests, file I/O, database queries
- ✅ CPU idle trong khi chờ I/O

**Ví dụ IO-Bound:**
```go
// 1. HTTP Requests
func fetchData(url string) ([]byte, error) {
    // CPU chờ network I/O
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return io.ReadAll(resp.Body)
}

// 2. Database Queries
func getUser(id int) (*User, error) {
    // CPU chờ database I/O
    var user User
    err := db.QueryRow("SELECT * FROM users WHERE id = $1", id).Scan(&user)
    return &user, err
}

// 3. File I/O
func readFile(path string) ([]byte, error) {
    // CPU chờ disk I/O
    return os.ReadFile(path)
}

// 4. Message Queue Operations
func publishMessage(topic string, msg []byte) error {
    // CPU chờ network I/O (NATS, Kafka, RabbitMQ)
    return natsConn.Publish(topic, msg)
}

// 5. gRPC Calls
func callService(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    // CPU chờ network I/O
    return client.Call(ctx, req)
}
```

**Thread Management cho IO-Bound:**
```go
// ✅ SỬ DỤNG: Default goroutine scheduling (KHÔNG pin thread)
go func() {
    // Go scheduler có thể preempt goroutine khi chờ I/O
    // Nhiều goroutines có thể chạy trên 1 OS thread
    resp, _ := http.Get("https://api.example.com/data")
    // Goroutine tự động yield khi chờ network I/O
}()

// ✅ SỬ DỤNG: Nhiều goroutines (không giới hạn bởi CPU cores)
for i := 0; i < 1000; i++ {
    go func(id int) {
        // Mỗi goroutine có thể chờ I/O độc lập
        fetchData(fmt.Sprintf("https://api.example.com/data/%d", id))
    }(i)
}
```

**Tối ưu cho IO-Bound:**
- Nhiều goroutines (không giới hạn bởi CPU cores)
- KHÔNG sử dụng `runtime.LockOSThread()` (lãng phí thread)
- Go scheduler tự động preempt khi chờ I/O
- Connection pooling để tái sử dụng connections

### 2.3 So Sánh CPU-Bound vs IO-Bound

| Đặc điểm | CPU-Bound | IO-Bound |
|----------|-----------|----------|
| **Tài nguyên giới hạn** | CPU cores | I/O bandwidth |
| **CPU utilization** | 80-100% | 10-30% |
| **Blocking operations** | Ít | Nhiều |
| **Số workers/goroutines** | ⚠️ **≤ CPU cores** (quan trọng!) | ✅ Không giới hạn |
| **Thread pinning** | ✅ Cần thiết (native code) | ❌ Không cần |
| **Goroutine scheduling** | Cooperative (có thể preempt) | Tự động yield khi I/O |
| **Ví dụ** | LLM inference, crypto, ML | HTTP, DB, file I/O |

**⚠️ LƯU Ý QUAN TRỌNG về số lượng goroutines:**

```go
// ❌ SAI: CPU-Bound với quá nhiều goroutines
for i := 0; i < 1000; i++ {
    go cpuIntensiveTask()  // 1000 goroutines cạnh tranh 8 CPU cores
}
// Kết quả: Context switching overhead, cache thrashing, chậm hơn!

// ✅ ĐÚNG: CPU-Bound với số goroutines = CPU cores
for i := 0; i < runtime.NumCPU(); i++ {
    go cpuIntensiveTask()  // 8 goroutines cho 8 CPU cores
}
// Hoặc dùng worker pool
pool := compute.NewComputePool(ctx, handler, compute.Config{
    Workers: runtime.NumCPU(),  // Giới hạn số workers
})

// ✅ ĐÚNG: IO-Bound với nhiều goroutines OK
for i := 0; i < 1000; i++ {
    go func() {
        http.Get("...")  // CPU idle khi chờ I/O, nhiều goroutines OK
    }()
}
```

**Tại sao CPU-Bound không nên quá nhiều goroutines?**

1. **GOMAXPROCS giới hạn số OS threads:**
   - Default: `GOMAXPROCS = CPU cores` (ví dụ: 8 cores = 8 threads)
   - Chỉ có 8 OS threads để chạy goroutines
   - 1000 goroutines CPU-bound = 992 goroutines chờ, overhead cao

2. **Context switching overhead:**
   - Go scheduler phải preempt liên tục
   - Mỗi lần switch tốn thời gian
   - Tích lũy = overhead đáng kể

3. **Cache thrashing:**
   - CPU cache bị đẩy ra liên tục
   - Cache miss rate tăng cao
   - Memory access chậm hơn

4. **Không cải thiện throughput:**
   - 8 CPU cores = tối đa 8 tasks song song
   - 1000 goroutines ≠ 1000x nhanh hơn
   - Chỉ làm tăng overhead, không có lợi

### 2.4 Mixed Workloads (Tác Vụ Hỗn Hợp)

**Định nghĩa:**
- Kết hợp cả CPU-bound và IO-bound operations
- Ví dụ: API endpoint nhận request (IO), xử lý dữ liệu (CPU), lưu database (IO)

**Ví dụ Mixed Workload:**
```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    // 1. IO-Bound: Đọc request body
    body, _ := io.ReadAll(r.Body)  // Chờ network I/O
    
    // 2. CPU-Bound: Parse và validate JSON
    var data RequestData
    json.Unmarshal(body, &data)  // CPU-intensive parsing
    
    // 3. CPU-Bound: Xử lý business logic
    result := processData(data)  // CPU-intensive computation
    
    // 4. IO-Bound: Lưu vào database
    db.Save(result)  // Chờ database I/O
    
    // 5. IO-Bound: Trả response
    json.NewEncoder(w).Encode(result)  // Chờ network I/O
}
```

**Thread Management cho Mixed Workloads:**
```go
// ✅ SỬ DỤNG: EventLoop cho IO-bound, WorkerPool cho CPU-bound
type Handler struct {
    eventLoop *eventloop.EventLoopGroup  // Cho IO operations
    cpuPool   *compute.ComputePool       // Cho CPU-bound work
}

func (h *Handler) HandleRequest(ctx context.Context, req *Request) {
    // IO-bound: Đọc từ network (event loop)
    data := h.readRequest(req)
    
    // CPU-bound: Xử lý (worker pool)
    future, _ := h.cpuPool.Submit(ctx, req.Key, data)
    result, _ := future.Get(ctx)
    
    // IO-bound: Ghi response (event loop)
    h.writeResponse(result)
}
```

### 2.5 Quyết Định: CPU-Bound hay IO-Bound?

**Câu hỏi để xác định:**

1. **CPU có idle không?**
   - ✅ CPU idle → IO-Bound
   - ❌ CPU busy → CPU-Bound

2. **Có blocking operations không?**
   - ✅ Nhiều blocking (network, file, DB) → IO-Bound
   - ❌ Ít blocking → CPU-Bound

3. **Tác vụ chủ yếu làm gì?**
   - Tính toán, xử lý dữ liệu → CPU-Bound
   - Đọc/ghi file, network calls → IO-Bound

4. **Số lượng goroutines nên dùng?**
   - ⚠️ CPU-Bound: **≤ CPU cores** (quan trọng!)
   - ✅ IO-Bound: Không giới hạn (nhiều goroutines OK)

**Ví dụ thực tế trong Fluxor:**

```go
// CPU-Bound: LLM Inference
// pkg/core/compute/pool.go
func (w *Worker[T]) Start() error {
    go func() {
        runtime.LockOSThread()  // ✅ Pin thread cho native code
        defer runtime.UnlockOSThread()
        
        // CPU-bound: Load model, inference
        client, _ := NewClient(w.config)
        for job := range w.jobChan {
            result := client.Chat(job.Context, job.Payload)  // CPU-intensive
        }
    }()
}

// IO-Bound: HTTP Handler
// pkg/web/http_handler.go
func (h *HTTPHandler) HandleRequest(w http.ResponseWriter, r *http.Request) {
    // ✅ KHÔNG pin thread - Go scheduler tự động yield khi I/O
    body, _ := io.ReadAll(r.Body)  // Chờ network I/O
    
    // Route đến event loop (IO-bound)
    h.eventLoop.Dispatch(ctx, event)  // Nhiều goroutines OK
}
```

## 3. Thread Management Strategies

### 2.1 Default (Cooperative Scheduling)

```go
// Normal goroutine - can be preempted
go func() {
    for i := 0; i < 1000000; i++ {
        // CPU-bound work
        // Go scheduler can preempt at function calls, channel ops
    }
}()
```

**Pros:**
- Efficient context switching
- Low overhead
- Good for most workloads

**Cons:**
- Can be preempted (not ideal for CPU-bound with native threads)
- No CPU affinity guarantee

### 2.2 OS Thread Pinning (runtime.LockOSThread)

```go
// Pin goroutine to OS thread (no preemption)
go func() {
    runtime.LockOSThread()  // Pin to current OS thread
    defer runtime.UnlockOSThread()
    
    // This goroutine will run on dedicated OS thread
    // Go scheduler cannot move it to another thread
    // Critical for native code (CGO, llama.cpp)
    
    // CPU-bound work with native threads
    nativeFunction()  // llama.cpp spawns native threads
}()
```

**When to Use:**
- **CGO calls** that spawn native threads (llama.cpp, FFmpeg)
- **CPU affinity** required (NUMA, cache locality)
- **Real-time** constraints (low latency)

**Example from Compute Framework:**

```go
// pkg/core/compute/pool.go
func (w *Worker[T]) Start() error {
    w.wg.Add(1)
    go func() {
        // Pin OS thread for CPU-bound work
        runtime.LockOSThread()
        defer runtime.UnlockOSThread()
        defer w.wg.Done()
        
        // Load model in this thread
        client, err := NewClient(w.config)
        // ... process jobs
    }()
}
```

### 2.3 Thread Affinity (Advanced)

```go
import (
    "runtime"
    "syscall"
)

// Set CPU affinity for current thread (Linux/Mac)
func setCPUAffinity(cpuID int) error {
    var mask syscall.CPUSet
    mask.Set(cpuID)
    return syscall.SchedSetaffinity(0, &mask)
}

// Use in pinned goroutine
go func() {
    runtime.LockOSThread()
    defer runtime.UnlockOSThread()
    
    setCPUAffinity(0)  // Pin to CPU 0
    // Work on CPU 0
}()
```

## 3. State Management Patterns

### 3.1 Thread-Local State (Per-Worker State)

```go
// Each worker has its own state
type Worker struct {
    id      int
    state   *WorkerState  // Thread-local state
    mu      sync.RWMutex
}

type WorkerState struct {
    model   *Model       // Loaded model
    cache   *Cache       // KV cache
    counter int64        // Per-worker counter
}

// Access pattern: Only from worker's goroutine
func (w *Worker) processJob(job Job) {
    // No locking needed - only this goroutine accesses state
    w.state.cache.Get(key)
    w.state.counter++
}
```

**Benefits:**
- **No contention**: Each worker has isolated state
- **No locking**: Single goroutine access
- **Cache locality**: State stays in CPU cache

### 3.2 Shared State (With Synchronization)

```go
type SharedState struct {
    mu      sync.RWMutex
    data    map[string]interface{}
    counter int64
}

// Read (multiple readers)
func (s *SharedState) Get(key string) interface{} {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.data[key]
}

// Write (single writer)
func (s *SharedState) Set(key string, value interface{}) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.data[key] = value
}

// Atomic operations
func (s *SharedState) Increment() {
    atomic.AddInt64(&s.counter, 1)
}
```

**When to Use:**
- **Shared resources**: Database connections, global cache
- **Coordination**: Worker coordination, statistics

### 3.3 Immutable State (Functional Style)

```go
type State struct {
    config  Config       // Immutable
    version int64        // Version number
}

// Create new state (immutable update)
func (s *State) UpdateConfig(newConfig Config) *State {
    return &State{
        config:  newConfig,
        version: s.version + 1,
    }
}

// No locking needed - read-only access
func (s *State) GetConfig() Config {
    return s.config  // Safe concurrent read
}
```

**Benefits:**
- **Lock-free reads**: No synchronization needed
- **Thread-safe**: Immutable = safe for concurrent access

## 4. Compute Framework Pattern

### 4.1 Worker Pool Architecture

```go
// pkg/core/compute/pool.go

type Worker[T any] struct {
    id      int
    jobChan chan *Job[T]
    handler func(context.Context, interface{}) (interface{}, error)
    
    // Thread-local state (no locking needed)
    stats   WorkerStats  // Per-worker statistics
    
    // Shared state (with locking)
    mu      sync.RWMutex
    running int32        // Atomic flag
}

type ComputePool[T any] struct {
    workers []*Worker[T]  // Each worker = 1 OS thread (pinned)
    jobChan chan *Job[T]  // Shared job queue
    
    // Shared state (with locking)
    mu      sync.RWMutex
    running int32         // Atomic flag
    stats   PoolStats     // Aggregated stats
}
```

### 4.2 State Access Patterns

```go
// Pattern 1: Thread-local (no locking)
func (w *Worker[T]) processJob(job *Job[T]) {
    // Access thread-local state (no lock)
    startTime := time.Now()
    result, err := w.handler(job.Context, job.Payload)
    
    // Update thread-local stats (no lock)
    w.stats.ProcessedJobs++
    w.stats.TotalLatency += time.Since(startTime)
}

// Pattern 2: Shared read (RWMutex)
func (p *ComputePool[T]) Stats() PoolStats {
    p.mu.RLock()  // Multiple readers allowed
    defer p.mu.RUnlock()
    
    // Aggregate from workers
    for _, worker := range p.workers {
        // Read worker stats (thread-safe)
        stats := worker.Stats()  // Worker has its own lock
        // Aggregate...
    }
}

// Pattern 3: Atomic operations
func (p *ComputePool[T]) Submit(ctx context.Context, key string, payload T) (*Future[T], error) {
    if atomic.LoadInt32(&p.running) == 0 {  // Lock-free read
        return nil, fmt.Errorf("pool not started")
    }
    
    jobID := fmt.Sprintf("job-%d", atomic.AddInt64(&p.jobIDGen, 1))  // Atomic increment
    // ...
}
```

## 5. Best Practices

### 5.1 Thread Management

```go
// ✅ DO: Pin OS threads for CPU-bound native code
go func() {
    runtime.LockOSThread()
    defer runtime.UnlockOSThread()
    nativeCPUWork()
}()

// ❌ DON'T: Pin threads for IO-bound work
go func() {
    runtime.LockOSThread()  // Wasteful - blocks thread during IO
    defer runtime.UnlockOSThread()
    http.Get("...")  // Thread blocked during IO
}()

// ✅ DO: Use default scheduling for IO-bound
go func() {
    http.Get("...")  // Go scheduler can preempt during IO
}()

// ⚠️ QUAN TRỌNG: Không nên spawn quá nhiều goroutines cho CPU-bound
// ❌ DON'T: Quá nhiều goroutines cho CPU-bound work
for i := 0; i < 1000; i++ {
    go func() {
        cpuIntensiveTask()  // 1000 goroutines cạnh tranh 8 CPU cores
    }()
}
// Vấn đề: Context switching overhead, cache thrashing, chậm hơn!

// ✅ DO: Giới hạn số goroutines = CPU cores cho CPU-bound
for i := 0; i < runtime.NumCPU(); i++ {
    go func() {
        cpuIntensiveTask()  // 8 goroutines cho 8 CPU cores
    }()
}
// Hoặc tốt hơn: Dùng worker pool
pool := compute.NewComputePool(ctx, handler, compute.Config{
    Workers: runtime.NumCPU(),  // Giới hạn số workers
})

// ✅ DO: Nhiều goroutines OK cho IO-bound
for i := 0; i < 1000; i++ {
    go func() {
        http.Get("...")  // CPU idle khi chờ I/O, nhiều goroutines OK
    }()
}
```

### 5.2 State Management

```go
// ✅ DO: Thread-local state for per-worker data
type Worker struct {
    cache *Cache  // Each worker has its own cache
}

// ❌ DON'T: Shared mutable state without locking
type Worker struct {
    cache *Cache  // Shared - needs locking!
}

// ✅ DO: Immutable shared state
type Config struct {
    ModelPath string  // Immutable - safe concurrent read
}

// ✅ DO: Atomic operations for counters/flags
var counter int64
atomic.AddInt64(&counter, 1)  // Lock-free
```

### 5.3 Lock Granularity

```go
// ✅ DO: Fine-grained locking
type Pool struct {
    mu      sync.RWMutex
    workers []*Worker
    stats   Stats
}

func (p *Pool) GetStats() Stats {
    p.mu.RLock()  // Only lock what you need
    defer p.mu.RUnlock()
    return p.stats
}

// ❌ DON'T: Coarse-grained locking
type Pool struct {
    mu sync.Mutex  // Locks everything
    // ...
}

func (p *Pool) GetStats() Stats {
    p.mu.Lock()  // Blocks all operations
    defer p.mu.Unlock()
    return p.stats
}
```

## 6. LLM-Specific Considerations

### 6.1 Model Loading (Per-Worker)

```go
// Each worker loads its own model instance
type LLMWorker struct {
    client *LlamaClient  // Thread-local (no sharing)
}

func (w *LLMWorker) Start() {
    go func() {
        runtime.LockOSThread()  // Pin thread
        
        // Load model in this thread
        w.client = NewClient(config)  // Thread-local
        
        // Process jobs (no locking needed)
        for job := range w.jobChan {
            w.client.Chat(job.Context, job.Request)  // Uses thread-local client
        }
    }()
}
```

**Why:**
- **Memory isolation**: Each model instance in separate memory
- **No contention**: No shared model access
- **Cache locality**: Model weights stay in CPU cache

### 6.2 KV Cache (Session Affinity)

```go
// Route jobs by session key to same worker
func (p *ComputePool[T]) Submit(ctx context.Context, key string, payload T) {
    // Hash key to worker index
    workerIdx := hash(key) % len(p.workers)
    worker := p.workers[workerIdx]
    
    // Job goes to same worker (session affinity)
    worker.jobChan <- job
}

// Worker maintains KV cache for its sessions
type LLMWorker struct {
    kvCache map[string]*KVCache  // Thread-local cache
}

func (w *LLMWorker) processJob(job *Job) {
    // Get cache for session (thread-local, no lock)
    cache := w.kvCache[job.Key]
    result := w.client.Chat(job.Context, job.Request, cache)
}
```

**Benefits:**
- **KV cache reuse**: Same session = same worker = cache hit
- **Lower latency**: No cache miss
- **No locking**: Thread-local cache access

## 7. Monitoring & Debugging

### 7.1 Thread Count

```go
import "runtime"

// Get number of OS threads
numThreads := runtime.GOMAXPROCS(0)

// Get number of goroutines
numGoroutines := runtime.NumGoroutine()

// Get thread creation stats
var m runtime.MemStats
runtime.ReadMemStats(&m)
// m.NumGC, m.NumForcedGC, etc.
```

### 7.2 State Inspection

```go
// Thread-local state (per worker)
func (w *Worker) Stats() WorkerStats {
    w.mu.RLock()
    defer w.mu.RUnlock()
    return w.stats  // Copy (safe)
}

// Shared state (pool-level)
func (p *ComputePool) Stats() PoolStats {
    p.mu.RLock()
    defer p.mu.RUnlock()
    
    // Aggregate from all workers
    workersStats := make([]WorkerStats, len(p.workers))
    for i, w := range p.workers {
        workersStats[i] = w.Stats()  // Each worker has its own lock
    }
    
    return PoolStats{
        Workers:      len(p.workers),
        QueueLength:  len(p.jobChan),
        WorkersStats: workersStats,
    }
}
```

### 7.3 Debugging Tools

```bash
# Go runtime trace
go tool trace trace.out

# CPU profiling
go tool pprof cpu.prof

# Memory profiling
go tool pprof mem.prof

# Goroutine dump
kill -QUIT <pid>  # Dumps all goroutines to stderr
```

## 8. Summary

### Thread Management Rules

1. **Default**: Use Go scheduler (no LockOSThread) for most cases
2. **CPU-bound native**: Use `runtime.LockOSThread()` for CGO/native threads
3. **GOMAXPROCS**: Default (CPU cores) is usually optimal
4. **⚠️ QUAN TRỌNG: CPU-bound goroutines**: **≤ CPU cores** (không nên quá nhiều!)
   - ❌ KHÔNG spawn 1000 goroutines cho CPU-bound work
   - ✅ Sử dụng worker pool với số workers = CPU cores
   - ✅ IO-bound: Nhiều goroutines OK (không giới hạn)

### State Management Rules

1. **Thread-local**: Per-worker state (no locking)
2. **Shared read-only**: Immutable state (no locking)
3. **Shared mutable**: Use locks (RWMutex for read-heavy)
4. **Counters/flags**: Use atomic operations (lock-free)

### Compute Framework Pattern

```
EventLoop (goroutine, can be preempted)
   |
   | Submit job
   v
ComputePool (shared job queue)
   |
   | Route by key
   v
Workers (pinned OS threads, thread-local state)
   |
   | Process job
   v
Result → Future → EventLoop
```

**Key Principles:**
- EventLoop: Cooperative scheduling (preemptible)
- Workers: OS thread pinning (non-preemptible)
- State: Thread-local per worker (no contention)
- Routing: Key-based for locality (KV cache)

