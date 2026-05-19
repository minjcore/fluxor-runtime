package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// LoadJSON loads configuration from a JSON file
// Fail-fast: Validates inputs before processing
func LoadJSON(path string, target interface{}) error {
	// Fail-fast: path cannot be empty
	if path == "" {
		return fmt.Errorf("fail-fast: JSON file path cannot be empty")
	}

	// Fail-fast: target cannot be nil
	failfast.NotNil(target, "target")

	// #nosec G304 -- path is provided by the caller (library function); callers should validate/lock down inputs if untrusted.
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("fail-fast: failed to read JSON file %s: %w", path, err)
	}

	// Fail-fast: file cannot be empty
	if len(data) == 0 {
		return fmt.Errorf("fail-fast: JSON file %s is empty", path)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("fail-fast: failed to unmarshal JSON from %s: %w", path, err)
	}

	return nil
}

// SaveJSON saves configuration to a JSON file
// Fail-fast: Validates inputs before processing
func SaveJSON(path string, config interface{}) error {
	// Fail-fast: path cannot be empty
	if path == "" {
		return fmt.Errorf("fail-fast: JSON file path cannot be empty")
	}

	// Fail-fast: config cannot be nil
	failfast.NotNil(config, "config")

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("fail-fast: failed to marshal JSON: %w", err)
	}

	// Use restrictive permissions by default since configs may contain secrets.
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("fail-fast: failed to write JSON file %s: %w", path, err)
	}

	return nil
}
