package dbruntime

import (
	"context"
	"database/sql"
	"time"
)

// PoolConfig configures database connection pool (similar to HikariConfig)
type PoolConfig struct {
	// DSN is the database connection string
	DSN string

	// MaxOpenConns is the maximum number of open connections (like maximumPoolSize in HikariCP)
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections (like minimumIdle in HikariCP)
	MaxIdleConns int

	// ConnMaxLifetime is the maximum amount of time a connection may be reused (like connectionTimeout)
	ConnMaxLifetime time.Duration

	// ConnMaxIdleTime is the maximum amount of time a connection may be idle (like idleTimeout)
	ConnMaxIdleTime time.Duration

	// DriverName is the database driver name (e.g., "postgres", "mysql")
	DriverName string

	// SkipInitialPing if true, NewPool does not ping the DB at creation (lazy connect).
	// Use when DB may not be ready at startup; PoolManager health check will reconnect when DB is up.
	SkipInitialPing bool
}

// DefaultPoolConfig returns HikariCP-like default configuration
func DefaultPoolConfig(dsn string, driverName string) PoolConfig {
	return PoolConfig{
		DSN:             dsn,
		DriverName:      driverName,
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}
}

// Pool represents a database connection pool
type Pool struct {
	db     *sql.DB
	config PoolConfig
}

// NewPool creates a new database connection pool (similar to HikariDataSource)
// Fail-fast: Validates configuration before creating pool
func NewPool(config PoolConfig) (*Pool, error) {
	// Fail-fast: Validate configuration
	if config.DSN == "" {
		return nil, &Error{Code: "INVALID_CONFIG", Message: "DSN cannot be empty"}
	}
	if config.DriverName == "" {
		return nil, &Error{Code: "INVALID_CONFIG", Message: "DriverName cannot be empty"}
	}
	if config.MaxOpenConns <= 0 {
		return nil, &Error{Code: "INVALID_CONFIG", Message: "MaxOpenConns must be positive"}
	}
	if config.MaxIdleConns < 0 {
		return nil, &Error{Code: "INVALID_CONFIG", Message: "MaxIdleConns cannot be negative"}
	}
	if config.MaxIdleConns > config.MaxOpenConns {
		return nil, &Error{Code: "INVALID_CONFIG", Message: "MaxIdleConns cannot exceed MaxOpenConns"}
	}
	if config.ConnMaxLifetime < 0 {
		return nil, &Error{Code: "INVALID_CONFIG", Message: "ConnMaxLifetime cannot be negative"}
	}
	if config.ConnMaxIdleTime < 0 {
		return nil, &Error{Code: "INVALID_CONFIG", Message: "ConnMaxIdleTime cannot be negative"}
	}

	// Open database (creates pool)
	db, err := sql.Open(config.DriverName, config.DSN)
	if err != nil {
		return nil, err
	}

	// Configure pool (similar to HikariConfig)
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	if !config.SkipInitialPing {
		// Test connection (fail-fast: verify connection works)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			if cerr := db.Close(); cerr != nil {
				// Best-effort cleanup; ignore on error.
			}
			return nil, err
		}
	}

	return &Pool{
		db:     db,
		config: config,
	}, nil
}

// Error represents a database error (fail-fast)
// Memory-efficient: simple struct with no allocations.
// Fast path for error creation: direct struct initialization.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

// Performance-related error codes
// These codes indicate performance issues that may require optimization
const (
	// ErrCodeSlowQuery indicates a query execution is slow
	// This is used for performance monitoring and optimization
	ErrCodeSlowQuery = "SLOW_QUERY"
	
	// ErrCodeInefficientOperation indicates an operation is not efficient
	// This helps identify operations that need optimization
	ErrCodeInefficientOperation = "INEFFICIENT_OPERATION"
	
	// ErrCodePerformanceDegradation indicates performance has degraded
	// This is used for performance monitoring and alerting
	ErrCodePerformanceDegradation = "PERFORMANCE_DEGRADATION"
	
	// ErrCodeOptimizationRequired indicates optimization is needed
	// This helps identify areas requiring performance improvements
	ErrCodeOptimizationRequired = "OPTIMIZATION_REQUIRED"
)

// DB returns the underlying *sql.DB
// Fail-fast: Panics if pool is nil (invalid state)
func (p *Pool) DB() *sql.DB {
	if p == nil {
		panic("pool cannot be nil")
	}
	if p.db == nil {
		panic("pool.db cannot be nil - pool not initialized")
	}
	return p.db
}

// Close closes the connection pool
// Fail-fast: Validates pool state before closing
func (p *Pool) Close() error {
	if p == nil {
		return &Error{Code: "INVALID_STATE", Message: "pool cannot be nil"}
	}
	if p.db == nil {
		return &Error{Code: "INVALID_STATE", Message: "pool already closed"}
	}
	return p.db.Close()
}

// Ping tests the connection
// Fail-fast: Validates inputs before pinging
func (p *Pool) Ping(ctx context.Context) error {
	if p == nil {
		return &Error{Code: "INVALID_STATE", Message: "pool cannot be nil"}
	}
	if p.db == nil {
		return &Error{Code: "INVALID_STATE", Message: "pool not initialized"}
	}
	if ctx == nil {
		return &Error{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}
	return p.db.PingContext(ctx)
}

// Stats returns pool statistics (similar to HikariPoolMXBean)
// Fail-fast: Validates pool state
func (p *Pool) Stats() sql.DBStats {
	if p == nil || p.db == nil {
		return sql.DBStats{}
	}
	return p.db.Stats()
}

// Query executes a query that returns rows
// Fail-fast: Validates inputs before querying
func (p *Pool) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if p == nil {
		return nil, &Error{Code: "INVALID_STATE", Message: "pool cannot be nil"}
	}
	if p.db == nil {
		return nil, &Error{Code: "INVALID_STATE", Message: "pool not initialized"}
	}
	if ctx == nil {
		return nil, &Error{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}
	if query == "" {
		return nil, &Error{Code: "INVALID_INPUT", Message: "query cannot be empty"}
	}
	return p.db.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns a single row
// Fail-fast: Validates inputs before querying
func (p *Pool) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if p == nil {
		panic("pool cannot be nil")
	}
	if p.db == nil {
		panic("pool not initialized")
	}
	if ctx == nil {
		panic("context cannot be nil")
	}
	if query == "" {
		panic("query cannot be empty")
	}
	return p.db.QueryRowContext(ctx, query, args...)
}

// Exec executes a command
// Fail-fast: Validates inputs before executing
func (p *Pool) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if p == nil {
		return nil, &Error{Code: "INVALID_STATE", Message: "pool cannot be nil"}
	}
	if p.db == nil {
		return nil, &Error{Code: "INVALID_STATE", Message: "pool not initialized"}
	}
	if ctx == nil {
		return nil, &Error{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}
	if query == "" {
		return nil, &Error{Code: "INVALID_INPUT", Message: "query cannot be empty"}
	}
	return p.db.ExecContext(ctx, query, args...)
}

// Begin starts a transaction
// Fail-fast: Validates inputs before beginning transaction
func (p *Pool) Begin(ctx context.Context) (*sql.Tx, error) {
	if p == nil {
		return nil, &Error{Code: "INVALID_STATE", Message: "pool cannot be nil"}
	}
	if p.db == nil {
		return nil, &Error{Code: "INVALID_STATE", Message: "pool not initialized"}
	}
	if ctx == nil {
		return nil, &Error{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}
	return p.db.BeginTx(ctx, nil)
}

// BeginTx starts a transaction with options
// Fail-fast: Validates inputs before beginning transaction
func (p *Pool) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if p == nil {
		return nil, &Error{Code: "INVALID_STATE", Message: "pool cannot be nil"}
	}
	if p.db == nil {
		return nil, &Error{Code: "INVALID_STATE", Message: "pool not initialized"}
	}
	if ctx == nil {
		return nil, &Error{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}
	return p.db.BeginTx(ctx, opts)
}
