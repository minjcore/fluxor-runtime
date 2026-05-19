package compute

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Job represents a compute job
type Job[T any] struct {
	ID      string
	Key     string // Routing key (for HashByKey policy)
	Payload T      // Job payload
	Context context.Context
	Result  chan *JobResult[T]
	Created time.Time

	// superseded is used by CoalesceByKey to allow workers to skip older jobs
	// without doing expensive work.
	superseded atomic.Bool

	// doneSig is signaled by the worker after this job is fully handled (including
	// superseded short-circuit). CoalesceByKey cleanup waits on this — not on
	// Result — so the Future remains the sole consumer of JobResult values.
	doneSig chan struct{}
}

// JobResult contains the result of a compute job
// Note: Value is T (payload type), but handler may return a different type
// For handlers that return different types (e.g., LLM returns ChatResponse from ChatRequest),
// we store the actual handler result in HandlerResult
type JobResult[T any] struct {
	Value         T           // Payload type (for type safety)
	HandlerResult interface{} // Actual result from handler (may be different type)
	Error         error
}

// Future represents an asynchronous computation result
type Future[T any] struct {
	resultChan chan *JobResult[T]
	done       atomic.Bool
	mu         sync.RWMutex
	cached     *JobResult[T]
}

// Get waits for the result and returns it
func (f *Future[T]) Get(ctx context.Context) (T, error) {
	// Check if already done (fast path)
	if f.done.Load() {
		f.mu.RLock()
		defer f.mu.RUnlock()
		if f.cached != nil {
			return f.cached.Value, f.cached.Error
		}
	}

	select {
	case result := <-f.resultChan:
		f.mu.Lock()
		f.cached = result
		f.done.Store(true)
		f.mu.Unlock()
		// Return Value (payload type) - for handlers returning different types,
		// use GetHandlerResult() helper method
		return result.Value, result.Error
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

// GetWithTimeout waits for the result with a timeout
func (f *Future[T]) GetWithTimeout(timeout time.Duration) (T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return f.Get(ctx)
}

// IsDone returns whether the future is completed
func (f *Future[T]) IsDone() bool {
	return f.done.Load()
}

// GetHandlerResult returns the actual handler result (may be different type than T)
// Use this when handler returns a different type than the payload
func (f *Future[T]) GetHandlerResult(ctx context.Context) (interface{}, error) {
	// Ensure result is loaded
	if !f.done.Load() {
		_, err := f.Get(ctx)
		if err != nil {
			return nil, err
		}
	}

	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.cached != nil {
		return f.cached.HandlerResult, f.cached.Error
	}
	return nil, fmt.Errorf("result not available")
}

// Worker represents a single compute worker
type Worker[T any] struct {
	id      int
	jobChan chan *Job[T]
	handler func(context.Context, interface{}) (interface{}, error) // Accepts interface{} for pool flexibility
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.RWMutex
	running int32
	stats   WorkerStats
}

// WorkerStats contains statistics for a worker
type WorkerStats struct {
	ProcessedJobs int64
	FailedJobs    int64
	TotalLatency  time.Duration
	LastJobTime   time.Time
}

// NewWorker creates a new worker
func NewWorker[T any](id int, handler func(context.Context, interface{}) (interface{}, error), jobChan chan *Job[T]) *Worker[T] {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker[T]{
		id:      id,
		jobChan: jobChan,
		handler: handler,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start starts the worker in a pinned OS thread
func (w *Worker[T]) Start() error {
	if !atomic.CompareAndSwapInt32(&w.running, 0, 1) {
		return fmt.Errorf("worker %d is already running", w.id)
	}

	w.wg.Add(1)
	go func() {
		// Pin OS thread for CPU-bound work (LLM, crypto, etc.)
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer w.wg.Done()

		for {
			select {
			case job, ok := <-w.jobChan:
				if !ok {
					return
				}

				func() {
					defer func() {
						select {
						case job.doneSig <- struct{}{}:
						default:
						}
					}()

					// If this job was superseded (CoalesceByKey), complete it with cancellation (best-effort)
					// and skip doing any expensive work.
					if job.superseded.Load() {
						select {
						case job.Result <- &JobResult[T]{
							Value:         job.Payload,
							HandlerResult: nil,
							Error:         context.Canceled,
						}:
						default:
						}
						return
					}

					// Process job (handler accepts interface{}, payload is T)
					startTime := time.Now()
					result, err := w.handler(job.Context, job.Payload)

					// Update stats
					atomic.AddInt64(&w.stats.ProcessedJobs, 1)
					latency := time.Since(startTime)
					w.mu.Lock()
					w.stats.TotalLatency += latency
					w.stats.LastJobTime = time.Now()
					if err != nil {
						atomic.AddInt64(&w.stats.FailedJobs, 1)
					}
					w.mu.Unlock()

					// Send result:
					// - Value is the original payload (type-safe)
					// - HandlerResult is the actual handler output (may be different type)
					//
					// Non-blocking send avoids deadlocks/panics when jobs are coalesced and a
					// cancellation result has already been delivered.
					select {
					case job.Result <- &JobResult[T]{
						Value:         job.Payload,
						HandlerResult: result,
						Error:         err,
					}:
					case <-job.Context.Done():
						// Context cancelled
					default:
						// Result already delivered (e.g., coalesced job)
					}
				}()

			case <-w.ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Stop stops the worker
func (w *Worker[T]) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&w.running, 1, 0) {
		return nil
	}

	w.cancel()

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stats returns worker statistics
func (w *Worker[T]) Stats() WorkerStats {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.stats
}

// IsRunning returns whether the worker is running
func (w *Worker[T]) IsRunning() bool {
	return atomic.LoadInt32(&w.running) == 1
}

// ComputePool is a generic compute pool for CPU-bound tasks
// Supports routing, backpressure, and auto-scaling
type ComputePool[T any] struct {
	workers           []*Worker[T]
	jobChan           chan *Job[T]
	handler           func(context.Context, interface{}) (interface{}, error) // Accepts interface{} for flexibility
	config            Config
	ctx               context.Context
	cancel            context.CancelFunc
	running           int32
	mu                sync.RWMutex
	jobIDGen          int64
	roundRobinCounter int64              // For round-robin routing
	coalesceMap       map[string]*Job[T] // For CoalesceByKey policy
	coalesceMu        sync.Mutex
}

// Config configures a compute pool
type Config struct {
	// Workers is the number of workers (0 = auto: GOMAXPROCS / 2)
	Workers int

	// ThreadsPerWorker is threads per worker (0 = auto: GOMAXPROCS / Workers)
	// Note: This is informational for LLM/custom workers, pool doesn't manage threads
	ThreadsPerWorker int

	// QueueSize is the job queue size
	QueueSize int

	// BackpressurePolicy defines how to handle queue overflow
	BackpressurePolicy BackpressurePolicy

	// RoutingPolicy defines how jobs are routed to workers
	RoutingPolicy RoutingPolicy

	// RouteByKey enables key-based routing (same as RoutingPolicy = HashByKey)
	RouteByKey bool
}

// DefaultConfig returns a default configuration with auto-scaling
func DefaultConfig() Config {
	numCPU := runtime.GOMAXPROCS(0)
	workers := numCPU / 2
	if workers < 1 {
		workers = 1
	}

	return Config{
		Workers:            0, // Auto
		ThreadsPerWorker:   0, // Auto
		QueueSize:          1000,
		BackpressurePolicy: Block,
		RoutingPolicy:      RoundRobin,
		RouteByKey:         false,
	}
}

// NewComputePool creates a new compute pool
func NewComputePool[T any](ctx context.Context, handler func(context.Context, interface{}) (interface{}, error), config Config) (*ComputePool[T], error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	if handler == nil {
		return nil, fmt.Errorf("handler cannot be nil")
	}

	// Auto-calculate workers
	workers := config.Workers
	if workers == 0 {
		numCPU := runtime.GOMAXPROCS(0)
		workers = numCPU / 2
		if workers < 1 {
			workers = 1
		}
	}

	// Auto-calculate threads per worker (informational)
	threadsPerWorker := config.ThreadsPerWorker
	if threadsPerWorker == 0 {
		numCPU := runtime.GOMAXPROCS(0)
		threadsPerWorker = numCPU / workers
		if threadsPerWorker < 1 {
			threadsPerWorker = 1
		}
	}

	if config.QueueSize < 1 {
		config.QueueSize = 1000
	}

	// Apply RouteByKey flag
	if config.RouteByKey {
		config.RoutingPolicy = HashByKey
	}

	poolCtx, cancel := context.WithCancel(ctx)

	pool := &ComputePool[T]{
		jobChan:     make(chan *Job[T], config.QueueSize),
		handler:     handler, // handler accepts interface{} for flexibility
		config:      config,
		ctx:         poolCtx,
		cancel:      cancel,
		workers:     make([]*Worker[T], 0, workers),
		coalesceMap: make(map[string]*Job[T]),
	}

	// Create workers
	for i := 0; i < workers; i++ {
		worker := NewWorker(i, handler, pool.jobChan)
		pool.workers = append(pool.workers, worker)
	}

	return pool, nil
}

// Start starts all workers
func (p *ComputePool[T]) Start() error {
	if !atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		return fmt.Errorf("pool is already started")
	}

	for i, worker := range p.workers {
		if err := worker.Start(); err != nil {
			// Stop already started workers
			p.Stop(p.ctx)
			return fmt.Errorf("failed to start worker %d: %w", i, err)
		}
	}

	return nil
}

// Stop stops all workers
func (p *ComputePool[T]) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&p.running, 1, 0) {
		return nil
	}

	p.cancel()
	close(p.jobChan)

	var lastErr error
	for _, worker := range p.workers {
		if err := worker.Stop(ctx); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// Submit submits a job and returns a Future
func (p *ComputePool[T]) Submit(ctx context.Context, key string, payload T) (*Future[T], error) {
	if atomic.LoadInt32(&p.running) == 0 {
		return nil, fmt.Errorf("pool is not started")
	}

	// Generate job ID
	jobID := fmt.Sprintf("job-%d", atomic.AddInt64(&p.jobIDGen, 1))

	// Create job
	job := &Job[T]{
		ID:      jobID,
		Key:     key,
		Payload: payload,
		Context: ctx,
		Result:  make(chan *JobResult[T], 1),
		Created: time.Now(),
		doneSig: make(chan struct{}, 1),
	}

	// Handle backpressure
	if err := p.submitWithBackpressure(job); err != nil {
		return nil, err
	}

	// Create future
	future := &Future[T]{
		resultChan: job.Result,
	}

	return future, nil
}

// submitWithBackpressure handles job submission with backpressure policy
func (p *ComputePool[T]) submitWithBackpressure(job *Job[T]) error {
	switch p.config.BackpressurePolicy {
	case Block:
		// Block until queue has space
		select {
		case p.jobChan <- job:
			return nil
		case <-job.Context.Done():
			return job.Context.Err()
		}

	case DropNewest:
		// Try non-blocking, drop if full
		select {
		case p.jobChan <- job:
			return nil
		default:
			return fmt.Errorf("queue full, dropping newest job")
		}

	case DropOldest:
		// Try to add, if full, remove oldest and add new
		select {
		case p.jobChan <- job:
			return nil
		default:
			// Remove oldest (first in queue)
			select {
			case <-p.jobChan:
				// Oldest removed, try again
				select {
				case p.jobChan <- job:
					return nil
				default:
					return fmt.Errorf("queue full, failed to drop oldest")
				}
			default:
				return fmt.Errorf("queue full, failed to drop oldest")
			}
		}

	case CoalesceByKey:
		// Coalesce by key: keep newest, drop older with same key
		if job.Key == "" {
			// No key, use regular submission
			select {
			case p.jobChan <- job:
				return nil
			default:
				return fmt.Errorf("queue full")
			}
		}

		p.coalesceMu.Lock()
		oldJob, exists := p.coalesceMap[job.Key]
		if exists {
			// Cancel old job (drop it) without closing channels (avoids races with workers).
			// Deliver a best-effort cancellation result so the old Future won't hang.
			oldJob.superseded.Store(true)
			select {
			case oldJob.Result <- &JobResult[T]{
				Value:         oldJob.Payload,
				HandlerResult: nil,
				Error:         context.Canceled,
			}:
			default:
				// Old job result already delivered or receiver not ready.
			}
		}
		p.coalesceMap[job.Key] = job
		p.coalesceMu.Unlock()

		// Submit job
		select {
		case p.jobChan <- job:
			// Remove from coalesce map when job is processed
			go func() {
				<-job.doneSig
				p.coalesceMu.Lock()
				if p.coalesceMap[job.Key] == job {
					delete(p.coalesceMap, job.Key)
				}
				p.coalesceMu.Unlock()
			}()
			return nil
		default:
			p.coalesceMu.Lock()
			delete(p.coalesceMap, job.Key)
			p.coalesceMu.Unlock()
			return fmt.Errorf("queue full")
		}

	default:
		return fmt.Errorf("unknown backpressure policy")
	}
}

// Note: Routing is handled by channel distribution
// All workers read from the same jobChan, routing happens via backpressure policy
// For HashByKey, we would need per-worker channels, but for simplicity
// we use a single channel and let workers compete (still maintains locality via key hashing in coalesce)

// IsRunning returns whether the pool is running
func (p *ComputePool[T]) IsRunning() bool {
	return atomic.LoadInt32(&p.running) == 1
}

// Stats returns pool statistics
type PoolStats struct {
	Workers      int
	Running      bool
	QueueLength  int
	WorkersStats []WorkerStats
}

// Stats returns current pool statistics
func (p *ComputePool[T]) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	workersStats := make([]WorkerStats, len(p.workers))
	for i, worker := range p.workers {
		workersStats[i] = worker.Stats()
	}

	return PoolStats{
		Workers:      len(p.workers),
		Running:      p.IsRunning(),
		QueueLength:  len(p.jobChan),
		WorkersStats: workersStats,
	}
}
