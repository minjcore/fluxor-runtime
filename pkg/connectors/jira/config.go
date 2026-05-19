package jira

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents Jira connector configuration.
// Uses environment variables: JIRA_BASE_URL, JIRA_EMAIL, JIRA_API_TOKEN.
type Config struct {
	config.BaseConfig

	// BaseURL is the Jira Cloud site URL (required), e.g. https://your-domain.atlassian.net.
	BaseURL string `json:"baseUrl" env:"JIRA_BASE_URL" description:"Jira base URL"`

	// Email is the Atlassian account email (required for basic auth with API token).
	Email string `json:"email" env:"JIRA_EMAIL" description:"Atlassian account email"`

	// APIToken is the Atlassian API token (required).
	APIToken string `json:"apiToken" env:"JIRA_API_TOKEN" description:"Atlassian API token"`

	// Timeout for Jira API calls (default: 30s).
	Timeout string `json:"timeout" env:"JIRA_TIMEOUT" default:"30s" description:"Timeout for Jira API calls"`
}

// DefaultConfig returns a default Jira configuration with BaseConfig initialized.
func DefaultConfig() Config {
	return Config{
		BaseConfig: *config.NewBaseConfig(),
		BaseURL:    os.Getenv("JIRA_BASE_URL"),
		Email:      os.Getenv("JIRA_EMAIL"),
		APIToken:   os.Getenv("JIRA_API_TOKEN"),
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

	if c.BaseURL == "" {
		c.BaseURL = os.Getenv("JIRA_BASE_URL")
		if c.BaseURL == "" {
			return &ConfigError{Code: "MISSING_BASE_URL", Message: "Jira base URL is required (set baseUrl or JIRA_BASE_URL)"}
		}
	}
	if c.Email == "" {
		c.Email = os.Getenv("JIRA_EMAIL")
		if c.Email == "" {
			return &ConfigError{Code: "MISSING_EMAIL", Message: "Jira email is required (set email or JIRA_EMAIL)"}
		}
	}
	if c.APIToken == "" {
		c.APIToken = os.Getenv("JIRA_API_TOKEN")
		if c.APIToken == "" {
			return &ConfigError{Code: "MISSING_API_TOKEN", Message: "Jira API token is required (set apiToken or JIRA_API_TOKEN)"}
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

