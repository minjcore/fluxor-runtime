package ws

import (
	"testing"
	"time"
)

func TestDefaultServerConfig(t *testing.T) {
	config := DefaultServerConfig(":8080", "/ws")

	if config.Addr != ":8080" {
		t.Errorf("Addr = %v, want :8080", config.Addr)
	}

	if config.Path != "/ws" {
		t.Errorf("Path = %v, want /ws", config.Path)
	}

	if config.ReadBufferSize <= 0 {
		t.Error("ReadBufferSize should be > 0")
	}

	if config.WriteBufferSize <= 0 {
		t.Error("WriteBufferSize should be > 0")
	}

	if config.ReadDeadline <= 0 {
		t.Error("ReadDeadline should be > 0")
	}

	if config.WriteDeadline <= 0 {
		t.Error("WriteDeadline should be > 0")
	}

	if config.PongWait <= 0 {
		t.Error("PongWait should be > 0")
	}

	if config.PingPeriod <= 0 {
		t.Error("PingPeriod should be > 0")
	}

	if config.PingPeriod >= config.PongWait {
		t.Error("PingPeriod should be less than PongWait")
	}

	if config.MaxQueue <= 0 {
		t.Error("MaxQueue should be > 0")
	}

	if config.Workers <= 0 {
		t.Error("Workers should be > 0")
	}
}

func TestDefaultServerConfig_EmptyArgs(t *testing.T) {
	config := DefaultServerConfig("", "")

	if config.Addr == "" {
		t.Error("Addr should have default value")
	}

	if config.Path == "" {
		t.Error("Path should have default value")
	}
}

func TestServerConfig_Defaults(t *testing.T) {
	config := DefaultServerConfig(":9000", "/test")

	// Verify defaults are sensible
	if config.ReadBufferSize < 1024 {
		t.Error("ReadBufferSize should be at least 1024")
	}

	if config.WriteBufferSize < 1024 {
		t.Error("WriteBufferSize should be at least 1024")
	}

	// PingPeriod should be 90% of PongWait (default implementation)
	expectedPingPeriod := (config.PongWait * 9) / 10
	if config.PingPeriod != expectedPingPeriod {
		t.Errorf("PingPeriod = %v, want %v", config.PingPeriod, expectedPingPeriod)
	}
}

func TestServerConfig_CustomValues(t *testing.T) {
	config := &ServerConfig{
		Addr:            ":9999",
		Path:            "/custom",
		ReadBufferSize:  8192,
		WriteBufferSize: 8192,
		ReadDeadline:    120 * time.Second,
		WriteDeadline:   30 * time.Second,
		PongWait:        120 * time.Second,
		PingPeriod:      108 * time.Second, // 90% of 120
		MaxConnections:  100,
	}

	if config.Addr != ":9999" {
		t.Errorf("Addr = %v, want :9999", config.Addr)
	}

	if config.Path != "/custom" {
		t.Errorf("Path = %v, want /custom", config.Path)
	}

	if config.ReadBufferSize != 8192 {
		t.Errorf("ReadBufferSize = %v, want 8192", config.ReadBufferSize)
	}

	if config.MaxConnections != 100 {
		t.Errorf("MaxConnections = %v, want 100", config.MaxConnections)
	}
}
