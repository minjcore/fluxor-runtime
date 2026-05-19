// Package osthread provides utilities for managing OS threads (system-level threads) concurrently.
//
// The osthread package provides thread pools, thread pinning, and thread-local storage for
// CPU-bound workloads. It supports pinning goroutines to OS threads using runtime.LockOSThread()
// for use cases requiring CPU affinity, cache locality, or real-time constraints.
//
// Important Distinction: OS Threads vs Goroutines
//
// OS Threads (what this package manages):
//   - System-level threads managed by the operating system
//   - Heavyweight (1-2MB stack per thread)
//   - Limited by GOMAXPROCS (typically = CPU cores)
//   - Pinned using runtime.LockOSThread()
//   - Used for CPU-bound native code (CGO, llama.cpp)
//
// Goroutines (Go's concurrency primitives):
//   - Lightweight threads managed by Go runtime
//   - Small stack (2KB initially, grows as needed)
//   - Can have millions of goroutines
//   - Scheduled by Go runtime onto OS threads
//   - Used for general concurrent work (IO-bound, event loops)
//
// This package manages OS threads, not goroutines. It pins goroutines to OS threads
// for CPU-bound work.
//
// Usage:
//
//	// Thread Pool
//	ctx := context.Background()
//	pool := osthread.NewThreadPool(ctx, osthread.DefaultConfig())
//	if err := pool.Start(); err != nil {
//	    log.Fatal(err)
//	}
//	defer pool.Stop(context.Background())
//
//	task := osthread.NewNamedTask("my-task", func(ctx context.Context) error {
//	    // CPU-bound work on pinned OS thread
//	    return doCPUWork()
//	})
//	pool.Submit(ctx, task)
//
//	// Thread Pinning
//	unpin := osthread.PinCurrentThread()
//	defer unpin()
//	// CPU-bound work on pinned thread
//
//	// Thread-Local Storage
//	var cache = osthread.NewThreadLocal[*MyCache]()
//	cache.Set(&MyCache{data: make(map[string]interface{})})
//	myCache := cache.Get()
//
// Use cases:
//   - CPU-bound native code (CGO, llama.cpp, FFmpeg)
//   - Work requiring CPU affinity (NUMA, cache locality)
//   - Real-time constraints (low latency)
//   - Thread-local state management on OS threads
//
// Do NOT use for:
//   - IO-bound work (HTTP, database, file I/O)
//   - Work that benefits from Go scheduler preemption
//   - General-purpose goroutines
package osthread
