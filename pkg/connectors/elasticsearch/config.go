package elasticsearch

import (
	"os"
	"strings"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents Elasticsearch connector configuration.
// It embeds BaseConfig from the config package, providing common configuration
// features like service name, server settings, profile, and environment.
//
// Example usage:
//
//	// Create config with BaseConfig defaults
//	cfg := elasticsearch.DefaultConfig()
//	cfg.Addresses = []string{"http://localhost:9200"}
//	cfg.Service.Name = "elasticsearch-service"
//
//	// Or load from file (BaseConfig supports YAML/JSON loading)
//	var cfg elasticsearch.Config
//	if err := config.Load("elasticsearch-config.yaml", &cfg); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Validate both Elasticsearch-specific and BaseConfig fields
//	if err := cfg.Validate(); err != nil {
//	    log.Fatal(err)
//	}
type Config struct {
	// Embed BaseConfig to inherit common configuration features
	// This provides: Service, Server, Profile, Environment, and lifecycle methods
	config.BaseConfig

	// Addresses is a list of Elasticsearch node addresses (required)
	// Format: ["http://localhost:9200", "http://localhost:9201"]
	// Can also use ELASTICSEARCH_ADDRESSES env var (comma-separated)
	Addresses []string `json:"addresses" env:"ELASTICSEARCH_ADDRESSES" description:"Elasticsearch node addresses"`

	// Username for authentication (optional, or use ELASTICSEARCH_USERNAME env var)
	Username string `json:"username" env:"ELASTICSEARCH_USERNAME" description:"Elasticsearch username"`

	// Password for authentication (optional, or use ELASTICSEARCH_PASSWORD env var)
	Password string `json:"password" env:"ELASTICSEARCH_PASSWORD" description:"Elasticsearch password"`

	// APIKey for API key authentication (optional, or use ELASTICSEARCH_API_KEY env var)
	APIKey string `json:"apiKey" env:"ELASTICSEARCH_API_KEY" description:"Elasticsearch API key"`

	// CloudID for Elastic Cloud (optional, or use ELASTICSEARCH_CLOUD_ID env var)
	CloudID string `json:"cloudID" env:"ELASTICSEARCH_CLOUD_ID" description:"Elastic Cloud ID"`

	// Timeout for Elasticsearch API calls (default: 30s)
	Timeout string `json:"timeout" env:"ELASTICSEARCH_TIMEOUT" default:"30s" description:"Timeout for Elasticsearch API calls"`

	// MaxRetries is the maximum number of retries for failed requests (default: 3)
	MaxRetries int `json:"maxRetries" env:"ELASTICSEARCH_MAX_RETRIES" default:"3" description:"Maximum retries for Elasticsearch API calls"`

	// Enable debug logging (default: false)
	Debug bool `json:"debug" env:"ELASTICSEARCH_DEBUG" default:"false" description:"Enable debug logging"`

	// DefaultIndex is the default index name for operations (optional)
	DefaultIndex string `json:"defaultIndex" env:"ELASTICSEARCH_DEFAULT_INDEX" description:"Default Elasticsearch index name"`
}

// DefaultConfig returns a default Elasticsearch configuration with BaseConfig initialized.
// Uses environment variables: ELASTICSEARCH_ADDRESSES, ELASTICSEARCH_USERNAME, ELASTICSEARCH_PASSWORD
// The BaseConfig is initialized with defaults from config.NewBaseConfig().
func DefaultConfig() Config {
	addresses := []string{"http://localhost:9200"}
	if envAddrs := os.Getenv("ELASTICSEARCH_ADDRESSES"); envAddrs != "" {
		addresses = strings.Split(envAddrs, ",")
		for i, addr := range addresses {
			addresses[i] = strings.TrimSpace(addr)
		}
	}

	cfg := Config{
		BaseConfig:  *config.NewBaseConfig(),
		Addresses:   addresses,
		Username:    os.Getenv("ELASTICSEARCH_USERNAME"),
		Password:    os.Getenv("ELASTICSEARCH_PASSWORD"),
		APIKey:      os.Getenv("ELASTICSEARCH_API_KEY"),
		CloudID:     os.Getenv("ELASTICSEARCH_CLOUD_ID"),
		Timeout:     "30s",
		MaxRetries:  3,
		Debug:       false,
		DefaultIndex: os.Getenv("ELASTICSEARCH_DEFAULT_INDEX"),
	}

	return cfg
}

// Validate validates the configuration, including both Elasticsearch-specific fields
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

	// Validate Elasticsearch-specific fields
	if len(c.Addresses) == 0 {
		// Try environment variable
		if envAddrs := os.Getenv("ELASTICSEARCH_ADDRESSES"); envAddrs != "" {
			c.Addresses = strings.Split(envAddrs, ",")
			for i, addr := range c.Addresses {
				c.Addresses[i] = strings.TrimSpace(addr)
			}
		}
		if len(c.Addresses) == 0 {
			return &ConfigError{
				Code:    "MISSING_ADDRESSES",
				Message: "Elasticsearch addresses are required (set addresses config or ELASTICSEARCH_ADDRESSES env var)",
			}
		}
	}

	// Validate addresses format
	for _, addr := range c.Addresses {
		if addr == "" {
			return &ConfigError{
				Code:    "INVALID_ADDRESS",
				Message: "Elasticsearch address cannot be empty",
			}
		}
		if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
			return &ConfigError{
				Code:    "INVALID_ADDRESS",
				Message: "Elasticsearch address must start with http:// or https://",
			}
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

// ConfigError represents a configuration error
type ConfigError struct {
	Code    string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}
