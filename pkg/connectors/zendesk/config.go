package zendesk

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents Zendesk connector configuration.
// Uses environment variables: ZENDESK_SUBDOMAIN, ZENDESK_EMAIL, ZENDESK_API_TOKEN.
type Config struct {
	config.BaseConfig

	// Subdomain is the Zendesk subdomain (required), e.g. "mycompany" for https://mycompany.zendesk.com.
	Subdomain string `json:"subdomain" env:"ZENDESK_SUBDOMAIN" description:"Zendesk subdomain"`

	// Email is the Zendesk account email (required for API token auth).
	Email string `json:"email" env:"ZENDESK_EMAIL" description:"Zendesk account email"`

	// APIToken is the Zendesk API token (required).
	APIToken string `json:"apiToken" env:"ZENDESK_API_TOKEN" description:"Zendesk API token"`

	// Timeout for Zendesk API calls (default: 30s).
	Timeout string `json:"timeout" env:"ZENDESK_TIMEOUT" default:"30s" description:"Timeout for Zendesk API calls"`
}

// DefaultConfig returns a default Zendesk configuration with BaseConfig initialized.
func DefaultConfig() Config {
	return Config{
		BaseConfig: *config.NewBaseConfig(),
		Subdomain:  os.Getenv("ZENDESK_SUBDOMAIN"),
		Email:      os.Getenv("ZENDESK_EMAIL"),
		APIToken:   os.Getenv("ZENDESK_API_TOKEN"),
		Timeout:    "30s",
	}
}

// Validate validates the configuration (fail-fast).
func (c *Config) Validate() error {
	for _, validator := range c.BaseConfig.GetValidators() {
		if err := validator.Validate(c); err != nil {
			return &ConfigError{Code: "BASE_CONFIG_VALIDATION", Message: err.Error()}
		}
	}

	if c.Subdomain == "" {
		c.Subdomain = os.Getenv("ZENDESK_SUBDOMAIN")
		if c.Subdomain == "" {
			return &ConfigError{Code: "MISSING_SUBDOMAIN", Message: "Zendesk subdomain is required (set subdomain or ZENDESK_SUBDOMAIN)"}
		}
	}
	if c.Email == "" {
		c.Email = os.Getenv("ZENDESK_EMAIL")
		if c.Email == "" {
			return &ConfigError{Code: "MISSING_EMAIL", Message: "Zendesk email is required (set email or ZENDESK_EMAIL)"}
		}
	}
	if c.APIToken == "" {
		c.APIToken = os.Getenv("ZENDESK_API_TOKEN")
		if c.APIToken == "" {
			return &ConfigError{Code: "MISSING_API_TOKEN", Message: "Zendesk API token is required (set apiToken or ZENDESK_API_TOKEN)"}
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

