package airtable

import (
	"os"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

func TestDefaultConfig(t *testing.T) {
	// Set environment variables
	os.Setenv("AIRTABLE_API_KEY", "keyTEST123")
	os.Setenv("AIRTABLE_BASE_ID", "appTEST456")
	defer func() {
		os.Unsetenv("AIRTABLE_API_KEY")
		os.Unsetenv("AIRTABLE_BASE_ID")
	}()

	cfg := DefaultConfig()

	if cfg.APIKey != "keyTEST123" {
		t.Errorf("DefaultConfig() APIKey = %v, want 'keyTEST123'", cfg.APIKey)
	}
	if cfg.BaseID != "appTEST456" {
		t.Errorf("DefaultConfig() BaseID = %v, want 'appTEST456'", cfg.BaseID)
	}
	if cfg.Timeout != "30s" {
		t.Errorf("DefaultConfig() Timeout = %v, want '30s'", cfg.Timeout)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("DefaultConfig() MaxRetries = %v, want 3", cfg.MaxRetries)
	}
	if cfg.RateLimit != 5 {
		t.Errorf("DefaultConfig() RateLimit = %v, want 5", cfg.RateLimit)
	}
}

func TestConfig_Validate(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "keyTEST123",
		BaseID:     "appTEST456",
		Timeout:    "30s",
		MaxRetries: 3,
		RateLimit:  5,
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}

func TestConfig_Validate_MissingAPIKey(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		BaseID:     "appTEST456",
		Timeout:    "30s",
		MaxRetries: 3,
		RateLimit:  5,
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

func TestConfig_Validate_MissingBaseID(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "keyTEST123",
		Timeout:    "30s",
		MaxRetries: 3,
		RateLimit:  5,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() expected error for missing base ID, got nil")
	}

	configErr, ok := err.(*ConfigError)
	if !ok {
		t.Errorf("Validate() error type = %T, want *ConfigError", err)
	}
	if configErr.Code != "MISSING_BASE_ID" {
		t.Errorf("Validate() error code = %v, want 'MISSING_BASE_ID'", configErr.Code)
	}
}

func TestConfig_Validate_InvalidRateLimit(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "keyTEST123",
		BaseID:     "appTEST456",
		Timeout:    "30s",
		MaxRetries: 3,
		RateLimit:  10, // Exceeds Airtable limit of 5
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() expected error for invalid rate limit, got nil")
	}

	configErr, ok := err.(*ConfigError)
	if !ok {
		t.Errorf("Validate() error type = %T, want *ConfigError", err)
	}
	if configErr.Code != "INVALID_RATE_LIMIT" {
		t.Errorf("Validate() error code = %v, want 'INVALID_RATE_LIMIT'", configErr.Code)
	}
}

func TestConfig_Validate_EnvVarFallback(t *testing.T) {
	// Set environment variables
	os.Setenv("AIRTABLE_API_KEY", "keyENV123")
	os.Setenv("AIRTABLE_BASE_ID", "appENV456")
	defer func() {
		os.Unsetenv("AIRTABLE_API_KEY")
		os.Unsetenv("AIRTABLE_BASE_ID")
	}()

	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		Timeout:    "30s",
		MaxRetries: 3,
		RateLimit:  5,
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() unexpected error with env vars: %v", err)
	}

	if cfg.APIKey != "keyENV123" {
		t.Errorf("Validate() APIKey = %v, want 'keyENV123' from env", cfg.APIKey)
	}
	if cfg.BaseID != "appENV456" {
		t.Errorf("Validate() BaseID = %v, want 'appENV456' from env", cfg.BaseID)
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
