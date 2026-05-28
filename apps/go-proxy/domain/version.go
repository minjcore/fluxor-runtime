// Copyright (c) 2026 Fluxor Framework
// SPDX-License-Identifier: MIT

package domain

import (
	"encoding/json"
	"time"
)

// VersionInfo contains metadata about a specific module version
// This matches the JSON format expected by the Go toolchain
type VersionInfo struct {
	// Version is the version string (e.g., "v1.2.3")
	Version string `json:"Version"`

	// Time is when this version was created/published
	Time time.Time `json:"Time"`

	// Origin contains information about where this module came from
	Origin *VersionOrigin `json:"Origin,omitempty"`
}

// VersionOrigin contains origin information for a module version
type VersionOrigin struct {
	// VCS is the version control system (e.g., "git")
	VCS string `json:"VCS,omitempty"`

	// URL is the repository URL
	URL string `json:"URL,omitempty"`

	// Ref is the VCS reference (e.g., branch, tag)
	Ref string `json:"Ref,omitempty"`

	// Hash is the VCS commit hash
	Hash string `json:"Hash,omitempty"`
}

// VersionDetails contains full details about a module version
type VersionDetails struct {
	VersionInfo

	// ModulePath is the full module path
	ModulePath string `json:"modulePath"`

	// GoMod is the contents of the go.mod file
	GoMod string `json:"goMod,omitempty"`

	// Size is the size of the zip archive in bytes
	Size int64 `json:"size"`

	// Checksum is the SHA-256 checksum of the zip archive
	Checksum string `json:"checksum,omitempty"`

	// Downloads is the number of times this version has been downloaded
	Downloads int64 `json:"downloads"`

	// Deprecated indicates if this version is deprecated
	Deprecated bool `json:"deprecated,omitempty"`

	// DeprecatedMessage is the deprecation message if deprecated
	DeprecatedMessage string `json:"deprecatedMessage,omitempty"`
}

// NewVersionInfo creates a new VersionInfo with the given version string
func NewVersionInfo(version string) *VersionInfo {
	return &VersionInfo{
		Version: version,
		Time:    time.Now().UTC(),
	}
}

// ToJSON serializes the VersionInfo to JSON
func (v *VersionInfo) ToJSON() ([]byte, error) {
	return json.Marshal(v)
}

// ParseVersionInfo parses JSON into a VersionInfo
func ParseVersionInfo(data []byte) (*VersionInfo, error) {
	var info VersionInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// IsValidVersion checks if a version string is valid
// Valid versions start with 'v' followed by semver (e.g., v1.2.3)
func IsValidVersion(version string) bool {
	if len(version) == 0 {
		return false
	}
	
	// Must start with 'v'
	if version[0] != 'v' {
		return false
	}
	
	// Simple validation: must have at least v0.0.0 format
	if len(version) < 6 {
		return false
	}
	
	// Check for pre-release or build metadata
	// Format: vMAJOR.MINOR.PATCH[-prerelease][+build]
	parts := 0
	for i := 1; i < len(version); i++ {
		c := version[i]
		if c == '.' {
			parts++
		} else if c == '-' || c == '+' {
			// Pre-release or build metadata starts
			break
		} else if c < '0' || c > '9' {
			return false
		}
	}
	
	// Must have at least major.minor.patch (2 dots = 3 parts)
	return parts >= 2
}

// IsPseudoVersion checks if a version is a pseudo-version
// Pseudo-versions have the format: vX.Y.Z-yyyymmddhhmmss-abcdefabcdef
func IsPseudoVersion(version string) bool {
	if len(version) < 20 {
		return false
	}
	
	// Check for the timestamp pattern
	// Format: vX.Y.Z-0.yyyymmddhhmmss-abcdef or vX.Y.Z-yyyymmddhhmmss-abcdef
	dashCount := 0
	for _, c := range version {
		if c == '-' {
			dashCount++
		}
	}
	
	return dashCount >= 2
}

// CompareVersions compares two version strings
// Returns -1 if a < b, 0 if a == b, 1 if a > b
// This is a simplified comparison; for full semver, use golang.org/x/mod/semver
func CompareVersions(a, b string) int {
	if a == b {
		return 0
	}
	if a < b {
		return -1
	}
	return 1
}
