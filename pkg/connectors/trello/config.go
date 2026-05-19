package trello

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents Trello connector configuration.
type Config struct {
	config.BaseConfig

	// APIKey is the Trello API Key (required)
	APIKey string `json:"apiKey" env:"TRELLO_API_KEY" description:"Trello API Key"`

	// Token is the Trello Token (required)
	Token string `json:"token" env:"TRELLO_TOKEN" description:"Trello Token"`

	// BaseURL is the Trello API base URL (default: https://api.trello.com)
	BaseURL string `json:"baseURL" env:"TRELLO_BASE_URL" default:"https://api.trello.com" description:"Trello API base URL"`

	// Timeout for Trello API calls (default: 30s)
	Timeout string `json:"timeout" env:"TRELLO_TIMEOUT" default:"30s" description:"Timeout for Trello API calls"`

	// MaxRetries is the maximum number of retries for failed requests (default: 3)
	MaxRetries int `json:"maxRetries" env:"TRELLO_MAX_RETRIES" default:"3" description:"Maximum retries"`

	// RateLimit is the maximum requests per 10 seconds (default: 100)
	// Trello has a rate limit of 100 requests per 10 seconds
	RateLimit int `json:"rateLimit" env:"TRELLO_RATE_LIMIT" default:"100" description:"Maximum requests per 10 seconds"`
}

// DefaultConfig returns a default Trello configuration
func DefaultConfig() Config {
	return Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     os.Getenv("TRELLO_API_KEY"),
		Token:      os.Getenv("TRELLO_TOKEN"),
		BaseURL:    "https://api.trello.com",
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

	if c.APIKey == "" {
		c.APIKey = os.Getenv("TRELLO_API_KEY")
		if c.APIKey == "" {
			return &ConfigError{
				Code:    "MISSING_API_KEY",
				Message: "Trello API Key is required (set apiKey config or TRELLO_API_KEY env var)",
			}
		}
	}

	if c.Token == "" {
		c.Token = os.Getenv("TRELLO_TOKEN")
		if c.Token == "" {
			return &ConfigError{
				Code:    "MISSING_TOKEN",
				Message: "Trello Token is required (set token config or TRELLO_TOKEN env var)",
			}
		}
	}

	if c.BaseURL == "" {
		c.BaseURL = "https://api.trello.com"
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
