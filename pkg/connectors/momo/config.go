package momo

import (
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents MoMo Payment Gateway connector configuration.
// Credentials from M4B (MoMo for Business): Partner Code, Access Key, Secret Key.
// See https://developers.momo.vn/v3/docs/payment/onboarding/integration-process
type Config struct {
	config.BaseConfig

	// PartnerCode is the business account identifier (required).
	PartnerCode string `json:"partnerCode" env:"MOMO_PARTNER_CODE" description:"MoMo partner code"`

	// AccessKey from M4B (required).
	AccessKey string `json:"accessKey" env:"MOMO_ACCESS_KEY" description:"MoMo access key"`

	// SecretKey for HMAC_SHA256 signature (required).
	SecretKey string `json:"secretKey" env:"MOMO_SECRET_KEY" description:"MoMo secret key"`

	// BaseURL: production https://payment.momo.vn, sandbox https://test-payment.momo.vn
	BaseURL string `json:"baseURL" env:"MOMO_BASE_URL" default:"https://test-payment.momo.vn" description:"MoMo API base URL"`

	// Timeout for API calls (min 30s recommended by MoMo).
	Timeout string `json:"timeout" env:"MOMO_TIMEOUT" default:"30s" description:"API timeout"`

	// Lang for response message: "vi" or "en".
	Lang string `json:"lang" env:"MOMO_LANG" default:"vi" description:"Response language"`

	// PartnerName optional display name.
	PartnerName string `json:"partnerName" env:"MOMO_PARTNER_NAME" description:"Partner name"`

	// StoreId optional store identifier.
	StoreId string `json:"storeId" env:"MOMO_STORE_ID" description:"Store ID"`
}

// DefaultConfig returns default MoMo configuration (sandbox).
func DefaultConfig() Config {
	return Config{
		BaseConfig:   *config.NewBaseConfig(),
		PartnerCode:  os.Getenv("MOMO_PARTNER_CODE"),
		AccessKey:    os.Getenv("MOMO_ACCESS_KEY"),
		SecretKey:   os.Getenv("MOMO_SECRET_KEY"),
		BaseURL:     "https://test-payment.momo.vn",
		Timeout:     "30s",
		Lang:        "vi",
		PartnerName: os.Getenv("MOMO_PARTNER_NAME"),
		StoreId:     os.Getenv("MOMO_STORE_ID"),
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if v := c.BaseConfig.GetValidators(); len(v) > 0 {
		for _, validator := range v {
			if err := validator.Validate(c); err != nil {
				return &ConfigError{Code: "BASE_CONFIG_VALIDATION", Message: err.Error()}
			}
		}
	}

	if c.PartnerCode == "" {
		c.PartnerCode = os.Getenv("MOMO_PARTNER_CODE")
		if c.PartnerCode == "" {
			return &ConfigError{Code: "MISSING_PARTNER_CODE", Message: "MoMo partner code is required (MOMO_PARTNER_CODE)"}
		}
	}
	if c.AccessKey == "" {
		c.AccessKey = os.Getenv("MOMO_ACCESS_KEY")
		if c.AccessKey == "" {
			return &ConfigError{Code: "MISSING_ACCESS_KEY", Message: "MoMo access key is required (MOMO_ACCESS_KEY)"}
		}
	}
	if c.SecretKey == "" {
		c.SecretKey = os.Getenv("MOMO_SECRET_KEY")
		if c.SecretKey == "" {
			return &ConfigError{Code: "MISSING_SECRET_KEY", Message: "MoMo secret key is required (MOMO_SECRET_KEY)"}
		}
	}
	if c.BaseURL == "" {
		c.BaseURL = "https://test-payment.momo.vn"
	}
	if c.Timeout == "" {
		c.Timeout = "30s"
	}
	if c.Lang != "vi" && c.Lang != "en" {
		c.Lang = "vi"
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
