package notion

import (
	"os"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

func TestDefaultConfig(t *testing.T) {
	// Set environment variables
	os.Setenv("NOTION_API_KEY", "secret_TEST123")
	defer func() {
		os.Unsetenv("NOTION_API_KEY")
	}()

	cfg := DefaultConfig()

	if cfg.APIKey != "secret_TEST123" {
		t.Errorf("DefaultConfig() APIKey = %v, want 'secret_TEST123'", cfg.APIKey)
	}
	if cfg.BaseURL != "https://api.notion.com" {
		t.Errorf("DefaultConfig() BaseURL = %v, want 'https://api.notion.com'", cfg.BaseURL)
	}
	if cfg.Version != "2022-06-28" {
		t.Errorf("DefaultConfig() Version = %v, want '2022-06-28'", cfg.Version)
	}
	if cfg.Timeout != "30s" {
		t.Errorf("DefaultConfig() Timeout = %v, want '30s'", cfg.Timeout)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("DefaultConfig() MaxRetries = %v, want 3", cfg.MaxRetries)
	}
	if cfg.RateLimit != 3 {
		t.Errorf("DefaultConfig() RateLimit = %v, want 3", cfg.RateLimit)
	}
}

func TestConfig_Validate(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "secret_TEST123",
		BaseURL:    "https://api.notion.com",
		Version:    "2022-06-28",
		Timeout:    "30s",
		MaxRetries: 3,
		RateLimit:  3,
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}

func TestConfig_Validate_MissingAPIKey(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		BaseURL:    "https://api.notion.com",
		Version:    "2022-06-28",
		Timeout:    "30s",
		MaxRetries: 3,
		RateLimit:  3,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() expected error for missing API key, got nil")
	}

	configErr, ok := err.(*ConfigError)
	if !ok {
		t.Errorf("Validate() error type = %T, want *ConfigError", err)
	}
	if configErr.Code != "MISSING_API_KEY" {
		t.Errorf("Validate() error code = %v, want 'MISSING_API_KEY'", configErr.Code)
	}
}

func TestConfig_Validate_EnvVarFallback(t *testing.T) {
	// Set environment variables
	os.Setenv("NOTION_API_KEY", "secret_ENV123")
	defer func() {
		os.Unsetenv("NOTION_API_KEY")
	}()

	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		BaseURL:    "https://api.notion.com",
		Version:    "2022-06-28",
		Timeout:    "30s",
		MaxRetries: 3,
		RateLimit:  3,
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() unexpected error with env vars: %v", err)
	}

	if cfg.APIKey != "secret_ENV123" {
		t.Errorf("Validate() APIKey = %v, want 'secret_ENV123' from env", cfg.APIKey)
	}
}

func TestConfig_Validate_DefaultValues(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "secret_TEST123",
		// BaseURL, Version, RateLimit not set
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}

	if cfg.BaseURL != "https://api.notion.com" {
		t.Errorf("Validate() BaseURL = %v, want 'https://api.notion.com'", cfg.BaseURL)
	}
	if cfg.Version != "2022-06-28" {
		t.Errorf("Validate() Version = %v, want '2022-06-28'", cfg.Version)
	}
	if cfg.RateLimit != 3 {
		t.Errorf("Validate() RateLimit = %v, want 3", cfg.RateLimit)
	}
}

func TestConfig_Validate_InvalidRateLimit(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "secret_TEST123",
		RateLimit:  -1, // Invalid
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}

	// Should default to 3
	if cfg.RateLimit != 3 {
		t.Errorf("Validate() RateLimit = %v, want 3 (default)", cfg.RateLimit)
	}
}

func TestConfig_Validate_InvalidMaxRetries(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "secret_TEST123",
		MaxRetries: -1, // Invalid
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}

	// Should default to 3
	if cfg.MaxRetries != 3 {
		t.Errorf("Validate() MaxRetries = %v, want 3 (default)", cfg.MaxRetries)
	}
}

func TestConfig_GetTimeout(t *testing.T) {
	tests := []struct {
		name     string
		timeout  string
		expected time.Duration
	}{
		{"default", "30s", 30 * time.Second},
		{"custom", "1m", 1 * time.Minute},
		{"invalid", "invalid", 30 * time.Second},
		{"empty", "", 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				BaseConfig: *config.NewBaseConfig(),
				Timeout:    tt.timeout,
			}

			result := cfg.GetTimeout()
			if result != tt.expected {
				t.Errorf("GetTimeout() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfigError_Error(t *testing.T) {
	err := &ConfigError{
		Code:    "TEST_CODE",
		Message: "test message",
	}

	if err.Error() != "test message" {
		t.Errorf("ConfigError.Error() = %v, want 'test message'", err.Error())
	}
}
