package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type AppConfig struct {
	Server  ServerConfig  `json:"server"`
	TLS     TLSConfig     `json:"tls"`
	Metrics MetricsConfig `json:"metrics"`
	Service ServiceConfig `json:"service"`
	GitHub  GitHubConfig  `json:"github"`
}

type ServerConfig struct {
	Addr            string `json:"addr"`
	Workers         int    `json:"workers"`
	MaxQueue        int    `json:"maxQueue"`
	ReadTimeoutSec  int    `json:"readTimeoutSec"`
	WriteTimeoutSec int    `json:"writeTimeoutSec"`
	MaxConns        int    `json:"maxConns"`
	ReadBufferSize  int    `json:"readBufferSize"`
	WriteBufferSize int    `json:"writeBufferSize"`
}

type TLSConfig struct {
	Enabled  bool   `json:"enabled"`
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
	CAFile   string `json:"caFile"`
	Domain   string `json:"domain"`
}

type MetricsConfig struct {
	Enabled     bool `json:"enabled"`
	IntervalSec int  `json:"intervalSec"`
}

type ServiceConfig struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type GitHubConfig struct {
	Token     string `json:"token"`
	BaseURL   string `json:"baseURL"`
	UserAgent string `json:"userAgent"`
}

func DefaultConfig() *AppConfig {
	return &AppConfig{
		Server: ServerConfig{
			Addr:            ":8088",
			Workers:         64,
			MaxQueue:        4096,
			ReadTimeoutSec:  10,
			WriteTimeoutSec: 10,
			MaxConns:        10000,
			ReadBufferSize:  8192,
			WriteBufferSize: 8192,
		},
		TLS:     TLSConfig{Enabled: false, Domain: "localhost"},
		Metrics: MetricsConfig{Enabled: true, IntervalSec: 10},
		Service: ServiceConfig{Name: "gitshare", Version: "1.0.0"},
		GitHub:  GitHubConfig{Token: os.Getenv("GITHUB_TOKEN"), BaseURL: "https://api.github.com", UserAgent: "playfluxor-gitshare"},
	}
}

func LoadConfig(path string) (*AppConfig, error) {
	cfg := DefaultConfig()
	if path == "" {
		return cfg, nil
	}

	// Validate config file path to prevent directory traversal attacks
	if err := validateConfigPath(path); err != nil {
		return nil, fmt.Errorf("invalid config file path: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("fail-fast: read config: %w", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("fail-fast: parse config: %w", err)
	}
	return cfg, nil
}

// validateConfigPath validates the config file path to prevent directory traversal attacks.
func validateConfigPath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check for directory traversal sequences in the original path
	if strings.Contains(path, "..") {
		return fmt.Errorf("path contains directory traversal sequence")
	}

	// Resolve absolute path to normalize it
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Clean the path to resolve any remaining ".." or "." components
	cleanPath := filepath.Clean(absPath)

	// After cleaning, if the path still contains "..", it indicates
	// an attempt to traverse outside the filesystem root
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path contains directory traversal sequence")
	}

	return nil
}
