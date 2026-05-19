package sheets

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents Google Sheets connector configuration.
// It embeds BaseConfig from the config package, providing common configuration
// features like service name, server settings, profile, and environment.
//
// Example usage:
//
//	// Create config with BaseConfig defaults
//	cfg := sheets.DefaultConfig()
//	cfg.CredentialsPath = "/path/to/credentials.json"
//	cfg.SpreadsheetID = "your-spreadsheet-id"
//	cfg.Service.Name = "sheets-service"
//
//	// Or load from file (BaseConfig supports YAML/JSON loading)
//	var cfg sheets.Config
//	if err := config.Load("sheets-config.yaml", &cfg); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Validate both Google Sheets-specific and BaseConfig fields
//	if err := cfg.Validate(); err != nil {
//	    log.Fatal(err)
//	}
type Config struct {
	// Embed BaseConfig to inherit common configuration features
	// This provides: Service, Server, Profile, Environment, and lifecycle methods
	config.BaseConfig

	// CredentialsPath is the path to the Google service account credentials JSON file
	// (required, or use GOOGLE_APPLICATION_CREDENTIALS env var)
	CredentialsPath string `json:"credentialsPath" env:"GOOGLE_APPLICATION_CREDENTIALS" description:"Path to Google service account credentials JSON file"`

	// SpreadsheetID is the Google Sheets spreadsheet ID (required, or use GOOGLE_SHEETS_SPREADSHEET_ID env var)
	SpreadsheetID string `json:"spreadsheetID" env:"GOOGLE_SHEETS_SPREADSHEET_ID" description:"Google Sheets spreadsheet ID"`

	// Timeout for Google Sheets API calls (default: 30s)
	Timeout string `json:"timeout" env:"GOOGLE_SHEETS_TIMEOUT" default:"30s" description:"Timeout for Google Sheets API calls"`

	// MaxRetries is the maximum number of retries for failed requests (default: 3)
	MaxRetries int `json:"maxRetries" env:"GOOGLE_SHEETS_MAX_RETRIES" default:"3" description:"Maximum retries for Google Sheets API calls"`

	// RateLimit is the maximum requests per second (default: 100)
	// Google Sheets API allows up to 100 requests per 100 seconds per user
	RateLimit int `json:"rateLimit" env:"GOOGLE_SHEETS_RATE_LIMIT" default:"100" description:"Maximum requests per second"`

	// OAuth2Token is an optional OAuth2 access token (alternative to service account)
	// If provided, will be used instead of credentials file
	OAuth2Token string `json:"oauth2Token" env:"GOOGLE_SHEETS_OAUTH2_TOKEN" description:"OAuth2 access token (alternative to service account)"`
}

// DefaultConfig returns a default Google Sheets configuration with BaseConfig initialized.
// Uses environment variables: GOOGLE_APPLICATION_CREDENTIALS, GOOGLE_SHEETS_SPREADSHEET_ID
// The BaseConfig is initialized with defaults from config.NewBaseConfig().
func DefaultConfig() Config {
	cfg := Config{
		BaseConfig:      *config.NewBaseConfig(),
		CredentialsPath: os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		SpreadsheetID:   os.Getenv("GOOGLE_SHEETS_SPREADSHEET_ID"),
		Timeout:         "30s",
		MaxRetries:      3,
		RateLimit:       100,
		OAuth2Token:     os.Getenv("GOOGLE_SHEETS_OAUTH2_TOKEN"),
	}

	return cfg
}

// Validate validates the configuration, including both Google Sheets-specific fields
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

	// Validate Google Sheets-specific fields
	// Either credentials path or OAuth2 token must be provided
	if c.CredentialsPath == "" && c.OAuth2Token == "" {
		c.CredentialsPath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		c.OAuth2Token = os.Getenv("GOOGLE_SHEETS_OAUTH2_TOKEN")
		if c.CredentialsPath == "" && c.OAuth2Token == "" {
			return &ConfigError{
				Code:    "MISSING_CREDENTIALS",
				Message: "Google Sheets credentials are required (set credentialsPath config, GOOGLE_APPLICATION_CREDENTIALS env var, or oauth2Token config/GOOGLE_SHEETS_OAUTH2_TOKEN env var)",
			}
		}
	}

	// If credentials path is set, check if file exists
	if c.CredentialsPath != "" {
		if _, err := os.Stat(c.CredentialsPath); os.IsNotExist(err) {
			return &ConfigError{
				Code:    "CREDENTIALS_FILE_NOT_FOUND",
				Message: "Credentials file not found: " + c.CredentialsPath,
			}
		}
	}

	if c.SpreadsheetID == "" {
		c.SpreadsheetID = os.Getenv("GOOGLE_SHEETS_SPREADSHEET_ID")
		if c.SpreadsheetID == "" {
			return &ConfigError{
				Code:    "MISSING_SPREADSHEET_ID",
				Message: "Google Sheets spreadsheet ID is required (set spreadsheetID config or GOOGLE_SHEETS_SPREADSHEET_ID env var)",
			}
		}
	}

	// Validate rate limit
	if c.RateLimit <= 0 {
		c.RateLimit = 100
	}
	if c.RateLimit > 100 {
		return &ConfigError{
			Code:    "INVALID_RATE_LIMIT",
			Message: "Google Sheets rate limit should not exceed 100 requests per second",
		}
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
