package dbruntime

import (
	"context"
	"database/sql"
	"runtime"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/runtime/state"
)

// ConnectionExecutor executes database operations in dedicated goroutines
// This separates connection operations from pool management
type ConnectionExecutor struct {
	poolManager *PoolManager
	workers     int
	queueSize   int
	timeout      time.Duration

	// Worker pool
	jobQueue chan *DBJob
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.RWMutex
	stateManager state.Manager
	queueClosed bool // Track if queue is closed to prevent double-close
}

// DBJob represents a database operation job
type DBJob struct {
	Operation func(*sql.DB) (interface{}, error)
	Result    chan *JobResult
	Context   context.Context
}

// JobResult represents the result of a database operation
type JobResult struct {
	Value interface{}
	Error error
}

// NewConnectionExecutor creates a new connection executor
func NewConnectionExecutor(poolManager *PoolManager, workers int, queueSize int, timeout time.Duration) *ConnectionExecutor {
	ctx, cancel := context.WithCancel(context.Background())
	if workers <= 0 {
		workers = runtime.NumCPU() // Default to CPU count
	}
	if queueSize <= 0 {
		queueSize = 1000 // Default queue size
	}
	if timeout <= 0 {
		timeout = 30 * time.Second // Default timeout
	}

	// Create state manager with history tracking
	stateConfig := state.DefaultConfig()
	stateConfig.HistorySize = 20 // Keep last 20 state transitions
	
	ce := &ConnectionExecutor{
		poolManager: poolManager,
		workers:     workers,
		queueSize:   queueSize,
		timeout:     timeout,
		jobQueue:    make(chan *DBJob, queueSize),
		ctx:         ctx,
		cancel:      cancel,
		stateManager: state.NewManager(stateConfig),
	}
	
	return ce
}

// Start starts the connection executor worker pool
func (ce *ConnectionExecutor) Start() error {
	// Use state manager for lifecycle management
	return ce.stateManager.Start(ce.ctx, func() error {
		ce.mu.Lock()
		defer ce.mu.Unlock()

		// Start worker goroutines
		for i := 0; i < ce.workers; i++ {
			ce.wg.Add(1)
			go ce.worker(i)
		}

		return nil
	})
}

// Stop stops the connection executor
func (ce *ConnectionExecutor) Stop() error {
	// Use state manager for lifecycle management
	return ce.stateManager.Stop(func() error {
		// Signal shutdown
		ce.cancel()

		// Close job queue to stop accepting new jobs (only once)
		ce.mu.Lock()
		if !ce.queueClosed {
			close(ce.jobQueue)
			ce.queueClosed = true
		}
		ce.mu.Unlock()

		// Wait for all workers to finish
		ce.wg.Wait()

		return nil
	})
}

// worker is the worker goroutine that executes database operations
func (ce *ConnectionExecutor) worker(id int) {
	defer ce.wg.Done()

	for {
		select {
		case <-ce.ctx.Done():
			return
		case job, ok := <-ce.jobQueue:
			if !ok {
				// Channel closed
				return
			}
			ce.executeJob(job, id)
		}
	}
}

// executeJob executes a database job in the worker goroutine
func (ce *ConnectionExecutor) executeJob(job *DBJob, workerID int) {
	// Get pool from pool manager
	pool := ce.poolManager.GetPool()
	if pool == nil {
		job.Result <- &JobResult{
			Error: &Error{Code: "POOL_NOT_AVAILABLE", Message: "pool not available"},
		}
		return
	}

	// Get DB connection from pool
	db := pool.DB()
	if db == nil {
		job.Result <- &JobResult{
			Error: &Error{Code: "DB_NOT_AVAILABLE", Message: "database connection not available"},
		}
		return
	}

	// Execute the operation with timeout
	ctx := job.Context
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), ce.timeout)
		defer cancel()
	}

	// Create a channel for the result
	resultChan := make(chan *JobResult, 1)

	// Execute operation in a goroutine to handle timeout
	go func() {
		value, err := job.Operation(db)
		resultChan <- &JobResult{
			Value: value,
			Error: err,
		}
	}()

	// Wait for result or timeout
	select {
	case <-ctx.Done():
		job.Result <- &JobResult{
			Error: ctx.Err(),
		}
	case result := <-resultChan:
		job.Result <- result
	}
}

// Execute executes a database operation asynchronously
// Returns a channel that will receive the result
func (ce *ConnectionExecutor) Execute(ctx context.Context, operation func(*sql.DB) (interface{}, error)) (<-chan *JobResult, error) {
	currentState := ce.stateManager.Current()
	if currentState != state.StateRunning {
		return nil, &Error{Code: "NOT_STARTED", Message: "connection executor not started, current state: " + currentState.String()}
	}

	// Create job
	job := &DBJob{
		Operation: operation,
		Result:    make(chan *JobResult, 1),
		Context:   ctx,
	}

	// Submit job (non-blocking if queue is full)
	select {
	case ce.jobQueue <- job:
		return job.Result, nil
	default:
		return nil, &Error{Code: "QUEUE_FULL", Message: "connection executor queue is full"}
	}
}

// ExecuteSync executes a database operation synchronously
func (ce *ConnectionExecutor) ExecuteSync(ctx context.Context, operation func(*sql.DB) (interface{}, error)) (interface{}, error) {
	resultChan, err := ce.Execute(ctx, operation)
	if err != nil {
		return nil, err
	}

	result := <-resultChan
	return result.Value, result.Error
}

// Query executes a query in a dedicated goroutine
func (ce *ConnectionExecutor) Query(ctx context.Context, query string, args ...interface{}) (<-chan *JobResult, error) {
	return ce.Execute(ctx, func(db *sql.DB) (interface{}, error) {
		return db.QueryContext(ctx, query, args...)
	})
}

// QueryRow executes a query that returns a single row in a dedicated goroutine
// Note: The result will be *sql.Row which needs to be scanned in the caller
func (ce *ConnectionExecutor) QueryRow(ctx context.Context, query string, args ...interface{}) (<-chan *JobResult, error) {
	return ce.Execute(ctx, func(db *sql.DB) (interface{}, error) {
		row := db.QueryRowContext(ctx, query, args...)
		// Return the row - caller will need to type assert to *sql.Row and scan
		return row, nil
	})
}

// Exec executes a command in a dedicated goroutine
func (ce *ConnectionExecutor) Exec(ctx context.Context, query string, args ...interface{}) (<-chan *JobResult, error) {
	return ce.Execute(ctx, func(db *sql.DB) (interface{}, error) {
		return db.ExecContext(ctx, query, args...)
	})
}

// Begin starts a transaction in a dedicated goroutine
func (ce *ConnectionExecutor) Begin(ctx context.Context) (<-chan *JobResult, error) {
	return ce.Execute(ctx, func(db *sql.DB) (interface{}, error) {
		return db.BeginTx(ctx, nil)
	})
}

// IsStarted returns whether the executor is started
func (ce *ConnectionExecutor) IsStarted() bool {
	currentState := ce.stateManager.Current()
	return currentState == state.StateRunning || currentState == state.StateStarting
}

// GetState returns the current state of the connection executor
func (ce *ConnectionExecutor) GetState() state.State {
	return ce.stateManager.Current()
}

// GetStateStats returns state transition statistics
func (ce *ConnectionExecutor) GetStateStats() state.Stats {
	return ce.stateManager.Stats()
}

// GetStateHistory returns the state transition history
func (ce *ConnectionExecutor) GetStateHistory() []state.Transition {
	return ce.stateManager.History()
}

// WaitForState waits for the connection executor to reach a specific state
func (ce *ConnectionExecutor) WaitForState(ctx context.Context, target state.State) error {
	return ce.stateManager.WaitForState(ctx, target)
}

// OnStateChange registers a callback for state changes
func (ce *ConnectionExecutor) OnStateChange(callback func(from, to state.State)) {
	ce.stateManager.OnStateChange(callback)
}

// GetStats returns executor statistics
func (ce *ConnectionExecutor) GetStats() map[string]interface{} {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	stateStats := ce.stateManager.Stats()
	return map[string]interface{}{
		"workers":    ce.workers,
		"queue_size": ce.queueSize,
		"queue_len":  len(ce.jobQueue),
		"state":      stateStats.CurrentState.String(),
		"uptime":     stateStats.Uptime.String(),
	}
}
