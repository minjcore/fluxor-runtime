package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// LoadWithProfile loads configuration with profile support
// Supports application-{profile}.properties files
// Profile selection via APP_PROFILE environment variable
// Fail-fast: Validates inputs before processing
func LoadWithProfile(basePath string, prefix string, target interface{}) error {
	// Fail-fast: basePath cannot be empty
	if basePath == "" {
		return fmt.Errorf("fail-fast: base config file path cannot be empty")
	}

	// Fail-fast: prefix cannot be empty
	if prefix == "" {
		return fmt.Errorf("fail-fast: environment variable prefix cannot be empty")
	}

	// Fail-fast: target cannot be nil
	failfast.NotNil(target, "target")

	// Get profile from environment variable
	profile := os.Getenv(prefix + "_PROFILE")
	if profile == "" {
		profile = "default"
	}

	// Determine config file path
	var configPath string
	if profile == "default" {
		configPath = basePath
	} else {
		// Try application-{profile}.properties
		ext := ""
		if strings.Contains(basePath, ".") {
			parts := strings.Split(basePath, ".")
			ext = "." + parts[len(parts)-1]
			basePath = strings.Join(parts[:len(parts)-1], ".")
		}
		configPath = fmt.Sprintf("%s-%s%s", basePath, profile, ext)
	}

	// Try to load profile-specific file
	if err := LoadProperties(configPath, target); err != nil {
		// If profile file doesn't exist, try base file
		if profile != "default" {
			if err2 := LoadProperties(basePath, target); err2 != nil {
				return fmt.Errorf("failed to load config: profile file %s: %v, base file %s: %v", configPath, err, basePath, err2)
			}
		} else {
			return err
		}
	}

	return nil
}
