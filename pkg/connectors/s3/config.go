package s3

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents S3 connector configuration.
// It embeds BaseConfig from the config package, providing common configuration
// features like service name, server settings, profile, and environment.
//
// Example usage:
//
//	// Create config with BaseConfig defaults
//	cfg := s3.DefaultConfig()
//	cfg.Bucket = "my-bucket"
//	cfg.Region = "us-east-1"
//	cfg.Service.Name = "s3-service"
//
//	// Or load from file (BaseConfig supports YAML/JSON loading)
//	var cfg s3.Config
//	if err := config.Load("s3-config.yaml", &cfg); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Validate both S3-specific and BaseConfig fields
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

	// Default bucket name (optional, can be specified per operation)
	Bucket string `json:"bucket" env:"S3_BUCKET" description:"Default S3 bucket name"`

	// Default content type for uploads (default: application/octet-stream)
	DefaultContentType string `json:"defaultContentType" env:"S3_DEFAULT_CONTENT_TYPE" default:"application/octet-stream" description:"Default content type for S3 uploads"`

	// Timeout for S3 API calls (default: 30s)
	Timeout string `json:"timeout" env:"S3_TIMEOUT" default:"30s" description:"Timeout for S3 API calls"`

	// Max retries for S3 API calls (default: 3)
	MaxRetries int `json:"maxRetries" env:"S3_MAX_RETRIES" default:"3" description:"Maximum retries for S3 API calls"`

	// Enable debug logging (default: false)
	Debug bool `json:"debug" env:"S3_DEBUG" default:"false" description:"Enable debug logging"`
}

// DefaultConfig returns a default S3 configuration with BaseConfig initialized.
// Uses environment variables: AWS_REGION, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, S3_BUCKET
// The BaseConfig is initialized with defaults from config.NewBaseConfig().
func DefaultConfig() Config {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1" // Default region
	}

	cfg := Config{
		BaseConfig:         *config.NewBaseConfig(),
		Region:             region,
		AccessKeyID:        os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretAccessKey:    os.Getenv("AWS_SECRET_ACCESS_KEY"),
		SessionToken:       os.Getenv("AWS_SESSION_TOKEN"),
		Bucket:             os.Getenv("S3_BUCKET"),
		DefaultContentType: "application/octet-stream",
		Timeout:            "30s",
		MaxRetries:         3,
		Debug:              false,
	}

	return cfg
}

// Validate validates the configuration, including both S3-specific fields
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
			return &ConfigError{
				Code:    "MISSING_REGION",
				Message: "AWS region is required (set region config or AWS_REGION env var)",
			}
		}
	}

	// Access key and secret are optional if using IAM roles
	// But if one is provided, both should be provided
	if c.AccessKeyID != "" && c.SecretAccessKey == "" {
		return &ConfigError{
			Code:    "INVALID_CREDENTIALS",
			Message: "AWS secret access key is required when access key ID is provided",
		}
	}
	if c.SecretAccessKey != "" && c.AccessKeyID == "" {
		return &ConfigError{
			Code:    "INVALID_CREDENTIALS",
			Message: "AWS access key ID is required when secret access key is provided",
		}
	}

	// Validate max retries
	if c.MaxRetries < 0 {
		c.MaxRetries = 3
	}

	// Validate default content type
	if c.DefaultContentType == "" {
		c.DefaultContentType = "application/octet-stream"
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

// ToAWSConfig converts S3 config to AWS config
func (c *Config) ToAWSConfig() interface{} {
	// This will be used by the component to create AWS client
	// We return a map that matches AWS config structure
	return map[string]interface{}{
		"region":          c.Region,
		"accessKeyID":     c.AccessKeyID,
		"secretAccessKey": c.SecretAccessKey,
		"sessionToken":    c.SessionToken,
		"endpoint":        c.Endpoint,
		"timeout":         c.Timeout,
		"maxRetries":      c.MaxRetries,
		"debug":           c.Debug,
	}
}

// ConfigError represents a configuration error
type ConfigError struct {
	Code    string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}
