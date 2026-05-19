package slack

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents Slack connector configuration.
// It embeds BaseConfig from the config package, providing common configuration
// features like service name, server settings, profile, and environment.
type Config struct {
	// Embed BaseConfig to inherit common configuration features
	config.BaseConfig

	// BotToken is the Slack Bot OAuth Token (required, or use SLACK_BOT_TOKEN env var)
	// Starts with "xoxb-"
	BotToken string `json:"botToken" env:"SLACK_BOT_TOKEN" description:"Slack Bot OAuth Token"`

	// AppToken is the Slack App-Level Token (optional, for Socket Mode)
	// Starts with "xapp-"
	AppToken string `json:"appToken" env:"SLACK_APP_TOKEN" description:"Slack App-Level Token for Socket Mode"`

	// SigningSecret is used to verify requests from Slack
	SigningSecret string `json:"signingSecret" env:"SLACK_SIGNING_SECRET" description:"Slack Signing Secret"`

	// Timeout for Slack API calls (default: 30s)
	Timeout string `json:"timeout" env:"SLACK_TIMEOUT" default:"30s" description:"Timeout for Slack API calls"`

	// MaxRetries is the maximum number of retries for failed requests (default: 3)
	MaxRetries int `json:"maxRetries" env:"SLACK_MAX_RETRIES" default:"3" description:"Maximum retries for Slack API calls"`

	// RateLimit is the maximum requests per second (default: 50)
	// Slack has tiered rate limits, typically around 50 requests per minute for most methods
	RateLimit int `json:"rateLimit" env:"SLACK_RATE_LIMIT" default:"50" description:"Maximum requests per minute"`
}

// DefaultConfig returns a default Slack configuration with BaseConfig initialized.
func DefaultConfig() Config {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		BotToken:   os.Getenv("SLACK_BOT_TOKEN"),
		AppToken:   os.Getenv("SLACK_APP_TOKEN"),
		SigningSecret: os.Getenv("SLACK_SIGNING_SECRET"),
		Timeout:    "30s",
		MaxRetries: 3,
		RateLimit:  50,
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

	// Validate Slack-specific fields
	if c.BotToken == "" {
		c.BotToken = os.Getenv("SLACK_BOT_TOKEN")
		if c.BotToken == "" {
			return &ConfigError{
				Code:    "MISSING_BOT_TOKEN",
				Message: "Slack Bot Token is required (set botToken config or SLACK_BOT_TOKEN env var)",
			}
		}
	}

	// Validate rate limit
	if c.RateLimit <= 0 {
		c.RateLimit = 50
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
