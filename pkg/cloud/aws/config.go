package aws

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents AWS client configuration.
// It embeds BaseConfig from the config package, providing common configuration
// features like service name, server settings, profile, and environment.
//
// Example usage:
//
//	// Create config with BaseConfig defaults
//	cfg := aws.DefaultConfig()
//	cfg.Region = "us-east-1"
//	cfg.Service.Name = "aws-service"
//	cfg.Server.Addr = ":8080"
//
//	// Or load from file (BaseConfig supports YAML/JSON loading)
//	var cfg aws.Config
//	if err := config.Load("aws-config.yaml", &cfg); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Validate both AWS-specific and BaseConfig fields
//	if err := cfg.Validate(); err != nil {
//	    log.Fatal(err)
//	}
type Config struct {
	// Embed BaseConfig to inherit common configuration features
	// This provides: Service, Server, Profile, Environment, and lifecycle methods
	config.BaseConfig

	// AWS Region (required, or use AWS_REGION env var)
	Region string `json:"region" env:"AWS_REGION" description:"AWS region"`

	// AWS Access Key ID (optional, uses AWS_ACCESS_KEY_ID env var or IAM role)
	AccessKeyID string `json:"accessKeyID" env:"AWS_ACCESS_KEY_ID" description:"AWS access key ID"`

	// AWS Secret Access Key (optional, uses AWS_SECRET_ACCESS_KEY env var or IAM role)
	SecretAccessKey string `json:"secretAccessKey" env:"AWS_SECRET_ACCESS_KEY" description:"AWS secret access key"`

	// AWS Session Token (optional, for temporary credentials)
	SessionToken string `json:"sessionToken" env:"AWS_SESSION_TOKEN" description:"AWS session token"`

	// Endpoint URL (optional, for local testing with LocalStack)
	Endpoint string `json:"endpoint" env:"AWS_ENDPOINT" description:"AWS endpoint URL (for LocalStack)"`

	// Timeout for AWS API calls (default: 30s)
	Timeout string `json:"timeout" env:"AWS_TIMEOUT" default:"30s" description:"Timeout for AWS API calls"`

	// Max retries for AWS API calls (default: 3)
	MaxRetries int `json:"maxRetries" env:"AWS_MAX_RETRIES" default:"3" description:"Maximum retries for AWS API calls"`

	// Enable debug logging (default: false)
	Debug bool `json:"debug" env:"AWS_DEBUG" default:"false" description:"Enable debug logging"`
}

// DefaultConfig returns a default AWS configuration with BaseConfig initialized.
// Uses environment variables: AWS_REGION, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY
// The BaseConfig is initialized with defaults from config.NewBaseConfig().
func DefaultConfig() Config {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1" // Default region
	}

	cfg := Config{
		BaseConfig:      *config.NewBaseConfig(),
		Region:          region,
		AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		SessionToken:    os.Getenv("AWS_SESSION_TOKEN"),
		Timeout:         "30s",
		MaxRetries:      3,
		Debug:           false,
	}

	return cfg
}

// Validate validates the configuration, including both AWS-specific fields
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

	// Validate AWS-specific fields
	if c.Region == "" {
		c.Region = os.Getenv("AWS_REGION")
		if c.Region == "" {
			return &ConfigError{Code: "MISSING_REGION", Message: "AWS region is required (set region config or AWS_REGION env var)"}
		}
	}

	// Access key and secret are optional if using IAM roles
	// But if one is provided, both should be provided
	if c.AccessKeyID != "" && c.SecretAccessKey == "" {
		return &ConfigError{Code: "INVALID_CREDENTIALS", Message: "AWS secret access key is required when access key ID is provided"}
	}
	if c.SecretAccessKey != "" && c.AccessKeyID == "" {
		return &ConfigError{Code: "INVALID_CREDENTIALS", Message: "AWS access key ID is required when secret access key is provided"}
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
