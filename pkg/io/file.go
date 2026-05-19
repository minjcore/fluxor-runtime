package io

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
	"gopkg.in/yaml.v3"
)

// ReadFile reads the entire contents of a file.
// Fail-fast: Validates path before reading
func ReadFile(path string) ([]byte, error) {
	failfast.If(path != "", "file path cannot be empty")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return data, nil
}

// WriteFile writes data to a file, creating it if it doesn't exist.
// Fail-fast: Validates inputs before writing
func WriteFile(path string, data []byte, perm os.FileMode) error {
	failfast.If(path != "", "file path cannot be empty")
	failfast.NotNil(data, "data")

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	return nil
}

// CopyFile copies a file from src to dst.
// Fail-fast: Validates paths before copying
func CopyFile(src, dst string) error {
	failfast.If(src != "", "source path cannot be empty")
	failfast.If(dst != "", "destination path cannot be empty")

	srcData, err := ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Get source file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	if err := WriteFile(dst, srcData, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}

// MoveFile moves (renames) a file from src to dst.
// Fail-fast: Validates paths before moving
func MoveFile(src, dst string) error {
	failfast.If(src != "", "source path cannot be empty")
	failfast.If(dst != "", "destination path cannot be empty")

	// Ensure destination directory exists
	dir := filepath.Dir(dst)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("failed to move file from %s to %s: %w", src, dst, err)
	}

	return nil
}

// DeleteFile deletes a file.
// Fail-fast: Validates path before deleting
func DeleteFile(path string) error {
	failfast.If(path != "", "file path cannot be empty")

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete file %s: %w", path, err)
	}

	return nil
}

// Exists checks if a file or directory exists.
func Exists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// IsDir checks if a path is a directory.
func IsDir(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsFile checks if a path is a regular file.
func IsFile(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// MkdirAll creates a directory and all parent directories if needed.
// Fail-fast: Validates path before creating
func MkdirAll(path string, perm os.FileMode) error {
	failfast.If(path != "", "directory path cannot be empty")

	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	return nil
}

// ReadJSON reads a JSON file and unmarshals it into the target.
// Fail-fast: Validates inputs before reading
func ReadJSON(path string, target interface{}) error {
	failfast.If(path != "", "file path cannot be empty")
	failfast.NotNil(target, "target")

	data, err := ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("JSON file %s is empty", path)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON from %s: %w", path, err)
	}

	return nil
}

// WriteJSON marshals the value to JSON and writes it to a file.
// Fail-fast: Validates inputs before writing
func WriteJSON(path string, value interface{}, perm os.FileMode) error {
	failfast.If(path != "", "file path cannot be empty")
	failfast.NotNil(value, "value")

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

// ReadYAML reads a YAML file and unmarshals it into the target.
// Fail-fast: Validates inputs before reading
func ReadYAML(path string, target interface{}) error {
	failfast.If(path != "", "file path cannot be empty")
	failfast.NotNil(target, "target")

	data, err := ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read YAML file: %w", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("YAML file %s is empty", path)
	}

	if err := yaml.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal YAML from %s: %w", path, err)
	}

	return nil
}

// WriteYAML marshals the value to YAML and writes it to a file.
// Fail-fast: Validates inputs before writing
func WriteYAML(path string, value interface{}, perm os.FileMode) error {
	failfast.If(path != "", "file path cannot be empty")
	failfast.NotNil(value, "value")

	data, err := yaml.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("failed to write YAML file: %w", err)
	}

	return nil
}
