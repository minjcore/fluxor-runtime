package vault

import (
	"fmt"
	"time"
)

// Config holds configuration for the HashiCorp Vault secret provider
type Config struct {
	// Address is the Vault server address (e.g. https://vault.example.com:8200)
	Address string

	// Token is the Vault authentication token
	Token string

	// MountPath is the KV v2 secrets engine mount path (e.g. "secret")
	MountPath string

	// Namespace is the Vault namespace (optional, for Vault Enterprise)
	Namespace string

	// Timeout is the client request timeout
	Timeout time.Duration
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		MountPath: "secret",
		Timeout:   30 * time.Second,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Address == "" {
		return &ErrVaultConfig{Field: "Address", Err: fmt.Errorf("vault address cannot be empty")}
	}
	if c.Token == "" {
		return &ErrVaultConfig{Field: "Token", Err: fmt.Errorf("vault token cannot be empty")}
	}
	if c.MountPath == "" {
		return &ErrVaultConfig{Field: "MountPath", Err: fmt.Errorf("mount path cannot be empty")}
	}
	return nil
}
