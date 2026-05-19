package http

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents HTTP connector configuration.
// It embeds BaseConfig from the config package, providing common configuration
// features like service name, server settings, profile, and environment.
//
// Example usage:
//
//	// Create config with BaseConfig defaults
//	cfg := http.DefaultConfig()
//	cfg.BaseURL = "https://api.example.com"
//	cfg.Service.Name = "http-service"
//
//	// Or load from file (BaseConfig supports YAML/JSON loading)
//	var cfg http.Config
//	if err := config.Load("http-config.yaml", &cfg); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Validate both HTTP-specific and BaseConfig fields
//	if err := cfg.Validate(); err != nil {
//	    log.Fatal(err)
//	}
type Config struct {
	// Embed BaseConfig to inherit common configuration features
	// This provides: Service, Server, Profile, Environment, and lifecycle methods
	config.BaseConfig

	// BaseURL is the base URL for all HTTP requests (optional)
	// If not set, full URLs must be provided in each request
	BaseURL string `json:"baseURL" env:"HTTP_BASE_URL" description:"Base URL for HTTP requests"`

	// Timeout for HTTP requests (default: 30s)
	Timeout string `json:"timeout" env:"HTTP_TIMEOUT" default:"30s" description:"Timeout for HTTP requests"`

	// MaxRetries is the maximum number of retries for failed requests (default: 3)
	MaxRetries int `json:"maxRetries" env:"HTTP_MAX_RETRIES" default:"3" description:"Maximum retries for HTTP requests"`

	// RateLimit is the maximum requests per second (default: 100)
	RateLimit float64 `json:"rateLimit" env:"HTTP_RATE_LIMIT" default:"100" description:"Maximum requests per second"`

	// DefaultHeaders are headers to include in all requests
	DefaultHeaders map[string]string `json:"defaultHeaders" description:"Default headers for all requests"`

	// AuthType specifies the authentication type
	// Options: "bearer", "basic", "apikey", "custom", "none" (default: "none")
	AuthType string `json:"authType" env:"HTTP_AUTH_TYPE" default:"none" description:"Authentication type"`

	// BearerToken for Bearer token authentication
	BearerToken string `json:"bearerToken" env:"HTTP_BEARER_TOKEN" description:"Bearer token for authentication"`

	// BasicAuthUsername for Basic authentication
	BasicAuthUsername string `json:"basicAuthUsername" env:"HTTP_BASIC_AUTH_USERNAME" description:"Basic auth username"`

	// BasicAuthPassword for Basic authentication
	BasicAuthPassword string `json:"basicAuthPassword" env:"HTTP_BASIC_AUTH_PASSWORD" description:"Basic auth password"`

	// APIKey for API key authentication
	APIKey string `json:"apiKey" env:"HTTP_API_KEY" description:"API key for authentication"`

	// APIKeyHeader is the header name for API key (default: "X-API-Key")
	APIKeyHeader string `json:"apiKeyHeader" env:"HTTP_API_KEY_HEADER" default:"X-API-Key" description:"Header name for API key"`

	// CustomAuthHeader is a custom header for authentication (e.g., "Authorization: Custom token")
	CustomAuthHeader string `json:"customAuthHeader" env:"HTTP_CUSTOM_AUTH_HEADER" description:"Custom authentication header"`

	// CustomAuthValue is the value for custom auth header
	CustomAuthValue string `json:"customAuthValue" env:"HTTP_CUSTOM_AUTH_VALUE" description:"Custom authentication header value"`

	// FollowRedirects enables following HTTP redirects (default: true)
	FollowRedirects bool `json:"followRedirects" env:"HTTP_FOLLOW_REDIRECTS" default:"true" description:"Follow HTTP redirects"`

	// MaxRedirects is the maximum number of redirects to follow (default: 10)
	MaxRedirects int `json:"maxRedirects" env:"HTTP_MAX_REDIRECTS" default:"10" description:"Maximum redirects to follow"`

	// InsecureSkipVerify skips TLS certificate verification (default: false)
	// WARNING: Only use for development/testing
	InsecureSkipVerify bool `json:"insecureSkipVerify" env:"HTTP_INSECURE_SKIP_VERIFY" default:"false" description:"Skip TLS certificate verification"`

	// EnableDebug enables debug logging (default: false)
	Debug bool `json:"debug" env:"HTTP_DEBUG" default:"false" description:"Enable debug logging"`
}

// DefaultConfig returns a default HTTP configuration with BaseConfig initialized.
// Uses environment variables: HTTP_BASE_URL, HTTP_BEARER_TOKEN, HTTP_API_KEY, etc.
// The BaseConfig is initialized with defaults from config.NewBaseConfig().
func DefaultConfig() Config {
	cfg := Config{
		BaseConfig:        *config.NewBaseConfig(),
		BaseURL:           os.Getenv("HTTP_BASE_URL"),
		Timeout:           "30s",
		MaxRetries:        3,
		RateLimit:         100.0,
		DefaultHeaders:    make(map[string]string),
		AuthType:          "none",
		BearerToken:       os.Getenv("HTTP_BEARER_TOKEN"),
		BasicAuthUsername: os.Getenv("HTTP_BASIC_AUTH_USERNAME"),
		BasicAuthPassword: os.Getenv("HTTP_BASIC_AUTH_PASSWORD"),
		APIKey:            os.Getenv("HTTP_API_KEY"),
		APIKeyHeader:      "X-API-Key",
		CustomAuthHeader:  os.Getenv("HTTP_CUSTOM_AUTH_HEADER"),
		CustomAuthValue:   os.Getenv("HTTP_CUSTOM_AUTH_VALUE"),
		FollowRedirects:   true,
		MaxRedirects:      10,
		InsecureSkipVerify: false,
		Debug:             false,
	}

	return cfg
}

// Validate validates the configuration, including both HTTP-specific fields
// and BaseConfig fields. This demonstrates how to extend BaseConfig validation.
// Fail-fast: Returns error if required fields are missing
func (c *Config) Validate() error {
	// Validate BaseConfig first (if it has validators)
	baseValidators := c.BaseConfig.GetValidators()
	for _, validator := range baseValidators {
		if err := validator.Validate(c); err != nil {
			return &ConfigError{Code: "BASE_CONFIG_VALIDATION", Message: err.Error()}
		}
	}

	// Validate timeout
	if c.Timeout == "" {
		c.Timeout = "30s"
	}

	// Validate max retries
	if c.MaxRetries < 0 {
		c.MaxRetries = 3
	}

	// Validate rate limit
	if c.RateLimit <= 0 {
		c.RateLimit = 100.0
	}

	// Validate auth configuration
	if c.AuthType == "" {
		c.AuthType = "none"
	}

	switch c.AuthType {
	case "bearer":
		if c.BearerToken == "" {
			c.BearerToken = os.Getenv("HTTP_BEARER_TOKEN")
			if c.BearerToken == "" {
				return &ConfigError{
					Code:    "MISSING_BEARER_TOKEN",
					Message: "Bearer token is required when authType is 'bearer' (set bearerToken config or HTTP_BEARER_TOKEN env var)",
				}
			}
		}
	case "basic":
		if c.BasicAuthUsername == "" {
			c.BasicAuthUsername = os.Getenv("HTTP_BASIC_AUTH_USERNAME")
		}
		if c.BasicAuthPassword == "" {
			c.BasicAuthPassword = os.Getenv("HTTP_BASIC_AUTH_PASSWORD")
		}
		if c.BasicAuthUsername == "" || c.BasicAuthPassword == "" {
			return &ConfigError{
				Code:    "MISSING_BASIC_AUTH",
				Message: "Basic auth requires both username and password (set basicAuthUsername/basicAuthPassword config or HTTP_BASIC_AUTH_USERNAME/HTTP_BASIC_AUTH_PASSWORD env vars)",
			}
		}
	case "apikey":
		if c.APIKey == "" {
			c.APIKey = os.Getenv("HTTP_API_KEY")
			if c.APIKey == "" {
				return &ConfigError{
					Code:    "MISSING_API_KEY",
					Message: "API key is required when authType is 'apikey' (set apiKey config or HTTP_API_KEY env var)",
				}
			}
		}
		if c.APIKeyHeader == "" {
			c.APIKeyHeader = "X-API-Key"
		}
	case "custom":
		if c.CustomAuthHeader == "" {
			c.CustomAuthHeader = os.Getenv("HTTP_CUSTOM_AUTH_HEADER")
		}
		if c.CustomAuthValue == "" {
			c.CustomAuthValue = os.Getenv("HTTP_CUSTOM_AUTH_VALUE")
		}
		if c.CustomAuthHeader == "" || c.CustomAuthValue == "" {
			return &ConfigError{
				Code:    "MISSING_CUSTOM_AUTH",
				Message: "Custom auth requires both header and value (set customAuthHeader/customAuthValue config or HTTP_CUSTOM_AUTH_HEADER/HTTP_CUSTOM_AUTH_VALUE env vars)",
			}
		}
	}

	// Validate max redirects
	if c.MaxRedirects <= 0 {
		c.MaxRedirects = 10
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
