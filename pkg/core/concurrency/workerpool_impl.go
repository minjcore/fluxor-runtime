package concurrency

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

// defaultWorkerPool implements WorkerPool
// Hides goroutine creation and management from public API
type defaultWorkerPool struct {
	workers   int
	queueSize int
	taskChan  chan Task // Hidden: internal channel
	wg        sync.WaitGroup
	running   int32 // Atomic flag
	ctx       context.Context
	cancel    context.CancelFunc
	logger    simpleLogger // Logger for error messages

	// Metrics (atomic for thread-safety)
	queuedTasks    int64 // Actual queue length (incremented on submit, decremented on receive)
	completedTasks int64
	rejectedTasks  int64
}

// WorkerPoolConfig configures a WorkerPool
type WorkerPoolConfig struct {
	Workers   int // Number of worker goroutines
	QueueSize int // Task queue size
}

// DefaultWorkerPoolConfig returns default worker pool configuration
func DefaultWorkerPoolConfig() WorkerPoolConfig {
	return WorkerPoolConfig{
		Workers:   10,
		QueueSize: 1000,
	}
}

// NewWorkerPool creates a new WorkerPool
// Hides goroutine and channel creation from callers
func NewWorkerPool(ctx context.Context, config WorkerPoolConfig) WorkerPool {
	// Fail-fast: context cannot be nil
	if ctx == nil {
		failFastIf(true, "context cannot be nil")
	}
	if config.Workers < 1 {
		config.Workers = 1
	}
	if config.QueueSize < 1 {
		config.QueueSize = 100
	}

	ctx, cancel := context.WithCancel(ctx)

	return &defaultWorkerPool{
		workers:   config.Workers,
		queueSize: config.QueueSize,
		taskChan:  make(chan Task, config.QueueSize), // Hidden: channel creation
		ctx:       ctx,
		cancel:    cancel,
		logger:    newDefaultSimpleLogger(),
	}
}

// Start implements WorkerPool interface
// Hides goroutine creation
// Optimized: Uses atomic CompareAndSwap for lock-free start check
func (wp *defaultWorkerPool) Start() error {
	// Lock-free start check using atomic CompareAndSwap
	if !atomic.CompareAndSwapInt32(&wp.running, 0, 1) {
		return fmt.Errorf("worker pool is already running")
	}

	wp.wg.Add(wp.workers)

	// Start worker goroutines (hidden: go func() calls)
	for i := 0; i < wp.workers; i++ {
		go wp.worker(i) // Hidden: goroutine creation
	}

	return nil
}

// worker processes tasks (hides channel operations)
func (wp *defaultWorkerPool) worker(id int) {
	defer wp.wg.Done()

	for {
		select {
		case task, ok := <-wp.taskChan: // Hidden: channel receive
			if !ok {
				return // Channel closed
			}

			// Decrement queued tasks counter when task is received
			atomic.AddInt64(&wp.queuedTasks, -1)

			// Execute task
			if err := task.Execute(wp.ctx); err != nil {
				// Don't log context.Canceled or ErrMailboxClosed - they're expected during shutdown
				if !errors.Is(err, context.Canceled) && !errors.Is(err, ErrMailboxClosed) {
					wp.logger.Errorf("worker %d: task %s failed: %v", id, task.Name(), err)
				}
			}
			atomic.AddInt64(&wp.completedTasks, 1)

		case <-wp.ctx.Done():
			return
		}
	}
}

// Stop implements WorkerPool interface
// Optimized: Uses atomic CompareAndSwap for lock-free stop check
func (wp *defaultWorkerPool) Stop(ctx context.Context) error {
	// Fail-fast: context cannot be nil
	if ctx == nil {
		failFastIf(true, "context cannot be nil")
	}

	// Lock-free stop check using atomic CompareAndSwap
	if !atomic.CompareAndSwapInt32(&wp.running, 1, 0) {
		return nil // Already stopped
	}

	wp.cancel()

	// Close task channel (hidden: channel close)
	close(wp.taskChan)

	// Wait for workers to finish or timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("stop timeout: %w", ctx.Err())
	}
}

// Submit implements WorkerPool interface
// Hides channel send operations
// Optimized: Lock-free hot path
func (wp *defaultWorkerPool) Submit(task Task) error {
	// Fail-fast: task cannot be nil
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	if atomic.LoadInt32(&wp.running) == 0 {
		return fmt.Errorf("worker pool is not running")
	}

	// Try to send (non-blocking for backpressure)
	select {
	case wp.taskChan <- task: // Hidden: channel send
		atomic.AddInt64(&wp.queuedTasks, 1)
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	default:
		atomic.AddInt64(&wp.rejectedTasks, 1)
		return ErrMailboxFull
	}
}

// SubmitBatch implements WorkerPool interface
// Optimized batch submission for high-throughput scenarios
func (wp *defaultWorkerPool) SubmitBatch(tasks []Task) error {
	if len(tasks) == 0 {
		return nil
	}

	if atomic.LoadInt32(&wp.running) == 0 {
		return fmt.Errorf("worker pool is not running")
	}

	// Submit tasks in batch - more efficient than individual submits
	submitted := 0
	for _, task := range tasks {
		if task == nil {
			continue // Skip nil tasks
		}

		select {
		case wp.taskChan <- task:
			atomic.AddInt64(&wp.queuedTasks, 1)
			submitted++
		case <-wp.ctx.Done():
			// If context is done, reject remaining tasks
			atomic.AddInt64(&wp.rejectedTasks, int64(len(tasks)-submitted))
			return wp.ctx.Err()
		default:
			// Queue full - reject remaining tasks
			atomic.AddInt64(&wp.rejectedTasks, int64(len(tasks)-submitted))
			return ErrMailboxFull
		}
	}

	return nil
}

// Workers implements WorkerPool interface
func (wp *defaultWorkerPool) Workers() int {
	return wp.workers
}

// IsRunning implements WorkerPool interface
func (wp *defaultWorkerPool) IsRunning() bool {
	return atomic.LoadInt32(&wp.running) == 1
}

// Stats implements WorkerPool interface
func (wp *defaultWorkerPool) Stats() WorkerPoolStats {
	queued := atomic.LoadInt64(&wp.queuedTasks)
	queueUtilization := float64(queued) / float64(wp.queueSize) * 100.0
	if queueUtilization > 100.0 {
		queueUtilization = 100.0
	}

	return WorkerPoolStats{
		QueuedTasks:      queued,
		ActiveWorkers:    wp.workers,
		CompletedTasks:   atomic.LoadInt64(&wp.completedTasks),
		RejectedTasks:    atomic.LoadInt64(&wp.rejectedTasks),
		QueueCapacity:    wp.queueSize,
		QueueUtilization: queueUtilization,
	}
}
