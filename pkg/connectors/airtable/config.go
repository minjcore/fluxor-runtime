package airtable

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents Airtable connector configuration.
// It embeds BaseConfig from the config package, providing common configuration
// features like service name, server settings, profile, and environment.
//
// Example usage:
//
//	// Create config with BaseConfig defaults
//	cfg := airtable.DefaultConfig()
//	cfg.APIKey = "keyXXXXXXXXXXXXXX"
//	cfg.BaseID = "appXXXXXXXXXXXXXX"
//	cfg.Service.Name = "airtable-service"
//
//	// Or load from file (BaseConfig supports YAML/JSON loading)
//	var cfg airtable.Config
//	if err := config.Load("airtable-config.yaml", &cfg); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Validate both Airtable-specific and BaseConfig fields
//	if err := cfg.Validate(); err != nil {
//	    log.Fatal(err)
//	}
type Config struct {
	// Embed BaseConfig to inherit common configuration features
	// This provides: Service, Server, Profile, Environment, and lifecycle methods
	config.BaseConfig

	// APIKey is the Airtable API key (required, or use AIRTABLE_API_KEY env var)
	APIKey string `json:"apiKey" env:"AIRTABLE_API_KEY" description:"Airtable API key"`

	// BaseID is the Airtable base ID (required, or use AIRTABLE_BASE_ID env var)
	BaseID string `json:"baseID" env:"AIRTABLE_BASE_ID" description:"Airtable base ID"`

	// Timeout for Airtable API calls (default: 30s)
	Timeout string `json:"timeout" env:"AIRTABLE_TIMEOUT" default:"30s" description:"Timeout for Airtable API calls"`

	// MaxRetries is the maximum number of retries for failed requests (default: 3)
	MaxRetries int `json:"maxRetries" env:"AIRTABLE_MAX_RETRIES" default:"3" description:"Maximum retries for Airtable API calls"`

	// RateLimit is the maximum requests per second (default: 5)
	// Airtable has a rate limit of 5 requests per second per base
	RateLimit int `json:"rateLimit" env:"AIRTABLE_RATE_LIMIT" default:"5" description:"Maximum requests per second"`
}

// DefaultConfig returns a default Airtable configuration with BaseConfig initialized.
// Uses environment variables: AIRTABLE_API_KEY, AIRTABLE_BASE_ID
// The BaseConfig is initialized with defaults from config.NewBaseConfig().
func DefaultConfig() Config {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     os.Getenv("AIRTABLE_API_KEY"),
		BaseID:     os.Getenv("AIRTABLE_BASE_ID"),
		Timeout:    "30s",
		MaxRetries: 3,
		RateLimit:  5,
	}

	return cfg
}

// Validate validates the configuration, including both Airtable-specific fields
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

	// Validate Airtable-specific fields
	if c.APIKey == "" {
		c.APIKey = os.Getenv("AIRTABLE_API_KEY")
		if c.APIKey == "" {
			return &ConfigError{
				Code:    "MISSING_API_KEY",
				Message: "Airtable API key is required (set apiKey config or AIRTABLE_API_KEY env var)",
			}
		}
	}

	if c.BaseID == "" {
		c.BaseID = os.Getenv("AIRTABLE_BASE_ID")
		if c.BaseID == "" {
			return &ConfigError{
				Code:    "MISSING_BASE_ID",
				Message: "Airtable base ID is required (set baseID config or AIRTABLE_BASE_ID env var)",
			}
		}
	}

	// Validate rate limit
	if c.RateLimit <= 0 {
		c.RateLimit = 5
	}
	if c.RateLimit > 5 {
		return &ConfigError{
			Code:    "INVALID_RATE_LIMIT",
			Message: "Airtable rate limit cannot exceed 5 requests per second",
		}
	}

	// Validate max retries
	if c.MaxRetries < 0 {
		c.MaxRetries = 3
	}

	return nil
}

// GetTimeout returns the timeout duration
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

// ConfigError represents a configuration error
type ConfigError struct {
	Code    string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}
