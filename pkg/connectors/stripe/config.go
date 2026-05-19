package stripe

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents Stripe connector configuration.
type Config struct {
	config.BaseConfig

	// SecretKey is the Stripe Secret API Key (required)
	SecretKey string `json:"secretKey" env:"STRIPE_SECRET_KEY" description:"Stripe Secret API Key"`

	// BaseURL is the Stripe API base URL (default: https://api.stripe.com)
	BaseURL string `json:"baseURL" env:"STRIPE_BASE_URL" default:"https://api.stripe.com" description:"Stripe API base URL"`

	// Version is the Stripe API version (default: 2023-10-16)
	Version string `json:"version" env:"STRIPE_API_VERSION" default:"2023-10-16" description:"Stripe API version"`

	// Timeout for Stripe API calls (default: 30s)
	Timeout string `json:"timeout" env:"STRIPE_TIMEOUT" default:"30s" description:"Timeout for Stripe API calls"`

	// MaxRetries is the maximum number of retries for failed requests (default: 3)
	MaxRetries int `json:"maxRetries" env:"STRIPE_MAX_RETRIES" default:"3" description:"Maximum retries"`

	// RateLimit is the maximum requests per second (default: 100)
	RateLimit int `json:"rateLimit" env:"STRIPE_RATE_LIMIT" default:"100" description:"Maximum requests per second"`
}

// DefaultConfig returns a default Stripe configuration
func DefaultConfig() Config {
	return Config{
		BaseConfig: *config.NewBaseConfig(),
		SecretKey:  os.Getenv("STRIPE_SECRET_KEY"),
		BaseURL:    "https://api.stripe.com",
		Version:    "2023-10-16",
		Timeout:    "30s",
		MaxRetries: 3,
		RateLimit:  100,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	baseValidators := c.BaseConfig.GetValidators()
	for _, validator := range baseValidators {
		if err := validator.Validate(c); err != nil {
			return &ConfigError{Code: "BASE_CONFIG_VALIDATION", Message: err.Error()}
		}
	}

	if c.SecretKey == "" {
		c.SecretKey = os.Getenv("STRIPE_SECRET_KEY")
		if c.SecretKey == "" {
			return &ConfigError{
				Code:    "MISSING_SECRET_KEY",
				Message: "Stripe Secret Key is required (set secretKey config or STRIPE_SECRET_KEY env var)",
			}
		}
	}

	if c.BaseURL == "" {
		c.BaseURL = "https://api.stripe.com"
	}
	if c.Version == "" {
		c.Version = "2023-10-16"
	}
	if c.RateLimit <= 0 {
		c.RateLimit = 100
	}
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
