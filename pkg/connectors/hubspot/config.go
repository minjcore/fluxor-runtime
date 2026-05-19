package hubspot

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents HubSpot connector configuration.
// Uses environment variables: HUBSPOT_PRIVATE_APP_TOKEN.
type Config struct {
	config.BaseConfig

	// PrivateAppToken is the HubSpot private app token (required).
	PrivateAppToken string `json:"privateAppToken" env:"HUBSPOT_PRIVATE_APP_TOKEN" description:"HubSpot private app token"`

	// Timeout for HubSpot API calls (default: 30s).
	Timeout string `json:"timeout" env:"HUBSPOT_TIMEOUT" default:"30s" description:"Timeout for HubSpot API calls"`
}

// DefaultConfig returns a default HubSpot configuration with BaseConfig initialized.
func DefaultConfig() Config {
	return Config{
		BaseConfig:       *config.NewBaseConfig(),
		PrivateAppToken:  os.Getenv("HUBSPOT_PRIVATE_APP_TOKEN"),
		Timeout:          "30s",
	}
}

// Validate validates the configuration (fail-fast).
func (c *Config) Validate() error {
	for _, validator := range c.BaseConfig.GetValidators() {
		if err := validator.Validate(c); err != nil {
			return &ConfigError{Code: "BASE_CONFIG_VALIDATION", Message: err.Error()}
		}
	}

	if c.PrivateAppToken == "" {
		c.PrivateAppToken = os.Getenv("HUBSPOT_PRIVATE_APP_TOKEN")
		if c.PrivateAppToken == "" {
			return &ConfigError{Code: "MISSING_PRIVATE_APP_TOKEN", Message: "HubSpot private app token is required (set privateAppToken or HUBSPOT_PRIVATE_APP_TOKEN)"}
		}
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

