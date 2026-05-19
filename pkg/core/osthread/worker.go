package osthread

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"runtime"
)

// worker represents a single worker goroutine pinned to an OS thread
// The worker is a goroutine that is locked to a system-level OS thread
// using runtime.LockOSThread().
// Each worker has its own dedicated channel - no global shared access.
type worker struct {
	id       int
	taskChan chan *taskWrapper // Dedicated channel for this worker (not shared)
	ctx      context.Context
	running  int32
	wg       sync.WaitGroup
	mu       sync.RWMutex
	statsVal workerStats
	closed   int32 // Atomic flag for closed state
}

// workerStats contains statistics for a worker
type workerStats struct {
	ProcessedTasks int64
	FailedTasks    int64
	TotalLatency   time.Duration
	LastTaskTime   time.Time
}

// newWorker creates a new worker with its own dedicated channel
// No global shared access - each worker has isolated task queue
func newWorker(id int, taskChan chan *taskWrapper, ctx context.Context) *worker {
	return &worker{
		id:       id,
		taskChan: taskChan,
		ctx:      ctx,
	}
}

// start starts the worker in a pinned OS thread
func (w *worker) start() error {
	if !atomic.CompareAndSwapInt32(&w.running, 0, 1) {
		return nil // Already running
	}

	w.wg.Add(1)
	go func() {
		// Pin OS thread for CPU-bound work (LLM, crypto, native code, etc.)
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer w.wg.Done()

		for {
			select {
			case wrapper, ok := <-w.taskChan:
				if !ok {
					return // Channel closed
				}

				// Execute task on pinned OS thread
				startTime := time.Now()
				err := w.executeTask(wrapper.ctx, wrapper.task)

				// Update stats
				latency := time.Since(startTime)
				w.mu.Lock()
				atomic.AddInt64(&w.statsVal.ProcessedTasks, 1)
				w.statsVal.TotalLatency += latency
				w.statsVal.LastTaskTime = time.Now()
				if err != nil {
					atomic.AddInt64(&w.statsVal.FailedTasks, 1)
				}
				w.mu.Unlock()

				// Send result (non-blocking)
				select {
				case wrapper.result <- err:
				default:
				}

			case <-w.ctx.Done():
				return
			}
		}
	}()

	return nil
}

// executeTask executes a task with proper context handling
func (w *worker) executeTask(ctx context.Context, task Task) error {
	// Create a combined context that respects both worker context and task context
	combinedCtx := ctx
	if w.ctx != nil {
		var cancel context.CancelFunc
		combinedCtx, cancel = context.WithCancel(ctx)
		defer cancel()

		// Cancel combined context if worker context is cancelled
		go func() {
			select {
			case <-w.ctx.Done():
				cancel()
			case <-combinedCtx.Done():
			}
		}()
	}

	return task.Execute(combinedCtx)
}

// submit submits a task to this worker's dedicated channel
// Returns true if submitted successfully, false if channel is full or closed
// This method is thread-safe and lock-free (uses channel operations)
func (w *worker) submit(wrapper *taskWrapper) bool {
	if atomic.LoadInt32(&w.closed) == 1 {
		return false // Worker is closed
	}
	
	// Non-blocking send to worker's dedicated channel
	select {
	case w.taskChan <- wrapper:
		return true
	default:
		return false // Channel full
	}
}

// close closes the worker's dedicated channel
func (w *worker) close() {
	if atomic.CompareAndSwapInt32(&w.closed, 0, 1) {
		close(w.taskChan)
	}
}

// queueLength returns the current queue length for this worker
func (w *worker) queueLength() int {
	return len(w.taskChan)
}

// stop stops the worker
func (w *worker) stop() {
	if !atomic.CompareAndSwapInt32(&w.running, 1, 0) {
		return // Already stopped
	}
	w.wg.Wait()
}

// getStats returns worker statistics
func (w *worker) getStats() workerStats {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.statsVal
}
