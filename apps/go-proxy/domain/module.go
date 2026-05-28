// Copyright (c) 2026 Fluxor Framework
// SPDX-License-Identifier: MIT

package domain

import (
	"strings"
	"time"
)

// Module represents a Go module in the registry
type Module struct {
	// Path is the module path (e.g., "github.com/example/module")
	Path string `json:"path"`

	// Versions is a list of available versions for this module
	Versions []string `json:"versions"`

	// LatestVersion is the latest version of the module
	LatestVersion string `json:"latestVersion"`

	// Description is an optional description of the module
	Description string `json:"description,omitempty"`

	// CreatedAt is when the module was first added
	CreatedAt time.Time `json:"createdAt"`

	// UpdatedAt is when the module was last updated
	UpdatedAt time.Time `json:"updatedAt"`

	// TotalDownloads is the total number of downloads across all versions
	TotalDownloads int64 `json:"totalDownloads"`
}

// ModuleSummary is a lightweight summary of a module for listing
type ModuleSummary struct {
	Path          string    `json:"path"`
	LatestVersion string    `json:"latestVersion"`
	Description   string    `json:"description,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Downloads     int64     `json:"downloads"`
}

// NewModule creates a new Module with the given path
func NewModule(path string) *Module {
	now := time.Now().UTC()
	return &Module{
		Path:      path,
		Versions:  make([]string, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddVersion adds a new version to the module
func (m *Module) AddVersion(version string) {
	// Check if version already exists
	for _, v := range m.Versions {
		if v == version {
			return
		}
	}
	m.Versions = append(m.Versions, version)
	m.UpdatedAt = time.Now().UTC()
	
	// Update latest version (simple comparison, proper semver comparison should be used)
	if m.LatestVersion == "" || version > m.LatestVersion {
		m.LatestVersion = version
	}
}

// RemoveVersion removes a version from the module
func (m *Module) RemoveVersion(version string) bool {
	for i, v := range m.Versions {
		if v == version {
			m.Versions = append(m.Versions[:i], m.Versions[i+1:]...)
			m.UpdatedAt = time.Now().UTC()
			
			// Update latest version if we removed the latest
			if m.LatestVersion == version {
				m.LatestVersion = ""
				for _, v := range m.Versions {
					if v > m.LatestVersion {
						m.LatestVersion = v
					}
				}
			}
			return true
		}
	}
	return false
}

// HasVersion checks if the module has a specific version
func (m *Module) HasVersion(version string) bool {
	for _, v := range m.Versions {
		if v == version {
			return true
		}
	}
	return false
}

// ToSummary converts a Module to a ModuleSummary
func (m *Module) ToSummary() ModuleSummary {
	return ModuleSummary{
		Path:          m.Path,
		LatestVersion: m.LatestVersion,
		Description:   m.Description,
		UpdatedAt:     m.UpdatedAt,
		Downloads:     m.TotalDownloads,
	}
}

// EscapeModulePath escapes a module path for use in URLs and file paths
// Go module paths use URL encoding with uppercase for special characters
// For example: github.com/Example/Module -> github.com/!example/!module
func EscapeModulePath(path string) string {
	var result strings.Builder
	for _, r := range path {
		if r >= 'A' && r <= 'Z' {
			// Uppercase letters are escaped with '!' prefix and lowercase
			result.WriteByte('!')
			result.WriteRune(r + 32) // Convert to lowercase
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// UnescapeModulePath unescapes a module path
func UnescapeModulePath(escaped string) string {
	var result strings.Builder
	escapeNext := false
	for _, r := range escaped {
		if r == '!' {
			escapeNext = true
			continue
		}
		if escapeNext {
			// Convert to uppercase
			if r >= 'a' && r <= 'z' {
				result.WriteRune(r - 32)
			} else {
				result.WriteRune(r)
			}
			escapeNext = false
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
