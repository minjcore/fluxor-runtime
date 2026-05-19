package managers

import "time"

// Config holds configuration for all Managers-managed components
// Car analogy: System settings for the car (car system settings)
type Config struct {
	// HTTP Server configuration
	HTTPAddr string // HTTP server address (e.g., ":8080")

	// Logging configuration
	LogLevel string // Logging level: DEBUG, INFO, ERROR
	LogJSON  bool   // Enable JSON logging output

	// Cache configuration
	CacheType   string      // Cache type: "memory", "redis"
	CacheConfig interface{} // Cache-specific configuration (can be Redis config, etc.)

	// Observability configuration
	EnableMetrics bool   // Enable observability metrics collection
	MetricsPort   string // Metrics endpoint port (e.g., ":9090")

	// Heartbeat configuration
	HeartbeatInterval        time.Duration // Interval between heartbeats (default: 10 seconds)
	EnableHeartbeat          bool          // Enable heartbeat system (default: true)
	HeartbeatEventBusAddress string        // EventBus address for heartbeat messages (default: "managers.heartbeat")
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		HTTPAddr:                 ":8080",
		LogLevel:                 "INFO",
		LogJSON:                  false,
		CacheType:                "memory",
		CacheConfig:              nil,
		EnableMetrics:            true,
		MetricsPort:              ":9090",
		HeartbeatInterval:        10 * time.Second,
		EnableHeartbeat:          true,
		HeartbeatEventBusAddress: "managers.heartbeat",
	}
}
