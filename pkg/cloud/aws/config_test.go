package aws

import (
	"os"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errCode string
		setup   func()
		cleanup func()
	}{
		{
			name: "valid config with region",
			config: Config{
				BaseConfig: *config.NewBaseConfig(),
				Region:     "us-east-1",
			},
			wantErr: false,
		},
		{
			name: "valid config with region from env",
			config: Config{
				BaseConfig: *config.NewBaseConfig(),
			},
			wantErr: false,
			setup: func() {
				os.Setenv("AWS_REGION", "us-west-2")
			},
			cleanup: func() {
				os.Unsetenv("AWS_REGION")
			},
		},
		{
			name: "missing region",
			config: Config{
				BaseConfig: *config.NewBaseConfig(),
			},
			wantErr: true,
			errCode: "MISSING_REGION",
			cleanup: func() {
				os.Unsetenv("AWS_REGION")
			},
		},
		{
			name: "valid config with credentials",
			config: Config{
				BaseConfig:      *config.NewBaseConfig(),
				Region:          "us-east-1",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
			},
			wantErr: false,
		},
		{
			name: "incomplete credentials - missing secret",
			config: Config{
				BaseConfig:  *config.NewBaseConfig(),
				Region:      "us-east-1",
				AccessKeyID: "test-key",
			},
			wantErr: true,
			errCode: "INVALID_CREDENTIALS",
		},
		{
			name: "incomplete credentials - missing key",
			config: Config{
				BaseConfig:      *config.NewBaseConfig(),
				Region:          "us-east-1",
				SecretAccessKey: "test-secret",
			},
			wantErr: true,
			errCode: "INVALID_CREDENTIALS",
		},
		{
			name: "valid config with session token",
			config: Config{
				BaseConfig:      *config.NewBaseConfig(),
				Region:          "us-east-1",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				SessionToken:    "test-token",
			},
			wantErr: false,
		},
		{
			name: "valid config with endpoint",
			config: Config{
				BaseConfig: *config.NewBaseConfig(),
				Region:     "us-east-1",
				Endpoint:   "http://localhost:4566",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			if tt.cleanup != nil {
				defer tt.cleanup()
			}

			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				configErr, ok := err.(*ConfigError)
				if !ok {
					t.Errorf("Config.Validate() error type = %T, want *ConfigError", err)
					return
				}
				if configErr.Code != tt.errCode {
					t.Errorf("Config.Validate() error code = %v, want %v", configErr.Code, tt.errCode)
				}
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	// Save original env vars
	originalRegion := os.Getenv("AWS_REGION")
	originalAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	originalSecret := os.Getenv("AWS_SECRET_ACCESS_KEY")
	originalToken := os.Getenv("AWS_SESSION_TOKEN")

	// Clean up after test
	defer func() {
		if originalRegion != "" {
			os.Setenv("AWS_REGION", originalRegion)
		} else {
			os.Unsetenv("AWS_REGION")
		}
		if originalAccessKey != "" {
			os.Setenv("AWS_ACCESS_KEY_ID", originalAccessKey)
		} else {
			os.Unsetenv("AWS_ACCESS_KEY_ID")
		}
		if originalSecret != "" {
			os.Setenv("AWS_SECRET_ACCESS_KEY", originalSecret)
		} else {
			os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		}
		if originalToken != "" {
			os.Setenv("AWS_SESSION_TOKEN", originalToken)
		} else {
			os.Unsetenv("AWS_SESSION_TOKEN")
		}
	}()

	tests := []struct {
		name           string
		envSetup       func()
		check          func(*testing.T, Config)
	}{
		{
			name: "default config with env vars",
			envSetup: func() {
				os.Setenv("AWS_REGION", "eu-west-1")
				os.Setenv("AWS_ACCESS_KEY_ID", "env-key")
				os.Setenv("AWS_SECRET_ACCESS_KEY", "env-secret")
				os.Setenv("AWS_SESSION_TOKEN", "env-token")
			},
			check: func(t *testing.T, cfg Config) {
				if cfg.Region != "eu-west-1" {
					t.Errorf("DefaultConfig() Region = %v, want eu-west-1", cfg.Region)
				}
				if cfg.AccessKeyID != "env-key" {
					t.Errorf("DefaultConfig() AccessKeyID = %v, want env-key", cfg.AccessKeyID)
				}
				if cfg.SecretAccessKey != "env-secret" {
					t.Errorf("DefaultConfig() SecretAccessKey = %v, want env-secret", cfg.SecretAccessKey)
				}
				if cfg.SessionToken != "env-token" {
					t.Errorf("DefaultConfig() SessionToken = %v, want env-token", cfg.SessionToken)
				}
			},
		},
		{
			name: "default config without env vars",
			envSetup: func() {
				os.Unsetenv("AWS_REGION")
				os.Unsetenv("AWS_ACCESS_KEY_ID")
				os.Unsetenv("AWS_SECRET_ACCESS_KEY")
				os.Unsetenv("AWS_SESSION_TOKEN")
			},
			check: func(t *testing.T, cfg Config) {
				if cfg.Region != "us-east-1" {
					t.Errorf("DefaultConfig() Region = %v, want us-east-1", cfg.Region)
				}
				if cfg.AccessKeyID != "" {
					t.Errorf("DefaultConfig() AccessKeyID = %v, want empty", cfg.AccessKeyID)
				}
				if cfg.SecretAccessKey != "" {
					t.Errorf("DefaultConfig() SecretAccessKey = %v, want empty", cfg.SecretAccessKey)
				}
				if cfg.Timeout != "30s" {
					t.Errorf("DefaultConfig() Timeout = %v, want 30s", cfg.Timeout)
				}
				if cfg.MaxRetries != 3 {
					t.Errorf("DefaultConfig() MaxRetries = %v, want 3", cfg.MaxRetries)
				}
				if cfg.Debug {
					t.Errorf("DefaultConfig() Debug = %v, want false", cfg.Debug)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.envSetup()
			cfg := DefaultConfig()
			tt.check(t, cfg)
		})
	}
}

func TestConfig_GetTimeout(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected time.Duration
	}{
		{
			name: "valid timeout string",
			config: Config{
				Timeout: "60s",
			},
			expected: 60 * time.Second,
		},
		{
			name: "timeout with minutes",
			config: Config{
				Timeout: "5m",
			},
			expected: 5 * time.Minute,
		},
		{
			name: "empty timeout - uses default",
			config: Config{
				Timeout: "",
			},
			expected: 30 * time.Second,
		},
		{
			name: "invalid timeout - uses default",
			config: Config{
				Timeout: "invalid",
			},
			expected: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := tt.config.GetTimeout()
			if duration != tt.expected {
				t.Errorf("Config.GetTimeout() = %v, want %v", duration, tt.expected)
			}
		})
	}
}

func TestConfigError_Error(t *testing.T) {
	err := &ConfigError{
		Code:    "TEST_ERROR",
		Message: "test error message",
	}

	msg := err.Error()
	if msg != "test error message" {
		t.Errorf("ConfigError.Error() = %v, want test error message", msg)
	}
}
