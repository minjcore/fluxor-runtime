package proxy

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents proxy server configuration.
// It embeds BaseConfig from the config package, providing common configuration
// features like service name, server settings, profile, and environment.
//
// Example usage:
//
//	// Create config with BaseConfig defaults
//	cfg := proxy.DefaultConfig()
//	cfg.ListenAddr = ":8080"
//	cfg.Backends = []proxy.Backend{
//	    {URL: "http://localhost:3000", Weight: 1},
//	    {URL: "http://localhost:3001", Weight: 2},
//	}
//	cfg.Service.Name = "proxy-service"
//
//	// Or load from file (BaseConfig supports YAML/JSON loading)
//	var cfg proxy.Config
//	if err := config.Load("proxy-config.yaml", &cfg); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Validate both proxy-specific and BaseConfig fields
//	if err := cfg.Validate(); err != nil {
//	    log.Fatal(err)
//	}
type Config struct {
	// Embed BaseConfig to inherit common configuration features
	// This provides: Service, Server, Profile, Environment, and lifecycle methods
	config.BaseConfig

	// ListenAddr is the address to listen on (e.g., ":8080")
	ListenAddr string `json:"listenAddr" env:"PROXY_LISTEN_ADDR" default:":8080" description:"Proxy server listen address"`

	// Protocol is the proxy protocol ("http", "tcp", or "both")
	Protocol string `json:"protocol" env:"PROXY_PROTOCOL" default:"http" description:"Proxy protocol (http, tcp, or both)"`

	// Backends is the list of backend servers to proxy to
	Backends []Backend `json:"backends" description:"Backend servers to proxy to"`

	// LoadBalancingStrategy is the load balancing algorithm ("round-robin", "least-connections", "weighted")
	LoadBalancingStrategy string `json:"loadBalancingStrategy" env:"PROXY_LB_STRATEGY" default:"round-robin" description:"Load balancing strategy"`

	// HealthCheckInterval is the interval between backend health checks
	HealthCheckInterval time.Duration `json:"healthCheckInterval" env:"PROXY_HEALTH_CHECK_INTERVAL" default:"30s" description:"Backend health check interval"`

	// HealthCheckTimeout is the timeout for health checks
	HealthCheckTimeout time.Duration `json:"healthCheckTimeout" env:"PROXY_HEALTH_CHECK_TIMEOUT" default:"5s" description:"Health check timeout"`

	// MaxConnections is the maximum concurrent connections (0 = unlimited)
	MaxConnections int `json:"maxConnections" env:"PROXY_MAX_CONNECTIONS" default:"0" description:"Maximum concurrent connections"`

	// ConnectionTimeout is the timeout for establishing backend connections
	ConnectionTimeout time.Duration `json:"connectionTimeout" env:"PROXY_CONNECTION_TIMEOUT" default:"30s" description:"Backend connection timeout"`

	// ReadTimeout is the read timeout for connections
	ReadTimeout time.Duration `json:"readTimeout" env:"PROXY_READ_TIMEOUT" default:"30s" description:"Read timeout"`

	// WriteTimeout is the write timeout for connections
	WriteTimeout time.Duration `json:"writeTimeout" env:"PROXY_WRITE_TIMEOUT" default:"30s" description:"Write timeout"`

	// IdleTimeout is the idle connection timeout
	IdleTimeout time.Duration `json:"idleTimeout" env:"PROXY_IDLE_TIMEOUT" default:"90s" description:"Idle connection timeout"`

	// EnableMetrics enables metrics collection
	EnableMetrics bool `json:"enableMetrics" env:"PROXY_ENABLE_METRICS" default:"true" description:"Enable metrics collection"`

	// RateLimit is the maximum requests per second (0 = unlimited)
	RateLimit int `json:"rateLimit" env:"PROXY_RATE_LIMIT" default:"0" description:"Rate limit (requests per second)"`

	// TLSConfig for HTTPS/TLS proxy (optional)
	TLSConfig interface{} `json:"tlsConfig,omitempty" description:"TLS configuration for HTTPS proxy"`
}

// DefaultConfig returns a default proxy configuration with BaseConfig initialized.
// Uses environment variables: PROXY_LISTEN_ADDR, PROXY_PROTOCOL
// The BaseConfig is initialized with defaults from config.NewBaseConfig().
func DefaultConfig() Config {
	cfg := Config{
		BaseConfig:            *config.NewBaseConfig(),
		ListenAddr:            getEnvOrDefault("PROXY_LISTEN_ADDR", ":8080"),
		Protocol:              getEnvOrDefault("PROXY_PROTOCOL", "http"),
		Backends:              []Backend{},
		LoadBalancingStrategy: getEnvOrDefault("PROXY_LB_STRATEGY", "round-robin"),
		HealthCheckInterval:   30 * time.Second,
		HealthCheckTimeout:    5 * time.Second,
		MaxConnections:        0,
		ConnectionTimeout:     30 * time.Second,
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           90 * time.Second,
		EnableMetrics:         true,
		RateLimit:             0,
	}

	return cfg
}

// Validate validates the configuration, including both proxy-specific fields
// and BaseConfig fields. This demonstrates how to extend BaseConfig validation.
// Fail-fast: Returns error if required fields are missing
func (c *Config) Validate() error {
	// Validate BaseConfig first (if it has validators)
	baseValidators := c.BaseConfig.GetValidators()
	for _, validator := range baseValidators {
		if err := validator.Validate(c); err != nil {
			return &ConfigError{Code: "BASE_CONFIG_VALIDATION", Message: err.Error()}
		}
	}

	// Validate listen address
	if c.ListenAddr == "" {
		c.ListenAddr = getEnvOrDefault("PROXY_LISTEN_ADDR", ":8080")
	}

	// Validate protocol
	if c.Protocol == "" {
		c.Protocol = getEnvOrDefault("PROXY_PROTOCOL", "http")
	}
	if c.Protocol != "http" && c.Protocol != "tcp" && c.Protocol != "both" {
		return &ConfigError{
			Code:    "INVALID_PROTOCOL",
			Message: "Protocol must be 'http', 'tcp', or 'both'",
		}
	}

	// Validate backends
	if len(c.Backends) == 0 {
		return &ConfigError{
			Code:    "MISSING_BACKENDS",
			Message: "At least one backend server is required",
		}
	}

	for i, backend := range c.Backends {
		if backend.URL == "" {
			return &ConfigError{
				Code:    "INVALID_BACKEND",
				Message: fmt.Sprintf("Backend %d: URL is required", i),
			}
		}

		// Validate URL format (basic check)
		if !isValidURL(backend.URL) {
			return &ConfigError{
				Code:    "INVALID_BACKEND_URL",
				Message: fmt.Sprintf("Backend %d: Invalid URL format: %s", i, backend.URL),
			}
		}

		// Set defaults
		if backend.Weight <= 0 {
			c.Backends[i].Weight = 1
		}
		if backend.Timeout == 0 {
			c.Backends[i].Timeout = c.ConnectionTimeout
		}
		if backend.HealthCheckInterval == 0 {
			c.Backends[i].HealthCheckInterval = c.HealthCheckInterval
		}
	}

	// Validate load balancing strategy
	if c.LoadBalancingStrategy == "" {
		c.LoadBalancingStrategy = getEnvOrDefault("PROXY_LB_STRATEGY", "round-robin")
	}
	validStrategies := map[string]bool{
		"round-robin":       true,
		"least-connections": true,
		"weighted":          true,
		"random":            true,
	}
	if !validStrategies[c.LoadBalancingStrategy] {
		return &ConfigError{
			Code:    "INVALID_LB_STRATEGY",
			Message: "Load balancing strategy must be one of: round-robin, least-connections, weighted, random",
		}
	}

	// Validate timeouts
	if c.ConnectionTimeout <= 0 {
		c.ConnectionTimeout = 30 * time.Second
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = 30 * time.Second
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = 30 * time.Second
	}
	if c.IdleTimeout <= 0 {
		c.IdleTimeout = 90 * time.Second
	}

	return nil
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func isValidURL(url string) bool {
	return strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "tcp://") ||
		strings.HasPrefix(url, "ws://") ||
		strings.HasPrefix(url, "wss://")
}
