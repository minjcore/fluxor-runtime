package osthread

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// ThreadPool manages a pool of OS threads (system-level threads) for CPU-bound concurrent work.
// Each worker is a goroutine pinned to an OS thread using runtime.LockOSThread().
//
// Important: This manages OS threads, not goroutines.
// - OS Threads: System-level threads (limited by GOMAXPROCS, typically = CPU cores)
// - Goroutines: Go's lightweight concurrency primitives (can have millions)
//
// This pool pins goroutines to OS threads for CPU-bound work.
//
// Use cases:
//   - CPU-bound native code (CGO, llama.cpp, FFmpeg)
//   - Work requiring CPU affinity (NUMA, cache locality)
//   - Real-time constraints (low latency)
//
// ThreadPool ensures:
//   - Each worker (goroutine) runs on a dedicated OS thread (pinned)
//   - Bounded queue for backpressure handling
//   - Graceful shutdown with context cancellation
//   - Thread-safe operations
type ThreadPool interface {
	// Start starts all worker threads
	Start() error

	// Stop stops all worker threads gracefully
	// Waits for all in-flight tasks to complete or context timeout
	Stop(ctx context.Context) error

	// Submit submits a task to the thread pool
	// Returns error if pool is not started or queue is full (backpressure)
	Submit(ctx context.Context, task Task) error

	// Execute executes a function on a pinned OS thread
	// Returns the result or error
	Execute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error)

	// IsRunning returns whether the pool is running
	IsRunning() bool

	// Stats returns current pool statistics
	Stats() PoolStats
}

// Task represents a unit of work to be executed on a pinned OS thread
type Task interface {
	// Execute performs the work
	// This runs on a pinned OS thread (runtime.LockOSThread)
	Execute(ctx context.Context) error

	// Name returns a human-readable name for the task
	Name() string
}

// TaskFunc is a function type that implements Task
type TaskFunc func(ctx context.Context) error

// Execute implements Task interface
func (f TaskFunc) Execute(ctx context.Context) error {
	return f(ctx)
}

// Name returns a default name for TaskFunc
func (f TaskFunc) Name() string {
	return "TaskFunc"
}

// NamedTask wraps a TaskFunc with a custom name
type NamedTask struct {
	name string
	task TaskFunc
}

// NewNamedTask creates a new NamedTask
func NewNamedTask(name string, task TaskFunc) *NamedTask {
	failfast.NotEmpty(name, "task name")
	failfast.NotNil(task, "task")
	return &NamedTask{
		name: name,
		task: task,
	}
}

// Execute implements Task interface
func (nt *NamedTask) Execute(ctx context.Context) error {
	return nt.task(ctx)
}

// Name returns the task name
func (nt *NamedTask) Name() string {
	return nt.name
}

// PoolStats contains statistics for a thread pool
type PoolStats struct {
	Workers      int           // Number of worker threads
	Running      bool          // Whether pool is running
	QueueLength  int           // Current queue length
	Processed    int64         // Total tasks processed
	Failed       int64         // Total tasks failed
	ActiveTasks  int64         // Currently executing tasks
	AvgLatency   time.Duration // Average task latency
}

// Config configures a ThreadPool
type Config struct {
	// Workers is the number of worker threads (0 = auto: GOMAXPROCS)
	Workers int

	// QueueSize is the maximum queue size (bounded for backpressure)
	QueueSize int

	// SharedQueueSize is the size of the shared queue before worker routing
	// Default: same as QueueSize (small queue - tasks wait for available threads)
	SharedQueueSize int

	// Timeout is the default timeout for task execution
	Timeout time.Duration
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	numCPU := runtime.GOMAXPROCS(0)
	workers := numCPU
	if workers < 1 {
		workers = 1
	}

	queueSize := 1000

	return Config{
		Workers:        workers,
		QueueSize:      queueSize,
		SharedQueueSize: queueSize, // Same size - small queue
		Timeout:        30 * time.Second,
	}
}

// defaultThreadPool implements ThreadPool
// Uses a shared queue before routing to workers for thread reuse
type defaultThreadPool struct {
	config       Config
	workers    []*worker
	sharedQueue chan *taskWrapper // Shared queue before routing
	ctx        context.Context
	cancel     context.CancelFunc
	running    int32
	wg         sync.WaitGroup
	dispatcherWg sync.WaitGroup // WaitGroup for dispatcher goroutine
	mu         sync.RWMutex
	stats      PoolStats
	processed  int64
	failed     int64
	activeTasks int64
	roundRobin int64 // Atomic counter for round-robin routing (fallback)
}

// taskWrapper wraps a task with metadata
type taskWrapper struct {
	task      Task
	ctx       context.Context
	result    chan error
	startTime time.Time
}

// NewThreadPool creates a new ThreadPool with the given configuration
func NewThreadPool(parentCtx context.Context, config Config) ThreadPool {
	failfast.NotNil(parentCtx, "parentCtx")

	// Auto-calculate workers if not specified
	workers := config.Workers
	if workers == 0 {
		numCPU := runtime.GOMAXPROCS(0)
		workers = numCPU
		if workers < 1 {
			workers = 1
		}
	}

	// Set default queue size
	queueSize := config.QueueSize
	if queueSize < 1 {
		queueSize = 1000
	}

	// Set default timeout
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// Calculate shared queue size (default: same as worker queue size - small queue)
	sharedQueueSize := config.SharedQueueSize
	if sharedQueueSize < 1 {
		sharedQueueSize = queueSize // Same size - small queue
	}

	ctx, cancel := context.WithCancel(parentCtx)

	pool := &defaultThreadPool{
		config: Config{
			Workers:        workers,
			QueueSize:      queueSize,
			SharedQueueSize: sharedQueueSize,
			Timeout:        timeout,
		},
		workers:     make([]*worker, 0, workers),
		sharedQueue: make(chan *taskWrapper, sharedQueueSize),
		ctx:         ctx,
		cancel:      cancel,
		stats: PoolStats{
			Workers: workers,
		},
	}

	// Create workers - each with its own dedicated channel (no global shared access)
	for i := 0; i < workers; i++ {
		// Each worker gets its own channel - no global shared state
		workerChan := make(chan *taskWrapper, queueSize)
		w := newWorker(i, workerChan, pool.ctx)
		pool.workers = append(pool.workers, w)
	}

	return pool
}

// startDispatcher starts the dispatcher goroutine that routes tasks from shared queue to workers
// Dispatcher finds first available worker (thread reuse) instead of round-robin
func (p *defaultThreadPool) startDispatcher() {
	p.dispatcherWg.Add(1)
	go func() {
		defer p.dispatcherWg.Done()

		for {
			select {
			case wrapper, ok := <-p.sharedQueue:
				if !ok {
					return // Queue closed
				}

				// Find first available worker (thread reuse)
				p.mu.RLock()
				workers := p.workers
				p.mu.RUnlock()

				if len(workers) == 0 {
					// No workers - send error result
					select {
					case wrapper.result <- fmt.Errorf("no workers available"):
					default:
					}
					continue
				}

				// Try to find available worker (not round-robin)
				assigned := false
				for _, worker := range workers {
					// Check if worker has capacity (queue not full)
					if worker.queueLength() < p.config.QueueSize {
						if worker.submit(wrapper) {
							assigned = true
							break // Thread reuse: assign to first available
						}
					}
				}

				// If no worker available, try round-robin as fallback
				if !assigned {
					workerIdx := int(atomic.AddInt64(&p.roundRobin, 1) % int64(len(workers)))
					worker := workers[workerIdx]
					if !worker.submit(wrapper) {
						// All workers full - send error
						select {
						case wrapper.result <- fmt.Errorf("all workers busy"):
						default:
						}
					}
				}

			case <-p.ctx.Done():
				return
			}
		}
	}()
}

// Start starts all worker threads
func (p *defaultThreadPool) Start() error {
	if !atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		return fmt.Errorf("thread pool is already started")
	}

	p.mu.Lock()
	p.stats.Running = true
	p.mu.Unlock()

	// Start dispatcher first
	p.startDispatcher()

	// Then start all workers
	for i, w := range p.workers {
		if err := w.start(); err != nil {
			// Stop already started workers
			p.Stop(p.ctx)
			return fmt.Errorf("failed to start worker %d: %w", i, err)
		}
	}

	return nil
}

// Stop stops all worker threads gracefully
func (p *defaultThreadPool) Stop(ctx context.Context) error {
	failfast.NotNil(ctx, "ctx")

	if !atomic.CompareAndSwapInt32(&p.running, 1, 0) {
		return nil // Already stopped
	}

	p.cancel()

	// Close shared queue first (stops dispatcher)
	close(p.sharedQueue)

	// Wait for dispatcher to finish
	dispatcherDone := make(chan struct{})
	go func() {
		p.dispatcherWg.Wait()
		close(dispatcherDone)
	}()

	select {
	case <-dispatcherDone:
	case <-ctx.Done():
		return fmt.Errorf("dispatcher stop timeout: %w", ctx.Err())
	}

	// Then close each worker's dedicated channel
	for _, w := range p.workers {
		w.close()
	}

	// Wait for all workers to finish
	done := make(chan struct{})
	go func() {
		for _, w := range p.workers {
			w.stop()
		}
		close(done)
	}()

	select {
	case <-done:
		p.mu.Lock()
		p.stats.Running = false
		p.mu.Unlock()
		return nil
	case <-ctx.Done():
		return fmt.Errorf("stop timeout: %w", ctx.Err())
	}
}

// Submit submits a task to the thread pool
// Tasks are sent to shared queue, then dispatcher routes to first available worker (thread reuse)
func (p *defaultThreadPool) Submit(ctx context.Context, task Task) error {
	failfast.NotNil(ctx, "ctx")
	failfast.NotNil(task, "task")

	if atomic.LoadInt32(&p.running) == 0 {
		return fmt.Errorf("thread pool is not started")
	}

	wrapper := &taskWrapper{
		task:      task,
		ctx:       ctx,
		result:    make(chan error, 1),
		startTime: time.Now(),
	}

	// Submit to shared queue (non-blocking for backpressure)
	select {
	case p.sharedQueue <- wrapper:
		return nil
	case <-p.ctx.Done():
		return p.ctx.Err()
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Shared queue full - backpressure
		return fmt.Errorf("thread pool queue full")
	}
}

// Execute executes a function on a pinned OS thread
func (p *defaultThreadPool) Execute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	failfast.NotNil(ctx, "ctx")
	failfast.NotNil(fn, "fn")

	if atomic.LoadInt32(&p.running) == 0 {
		return nil, fmt.Errorf("thread pool is not started")
	}

	resultChan := make(chan interface{}, 1)
	errChan := make(chan error, 1)

	task := TaskFunc(func(ctx context.Context) error {
		result, err := fn()
		if err != nil {
			errChan <- err
			return err
		}
		resultChan <- result
		return nil
	})

	if err := p.Submit(ctx, task); err != nil {
		return nil, err
	}

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.ctx.Done():
		return nil, fmt.Errorf("thread pool is shutting down")
	}
}

// IsRunning returns whether the pool is running
func (p *defaultThreadPool) IsRunning() bool {
	return atomic.LoadInt32(&p.running) == 1
}

// Stats returns current pool statistics
func (p *defaultThreadPool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := p.stats
	stats.ActiveTasks = atomic.LoadInt64(&p.activeTasks)
	
	// Track shared queue length
	stats.QueueLength = len(p.sharedQueue)

	// Aggregate worker stats
	var totalLatency time.Duration
	var totalProcessed int64
	for _, w := range p.workers {
		wStats := w.getStats()
		totalLatency += wStats.TotalLatency
		totalProcessed += wStats.ProcessedTasks
	}

	stats.Processed = totalProcessed
	if totalProcessed > 0 {
		stats.AvgLatency = totalLatency / time.Duration(totalProcessed)
	}

	return stats
}
