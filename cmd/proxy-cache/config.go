package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config represents the proxy cache configuration
type Config struct {
	// Server configuration
	Port    int    `json:"port"`
	Host    string `json:"host"`
	Verbose bool   `json:"verbose"`

	// Cache configuration
	CacheDir     string        `json:"cache_dir"`
	CacheType    string        `json:"cache_type"` // memory, redis, disk
	CacheTTL     time.Duration `json:"cache_ttl"`
	MaxCacheSize int64         `json:"max_cache_size"`

	// Redis configuration (if using redis cache)
	RedisAddr     string `json:"redis_addr"`
	RedisPassword string `json:"redis_password"`
	RedisDB       int    `json:"redis_db"`

	// Upstream configuration
	Upstream        string        `json:"upstream"`
	UpstreamTimeout time.Duration `json:"upstream_timeout"`

	// Security
	AllowedOrigins []string `json:"allowed_origins"`
	APIKey         string   `json:"api_key"`

	// Performance
	MaxConcurrent int `json:"max_concurrent"`
	RateLimit     int `json:"rate_limit"` // requests per second

	// Cleanup
	CleanupInterval time.Duration `json:"cleanup_interval"`
	MaxFileAge      time.Duration `json:"max_file_age"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Port:            8080,
		Host:            "0.0.0.0",
		Verbose:         false,
		CacheDir:        "./proxycache",
		CacheType:       "memory",
		CacheTTL:        24 * time.Hour,
		MaxCacheSize:    10 * 1024 * 1024 * 1024, // 10GB
		RedisAddr:       "localhost:6379",
		RedisPassword:   "",
		RedisDB:         0,
		Upstream:        "https://proxy.golang.org",
		UpstreamTimeout: 30 * time.Second,
		AllowedOrigins:  []string{"*"},
		APIKey:          "",
		MaxConcurrent:   100,
		RateLimit:       1000,
		CleanupInterval: 10 * time.Minute,
		MaxFileAge:      7 * 24 * time.Hour,
	}
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(path string) (*Config, error) {
	config := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return config, nil
}

// SaveConfig saves configuration to a JSON file
func (c *Config) SaveConfig(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Use restrictive permissions (0600) to prevent unauthorized access to config files
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	if c.CacheDir == "" {
		return fmt.Errorf("cache_dir cannot be empty")
	}

	if c.CacheType != "memory" && c.CacheType != "redis" && c.CacheType != "disk" {
		return fmt.Errorf("invalid cache_type: %s (must be memory, redis, or disk)", c.CacheType)
	}

	if c.Upstream == "" {
		return fmt.Errorf("upstream cannot be empty")
	}

	if c.MaxCacheSize < 1024*1024 {
		return fmt.Errorf("max_cache_size too small (minimum 1MB)")
	}

	if c.CacheTTL < 0 {
		return fmt.Errorf("cache_ttl cannot be negative")
	}

	return nil
}
