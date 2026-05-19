package bulkhead

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Executable is a function that can be executed within a bulkhead.
type Executable func(ctx context.Context) error

// Config configures bulkhead behavior.
type Config struct {
	// MaxConcurrency is the maximum number of concurrent executions allowed.
	// Must be greater than 0.
	MaxConcurrency int

	// MaxQueueSize is the maximum number of queued executions waiting for a slot.
	// Zero means no queuing (immediate rejection when full).
	// Defaults to 0.
	MaxQueueSize int

	// QueueTimeout is the maximum time to wait in the queue for a slot.
	// Only used when MaxQueueSize > 0.
	// Zero means wait indefinitely.
	// Defaults to 0.
	QueueTimeout time.Duration

	// OnRejected is called when an execution is rejected (bulkhead full and queue full or timeout).
	OnRejected func(ctx context.Context, reason string)

	// OnRejectedAsync is called asynchronously when an execution is rejected.
	OnRejectedAsync func(ctx context.Context, reason string)

	// OnExecuting is called when execution starts (enters bulkhead).
	OnExecuting func(ctx context.Context)

	// OnExecutingAsync is called asynchronously when execution starts.
	OnExecutingAsync func(ctx context.Context)

	// OnCompleted is called when execution completes (exits bulkhead).
	OnCompleted func(ctx context.Context, err error)

	// OnCompletedAsync is called asynchronously when execution completes.
	OnCompletedAsync func(ctx context.Context, err error)

	// EnableMetrics enables detailed metrics collection.
	EnableMetrics bool
}

// DefaultConfig returns the default bulkhead configuration.
func DefaultConfig() Config {
	return Config{
		MaxConcurrency: 10,
		MaxQueueSize:   0, // No queuing by default
		QueueTimeout:   0, // No timeout by default
	}
}

// Manager provides bulkhead functionality to limit concurrent executions.
type Manager interface {
	// Execute executes a function within the bulkhead limits using default config.
	Execute(ctx context.Context, fn Executable) error

	// ExecuteWithConfig executes a function with the specified bulkhead config.
	ExecuteWithConfig(ctx context.Context, fn Executable, config Config) error

	// Stats returns statistics about bulkhead operations.
	Stats() Stats
}

// Stats contains statistics about bulkhead operations.
type Stats struct {
	// TotalExecutions is the total number of executions attempted.
	TotalExecutions int64

	// TotalAccepted is the total number of executions that were accepted (started).
	TotalAccepted int64

	// TotalRejected is the total number of executions that were rejected (bulkhead full).
	TotalRejected int64

	// TotalSuccessful is the total number of successful executions.
	TotalSuccessful int64

	// TotalFailed is the total number of failed executions.
	TotalFailed int64

	// CurrentConcurrency is the current number of concurrent executions.
	CurrentConcurrency int

	// CurrentQueueSize is the current number of executions waiting in queue.
	CurrentQueueSize int

	// LastExecutionTime is when the last execution completed.
	LastExecutionTime time.Time
}

// bulkheadManager implements the Manager interface.
type bulkheadManager struct {
	config Config

	// Semaphore for limiting concurrent executions
	semaphore chan struct{}

	// Queue for waiting executions (if MaxQueueSize > 0)
	queue chan queuedExecution

	// Current concurrency counter
	currentConcurrency int32

	// Statistics
	stats Stats
	mu    sync.RWMutex

	// Stop channel for queue processor
	stopChan chan struct{}
	doneChan chan struct{}
}

// queuedExecution represents an execution waiting in the queue.
type queuedExecution struct {
	fn       Executable
	ctx      context.Context
	resultCh chan error
}

// NewManager creates a new bulkhead manager with default config.
func NewManager() Manager {
	return NewManagerWithConfig(DefaultConfig())
}

// NewManagerWithConfig creates a new bulkhead manager with the specified config.
func NewManagerWithConfig(config Config) Manager {
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = DefaultConfig().MaxConcurrency
	}
	if config.MaxQueueSize < 0 {
		config.MaxQueueSize = 0
	}

	manager := &bulkheadManager{
		config:    config,
		semaphore: make(chan struct{}, config.MaxConcurrency),
		stopChan:  make(chan struct{}),
		doneChan:  make(chan struct{}),
	}

	// Initialize queue if queuing is enabled
	if config.MaxQueueSize > 0 {
		manager.queue = make(chan queuedExecution, config.MaxQueueSize)
		go manager.queueProcessor()
	} else {
		close(manager.doneChan)
	}

	return manager
}

// queueProcessor processes queued executions.
func (m *bulkheadManager) queueProcessor() {
	defer close(m.doneChan)

	for {
		select {
		case <-m.stopChan:
			return
		case queued := <-m.queue:
			// Try to acquire semaphore with optional queue timeout
			if m.config.QueueTimeout > 0 {
				// With timeout
				timeoutChan := time.After(m.config.QueueTimeout)
				select {
				case m.semaphore <- struct{}{}:
					// Acquired semaphore, execute
					go m.executeWithSemaphore(queued.ctx, queued.fn, queued.resultCh)
				case <-timeoutChan:
					// Queue timeout
					atomic.AddInt64(&m.stats.TotalRejected, 1)
					m.onRejected(queued.ctx, "queue timeout")
					select {
					case queued.resultCh <- NewError(ErrCodeContextTimeout, "queue timeout while waiting for bulkhead slot"):
					default:
					}
				case <-queued.ctx.Done():
					// Context cancelled
					atomic.AddInt64(&m.stats.TotalRejected, 1)
					m.onRejected(queued.ctx, "context cancelled")
					select {
					case queued.resultCh <- NewError(ErrCodeContextCanceled, "context cancelled while waiting for bulkhead slot"):
					default:
					}
				case <-m.stopChan:
					// Manager stopped
					atomic.AddInt64(&m.stats.TotalRejected, 1)
					m.onRejected(queued.ctx, "bulkhead stopped")
					select {
					case queued.resultCh <- NewError(ErrCodeBulkheadFull, "bulkhead stopped"):
					default:
					}
					return
				}
			} else {
				// Without timeout (wait indefinitely or until context cancelled)
				select {
				case m.semaphore <- struct{}{}:
					// Acquired semaphore, execute
					go m.executeWithSemaphore(queued.ctx, queued.fn, queued.resultCh)
				case <-queued.ctx.Done():
					// Context cancelled
					atomic.AddInt64(&m.stats.TotalRejected, 1)
					m.onRejected(queued.ctx, "context cancelled")
					select {
					case queued.resultCh <- NewError(ErrCodeContextCanceled, "context cancelled while waiting for bulkhead slot"):
					default:
					}
				case <-m.stopChan:
					// Manager stopped
					atomic.AddInt64(&m.stats.TotalRejected, 1)
					m.onRejected(queued.ctx, "bulkhead stopped")
					select {
					case queued.resultCh <- NewError(ErrCodeBulkheadFull, "bulkhead stopped"):
					default:
					}
					return
				}
			}
		}
	}
}

// Execute executes a function within the bulkhead limits using default config.
func (m *bulkheadManager) Execute(ctx context.Context, fn Executable) error {
	return m.ExecuteWithConfig(ctx, fn, m.config)
}

// ExecuteWithConfig executes a function with the specified bulkhead config.
// Note: Queue settings (MaxQueueSize, QueueTimeout) use the manager's config,
// but other settings (callbacks, etc.) use the provided config.
func (m *bulkheadManager) ExecuteWithConfig(ctx context.Context, fn Executable, config Config) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}
	if fn == nil {
		return NewError(ErrCodeNilFunction, "function cannot be nil")
	}

	atomic.AddInt64(&m.stats.TotalExecutions, 1)

	// Use manager's config for queue settings (queue is part of manager state)
	// But use provided config for callbacks
	m.mu.Lock()
	execConfig := m.config
	m.mu.Unlock()

	// Override callbacks from provided config
	if config.OnRejected != nil {
		execConfig.OnRejected = config.OnRejected
	}
	if config.OnRejectedAsync != nil {
		execConfig.OnRejectedAsync = config.OnRejectedAsync
	}
	if config.OnExecuting != nil {
		execConfig.OnExecuting = config.OnExecuting
	}
	if config.OnExecutingAsync != nil {
		execConfig.OnExecutingAsync = config.OnExecutingAsync
	}
	if config.OnCompleted != nil {
		execConfig.OnCompleted = config.OnCompleted
	}
	if config.OnCompletedAsync != nil {
		execConfig.OnCompletedAsync = config.OnCompletedAsync
	}

	// Check if queuing is enabled (using manager's config)
	if execConfig.MaxQueueSize > 0 && m.queue != nil {
		return m.executeWithQueue(ctx, fn, execConfig)
	}

	// Execute without queue (immediate rejection if full)
	return m.executeWithoutQueue(ctx, fn, execConfig)
}

// executeWithoutQueue executes without queuing (immediate rejection if bulkhead is full).
func (m *bulkheadManager) executeWithoutQueue(ctx context.Context, fn Executable, config Config) error {
	// Try to acquire semaphore immediately
	select {
	case m.semaphore <- struct{}{}:
		// Acquired, execute
		return m.executeWithSemaphore(ctx, fn, nil)
	case <-ctx.Done():
		// Context cancelled
		atomic.AddInt64(&m.stats.TotalRejected, 1)
		m.onRejected(ctx, "context cancelled")
		if ctx.Err() == context.DeadlineExceeded {
			return NewError(ErrCodeContextTimeout, fmt.Sprintf("context timeout: %v", ctx.Err()))
		}
		return NewError(ErrCodeContextCanceled, fmt.Sprintf("context cancelled: %v", ctx.Err()))
	default:
		// Bulkhead full
		atomic.AddInt64(&m.stats.TotalRejected, 1)
		m.onRejected(ctx, "bulkhead full")
		return NewError(ErrCodeBulkheadFull, fmt.Sprintf("bulkhead full (max concurrency: %d)", config.MaxConcurrency))
	}
}

// executeWithQueue queues the execution if bulkhead is full.
func (m *bulkheadManager) executeWithQueue(ctx context.Context, fn Executable, config Config) error {
	resultCh := make(chan error, 1)

	// Try to acquire semaphore immediately
	select {
	case m.semaphore <- struct{}{}:
		// Acquired immediately, execute
		return m.executeWithSemaphore(ctx, fn, nil)
	case <-ctx.Done():
		// Context cancelled
		atomic.AddInt64(&m.stats.TotalRejected, 1)
		m.onRejected(ctx, "context cancelled")
		if ctx.Err() == context.DeadlineExceeded {
			return NewError(ErrCodeContextTimeout, fmt.Sprintf("context timeout: %v", ctx.Err()))
		}
		return NewError(ErrCodeContextCanceled, fmt.Sprintf("context cancelled: %v", ctx.Err()))
	default:
		// Bulkhead full, try to queue
		select {
		case m.queue <- queuedExecution{fn: fn, ctx: ctx, resultCh: resultCh}:
			// Queued successfully, wait for result
			m.mu.Lock()
			m.stats.CurrentQueueSize++
			m.mu.Unlock()

			select {
			case err := <-resultCh:
				m.mu.Lock()
				m.stats.CurrentQueueSize--
				m.mu.Unlock()
				return err
			case <-ctx.Done():
				// Context cancelled while waiting
				m.mu.Lock()
				m.stats.CurrentQueueSize--
				m.mu.Unlock()
				atomic.AddInt64(&m.stats.TotalRejected, 1)
				m.onRejected(ctx, "context cancelled while queued")
				if ctx.Err() == context.DeadlineExceeded {
					return NewError(ErrCodeContextTimeout, fmt.Sprintf("context timeout while queued: %v", ctx.Err()))
				}
				return NewError(ErrCodeContextCanceled, fmt.Sprintf("context cancelled while queued: %v", ctx.Err()))
			}
		default:
			// Queue full, reject
			atomic.AddInt64(&m.stats.TotalRejected, 1)
			m.onRejected(ctx, "bulkhead and queue full")
			return NewError(ErrCodeBulkheadFull, fmt.Sprintf("bulkhead and queue full (max concurrency: %d, max queue: %d)", config.MaxConcurrency, config.MaxQueueSize))
		}
	}
}

// executeWithSemaphore executes the function after acquiring semaphore.
func (m *bulkheadManager) executeWithSemaphore(ctx context.Context, fn Executable, resultCh chan error) error {
	// Increment concurrency
	atomic.AddInt32(&m.currentConcurrency, 1)
	atomic.AddInt64(&m.stats.TotalAccepted, 1)

	m.onExecuting(ctx)

	// Release semaphore when done
	defer func() {
		<-m.semaphore
		atomic.AddInt32(&m.currentConcurrency, -1)
	}()

	// Execute function
	err := fn(ctx)

	// Update statistics
	m.mu.Lock()
	m.stats.LastExecutionTime = time.Now()
	m.mu.Unlock()

	if err == nil {
		atomic.AddInt64(&m.stats.TotalSuccessful, 1)
	} else {
		atomic.AddInt64(&m.stats.TotalFailed, 1)
	}

	m.onCompleted(ctx, err)

	// Send result if channel provided
	if resultCh != nil {
		select {
		case resultCh <- err:
		default:
		}
	}

	return err
}

// onRejected invokes rejection callbacks.
func (m *bulkheadManager) onRejected(ctx context.Context, reason string) {
	if m.config.OnRejected != nil {
		m.config.OnRejected(ctx, reason)
	}

	if m.config.OnRejectedAsync != nil {
		go m.config.OnRejectedAsync(ctx, reason)
	}
}

// onExecuting invokes execution start callbacks.
func (m *bulkheadManager) onExecuting(ctx context.Context) {
	if m.config.OnExecuting != nil {
		m.config.OnExecuting(ctx)
	}

	if m.config.OnExecutingAsync != nil {
		go m.config.OnExecutingAsync(ctx)
	}
}

// onCompleted invokes completion callbacks.
func (m *bulkheadManager) onCompleted(ctx context.Context, err error) {
	if m.config.OnCompleted != nil {
		m.config.OnCompleted(ctx, err)
	}

	if m.config.OnCompletedAsync != nil {
		go m.config.OnCompletedAsync(ctx, err)
	}
}

// Stats returns statistics about bulkhead operations.
func (m *bulkheadManager) Stats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return Stats{
		TotalExecutions:     atomic.LoadInt64(&m.stats.TotalExecutions),
		TotalAccepted:       atomic.LoadInt64(&m.stats.TotalAccepted),
		TotalRejected:       atomic.LoadInt64(&m.stats.TotalRejected),
		TotalSuccessful:     atomic.LoadInt64(&m.stats.TotalSuccessful),
		TotalFailed:         atomic.LoadInt64(&m.stats.TotalFailed),
		CurrentConcurrency:  int(atomic.LoadInt32(&m.currentConcurrency)),
		CurrentQueueSize:    m.stats.CurrentQueueSize,
		LastExecutionTime:   m.stats.LastExecutionTime,
	}
}

// Stop stops the bulkhead manager (stops queue processor if running).
func (m *bulkheadManager) Stop() {
	select {
	case <-m.stopChan:
		// Already stopped
	default:
		close(m.stopChan)
		<-m.doneChan
	}
}
