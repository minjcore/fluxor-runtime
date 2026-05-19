package dbruntime

import (
	"testing"
	"time"
)

func TestDefaultPoolConfig(t *testing.T) {
	config := DefaultPoolConfig("test-dsn", "postgres")

	if config.DSN != "test-dsn" {
		t.Errorf("DSN = %v, want test-dsn", config.DSN)
	}
	if config.DriverName != "postgres" {
		t.Errorf("DriverName = %v, want postgres", config.DriverName)
	}
	if config.MaxOpenConns != 25 {
		t.Errorf("MaxOpenConns = %v, want 25", config.MaxOpenConns)
	}
	if config.MaxIdleConns != 5 {
		t.Errorf("MaxIdleConns = %v, want 5", config.MaxIdleConns)
	}
	if config.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("ConnMaxLifetime = %v, want 5m", config.ConnMaxLifetime)
	}
	if config.ConnMaxIdleTime != 10*time.Minute {
		t.Errorf("ConnMaxIdleTime = %v, want 10m", config.ConnMaxIdleTime)
	}
}

func TestPoolConfig(t *testing.T) {
	config := PoolConfig{
		DSN:             "test-dsn",
		DriverName:      "mysql",
		MaxOpenConns:    50,
		MaxIdleConns:    10,
		ConnMaxLifetime: 10 * time.Minute,
		ConnMaxIdleTime: 20 * time.Minute,
	}

	if config.DSN != "test-dsn" {
		t.Errorf("DSN = %v, want test-dsn", config.DSN)
	}
	if config.MaxOpenConns != 50 {
		t.Errorf("MaxOpenConns = %v, want 50", config.MaxOpenConns)
	}
}

// Note: Actual pool tests would require a real database connection
// These are unit tests for configuration only
