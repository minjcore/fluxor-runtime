package dbruntime

import (
	"testing"
	"time"
)

func TestOpen_MissingDSN(t *testing.T) {
	_, err := Open(Config{Driver: "postgres"})
	if err == nil {
		t.Fatal("expected error for empty DSN")
	}
}

func TestOpen_MissingDriver(t *testing.T) {
	_, err := Open(Config{DSN: "postgres://localhost/db"})
	if err == nil {
		t.Fatal("expected error for empty Driver")
	}
}

func TestConfig_Defaults(t *testing.T) {
	cfg := Config{DSN: "x", Driver: "postgres"}
	cfg.setDefaults()

	if cfg.MaxOpen != 25 {
		t.Errorf("MaxOpen = %d, want 25", cfg.MaxOpen)
	}
	if cfg.MaxIdle != 5 {
		t.Errorf("MaxIdle = %d, want 5", cfg.MaxIdle)
	}
	if cfg.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("ConnMaxLifetime = %v, want 5m", cfg.ConnMaxLifetime)
	}
	if cfg.HealthCheck != 30*time.Second {
		t.Errorf("HealthCheck = %v, want 30s", cfg.HealthCheck)
	}
}

func TestConfig_MaxIdleCappedAtMaxOpen(t *testing.T) {
	cfg := Config{DSN: "x", Driver: "postgres", MaxOpen: 3, MaxIdle: 10}
	cfg.setDefaults()
	if cfg.MaxIdle > cfg.MaxOpen {
		t.Errorf("MaxIdle %d > MaxOpen %d", cfg.MaxIdle, cfg.MaxOpen)
	}
}
