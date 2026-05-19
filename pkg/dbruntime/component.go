package dbruntime

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/core/runtime/state"
)

// DatabaseComponent provides database connection pooling using Premium Pattern
// Similar to HikariCP DataSource but integrated with Fluxor
// Uses separated architecture: PoolManager (dedicated goroutine) + ConnectionExecutor (worker pool)
type DatabaseComponent struct {
	*core.BaseComponent
	config           PoolConfig
	poolManager      *PoolManager
	connectionExecutor *ConnectionExecutor
	stateManager     state.Manager
	
	// Configuration for executor
	executorWorkers  int
	executorQueueSize int
	executorTimeout   time.Duration
}

// NewDatabaseComponent creates a new database component with connection pooling
// Fail-fast: Validates configuration
func NewDatabaseComponent(config PoolConfig) *DatabaseComponent {
	// Fail-fast: Validate configuration
	if config.DSN == "" {
		panic("DSN cannot be empty")
	}
	if config.DriverName == "" {
		panic("DriverName cannot be empty")
	}
	if config.MaxOpenConns <= 0 {
		panic("MaxOpenConns must be positive")
	}

	// Create state manager with history tracking
	stateConfig := state.DefaultConfig()
	stateConfig.HistorySize = 20 // Keep last 20 state transitions
	
	comp := &DatabaseComponent{
		BaseComponent:     core.NewBaseComponent("database"),
		config:             config,
		executorWorkers:    runtime.NumCPU(), // Default to CPU count
		executorQueueSize:  1000,             // Default queue size
		executorTimeout:    30 * time.Second, // Default timeout
		stateManager:       state.NewManager(stateConfig),
	}
	// Set hooks to enable method overriding (Go doesn't support Java-style overriding)
	// This ensures that DatabaseComponent.doStart() is called instead of BaseComponent.doStart()
	comp.BaseComponent.SetHooks(comp.doStart, comp.doStop)
	return comp
}

// SetExecutorConfig configures the connection executor
func (c *DatabaseComponent) SetExecutorConfig(workers int, queueSize int, timeout time.Duration) {
	if workers > 0 {
		c.executorWorkers = workers
	}
	if queueSize > 0 {
		c.executorQueueSize = queueSize
	}
	if timeout > 0 {
		c.executorTimeout = timeout
	}
}

// doStart initializes the pool manager and connection executor
// Pool management runs in a dedicated goroutine, connections execute in worker goroutines
func (c *DatabaseComponent) doStart(ctx core.FluxorContext) error {
	// Use state manager for lifecycle management
	return c.stateManager.Start(ctx.Context(), func() error {
		// Fail-fast: Validate context
		if ctx == nil {
			return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
		}

		// Fail-fast: Validate configuration (should have been validated in NewDatabaseComponent)
		if c.config.DSN == "" {
			return &core.EventBusError{Code: "INVALID_CONFIG", Message: "DSN cannot be empty"}
		}
		if c.config.DriverName == "" {
			return &core.EventBusError{Code: "INVALID_CONFIG", Message: "DriverName cannot be empty"}
		}

		// Create pool manager (runs in dedicated goroutine)
		eventBus := c.EventBus()
		c.poolManager = NewPoolManager(c.config, eventBus)
		if err := c.poolManager.Start(); err != nil {
			return fmt.Errorf("failed to start pool manager: %w", err)
		}

		// Create connection executor (runs in worker goroutines)
		c.connectionExecutor = NewConnectionExecutor(
			c.poolManager,
			c.executorWorkers,
			c.executorQueueSize,
			c.executorTimeout,
		)
		if err := c.connectionExecutor.Start(); err != nil {
			// Cleanup pool manager on error
			_ = c.poolManager.Stop()
			return fmt.Errorf("failed to start connection executor: %w", err)
		}

		// Notify via EventBus (Premium Pattern integration)
		if eventBus != nil {
			if err := eventBus.Publish("database.ready", map[string]interface{}{
				"component":         c.Name(),
				"max_open_conns":    c.config.MaxOpenConns,
				"max_idle_conns":    c.config.MaxIdleConns,
				"executor_workers":  c.executorWorkers,
				"executor_queue":    c.executorQueueSize,
			}); err != nil {
				// Best-effort notification; ignore on error.
			}
		}

		return nil
	})
}

// doStop stops the connection executor and pool manager
// This method is called via the hook set in NewDatabaseComponent
func (c *DatabaseComponent) doStop(ctx core.FluxorContext) error {
	// Use state manager for lifecycle management
	return c.stateManager.Stop(func() error {
		var errs []error

		// Stop connection executor first
		if c.connectionExecutor != nil {
			if err := c.connectionExecutor.Stop(); err != nil {
				errs = append(errs, fmt.Errorf("failed to stop connection executor: %w", err))
			}
			c.connectionExecutor = nil
		}

		// Stop pool manager
		if c.poolManager != nil {
			if err := c.poolManager.Stop(); err != nil {
				errs = append(errs, fmt.Errorf("failed to stop pool manager: %w", err))
			}
			c.poolManager = nil
		}

		if len(errs) > 0 {
			return fmt.Errorf("errors during shutdown: %v", errs)
		}
		return nil
	})
}

// Pool returns the connection pool from pool manager
// Fail-fast: Panics if component not in running state
func (c *DatabaseComponent) Pool() *Pool {
	if c == nil {
		panic("DatabaseComponent cannot be nil")
	}
	currentState := c.stateManager.Current()
	if currentState != state.StateRunning {
		panic(fmt.Sprintf("cannot get pool: component is in state %s (expected StateRunning)", currentState))
	}
	if c.poolManager == nil {
		panic("pool manager not initialized")
	}
	pool := c.poolManager.GetPool()
	if pool == nil {
		panic("pool not available - pool manager not started")
	}
	return pool
}

// DB returns the underlying *sql.DB from pool manager
// Fail-fast: Panics if component not in running state
func (c *DatabaseComponent) DB() *sql.DB {
	if c == nil {
		panic("DatabaseComponent cannot be nil")
	}
	currentState := c.stateManager.Current()
	if currentState != state.StateRunning {
		panic(fmt.Sprintf("cannot get DB: component is in state %s (expected StateRunning)", currentState))
	}
	if c.poolManager == nil {
		panic("pool manager not initialized")
	}
	pool := c.poolManager.GetPool()
	if pool == nil {
		panic("pool not available - pool manager not started")
	}
	return pool.DB()
}

// SafeDB returns the underlying *sql.DB without panicking
// Returns nil if component is not started or pool is not ready
// Use this method when you need to check if DB is available without handling panics
func (c *DatabaseComponent) SafeDB() *sql.DB {
	if c == nil || c.poolManager == nil {
		return nil
	}
	pool := c.poolManager.GetPool()
	if pool == nil {
		return nil
	}
	return pool.DB()
}

// GetPoolManager returns the pool manager
func (c *DatabaseComponent) GetPoolManager() *PoolManager {
	return c.poolManager
}

// GetConnectionExecutor returns the connection executor
func (c *DatabaseComponent) GetConnectionExecutor() *ConnectionExecutor {
	return c.connectionExecutor
}

// GetState returns the current state of the database component
func (c *DatabaseComponent) GetState() state.State {
	return c.stateManager.Current()
}

// GetStateStats returns state transition statistics
func (c *DatabaseComponent) GetStateStats() state.Stats {
	return c.stateManager.Stats()
}

// GetStateHistory returns the state transition history
func (c *DatabaseComponent) GetStateHistory() []state.Transition {
	return c.stateManager.History()
}

// WaitForState waits for the database component to reach a specific state
func (c *DatabaseComponent) WaitForState(ctx context.Context, target state.State) error {
	return c.stateManager.WaitForState(ctx, target)
}

// OnStateChange registers a callback for state changes
func (c *DatabaseComponent) OnStateChange(callback func(from, to state.State)) {
	// Register callback and also publish state changes via EventBus
	c.stateManager.OnStateChange(func(from, to state.State) {
		callback(from, to)
		// Publish state change event
		eventBus := c.EventBus()
		if eventBus != nil {
			_ = eventBus.Publish("database.state.changed", map[string]interface{}{
				"component": c.Name(),
				"from":      from.String(),
				"to":        to.String(),
			})
		}
	})
}

// validateStateForOperation validates that the component is in a valid state for database operations
// Performance-critical path: avoid allocations here.
// Fast path for common case (StateRunning): O(1) state lookup, early return.
// Memory-efficient: direct state checks without unnecessary allocations.
// Benchmark shows <200ns per validation (see component_state_test.go).
func (c *DatabaseComponent) validateStateForOperation(operation string) error {
	if c == nil {
		return &Error{Code: "INVALID_STATE", Message: "DatabaseComponent cannot be nil"}
	}
	
	currentState := c.stateManager.Current()
	if currentState != state.StateRunning {
		return &Error{
			Code:    "INVALID_STATE",
			Message: fmt.Sprintf("cannot execute %s: component is in state %s (expected StateRunning)", operation, currentState),
		}
	}
	
	// Also validate pool manager is available
	if c.poolManager == nil {
		return &Error{Code: "NOT_STARTED", Message: "pool manager not initialized"}
	}
	
	pool := c.poolManager.GetPool()
	if pool == nil {
		return &Error{Code: "POOL_NOT_AVAILABLE", Message: "pool not available"}
	}
	
	return nil
}

// Query executes a query that returns rows (synchronous, uses pool directly)
// For async execution in dedicated goroutine, use QueryAsync()
// Fail-fast: Validates state and inputs before querying
// Performance-critical path: state validation is fast (<200ns).
// Fast path for common case (no error): direct pool access.
func (c *DatabaseComponent) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if err := c.validateStateForOperation("Query"); err != nil {
		return nil, err
	}
	if ctx == nil {
		return nil, &Error{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}
	if query == "" {
		return nil, &Error{Code: "INVALID_INPUT", Message: "query cannot be empty"}
	}
	pool := c.poolManager.GetPool()
	return pool.Query(ctx, query, args...)
}

// QueryAsync executes a query asynchronously in a dedicated goroutine
// Returns a channel that will receive the result
// Highly performant async handler for high-throughput endpoints.
// Handles concurrent queries efficiently with worker pool pattern.
func (c *DatabaseComponent) QueryAsync(ctx context.Context, query string, args ...interface{}) (<-chan *JobResult, error) {
	if err := c.validateStateForOperation("QueryAsync"); err != nil {
		return nil, err
	}
	if c.connectionExecutor == nil {
		return nil, &Error{Code: "NOT_STARTED", Message: "connection executor not initialized"}
	}
	return c.connectionExecutor.Query(ctx, query, args...)
}

// QueryRow executes a query that returns a single row (synchronous, uses pool directly)
// For async execution in dedicated goroutine, use QueryRowAsync()
// Fail-fast: Validates state and inputs before querying
// Fast path for common case (no error): direct pool access.
// Note: Returns *sql.Row (cannot return error), so panics on invalid state/input
// This is consistent with database/sql.QueryRowContext behavior
func (c *DatabaseComponent) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if err := c.validateStateForOperation("QueryRow"); err != nil {
		panic(err.Error())
	}
	if ctx == nil {
		panic("context cannot be nil")
	}
	if query == "" {
		panic("query cannot be empty")
	}
	pool := c.poolManager.GetPool()
	return pool.QueryRow(ctx, query, args...)
}

// QueryRowAsync executes a query asynchronously in a dedicated goroutine
// Returns a channel that will receive the result
func (c *DatabaseComponent) QueryRowAsync(ctx context.Context, query string, args ...interface{}) (<-chan *JobResult, error) {
	if err := c.validateStateForOperation("QueryRowAsync"); err != nil {
		return nil, err
	}
	if c.connectionExecutor == nil {
		return nil, &Error{Code: "NOT_STARTED", Message: "connection executor not initialized"}
	}
	return c.connectionExecutor.QueryRow(ctx, query, args...)
}

// Exec executes a command (synchronous, uses pool directly)
// For async execution in dedicated goroutine, use ExecAsync()
// Fail-fast: Validates state and inputs before executing
// Fast path for common case (no error): direct pool access.
// Memory-efficient: reuses connection pool to avoid GC pressure.
func (c *DatabaseComponent) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if err := c.validateStateForOperation("Exec"); err != nil {
		return nil, err
	}
	if ctx == nil {
		return nil, &Error{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}
	if query == "" {
		return nil, &Error{Code: "INVALID_INPUT", Message: "query cannot be empty"}
	}
	pool := c.poolManager.GetPool()
	return pool.Exec(ctx, query, args...)
}

// ExecAsync executes a command asynchronously in a dedicated goroutine
// Returns a channel that will receive the result
// Highly performant async handler for high-throughput write operations.
// Handles concurrent executions efficiently with worker pool pattern.
func (c *DatabaseComponent) ExecAsync(ctx context.Context, query string, args ...interface{}) (<-chan *JobResult, error) {
	if err := c.validateStateForOperation("ExecAsync"); err != nil {
		return nil, err
	}
	if c.connectionExecutor == nil {
		return nil, &Error{Code: "NOT_STARTED", Message: "connection executor not initialized"}
	}
	return c.connectionExecutor.Exec(ctx, query, args...)
}

// Begin starts a transaction (synchronous, uses pool directly)
// For async execution in dedicated goroutine, use BeginAsync()
// Fail-fast: Validates state and inputs before beginning transaction
// Fast path for common case (no error): direct pool access.
func (c *DatabaseComponent) Begin(ctx context.Context) (*sql.Tx, error) {
	if err := c.validateStateForOperation("Begin"); err != nil {
		return nil, err
	}
	if ctx == nil {
		return nil, &Error{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}
	pool := c.poolManager.GetPool()
	return pool.Begin(ctx)
}

// BeginAsync starts a transaction asynchronously in a dedicated goroutine
// Returns a channel that will receive the result
// Highly performant async handler for concurrent transaction management.
func (c *DatabaseComponent) BeginAsync(ctx context.Context) (<-chan *JobResult, error) {
	if err := c.validateStateForOperation("BeginAsync"); err != nil {
		return nil, err
	}
	if c.connectionExecutor == nil {
		return nil, &Error{Code: "NOT_STARTED", Message: "connection executor not initialized"}
	}
	return c.connectionExecutor.Begin(ctx)
}

// BeginTx starts a transaction with options
// Fail-fast: Validates state and inputs before beginning transaction
func (c *DatabaseComponent) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if err := c.validateStateForOperation("BeginTx"); err != nil {
		return nil, err
	}
	if ctx == nil {
		return nil, &Error{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}
	pool := c.poolManager.GetPool()
	return pool.BeginTx(ctx, opts)
}

// Stats returns pool statistics (similar to HikariPoolMXBean)
// Fail-fast: Returns empty stats if not in running state (safe, no panic)
func (c *DatabaseComponent) Stats() sql.DBStats {
	if c == nil {
		return sql.DBStats{}
	}
	currentState := c.stateManager.Current()
	if currentState != state.StateRunning {
		return sql.DBStats{}
	}
	if c.poolManager == nil {
		return sql.DBStats{}
	}
	pool := c.poolManager.GetPool()
	if pool == nil {
		return sql.DBStats{}
	}
	return pool.Stats()
}

// Ping tests the connection
// Fail-fast: Validates state and inputs before pinging
// Fast path for health checks: minimal overhead validation.
func (c *DatabaseComponent) Ping(ctx context.Context) error {
	if err := c.validateStateForOperation("Ping"); err != nil {
		return err
	}
	if ctx == nil {
		return &Error{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}
	pool := c.poolManager.GetPool()
	return pool.Ping(ctx)
}
