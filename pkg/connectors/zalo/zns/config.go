package zns

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents Zalo ZNS connector configuration.
//
// Common env vars:
// - ZALO_ZNS_OA_ID
// - ZALO_ZNS_ACCESS_TOKEN
// - ZALO_ZNS_APP_ID
// - ZALO_ZNS_APP_SECRET
type Config struct {
	config.BaseConfig

	// OAID is the Official Account ID (optional but recommended for validation / metadata).
	OAID string `json:"oaId" env:"ZALO_ZNS_OA_ID" description:"Zalo Official Account ID"`

	// AccessToken is a Zalo access token used to call ZNS APIs (optional if you plan to mint tokens).
	AccessToken string `json:"accessToken" env:"ZALO_ZNS_ACCESS_TOKEN" description:"Zalo access token for ZNS APIs"`

	// AppID is the Zalo app id (optional; used if implementing OAuth/token minting).
	AppID string `json:"appId" env:"ZALO_ZNS_APP_ID" description:"Zalo App ID"`

	// AppSecret is the Zalo app secret (optional; used if implementing OAuth/token minting).
	AppSecret string `json:"appSecret" env:"ZALO_ZNS_APP_SECRET" description:"Zalo App Secret"`

	// BaseURL is the Zalo OpenAPI base URL (default: https://business.openapi.zalo.me).
	BaseURL string `json:"baseUrl" env:"ZALO_ZNS_BASE_URL" default:"https://business.openapi.zalo.me" description:"Zalo OpenAPI base URL"`

	// Timeout for ZNS API calls (default: 30s).
	Timeout string `json:"timeout" env:"ZALO_ZNS_TIMEOUT" default:"30s" description:"Timeout for ZNS API calls"`
}

// DefaultConfig returns a default ZNS configuration with BaseConfig initialized.
func DefaultConfig() Config {
	baseURL := os.Getenv("ZALO_ZNS_BASE_URL")
	if baseURL == "" {
		baseURL = "https://business.openapi.zalo.me"
	}
	return Config{
		BaseConfig:   *config.NewBaseConfig(),
		OAID:        os.Getenv("ZALO_ZNS_OA_ID"),
		AccessToken: os.Getenv("ZALO_ZNS_ACCESS_TOKEN"),
		AppID:       os.Getenv("ZALO_ZNS_APP_ID"),
		AppSecret:   os.Getenv("ZALO_ZNS_APP_SECRET"),
		BaseURL:     baseURL,
		Timeout:     "30s",
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
		c.BaseURL = os.Getenv("ZALO_ZNS_BASE_URL")
		if c.BaseURL == "" {
			c.BaseURL = "https://business.openapi.zalo.me"
		}
	}

	// Require at least one auth path:
	// - an access token, OR
	// - app id + secret (token minting can be implemented later)
	if c.AccessToken == "" {
		c.AccessToken = os.Getenv("ZALO_ZNS_ACCESS_TOKEN")
	}
	if c.AppID == "" {
		c.AppID = os.Getenv("ZALO_ZNS_APP_ID")
	}
	if c.AppSecret == "" {
		c.AppSecret = os.Getenv("ZALO_ZNS_APP_SECRET")
	}

	if c.AccessToken == "" && (c.AppID == "" || c.AppSecret == "") {
		return &ConfigError{
			Code:    "MISSING_AUTH",
			Message: "Zalo ZNS auth is required: provide accessToken (ZALO_ZNS_ACCESS_TOKEN) or appId+appSecret (ZALO_ZNS_APP_ID + ZALO_ZNS_APP_SECRET)",
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

