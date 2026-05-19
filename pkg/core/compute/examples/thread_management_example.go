package main

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/compute"
)

// Example 1: Thread-local state (no locking)
type ThreadLocalWorker struct {
	id      int
	counter int64                  // Thread-local (no locking needed)
	cache   map[string]interface{} // Thread-local cache
}

func (w *ThreadLocalWorker) Process() {
	// No locking - only this goroutine accesses state
	w.counter++
	w.cache["key"] = w.counter
}

// Example 2: Shared state (with locking)
type SharedState struct {
	mu    sync.RWMutex
	data  map[string]interface{}
	count int64
}

func (s *SharedState) Get(key string) interface{} {
	s.mu.RLock() // Multiple readers allowed
	defer s.mu.RUnlock()
	return s.data[key]
}

func (s *SharedState) Set(key string, value interface{}) {
	s.mu.Lock() // Single writer
	defer s.mu.Unlock()
	s.data[key] = value
}

func (s *SharedState) Increment() {
	atomic.AddInt64(&s.count, 1) // Atomic (lock-free)
}

// Example 3: OS Thread Pinning
func ExampleThreadPinning() {
	fmt.Println("=== Example: OS Thread Pinning ===")

	// Normal goroutine (can be preempted)
	go func() {
		fmt.Printf("Goroutine 1: Thread ID = %d\n", getThreadID())
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("Goroutine 1: Thread ID = %d (may change)\n", getThreadID())
	}()

	// Pinned goroutine (stays on same OS thread)
	go func() {
		runtime.LockOSThread() // Pin to OS thread
		defer runtime.UnlockOSThread()

		threadID := getThreadID()
		fmt.Printf("Goroutine 2: Thread ID = %d (pinned)\n", threadID)

		time.Sleep(100 * time.Millisecond)

		// Still on same thread
		if getThreadID() == threadID {
			fmt.Printf("Goroutine 2: Thread ID = %d (still same)\n", threadID)
		}
	}()

	time.Sleep(200 * time.Millisecond)
}

// Example 4: Compute Pool with Thread-Local State
func ExampleComputePoolState() {
	fmt.Println("\n=== Example: Compute Pool State Management ===")

	ctx := context.Background()

	// Handler with thread-local state
	type WorkerState struct {
		processed int64
		lastJob   time.Time
	}

	workerStates := make(map[int]*WorkerState)
	var statesMu sync.RWMutex

	handler := func(ctx context.Context, payload interface{}) (interface{}, error) {
		// Get current goroutine ID (approximate)
		goroutineID := getGoroutineID()

		// Get or create thread-local state
		statesMu.Lock()
		state, exists := workerStates[goroutineID]
		if !exists {
			state = &WorkerState{}
			workerStates[goroutineID] = state
		}
		statesMu.Unlock()

		// Update thread-local state (no locking needed after retrieval)
		state.processed++
		state.lastJob = time.Now()

		// Process job
		val := payload.(int)
		return val * 2, nil
	}

	config := compute.DefaultConfig()
	config.Workers = 2

	pool, err := compute.NewComputePool[int](ctx, handler, config)
	if err != nil {
		panic(err)
	}

	if err := pool.Start(); err != nil {
		panic(err)
	}
	defer pool.Stop(ctx)

	// Submit jobs
	for i := 0; i < 10; i++ {
		_, _ = pool.Submit(ctx, "", i)
	}

	time.Sleep(500 * time.Millisecond)

	// Print thread-local stats
	statesMu.RLock()
	fmt.Printf("Worker states: %d workers\n", len(workerStates))
	for id, state := range workerStates {
		fmt.Printf("  Worker %d: processed=%d, lastJob=%v\n",
			id, state.processed, state.lastJob)
	}
	statesMu.RUnlock()
}

// Example 5: Atomic Operations (Lock-Free)
func ExampleAtomicOperations() {
	fmt.Println("\n=== Example: Atomic Operations ===")

	var counter int64
	var wg sync.WaitGroup

	// Multiple goroutines incrementing counter
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				atomic.AddInt64(&counter, 1) // Lock-free
			}
		}()
	}

	wg.Wait()
	fmt.Printf("Counter: %d (expected: 10000)\n", atomic.LoadInt64(&counter))
}

// Example 6: GOMAXPROCS Management
func ExampleGOMAXPROCS() {
	fmt.Println("\n=== Example: GOMAXPROCS Management ===")

	// Get current setting
	current := runtime.GOMAXPROCS(0)
	fmt.Printf("Current GOMAXPROCS: %d\n", current)
	fmt.Printf("CPU cores: %d\n", runtime.NumCPU())

	// Set new value
	runtime.GOMAXPROCS(4)
	fmt.Printf("New GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))

	// Restore
	runtime.GOMAXPROCS(current)
	fmt.Printf("Restored GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
}

// Helper functions
func getThreadID() int {
	// Approximate thread ID (not exact, but good enough for demo)
	return int(time.Now().UnixNano() % 1000)
}

func getGoroutineID() int {
	// Approximate goroutine ID
	buf := make([]byte, 64)
	n := runtime.Stack(buf, false)
	// Parse stack trace to get goroutine ID (simplified)
	return n % 1000
}

// Example 7: CPU-Bound Workload
func ExampleCPUBound() {
	fmt.Println("\n=== Example: CPU-Bound Workload ===")

	// CPU-bound: Tính toán số Pi (CPU-intensive)
	cpuBoundTask := func(n int) float64 {
		sum := 0.0
		for i := 0; i < n; i++ {
			sum += 1.0 / float64(i*2+1)
		}
		return sum * 4
	}

	// ✅ SỬ DỤNG: Worker pool với số lượng = CPU cores
	ctx := context.Background()
	handler := func(ctx context.Context, payload interface{}) (interface{}, error) {
		n := payload.(int)
		result := cpuBoundTask(n) // CPU-intensive computation
		return result, nil
	}

	config := compute.DefaultConfig()
	config.Workers = runtime.NumCPU() // 1 worker per CPU core

	pool, _ := compute.NewComputePool[int](ctx, handler, config)
	pool.Start()
	defer pool.Stop(ctx)

	// Submit CPU-bound jobs
	start := time.Now()
	for i := 0; i < 10; i++ {
		pool.Submit(ctx, "", 1000000)
	}
	time.Sleep(2 * time.Second)

	fmt.Printf("CPU-bound: Completed in %v\n", time.Since(start))
	fmt.Printf("Workers: %d (optimal for CPU-bound)\n", config.Workers)
}

// Example 7b: ❌ SAI - Quá nhiều goroutines cho CPU-Bound
func ExampleCPUBoundWrong() {
	fmt.Println("\n=== Example: ❌ SAI - Quá nhiều goroutines cho CPU-Bound ===")

	cpuBoundTask := func(n int) float64 {
		sum := 0.0
		for i := 0; i < n; i++ {
			sum += 1.0 / float64(i*2+1)
		}
		return sum * 4
	}

	// ❌ SAI: Spawn quá nhiều goroutines cho CPU-bound work
	numCPU := runtime.NumCPU()
	fmt.Printf("CPU cores: %d\n", numCPU)

	var wg sync.WaitGroup
	start := time.Now()

	// ❌ 1000 goroutines cạnh tranh cho 8 CPU cores
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cpuBoundTask(1000000) // CPU-intensive
		}()
	}

	wg.Wait()
	fmt.Printf("❌ Wrong approach: 1000 goroutines completed in %v\n", time.Since(start))
	fmt.Println("Problems:")
	fmt.Println("  - Context switching overhead")
	fmt.Println("  - Cache thrashing")
	fmt.Println("  - Preemption overhead")
	fmt.Println("  - No throughput improvement (CPU already busy)")

	// ✅ ĐÚNG: Chỉ spawn số goroutines = CPU cores
	start2 := time.Now()
	var wg2 sync.WaitGroup

	for i := 0; i < numCPU; i++ {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			cpuBoundTask(1000000)
		}()
	}

	wg2.Wait()
	fmt.Printf("✅ Correct approach: %d goroutines completed in %v\n", numCPU, time.Since(start2))
	fmt.Println("Benefits:")
	fmt.Println("  - No context switching overhead")
	fmt.Println("  - Better cache locality")
	fmt.Println("  - Optimal CPU utilization")
}

// Example 8: IO-Bound Workload
func ExampleIOBound() {
	fmt.Println("\n=== Example: IO-Bound Workload ===")

	// IO-bound: Simulate HTTP request (network I/O)
	ioBoundTask := func(url string) (string, error) {
		// Simulate network latency
		time.Sleep(100 * time.Millisecond) // Chờ network I/O
		return fmt.Sprintf("Response from %s", url), nil
	}

	// ✅ SỬ DỤNG: Nhiều goroutines (không giới hạn bởi CPU cores)
	var wg sync.WaitGroup
	start := time.Now()

	// Có thể spawn nhiều goroutines vì CPU idle khi chờ I/O
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			url := fmt.Sprintf("https://api.example.com/data/%d", id)
			ioBoundTask(url) // CPU chờ network I/O
		}(i)
	}

	wg.Wait()
	fmt.Printf("IO-bound: Completed 100 requests in %v\n", time.Since(start))
	fmt.Printf("Goroutines: 100 (OK for IO-bound, CPU idle during I/O)\n")
}

// Example 9: Mixed Workload (CPU + IO)
func ExampleMixedWorkload() {
	fmt.Println("\n=== Example: Mixed Workload (CPU + IO) ===")

	// Mixed: API handler với cả CPU và IO operations
	handleRequest := func(id int) {
		// 1. IO-Bound: Đọc từ database (simulate)
		time.Sleep(50 * time.Millisecond) // Chờ DB I/O
		data := fmt.Sprintf("data-%d", id)

		// 2. CPU-Bound: Xử lý dữ liệu
		result := ""
		for i := 0; i < 10000; i++ {
			result += data // CPU-intensive string processing
		}

		// 3. IO-Bound: Ghi vào database (simulate)
		time.Sleep(50 * time.Millisecond) // Chờ DB I/O

		_ = result
	}

	// ✅ SỬ DỤNG: EventLoop cho IO, WorkerPool cho CPU
	start := time.Now()
	var wg sync.WaitGroup

	// IO-bound: Nhiều goroutines OK
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			handleRequest(id) // Mixed workload
		}(i)
	}

	wg.Wait()
	fmt.Printf("Mixed workload: Completed 20 requests in %v\n", time.Since(start))
	fmt.Println("Strategy: EventLoop (IO) + WorkerPool (CPU)")
}

// Example 10: So Sánh CPU-Bound vs IO-Bound
func ExampleCompareCPUBoundIOBound() {
	fmt.Println("\n=== Example: So Sánh CPU-Bound vs IO-Bound ===")

	fmt.Println("\n1. CPU-Bound Characteristics:")
	fmt.Println("   - CPU utilization: 80-100%")
	fmt.Println("   - Workers: ≤ CPU cores")
	fmt.Println("   - Thread pinning: ✅ Cần thiết (native code)")
	fmt.Println("   - Example: LLM inference, crypto, ML")

	fmt.Println("\n2. IO-Bound Characteristics:")
	fmt.Println("   - CPU utilization: 10-30%")
	fmt.Println("   - Workers: Không giới hạn")
	fmt.Println("   - Thread pinning: ❌ Không cần")
	fmt.Println("   - Example: HTTP, DB, file I/O")

	fmt.Println("\n3. Decision Tree:")
	fmt.Println("   - CPU idle? → IO-Bound")
	fmt.Println("   - CPU busy? → CPU-Bound")
	fmt.Println("   - Nhiều blocking? → IO-Bound")
	fmt.Println("   - Ít blocking? → CPU-Bound")
}
func main() {
	fmt.Println("Go Runtime Thread & State Management Examples")

	ExampleGOMAXPROCS()
	ExampleThreadPinning()
	ExampleAtomicOperations()
	ExampleComputePoolState()
	ExampleCPUBound()
	ExampleCPUBoundWrong() // Ví dụ về cách SAI
	ExampleIOBound()
	ExampleMixedWorkload()
	ExampleCompareCPUBoundIOBound()

	fmt.Println("\n=== Summary ===")
	fmt.Println("1. Use runtime.LockOSThread() for CPU-bound native code")
	fmt.Println("2. Use thread-local state to avoid locking")
	fmt.Println("3. Use atomic operations for counters/flags")
	fmt.Println("4. Use RWMutex for shared read-heavy state")
	fmt.Println("5. GOMAXPROCS defaults to CPU cores (usually optimal)")
	fmt.Println("6. CPU-Bound: Workers ≤ CPU cores, pin threads")
	fmt.Println("7. IO-Bound: Many goroutines OK, no thread pinning")
	fmt.Println("8. ⚠️ QUAN TRỌNG: KHÔNG spawn quá nhiều goroutines cho CPU-bound!")
	fmt.Println("   - 1000 goroutines CPU-bound ≠ 1000x nhanh hơn")
	fmt.Println("   - Chỉ làm tăng overhead, không có lợi")
	fmt.Println("   - Sử dụng worker pool với số workers = CPU cores")
}
