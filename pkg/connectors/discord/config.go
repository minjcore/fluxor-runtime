package discord

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents Discord connector configuration.
type Config struct {
	config.BaseConfig

	// BotToken is the Discord Bot Token (required)
	BotToken string `json:"botToken" env:"DISCORD_BOT_TOKEN" description:"Discord Bot Token"`

	// ApplicationID is the Discord Application ID
	ApplicationID string `json:"applicationID" env:"DISCORD_APPLICATION_ID" description:"Discord Application ID"`

	// Timeout for Discord API calls (default: 30s)
	Timeout string `json:"timeout" env:"DISCORD_TIMEOUT" default:"30s" description:"Timeout for Discord API calls"`

	// MaxRetries is the maximum number of retries for failed requests (default: 3)
	MaxRetries int `json:"maxRetries" env:"DISCORD_MAX_RETRIES" default:"3" description:"Maximum retries"`

	// RateLimit is the maximum requests per second (default: 50)
	RateLimit int `json:"rateLimit" env:"DISCORD_RATE_LIMIT" default:"50" description:"Maximum requests per second"`
}

// DefaultConfig returns a default Discord configuration
func DefaultConfig() Config {
	return Config{
		BaseConfig:    *config.NewBaseConfig(),
		BotToken:      os.Getenv("DISCORD_BOT_TOKEN"),
		ApplicationID: os.Getenv("DISCORD_APPLICATION_ID"),
		Timeout:       "30s",
		MaxRetries:    3,
		RateLimit:     50,
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

	if c.BotToken == "" {
		c.BotToken = os.Getenv("DISCORD_BOT_TOKEN")
		if c.BotToken == "" {
			return &ConfigError{
				Code:    "MISSING_BOT_TOKEN",
				Message: "Discord Bot Token is required (set botToken config or DISCORD_BOT_TOKEN env var)",
			}
		}
	}

	if c.RateLimit <= 0 {
		c.RateLimit = 50
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
