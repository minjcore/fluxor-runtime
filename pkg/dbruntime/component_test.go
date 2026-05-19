package dbruntime

import (
	"context"
	"testing"
)

func TestNewDatabaseComponent(t *testing.T) {
	config := DefaultPoolConfig("test-dsn", "postgres")
	component := NewDatabaseComponent(config)

	// Fail-fast: NewDatabaseComponent panics on invalid config, so component is never nil
	if component.Name() != "database" {
		t.Errorf("Name() = %v, want database", component.Name())
	}
}

func TestDatabaseComponent_NotStarted(t *testing.T) {
	config := DefaultPoolConfig("test-dsn", "postgres")
	component := NewDatabaseComponent(config)

	// Component not started, operations should fail
	ctx := context.Background()

	_, err := component.Query(ctx, "SELECT 1")
	if err == nil {
		t.Error("Query() should return error when not started")
	}

	_, err = component.Exec(ctx, "SELECT 1")
	if err == nil {
		t.Error("Exec() should return error when not started")
	}

	_, err = component.Begin(ctx)
	if err == nil {
		t.Error("Begin() should return error when not started")
	}

	err = component.Ping(ctx)
	if err == nil {
		t.Error("Ping() should return error when not started")
	}
}

func TestDatabaseComponent_Stats_NotStarted(t *testing.T) {
	config := DefaultPoolConfig("test-dsn", "postgres")
	component := NewDatabaseComponent(config)

	// Stats should return empty stats when not started
	stats := component.Stats()
	if stats.OpenConnections != 0 {
		t.Errorf("Stats().OpenConnections = %v, want 0", stats.OpenConnections)
	}
}

// Note: Integration tests with real database would require:
// - Database setup
// - Actual connection string
// - Test database cleanup
// These are unit tests for component structure only
