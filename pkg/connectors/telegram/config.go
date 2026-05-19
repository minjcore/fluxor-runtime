package telegram

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents Telegram connector configuration.
type Config struct {
	config.BaseConfig

	// BotToken is the Telegram Bot Token (required)
	BotToken string `json:"botToken" env:"TELEGRAM_BOT_TOKEN" description:"Telegram Bot Token"`

	// BaseURL is the Telegram API base URL (default: https://api.telegram.org)
	BaseURL string `json:"baseURL" env:"TELEGRAM_BASE_URL" default:"https://api.telegram.org" description:"Telegram API base URL"`

	// Timeout for Telegram API calls (default: 30s)
	Timeout string `json:"timeout" env:"TELEGRAM_TIMEOUT" default:"30s" description:"Timeout for Telegram API calls"`

	// MaxRetries is the maximum number of retries for failed requests (default: 3)
	MaxRetries int `json:"maxRetries" env:"TELEGRAM_MAX_RETRIES" default:"3" description:"Maximum retries"`

	// RateLimit is the maximum requests per second (default: 30)
	RateLimit int `json:"rateLimit" env:"TELEGRAM_RATE_LIMIT" default:"30" description:"Maximum requests per second"`
}

// DefaultConfig returns a default Telegram configuration
func DefaultConfig() Config {
	return Config{
		BaseConfig: *config.NewBaseConfig(),
		BotToken:   os.Getenv("TELEGRAM_BOT_TOKEN"),
		BaseURL:    "https://api.telegram.org",
		Timeout:    "30s",
		MaxRetries: 3,
		RateLimit:  30,
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
		c.BotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
		if c.BotToken == "" {
			return &ConfigError{
				Code:    "MISSING_BOT_TOKEN",
				Message: "Telegram Bot Token is required (set botToken config or TELEGRAM_BOT_TOKEN env var)",
			}
		}
	}

	if c.BaseURL == "" {
		c.BaseURL = "https://api.telegram.org"
	}
	if c.RateLimit <= 0 {
		c.RateLimit = 30
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
