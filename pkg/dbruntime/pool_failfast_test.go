package dbruntime

import (
	"context"
	"testing"
)

func TestNewPool_FailFast_EmptyDSN(t *testing.T) {
	config := PoolConfig{
		DSN:        "",
		DriverName: "postgres",
	}

	_, err := NewPool(config)
	if err == nil {
		t.Error("NewPool() should fail-fast with empty DSN")
	}
	if err.Error() != "DSN cannot be empty" {
		t.Errorf("Error message = %v, want 'DSN cannot be empty'", err)
	}
}

func TestNewPool_FailFast_EmptyDriverName(t *testing.T) {
	config := PoolConfig{
		DSN:        "postgres://localhost/db",
		DriverName: "",
	}

	_, err := NewPool(config)
	if err == nil {
		t.Error("NewPool() should fail-fast with empty DriverName")
	}
}

func TestNewPool_FailFast_InvalidMaxOpenConns(t *testing.T) {
	config := PoolConfig{
		DSN:          "postgres://localhost/db",
		DriverName:   "postgres",
		MaxOpenConns: 0, // Invalid
	}

	_, err := NewPool(config)
	if err == nil {
		t.Error("NewPool() should fail-fast with MaxOpenConns <= 0")
	}
}

func TestNewPool_FailFast_InvalidMaxIdleConns(t *testing.T) {
	config := PoolConfig{
		DSN:          "postgres://localhost/db",
		DriverName:   "postgres",
		MaxOpenConns: 10,
		MaxIdleConns: -1, // Invalid
	}

	_, err := NewPool(config)
	if err == nil {
		t.Error("NewPool() should fail-fast with negative MaxIdleConns")
	}
}

func TestNewPool_FailFast_MaxIdleExceedsMaxOpen(t *testing.T) {
	config := PoolConfig{
		DSN:          "postgres://localhost/db",
		DriverName:   "postgres",
		MaxOpenConns: 10,
		MaxIdleConns: 20, // Exceeds MaxOpenConns
	}

	_, err := NewPool(config)
	if err == nil {
		t.Error("NewPool() should fail-fast when MaxIdleConns > MaxOpenConns")
	}
}

func TestPool_Query_FailFast_NilPool(t *testing.T) {
	var pool *Pool = nil

	ctx := context.Background()
	_, err := pool.Query(ctx, "SELECT 1")
	if err == nil {
		t.Error("Query() should fail-fast with nil pool")
	}
}

func TestPool_Query_FailFast_NilContext(t *testing.T) {
	// Note: This test would require a real database connection
	// For unit testing, we test the validation logic
	config := DefaultPoolConfig("test-dsn", "postgres")
	pool := &Pool{config: config} // pool.db is nil

	// Test nil context validation (fail-fast)
	var nilCtx context.Context = nil
	_, err := pool.Query(nilCtx, "SELECT 1")
	if err == nil {
		t.Error("Query() should fail-fast with nil context")
	}
}

func TestPool_Query_FailFast_EmptyQuery(t *testing.T) {
	config := DefaultPoolConfig("test-dsn", "postgres")
	pool := &Pool{config: config} // pool.db is nil

	ctx := context.Background()
	_, err := pool.Query(ctx, "")
	if err == nil {
		t.Error("Query() should fail-fast with empty query")
	}
}

func TestPool_QueryRow_FailFast_NilPool(t *testing.T) {
	var pool *Pool = nil

	ctx := context.Background()

	defer func() {
		if r := recover(); r == nil {
			t.Error("QueryRow() should panic with nil pool")
		}
	}()

	pool.QueryRow(ctx, "SELECT 1")
}

func TestPool_Exec_FailFast_NilPool(t *testing.T) {
	var pool *Pool = nil

	ctx := context.Background()
	_, err := pool.Exec(ctx, "SELECT 1")
	if err == nil {
		t.Error("Exec() should fail-fast with nil pool")
	}
}

func TestPool_Begin_FailFast_NilPool(t *testing.T) {
	var pool *Pool = nil

	ctx := context.Background()
	_, err := pool.Begin(ctx)
	if err == nil {
		t.Error("Begin() should fail-fast with nil pool")
	}
}

func TestPool_Begin_FailFast_NilContext(t *testing.T) {
	config := DefaultPoolConfig("test-dsn", "postgres")
	pool := &Pool{config: config}

	// Test nil context validation (fail-fast)
	var nilCtx context.Context = nil
	_, err := pool.Begin(nilCtx)
	if err == nil {
		t.Error("Begin() should fail-fast with nil context")
	}
}

func TestPool_Ping_FailFast_NilPool(t *testing.T) {
	var pool *Pool = nil

	ctx := context.Background()
	err := pool.Ping(ctx)
	if err == nil {
		t.Error("Ping() should fail-fast with nil pool")
	}
}

func TestPool_Ping_FailFast_NilContext(t *testing.T) {
	config := DefaultPoolConfig("test-dsn", "postgres")
	pool := &Pool{config: config}

	// Test nil context validation (fail-fast)
	var nilCtx context.Context = nil
	err := pool.Ping(nilCtx)
	if err == nil {
		t.Error("Ping() should fail-fast with nil context")
	}
}

func TestPool_DB_FailFast_NilPool(t *testing.T) {
	var pool *Pool = nil

	defer func() {
		if r := recover(); r == nil {
			t.Error("DB() should panic with nil pool")
		}
	}()

	pool.DB()
}

func TestNewDatabaseComponent_FailFast_EmptyDSN(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewDatabaseComponent() should panic with empty DSN")
		}
	}()

	config := PoolConfig{
		DSN:        "",
		DriverName: "postgres",
	}
	NewDatabaseComponent(config)
}

func TestNewDatabaseComponent_FailFast_EmptyDriverName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewDatabaseComponent() should panic with empty DriverName")
		}
	}()

	config := PoolConfig{
		DSN:        "postgres://localhost/db",
		DriverName: "",
	}
	NewDatabaseComponent(config)
}

func TestNewDatabaseComponent_FailFast_InvalidMaxOpenConns(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewDatabaseComponent() should panic with MaxOpenConns <= 0")
		}
	}()

	config := PoolConfig{
		DSN:          "postgres://localhost/db",
		DriverName:   "postgres",
		MaxOpenConns: 0,
	}
	NewDatabaseComponent(config)
}

func TestDatabaseComponent_Query_FailFast_NilComponent(t *testing.T) {
	var component *DatabaseComponent = nil

	ctx := context.Background()
	_, err := component.Query(ctx, "SELECT 1")
	if err == nil {
		t.Error("Query() should fail-fast with nil component")
	}
}

func TestDatabaseComponent_Query_FailFast_EmptyQuery(t *testing.T) {
	config := DefaultPoolConfig("test-dsn", "postgres")
	component := NewDatabaseComponent(config)

	ctx := context.Background()
	_, err := component.Query(ctx, "")
	if err == nil {
		t.Error("Query() should fail-fast with empty query")
	}
}

func TestDatabaseComponent_QueryRow_FailFast_NilComponent(t *testing.T) {
	var component *DatabaseComponent = nil

	ctx := context.Background()

	defer func() {
		if r := recover(); r == nil {
			t.Error("QueryRow() should panic with nil component")
		}
	}()

	component.QueryRow(ctx, "SELECT 1")
}

func TestDatabaseComponent_Pool_FailFast_NotStarted(t *testing.T) {
	config := DefaultPoolConfig("test-dsn", "postgres")
	component := NewDatabaseComponent(config)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Pool() should panic when component not started")
		}
	}()

	component.Pool()
}

func TestDatabaseComponent_DB_FailFast_NotStarted(t *testing.T) {
	config := DefaultPoolConfig("test-dsn", "postgres")
	component := NewDatabaseComponent(config)

	defer func() {
		if r := recover(); r == nil {
			t.Error("DB() should panic when component not started")
		}
	}()

	component.DB()
}
