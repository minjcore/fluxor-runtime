package github

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents GitHub connector configuration.
type Config struct {
	// Embed BaseConfig to inherit common configuration features
	config.BaseConfig

	// Token is the GitHub Personal Access Token or OAuth Token (required)
	Token string `json:"token" env:"GITHUB_TOKEN" description:"GitHub Personal Access Token"`

	// BaseURL is the GitHub API base URL (default: https://api.github.com)
	// Change this for GitHub Enterprise
	BaseURL string `json:"baseURL" env:"GITHUB_BASE_URL" default:"https://api.github.com" description:"GitHub API base URL"`

	// UploadURL is the GitHub upload URL (default: https://uploads.github.com)
	UploadURL string `json:"uploadURL" env:"GITHUB_UPLOAD_URL" default:"https://uploads.github.com" description:"GitHub upload URL"`

	// Timeout for GitHub API calls (default: 30s)
	Timeout string `json:"timeout" env:"GITHUB_TIMEOUT" default:"30s" description:"Timeout for GitHub API calls"`

	// MaxRetries is the maximum number of retries for failed requests (default: 3)
	MaxRetries int `json:"maxRetries" env:"GITHUB_MAX_RETRIES" default:"3" description:"Maximum retries for GitHub API calls"`

	// RateLimit is the maximum requests per hour (default: 5000)
	// GitHub has a rate limit of 5000 requests per hour for authenticated requests
	RateLimit int `json:"rateLimit" env:"GITHUB_RATE_LIMIT" default:"5000" description:"Maximum requests per hour"`
}

// DefaultConfig returns a default GitHub configuration with BaseConfig initialized.
func DefaultConfig() Config {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		Token:      os.Getenv("GITHUB_TOKEN"),
		BaseURL:    "https://api.github.com",
		UploadURL:  "https://uploads.github.com",
		Timeout:    "30s",
		MaxRetries: 3,
		RateLimit:  5000,
	}

	if envBaseURL := os.Getenv("GITHUB_BASE_URL"); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	}

	return cfg
}

// Validate validates the configuration.
// Fail-fast: Returns error if required fields are missing
func (c *Config) Validate() error {
	// Validate BaseConfig first
	baseValidators := c.BaseConfig.GetValidators()
	for _, validator := range baseValidators {
		if err := validator.Validate(c); err != nil {
			return &ConfigError{Code: "BASE_CONFIG_VALIDATION", Message: err.Error()}
		}
	}

	// Validate GitHub-specific fields
	if c.Token == "" {
		c.Token = os.Getenv("GITHUB_TOKEN")
		if c.Token == "" {
			return &ConfigError{
				Code:    "MISSING_TOKEN",
				Message: "GitHub Token is required (set token config or GITHUB_TOKEN env var)",
			}
		}
	}

	// Set defaults
	if c.BaseURL == "" {
		c.BaseURL = "https://api.github.com"
	}
	if c.UploadURL == "" {
		c.UploadURL = "https://uploads.github.com"
	}

	// Validate rate limit
	if c.RateLimit <= 0 {
		c.RateLimit = 5000
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
