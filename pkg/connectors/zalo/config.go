package zalo

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents Zalo connector configuration (OAuth + ZNS).
// Env vars: ZALO_ZNS_OA_ID, ZALO_ZNS_ACCESS_TOKEN, ZALO_ZNS_APP_ID, ZALO_ZNS_APP_SECRET,
// ZALO_ZNS_BASE_URL, ZALO_ZNS_TIMEOUT, ZALO_CODE_VERIFIER (optional, for PKCE).
type Config struct {
	config.BaseConfig

	OAID         string `json:"oaId" env:"ZALO_ZNS_OA_ID" description:"Zalo Official Account ID"`
	AccessToken  string `json:"accessToken" env:"ZALO_ZNS_ACCESS_TOKEN" description:"Zalo access token for ZNS APIs"`
	AppID        string `json:"appId" env:"ZALO_ZNS_APP_ID" description:"Zalo App ID"`
	AppSecret    string `json:"appSecret" env:"ZALO_ZNS_APP_SECRET" description:"Zalo App Secret"`
	CodeVerifier string `json:"codeVerifier" env:"ZALO_CODE_VERIFIER" description:"PKCE code verifier for OAuth"`
	BaseURL      string `json:"baseUrl" env:"ZALO_ZNS_BASE_URL" default:"https://business.openapi.zalo.me" description:"Zalo OpenAPI base URL"`
	OAuthURL     string `json:"oauthUrl" env:"ZALO_OAUTH_URL" default:"https://oauth.zaloapp.com/v4/oa/access_token" description:"Zalo OAuth token URL"`
	Timeout      string `json:"timeout" env:"ZALO_ZNS_TIMEOUT" default:"30s" description:"Timeout for API calls"`
}

// DefaultConfig returns default Zalo config with env fallbacks.
func DefaultConfig() Config {
	baseURL := os.Getenv("ZALO_ZNS_BASE_URL")
	if baseURL == "" {
		baseURL = "https://business.openapi.zalo.me"
	}
	oauthURL := os.Getenv("ZALO_OAUTH_URL")
	if oauthURL == "" {
		oauthURL = "https://oauth.zaloapp.com/v4/oa/access_token"
	}
	return Config{
		BaseConfig:   *config.NewBaseConfig(),
		OAID:         os.Getenv("ZALO_ZNS_OA_ID"),
		AccessToken:  os.Getenv("ZALO_ZNS_ACCESS_TOKEN"),
		AppID:        os.Getenv("ZALO_ZNS_APP_ID"),
		AppSecret:    os.Getenv("ZALO_ZNS_APP_SECRET"),
		CodeVerifier: os.Getenv("ZALO_CODE_VERIFIER"),
		BaseURL:      baseURL,
		OAuthURL:     oauthURL,
		Timeout:      "30s",
	}
}

// Validate validates the configuration.
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
	if c.OAuthURL == "" {
		c.OAuthURL = "https://oauth.zaloapp.com/v4/oa/access_token"
	}
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
			Message: "Zalo auth required: set accessToken (ZALO_ZNS_ACCESS_TOKEN) or appId+appSecret (ZALO_ZNS_APP_ID + ZALO_ZNS_APP_SECRET)",
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
