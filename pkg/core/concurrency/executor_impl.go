package concurrency

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// defaultExecutor implements Executor using channels and goroutines internally
// Hides all Go concurrency primitives from public API
type defaultExecutor struct {
	taskChan  chan Task // Hidden: internal channel
	workers   int
	queueSize int
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	closed    int32        // Atomic flag for lock-free closed check
	logger    simpleLogger // Logger for error messages

	// Shutdown synchronization - prevents race between Submit and channel close
	shutdownMu sync.RWMutex

	// Metrics (atomic for thread-safety)
	queuedTasks    int64 // Actual queue length (incremented on submit, decremented on receive)
	completedTasks int64
	rejectedTasks  int64
}

// ExecutorConfig configures an Executor
type ExecutorConfig struct {
	Workers   int // Number of worker goroutines
	QueueSize int // Maximum queue size (bounded for backpressure)
}

// DefaultExecutorConfig returns default executor configuration
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		Workers:   10,
		QueueSize: 1000,
	}
}

// NewExecutor creates a new Executor with the given configuration
// Hides goroutine and channel creation from callers
func NewExecutor(ctx context.Context, config ExecutorConfig) Executor {
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

	exec := &defaultExecutor{
		taskChan:  make(chan Task, config.QueueSize), // Hidden channel
		workers:   config.Workers,
		queueSize: config.QueueSize,
		ctx:       ctx,
		cancel:    cancel,
		logger:    newDefaultSimpleLogger(),
	}

	// Start worker goroutines (hidden from public API)
	exec.startWorkers()

	return exec
}

// startWorkers starts worker goroutines (hides go func() calls)
func (e *defaultExecutor) startWorkers() {
	e.wg.Add(e.workers)
	for i := 0; i < e.workers; i++ {
		go e.worker(i) // Hidden: goroutine creation
	}
}

// worker processes tasks from the queue (hides channel operations)
// Implements the event loop pattern for race-free sequential task processing:
//
// Event Loop Pattern (Race-Free Guarantee):
// - This is the classic Go event loop: for { select { case task := <-chan: ... } }
// - With Workers=1, only one goroutine runs this loop, ensuring sequential execution
// - Go channels guarantee only one receiver gets each message (serialization)
// - Tasks execute one at a time in the same goroutine (no concurrent access)
//
// Equivalent to explicit pattern:
//
//	func (v *BaseVerticle) eventLoop() {
//	    for {
//	        select {
//	        case msg := <-v.inbox:           // Message từ EventBus
//	            v.handleMessage(msg)         // Xử lý tuần tự – KHÔNG concurrent
//	        case <-v.ctx.Done():             // Stop signal
//	            v.cleanup()
//	            return
//	        }
//	    }
//	}
func (e *defaultExecutor) worker(id int) {
	defer e.wg.Done()

	for {
		select {
		case task, ok := <-e.taskChan: // Hidden: channel receive - serializes task delivery
			if !ok {
				return // Channel closed
			}
			// Decrement queued tasks counter when task is received
			atomic.AddInt64(&e.queuedTasks, -1)

			// Execute task sequentially (race-free: only one worker processes tasks when Workers=1)
			// Event loop tasks must complete in < 20µs (iron rule)
			startTime := time.Now()
			err := task.Execute(e.ctx)
			if err != nil {
				// Don't log context.Canceled or ErrMailboxClosed - they're expected during shutdown
				if !errors.Is(err, context.Canceled) && !errors.Is(err, ErrMailboxClosed) {
					e.logger.Errorf("task %s failed: %v", task.Name(), err)
				}
			}
			duration := time.Since(startTime)
			atomic.AddInt64(&e.completedTasks, 1)

			// Warn if task execution exceeds 20µs (event loop should not block).
			// Skip when task ended due to shutdown (long-running workers e.g. udp-worker-N).
			if duration > 20*time.Microsecond &&
				!errors.Is(err, context.Canceled) && !errors.Is(err, ErrMailboxClosed) {
				e.logger.Errorf("WARNING: task %s took %v (exceeds 20µs limit) - consider using SubmitBlocking() for blocking work",
					task.Name(), duration)
			}

		case <-e.ctx.Done():
			return // Context cancelled - graceful shutdown
		}
	}
}

// Submit implements Executor interface
// Hides channel send operations and select statements
// Thread-safe: Uses RWMutex to prevent race with Shutdown
func (e *defaultExecutor) Submit(task Task) error {
	// Fail-fast: task cannot be nil
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	// Acquire read lock to prevent race with Shutdown closing the channel
	// Multiple Submit() calls can run concurrently (RLock)
	// But Shutdown() will wait for all Submit() to finish before closing channel
	e.shutdownMu.RLock()
	defer e.shutdownMu.RUnlock()

	// Check closed after acquiring lock (Shutdown sets this before acquiring write lock)
	if atomic.LoadInt32(&e.closed) == 1 {
		return fmt.Errorf("executor is closed")
	}

	// Try to send to channel (non-blocking for backpressure)
	// Safe: channel cannot be closed while we hold RLock
	select {
	case e.taskChan <- task: // Hidden: channel send
		atomic.AddInt64(&e.queuedTasks, 1)
		return nil
	case <-e.ctx.Done():
		return e.ctx.Err()
	default:
		// Queue full - backpressure
		atomic.AddInt64(&e.rejectedTasks, 1)
		return ErrMailboxFull
	}
}

// SubmitWithTimeout implements Executor interface
// Thread-safe: Uses RWMutex to prevent race with Shutdown
func (e *defaultExecutor) SubmitWithTimeout(task Task, timeout time.Duration) error {
	// Fail-fast: task cannot be nil
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}
	// Fail-fast: timeout must be positive
	if timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	// Acquire read lock to prevent race with Shutdown closing the channel
	e.shutdownMu.RLock()
	defer e.shutdownMu.RUnlock()

	// Check closed after acquiring lock
	if atomic.LoadInt32(&e.closed) == 1 {
		return fmt.Errorf("executor is closed")
	}

	// Try to send with timeout
	// Safe: channel cannot be closed while we hold RLock
	select {
	case e.taskChan <- task: // Hidden: channel send
		atomic.AddInt64(&e.queuedTasks, 1)
		return nil
	case <-time.After(timeout):
		atomic.AddInt64(&e.rejectedTasks, 1)
		return fmt.Errorf("submit timeout after %v", timeout)
	case <-e.ctx.Done():
		return e.ctx.Err()
	}
}

// Shutdown implements Executor interface
// Thread-safe: Uses mutex to synchronize with Submit calls
func (e *defaultExecutor) Shutdown(ctx context.Context) error {
	// Fail-fast: context cannot be nil
	if ctx == nil {
		failFastIf(true, "context cannot be nil")
	}

	// Set closed flag first (Submit will check this after acquiring RLock)
	if !atomic.CompareAndSwapInt32(&e.closed, 0, 1) {
		return nil // Already closed
	}

	// Acquire write lock - waits for all in-flight Submit() calls to complete
	// This ensures no Submit() is in the middle of sending to taskChan
	e.shutdownMu.Lock()

	// Cancel context to stop workers
	e.cancel()

	// Close task channel (safe: no Submit() can be sending now)
	close(e.taskChan)

	// Release lock after channel is closed
	e.shutdownMu.Unlock()

	// Wait for workers to finish or timeout
	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout: %w", ctx.Err())
	}
}

// SubmitBatch implements Executor interface
// Optimized batch submission for high-throughput scenarios
func (e *defaultExecutor) SubmitBatch(tasks []Task) error {
	if len(tasks) == 0 {
		return nil
	}

	// Lock-free closed check using atomic
	if atomic.LoadInt32(&e.closed) == 1 {
		return fmt.Errorf("executor is closed")
	}

	// Submit tasks in batch - more efficient than individual submits
	submitted := 0
	for _, task := range tasks {
		if task == nil {
			continue // Skip nil tasks
		}

		select {
		case e.taskChan <- task:
			atomic.AddInt64(&e.queuedTasks, 1)
			submitted++
		case <-e.ctx.Done():
			// If context is done, reject remaining tasks
			atomic.AddInt64(&e.rejectedTasks, int64(len(tasks)-submitted))
			return e.ctx.Err()
		default:
			// Queue full - reject remaining tasks
			atomic.AddInt64(&e.rejectedTasks, int64(len(tasks)-submitted))
			return ErrMailboxFull
		}
	}

	return nil
}

// Stats implements Executor interface
func (e *defaultExecutor) Stats() ExecutorStats {
	queued := atomic.LoadInt64(&e.queuedTasks)
	queueUtilization := float64(queued) / float64(e.queueSize) * 100.0
	if queueUtilization > 100.0 {
		queueUtilization = 100.0
	}

	return ExecutorStats{
		QueuedTasks:      queued,
		ActiveWorkers:    e.workers,
		CompletedTasks:   atomic.LoadInt64(&e.completedTasks),
		RejectedTasks:    atomic.LoadInt64(&e.rejectedTasks),
		QueueCapacity:    e.queueSize,
		QueueUtilization: queueUtilization,
	}
}
