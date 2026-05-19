package file

import (
	"fmt"
	"time"
)

// Config holds configuration for the file-based secret provider
type Config struct {
	// Path is the path to the secrets file
	Path string

	// Format specifies the file format: "yaml", "json", or "properties"
	// If empty, format is auto-detected from file extension
	Format string

	// KeyPrefix is an optional prefix to prepend to all secret keys
	// For example, if KeyPrefix is "secrets.", then "db.password" becomes "secrets.db.password"
	KeyPrefix string

	// DefaultPermissions sets the file permissions when creating new files
	// Default is 0600 (read/write for owner only)
	DefaultPermissions uint32

	// WatchInterval enables automatic file watching and reloading
	// If set to a positive duration, the provider will watch the file for changes
	// and automatically reload secrets when the file is modified
	// Set to 0 to disable watching (default)
	WatchInterval time.Duration
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Format:             "",
		KeyPrefix:          "",
		DefaultPermissions: 0600,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Path == "" {
		return &ErrFileNotReadable{Path: "", Err: fmt.Errorf("file path cannot be empty")}
	}

	if c.Format != "" && c.Format != "yaml" && c.Format != "json" && c.Format != "properties" {
		return &ErrInvalidFileFormat{
			Path:   c.Path,
			Format: c.Format,
			Err:    fmt.Errorf("unsupported format: %s (supported: yaml, json, properties)", c.Format),
		}
	}

	return nil
}
