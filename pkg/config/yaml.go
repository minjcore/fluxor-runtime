package config

import (
	"fmt"
	"os"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
	"gopkg.in/yaml.v3"
)

// LoadYAML loads configuration from a YAML file
// Fail-fast: Validates inputs before processing
func LoadYAML(path string, target interface{}) error {
	// Fail-fast: path cannot be empty
	if path == "" {
		return fmt.Errorf("fail-fast: YAML file path cannot be empty")
	}

	// Fail-fast: target cannot be nil
	failfast.NotNil(target, "target")

	// #nosec G304 -- path is provided by the caller (library function); callers should validate/lock down inputs if untrusted.
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("fail-fast: failed to read YAML file %s: %w", path, err)
	}

	// Fail-fast: file cannot be empty
	if len(data) == 0 {
		return fmt.Errorf("fail-fast: YAML file %s is empty", path)
	}

	if err := yaml.Unmarshal(data, target); err != nil {
		return fmt.Errorf("fail-fast: failed to unmarshal YAML from %s: %w", path, err)
	}

	return nil
}

// SaveYAML saves configuration to a YAML file
// Fail-fast: Validates inputs before processing
func SaveYAML(path string, config interface{}) error {
	// Fail-fast: path cannot be empty
	if path == "" {
		return fmt.Errorf("fail-fast: YAML file path cannot be empty")
	}

	// Fail-fast: config cannot be nil
	failfast.NotNil(config, "config")

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("fail-fast: failed to marshal YAML: %w", err)
	}

	// Use restrictive permissions by default since configs may contain secrets.
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("fail-fast: failed to write YAML file %s: %w", path, err)
	}

	return nil
}
