package dbruntime

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Config holds database pool configuration.
type Config struct {
	DSN    string // connection string
	Driver string // "postgres", "mysql", "sqlite3", etc.

	MaxOpen         int           // max open connections (default 25)
	MaxIdle         int           // max idle connections (default 5)
	ConnMaxLifetime time.Duration // max connection reuse time (default 5m)
	ConnMaxIdleTime time.Duration // max idle time (default 10m)

	// HealthCheck interval for background ping + reconnect. 0 = disabled.
	HealthCheck time.Duration

	// CircuitBreaker enables the circuit breaker on the health-check loop.
	// nil = disabled.
	CircuitBreaker *CircuitConfig
}

func (c *Config) setDefaults() {
	if c.MaxOpen <= 0 {
		c.MaxOpen = 25
	}
	if c.MaxIdle <= 0 {
		c.MaxIdle = 5
	}
	if c.MaxIdle > c.MaxOpen {
		c.MaxIdle = c.MaxOpen
	}
	if c.ConnMaxLifetime <= 0 {
		c.ConnMaxLifetime = 5 * time.Minute
	}
	if c.ConnMaxIdleTime <= 0 {
		c.ConnMaxIdleTime = 10 * time.Minute
	}
	if c.HealthCheck <= 0 {
		c.HealthCheck = 30 * time.Second
	}
}

func (c Config) validate() error {
	if c.DSN == "" {
		return fmt.Errorf("dbruntime: DSN is required")
	}
	if c.Driver == "" {
		return fmt.Errorf("dbruntime: Driver is required")
	}
	return nil
}

// DB wraps *sql.DB with config and an optional background health-check goroutine.
// All standard database/sql methods are available via embedding.
type DB struct {
	*sql.DB
	cfg     Config
	circuit *Circuit
	stop    chan struct{}
	once    sync.Once
}

// Open opens a database, validates config, pings, and starts the health-check loop.
func Open(cfg Config) (*DB, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	cfg.setDefaults()

	sqlDB, err := sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("dbruntime: open: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpen)
	sqlDB.SetMaxIdleConns(cfg.MaxIdle)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("dbruntime: ping: %w", err)
	}

	d := &DB{DB: sqlDB, cfg: cfg, stop: make(chan struct{})}
	if cfg.CircuitBreaker != nil {
		d.circuit = NewCircuit(*cfg.CircuitBreaker)
	}
	if cfg.HealthCheck > 0 {
		go d.healthLoop()
	}
	return d, nil
}

// Close stops the health-check goroutine and closes the underlying connection pool.
func (d *DB) Close() error {
	d.once.Do(func() { close(d.stop) })
	return d.DB.Close()
}

// healthLoop pings the DB on the configured interval and logs failures.
// database/sql reconnects automatically; this loop surfaces persistent failures.
func (d *DB) healthLoop() {
	ticker := time.NewTicker(d.cfg.HealthCheck)
	defer ticker.Stop()
	for {
		select {
		case <-d.stop:
			return
		case <-ticker.C:
			if d.circuit != nil && !d.circuit.Allow() {
				continue // circuit open — fail fast, skip ping
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := d.DB.PingContext(ctx)
			cancel()
			if err != nil {
				slog.Warn("dbruntime: health check failed", "driver", d.cfg.Driver, "err", err)
				if d.circuit != nil {
					d.circuit.RecordFailure(err)
				}
			} else if d.circuit != nil {
				d.circuit.RecordSuccess()
			}
		}
	}
}

// CircuitStats returns a snapshot of the circuit breaker state for metrics/observability.
// Returns nil if no circuit breaker is configured.
func (d *DB) CircuitStats() *CircuitStats {
	if d.circuit == nil {
		return nil
	}
	s := d.circuit.Stats()
	return &s
}
