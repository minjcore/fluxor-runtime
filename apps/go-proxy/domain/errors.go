// Copyright (c) 2026 Fluxor Framework
// SPDX-License-Identifier: MIT

package domain

import (
	"errors"
	"fmt"
)

// Common domain errors
var (
	// ErrModuleNotFound is returned when a module is not found in storage
	ErrModuleNotFound = errors.New("module not found")

	// ErrVersionNotFound is returned when a specific version is not found
	ErrVersionNotFound = errors.New("version not found")

	// ErrInvalidModulePath is returned when a module path is invalid
	ErrInvalidModulePath = errors.New("invalid module path")

	// ErrInvalidVersion is returned when a version string is invalid
	ErrInvalidVersion = errors.New("invalid version")

	// ErrModuleExists is returned when trying to create a module that already exists
	ErrModuleExists = errors.New("module already exists")

	// ErrVersionExists is returned when trying to create a version that already exists
	ErrVersionExists = errors.New("version already exists")

	// ErrStorageUnavailable is returned when the storage backend is unavailable
	ErrStorageUnavailable = errors.New("storage unavailable")

	// ErrUnauthorized is returned when authentication fails
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned when access is denied
	ErrForbidden = errors.New("forbidden")

	// ErrInvalidZip is returned when the module zip archive is invalid
	ErrInvalidZip = errors.New("invalid zip archive")

	// ErrInvalidGoMod is returned when the go.mod file is invalid
	ErrInvalidGoMod = errors.New("invalid go.mod file")

	// ErrChecksumMismatch is returned when the checksum verification fails
	ErrChecksumMismatch = errors.New("checksum mismatch")
)

// ModuleError represents an error related to a specific module
type ModuleError struct {
	Module  string
	Version string
	Op      string
	Err     error
}

// Error returns the error message
func (e *ModuleError) Error() string {
	if e.Version != "" {
		return fmt.Sprintf("%s %s@%s: %v", e.Op, e.Module, e.Version, e.Err)
	}
	return fmt.Sprintf("%s %s: %v", e.Op, e.Module, e.Err)
}

// Unwrap returns the underlying error
func (e *ModuleError) Unwrap() error {
	return e.Err
}

// NewModuleError creates a new ModuleError
func NewModuleError(op, module, version string, err error) *ModuleError {
	return &ModuleError{
		Module:  module,
		Version: version,
		Op:      op,
		Err:     err,
	}
}

// StorageError represents an error from the storage backend
type StorageError struct {
	Backend string
	Op      string
	Key     string
	Err     error
}

// Error returns the error message
func (e *StorageError) Error() string {
	if e.Key != "" {
		return fmt.Sprintf("storage %s %s [%s]: %v", e.Backend, e.Op, e.Key, e.Err)
	}
	return fmt.Sprintf("storage %s %s: %v", e.Backend, e.Op, e.Err)
}

// Unwrap returns the underlying error
func (e *StorageError) Unwrap() error {
	return e.Err
}

// NewStorageError creates a new StorageError
func NewStorageError(backend, op, key string, err error) *StorageError {
	return &StorageError{
		Backend: backend,
		Op:      op,
		Key:     key,
		Err:     err,
	}
}

// IsNotFound checks if an error is a "not found" error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrModuleNotFound) || errors.Is(err, ErrVersionNotFound)
}

// IsInvalid checks if an error is a validation error
func IsInvalid(err error) bool {
	return errors.Is(err, ErrInvalidModulePath) ||
		errors.Is(err, ErrInvalidVersion) ||
		errors.Is(err, ErrInvalidZip) ||
		errors.Is(err, ErrInvalidGoMod)
}

// IsConflict checks if an error is a conflict error
func IsConflict(err error) bool {
	return errors.Is(err, ErrModuleExists) || errors.Is(err, ErrVersionExists)
}

// IsAuth checks if an error is an authentication/authorization error
func IsAuth(err error) bool {
	return errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrForbidden)
}
