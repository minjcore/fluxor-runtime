package hoadondientu

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Provider represents supported Vietnam e-invoice providers.
// BaseURL can override for custom or other providers (VNPT, eHoaDon, etc.).
type Provider string

const (
	// ProviderMISA meInvoice: https://doc.meinvoice.vn
	ProviderMISA Provider = "misa"
	// ProviderCustom uses BaseURL as-is (VNPT, eHoaDon, etc.)
	ProviderCustom Provider = "custom"
)

// Config represents Hóa đơn điện tử (Vietnam e-invoice) connector configuration.
// Compatible with MISA meInvoice (https://doc.meinvoice.vn) and configurable for
// other providers via BaseURL.
type Config struct {
	config.BaseConfig

	// BaseURL: MISA test https://testapi.meinvoice.vn, prod https://api.meinvoice.vn.
	// For other providers set Provider=custom and BaseURL accordingly.
	BaseURL string `json:"baseURL" env:"HOADONDIENTU_BASE_URL" default:"https://testapi.meinvoice.vn" description:"E-invoice API base URL"`

	// Token is the Bearer token for API auth. If empty, Username/Password may be used to obtain token.
	Token string `json:"token" env:"HOADONDIENTU_TOKEN" description:"Bearer token for API"`

	// Username for token-based auth (e.g. MISA meInvoice integration account).
	Username string `json:"username" env:"HOADONDIENTU_USERNAME" description:"Username for token"`

	// Password for token-based auth.
	Password string `json:"password" env:"HOADONDIENTU_PASSWORD" description:"Password for token"`

	// Provider: "misa" (default base URLs) or "custom" (use BaseURL as-is).
	Provider Provider `json:"provider" env:"HOADONDIENTU_PROVIDER" default:"misa" description:"Provider: misa | custom"`

	// Timeout for API calls (default: 30s).
	Timeout string `json:"timeout" env:"HOADONDIENTU_TIMEOUT" default:"30s" description:"API timeout"`

	// MaxRetries for failed requests (default: 2).
	MaxRetries int `json:"maxRetries" env:"HOADONDIENTU_MAX_RETRIES" default:"2" description:"Max retries"`
}

// DefaultConfig returns default config (MISA meInvoice test environment).
func DefaultConfig() Config {
	return Config{
		BaseConfig: *config.NewBaseConfig(),
		BaseURL:    "https://testapi.meinvoice.vn",
		Token:      os.Getenv("HOADONDIENTU_TOKEN"),
		Username:   os.Getenv("HOADONDIENTU_USERNAME"),
		Password:   os.Getenv("HOADONDIENTU_PASSWORD"),
		Provider:   ProviderMISA,
		Timeout:    "30s",
		MaxRetries: 2,
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	baseValidators := c.BaseConfig.GetValidators()
	for _, validator := range baseValidators {
		if err := validator.Validate(c); err != nil {
			return &ConfigError{Code: "BASE_CONFIG_VALIDATION", Message: err.Error()}
		}
	}

	if c.BaseURL == "" {
		if c.Provider == ProviderMISA {
			c.BaseURL = "https://testapi.meinvoice.vn"
		} else {
			return &ConfigError{Code: "MISSING_BASE_URL", Message: "BaseURL is required when provider is custom"}
		}
	}

	// Either Token or (Username + Password) required
	if c.Token == "" {
		c.Token = os.Getenv("HOADONDIENTU_TOKEN")
	}
	if c.Username == "" {
		c.Username = os.Getenv("HOADONDIENTU_USERNAME")
	}
	if c.Password == "" {
		c.Password = os.Getenv("HOADONDIENTU_PASSWORD")
	}
	if c.Token == "" && (c.Username == "" || c.Password == "") {
		return &ConfigError{
			Code:    "MISSING_AUTH",
			Message: "Either token or (username + password) is required (HOADONDIENTU_TOKEN or HOADONDIENTU_USERNAME/HOADONDIENTU_PASSWORD)",
		}
	}

	if c.Timeout == "" {
		c.Timeout = "30s"
	}
	if c.MaxRetries < 0 {
		c.MaxRetries = 2
	}
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

func (e *ConfigError) Error() string {
	return e.Message
}
