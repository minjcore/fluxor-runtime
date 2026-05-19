package notion

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents Notion connector configuration.
type Config struct {
	config.BaseConfig

	// APIKey is the Notion Integration Token (required)
	APIKey string `json:"apiKey" env:"NOTION_API_KEY" description:"Notion Integration Token"`

	// BaseURL is the Notion API base URL (default: https://api.notion.com)
	BaseURL string `json:"baseURL" env:"NOTION_BASE_URL" default:"https://api.notion.com" description:"Notion API base URL"`

	// Version is the Notion API version (default: 2022-06-28)
	Version string `json:"version" env:"NOTION_API_VERSION" default:"2022-06-28" description:"Notion API version"`

	// Timeout for Notion API calls (default: 30s)
	Timeout string `json:"timeout" env:"NOTION_TIMEOUT" default:"30s" description:"Timeout for Notion API calls"`

	// MaxRetries is the maximum number of retries for failed requests (default: 3)
	MaxRetries int `json:"maxRetries" env:"NOTION_MAX_RETRIES" default:"3" description:"Maximum retries"`

	// RateLimit is the maximum requests per second (default: 3)
	// Notion has a rate limit of 3 requests per second
	RateLimit int `json:"rateLimit" env:"NOTION_RATE_LIMIT" default:"3" description:"Maximum requests per second"`
}

// DefaultConfig returns a default Notion configuration
func DefaultConfig() Config {
	return Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     os.Getenv("NOTION_API_KEY"),
		BaseURL:    "https://api.notion.com",
		Version:    "2022-06-28",
		Timeout:    "30s",
		MaxRetries: 3,
		RateLimit:  3,
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
		c.APIKey = os.Getenv("NOTION_API_KEY")
		if c.APIKey == "" {
			return &ConfigError{
				Code:    "MISSING_API_KEY",
				Message: "Notion API Key is required (set apiKey config or NOTION_API_KEY env var)",
			}
		}
	}

	if c.BaseURL == "" {
		c.BaseURL = "https://api.notion.com"
	}
	if c.Version == "" {
		c.Version = "2022-06-28"
	}
	if c.RateLimit <= 0 {
		c.RateLimit = 3
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
