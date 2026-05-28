// Copyright (c) 2026 Fluxor Framework
// SPDX-License-Identifier: MIT

package storage

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/fluxorio/goproxy/domain"
)

// MetadataIndex provides fast in-memory search and lookup for modules
// This is synchronized with the persistent storage backend
type MetadataIndex struct {
	mu sync.RWMutex

	// modules maps module path to module metadata
	modules map[string]*ModuleMetadata

	// versionDownloads tracks download counts per version
	versionDownloads map[string]map[string]int64

	// lastUpdated is when the index was last modified
	lastUpdated time.Time
}

// ModuleMetadata contains cached metadata for a module
type ModuleMetadata struct {
	Path           string    `json:"path"`
	Description    string    `json:"description,omitempty"`
	LatestVersion  string    `json:"latestVersion"`
	Versions       []string  `json:"versions"`
	TotalDownloads int64     `json:"totalDownloads"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// NewMetadataIndex creates a new metadata index
func NewMetadataIndex() *MetadataIndex {
	return &MetadataIndex{
		modules:          make(map[string]*ModuleMetadata),
		versionDownloads: make(map[string]map[string]int64),
		lastUpdated:      time.Now().UTC(),
	}
}

// AddModule adds or updates a module in the index
func (idx *MetadataIndex) AddModule(module *domain.Module) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.modules[module.Path] = &ModuleMetadata{
		Path:           module.Path,
		Description:    module.Description,
		LatestVersion:  module.LatestVersion,
		Versions:       module.Versions,
		TotalDownloads: module.TotalDownloads,
		CreatedAt:      module.CreatedAt,
		UpdatedAt:      module.UpdatedAt,
	}
	idx.lastUpdated = time.Now().UTC()
}

// RemoveModule removes a module from the index
func (idx *MetadataIndex) RemoveModule(path string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	delete(idx.modules, path)
	delete(idx.versionDownloads, path)
	idx.lastUpdated = time.Now().UTC()
}

// GetModule returns metadata for a module
func (idx *MetadataIndex) GetModule(path string) (*ModuleMetadata, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	mod, ok := idx.modules[path]
	return mod, ok
}

// AddVersion adds a version to a module
func (idx *MetadataIndex) AddVersion(modulePath, version string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	mod, ok := idx.modules[modulePath]
	if !ok {
		mod = &ModuleMetadata{
			Path:      modulePath,
			Versions:  make([]string, 0),
			CreatedAt: time.Now().UTC(),
		}
		idx.modules[modulePath] = mod
	}

	// Check if version already exists
	for _, v := range mod.Versions {
		if v == version {
			return
		}
	}

	mod.Versions = append(mod.Versions, version)
	mod.UpdatedAt = time.Now().UTC()

	// Update latest version (simple comparison)
	if mod.LatestVersion == "" || version > mod.LatestVersion {
		mod.LatestVersion = version
	}

	idx.lastUpdated = time.Now().UTC()
}

// RemoveVersion removes a version from a module
func (idx *MetadataIndex) RemoveVersion(modulePath, version string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	mod, ok := idx.modules[modulePath]
	if !ok {
		return
	}

	for i, v := range mod.Versions {
		if v == version {
			mod.Versions = append(mod.Versions[:i], mod.Versions[i+1:]...)
			break
		}
	}

	// Update latest version
	if mod.LatestVersion == version {
		mod.LatestVersion = ""
		for _, v := range mod.Versions {
			if v > mod.LatestVersion {
				mod.LatestVersion = v
			}
		}
	}

	mod.UpdatedAt = time.Now().UTC()
	idx.lastUpdated = time.Now().UTC()

	// Remove module if no versions left
	if len(mod.Versions) == 0 {
		delete(idx.modules, modulePath)
	}
}

// IncrementDownloads increments download count for a version
func (idx *MetadataIndex) IncrementDownloads(modulePath, version string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Update module total
	if mod, ok := idx.modules[modulePath]; ok {
		mod.TotalDownloads++
	}

	// Update version count
	if idx.versionDownloads[modulePath] == nil {
		idx.versionDownloads[modulePath] = make(map[string]int64)
	}
	idx.versionDownloads[modulePath][version]++
}

// GetDownloads returns download count for a version
func (idx *MetadataIndex) GetDownloads(modulePath, version string) int64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if versions, ok := idx.versionDownloads[modulePath]; ok {
		return versions[version]
	}
	return 0
}

// Search searches for modules matching a query
func (idx *MetadataIndex) Search(query string, limit int) []domain.ModuleSummary {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	results := make([]domain.ModuleSummary, 0, limit)
	query = normalizeQuery(query)

	for _, mod := range idx.modules {
		if matchesQuery(mod.Path, mod.Description, query) {
			results = append(results, domain.ModuleSummary{
				Path:          mod.Path,
				LatestVersion: mod.LatestVersion,
				Description:   mod.Description,
				UpdatedAt:     mod.UpdatedAt,
				Downloads:     mod.TotalDownloads,
			})
			if len(results) >= limit {
				break
			}
		}
	}

	return results
}

// List lists all modules with optional prefix filter
func (idx *MetadataIndex) List(prefix string, limit int) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	results := make([]string, 0, limit)

	for path := range idx.modules {
		if prefix == "" || hasPrefix(path, prefix) {
			results = append(results, path)
			if len(results) >= limit {
				break
			}
		}
	}

	return results
}

// Count returns the total number of modules
func (idx *MetadataIndex) Count() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.modules)
}

// LastUpdated returns when the index was last modified
func (idx *MetadataIndex) LastUpdated() time.Time {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.lastUpdated
}

// ToJSON serializes the index to JSON
func (idx *MetadataIndex) ToJSON() ([]byte, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	data := struct {
		Modules          map[string]*ModuleMetadata      `json:"modules"`
		VersionDownloads map[string]map[string]int64     `json:"versionDownloads"`
		LastUpdated      time.Time                       `json:"lastUpdated"`
	}{
		Modules:          idx.modules,
		VersionDownloads: idx.versionDownloads,
		LastUpdated:      idx.lastUpdated,
	}

	return json.Marshal(data)
}

// FromJSON deserializes the index from JSON
func (idx *MetadataIndex) FromJSON(data []byte) error {
	var parsed struct {
		Modules          map[string]*ModuleMetadata  `json:"modules"`
		VersionDownloads map[string]map[string]int64 `json:"versionDownloads"`
		LastUpdated      time.Time                   `json:"lastUpdated"`
	}

	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.modules = parsed.Modules
	idx.versionDownloads = parsed.VersionDownloads
	idx.lastUpdated = parsed.LastUpdated

	// Initialize nil maps
	if idx.modules == nil {
		idx.modules = make(map[string]*ModuleMetadata)
	}
	if idx.versionDownloads == nil {
		idx.versionDownloads = make(map[string]map[string]int64)
	}

	return nil
}

// Helper functions

func normalizeQuery(query string) string {
	// Simple normalization: lowercase
	return query
}

func matchesQuery(path, description, query string) bool {
	// Simple substring match
	return contains(path, query) || contains(description, query)
}

func contains(s, substr string) bool {
	// Case-insensitive contains
	return len(substr) == 0 ||
		len(s) >= len(substr) &&
			(s == substr ||
				containsLower(s, substr))
}

func containsLower(s, substr string) bool {
	// Simple case-insensitive substring search
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc := s[i+j]
			qc := substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if qc >= 'A' && qc <= 'Z' {
				qc += 32
			}
			if sc != qc {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
