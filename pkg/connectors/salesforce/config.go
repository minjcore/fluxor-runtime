package salesforce

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents Salesforce connector configuration.
// Uses environment variables: SALESFORCE_INSTANCE_URL, SALESFORCE_CLIENT_ID,
// SALESFORCE_CLIENT_SECRET, SALESFORCE_REFRESH_TOKEN.
type Config struct {
	config.BaseConfig

	// InstanceURL is the Salesforce instance URL (required), e.g. https://your-instance.my.salesforce.com.
	InstanceURL string `json:"instanceUrl" env:"SALESFORCE_INSTANCE_URL" description:"Salesforce instance URL"`

	// ClientID is the OAuth client id (required).
	ClientID string `json:"clientId" env:"SALESFORCE_CLIENT_ID" description:"Salesforce OAuth client id"`

	// ClientSecret is the OAuth client secret (required).
	ClientSecret string `json:"clientSecret" env:"SALESFORCE_CLIENT_SECRET" description:"Salesforce OAuth client secret"`

	// RefreshToken is the OAuth refresh token (required).
	RefreshToken string `json:"refreshToken" env:"SALESFORCE_REFRESH_TOKEN" description:"Salesforce OAuth refresh token"`

	// Timeout for Salesforce API calls (default: 30s).
	Timeout string `json:"timeout" env:"SALESFORCE_TIMEOUT" default:"30s" description:"Timeout for Salesforce API calls"`
}

// DefaultConfig returns a default Salesforce configuration with BaseConfig initialized.
func DefaultConfig() Config {
	return Config{
		BaseConfig:   *config.NewBaseConfig(),
		InstanceURL:  os.Getenv("SALESFORCE_INSTANCE_URL"),
		ClientID:     os.Getenv("SALESFORCE_CLIENT_ID"),
		ClientSecret: os.Getenv("SALESFORCE_CLIENT_SECRET"),
		RefreshToken: os.Getenv("SALESFORCE_REFRESH_TOKEN"),
		Timeout:      "30s",
	}
}

// Validate validates the configuration (fail-fast).
func (c *Config) Validate() error {
	for _, validator := range c.BaseConfig.GetValidators() {
		if err := validator.Validate(c); err != nil {
			return &ConfigError{Code: "BASE_CONFIG_VALIDATION", Message: err.Error()}
		}
	}

	if c.InstanceURL == "" {
		c.InstanceURL = os.Getenv("SALESFORCE_INSTANCE_URL")
		if c.InstanceURL == "" {
			return &ConfigError{Code: "MISSING_INSTANCE_URL", Message: "Salesforce instance URL is required (set instanceUrl or SALESFORCE_INSTANCE_URL)"}
		}
	}
	if c.ClientID == "" {
		c.ClientID = os.Getenv("SALESFORCE_CLIENT_ID")
		if c.ClientID == "" {
			return &ConfigError{Code: "MISSING_CLIENT_ID", Message: "Salesforce client id is required (set clientId or SALESFORCE_CLIENT_ID)"}
		}
	}
	if c.ClientSecret == "" {
		c.ClientSecret = os.Getenv("SALESFORCE_CLIENT_SECRET")
		if c.ClientSecret == "" {
			return &ConfigError{Code: "MISSING_CLIENT_SECRET", Message: "Salesforce client secret is required (set clientSecret or SALESFORCE_CLIENT_SECRET)"}
		}
	}
	if c.RefreshToken == "" {
		c.RefreshToken = os.Getenv("SALESFORCE_REFRESH_TOKEN")
		if c.RefreshToken == "" {
			return &ConfigError{Code: "MISSING_REFRESH_TOKEN", Message: "Salesforce refresh token is required (set refreshToken or SALESFORCE_REFRESH_TOKEN)"}
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

