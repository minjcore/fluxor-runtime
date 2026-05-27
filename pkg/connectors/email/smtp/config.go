package smtp

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents SMTP (mail) connector configuration.
// Env vars: SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD (or SMTP_PASS), SMTP_FROM, SMTP_FROM_NAME, SMTP_USE_TLS, SMTP_TIMEOUT.
type Config struct {
	config.BaseConfig

	Host     string `json:"host" env:"SMTP_HOST" description:"SMTP server host"`
	Port     int    `json:"port" env:"SMTP_PORT" description:"SMTP server port (default 587)"`
	User     string `json:"user" env:"SMTP_USER" description:"SMTP username"`
	Password string `json:"password" env:"SMTP_PASSWORD" description:"SMTP password"`
	From     string `json:"from" env:"SMTP_FROM" description:"From email address"`
	FromName string `json:"fromName" env:"SMTP_FROM_NAME" description:"From display name"`
	UseTLS   bool   `json:"useTls" env:"SMTP_USE_TLS" description:"Use TLS (default true for port 587)"`
	Timeout  string `json:"timeout" env:"SMTP_TIMEOUT" default:"30s" description:"Timeout for send"`
}

// DefaultConfig returns default SMTP config with env fallbacks.
func DefaultConfig() Config {
	port := 587
	if v := os.Getenv("SMTP_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			port = p
		}
	}
	useTLS := true
	if v := strings.ToLower(os.Getenv("SMTP_USE_TLS")); v == "false" || v == "0" {
		useTLS = false
	}
	return Config{
		BaseConfig: *config.NewBaseConfig(),
		Host:       os.Getenv("SMTP_HOST"),
		Port:       port,
		User:       os.Getenv("SMTP_USER"),
		Password:   getEnv("SMTP_PASSWORD", os.Getenv("SMTP_PASS")),
		From:       os.Getenv("SMTP_FROM"),
		FromName:   getEnv("SMTP_FROM_NAME", "Fluxor Mail"),
		UseTLS:     useTLS,
		Timeout:    getEnv("SMTP_TIMEOUT", "30s"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	for _, validator := range c.BaseConfig.GetValidators() {
		if err := validator.Validate(c); err != nil {
			return &ConfigError{Code: "BASE_CONFIG_VALIDATION", Message: err.Error()}
		}
	}
	if c.Host == "" {
		c.Host = os.Getenv("SMTP_HOST")
	}
	if c.Port <= 0 {
		c.Port = 587
	}
	if c.FromName == "" {
		c.FromName = "Fluxor Mail"
	}
	if c.Timeout == "" {
		c.Timeout = "30s"
	}
	_ = c.GetTimeout()
	return nil
}

// GetTimeout returns the timeout duration.
func (c *Config) GetTimeout() time.Duration {
	if c.Timeout == "" {
		return 30 * time.Second
	}
	d, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return 30 * time.Second
	}
	return d
}

// ConfigError represents a configuration error.
type ConfigError struct {
	Code    string
	Message string
}

func (e *ConfigError) Error() string { return e.Message }
