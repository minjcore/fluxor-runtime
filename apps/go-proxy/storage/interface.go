// Copyright (c) 2026 Fluxor Framework
// SPDX-License-Identifier: MIT

package storage

import (
	"context"
	"io"

	"github.com/fluxorio/goproxy/domain"
)

// Storage defines the interface for module storage backends
type Storage interface {
	// ListVersions returns all versions available for a module
	ListVersions(ctx context.Context, module string) ([]string, error)

	// GetVersionInfo returns metadata for a specific module version
	GetVersionInfo(ctx context.Context, module, version string) (*domain.VersionInfo, error)

	// GetModFile returns the go.mod file contents for a specific version
	GetModFile(ctx context.Context, module, version string) ([]byte, error)

	// GetZip returns the module source archive
	GetZip(ctx context.Context, module, version string) (io.ReadCloser, error)

	// GetLatest returns the latest version info for a module
	GetLatest(ctx context.Context, module string) (*domain.VersionInfo, error)

	// PutModule stores a new module version with its zip archive and go.mod
	PutModule(ctx context.Context, module, version string, zip io.Reader, mod []byte) error

	// DeleteVersion removes a specific version of a module
	DeleteVersion(ctx context.Context, module, version string) error

	// SearchModules searches for modules matching a query string
	SearchModules(ctx context.Context, query string, limit int) ([]domain.ModuleSummary, error)

	// ListModules lists all modules with an optional prefix filter
	ListModules(ctx context.Context, prefix string, limit int) ([]string, error)

	// GetModuleStats returns statistics for a module
	GetModuleStats(ctx context.Context, module string) (*ModuleStats, error)

	// IncrementDownloads increments the download counter for a version
	IncrementDownloads(ctx context.Context, module, version string) error

	// Close closes the storage backend and releases resources
	Close() error
}

// ModuleStats contains statistics about a module
type ModuleStats struct {
	// Module is the module path
	Module string `json:"module"`

	// VersionCount is the number of available versions
	VersionCount int `json:"versionCount"`

	// TotalDownloads is the total downloads across all versions
	TotalDownloads int64 `json:"totalDownloads"`

	// TotalSize is the total size of all versions in bytes
	TotalSize int64 `json:"totalSize"`

	// VersionDownloads is a map of version -> download count
	VersionDownloads map[string]int64 `json:"versionDownloads,omitempty"`
}

// StorageConfig contains common storage configuration
type StorageConfig struct {
	// Type is the storage backend type ("s3", "filesystem")
	Type string

	// S3 specific configuration
	S3Endpoint        string
	S3Bucket          string
	S3Region          string
	S3AccessKey       string
	S3SecretKey       string
	S3ForcePathStyle  bool
	S3DisableSSL      bool

	// Filesystem specific configuration
	BasePath string

	// Mirror: optional S3/OSS config to mirror writes from filesystem primary
	MirrorEnabled        bool
	MirrorS3Endpoint     string
	MirrorS3Bucket       string
	MirrorS3Region       string
	MirrorS3AccessKey    string
	MirrorS3SecretKey    string
	MirrorS3ForcePathStyle bool
	MirrorS3DisableSSL   bool
}

// NewStorage creates a new storage backend based on configuration
func NewStorage(cfg StorageConfig) (Storage, error) {
	switch cfg.Type {
	case "s3":
		return NewS3Storage(S3Config{
			Endpoint:       cfg.S3Endpoint,
			Bucket:         cfg.S3Bucket,
			Region:         cfg.S3Region,
			AccessKey:      cfg.S3AccessKey,
			SecretKey:      cfg.S3SecretKey,
			ForcePathStyle: cfg.S3ForcePathStyle,
			DisableSSL:     cfg.S3DisableSSL,
		})
	case "filesystem":
		basePath := cfg.BasePath
		if basePath == "" {
			basePath = "/var/lib/goproxy"
		}
		fs, err := NewFilesystemStorage(basePath)
		if err != nil {
			return nil, err
		}
		if !cfg.MirrorEnabled {
			return fs, nil
		}
		// Wrap with OSS mirror
		mirror, err := NewS3Storage(S3Config{
			Endpoint:       cfg.MirrorS3Endpoint,
			Bucket:         cfg.MirrorS3Bucket,
			Region:         cfg.MirrorS3Region,
			AccessKey:      cfg.MirrorS3AccessKey,
			SecretKey:      cfg.MirrorS3SecretKey,
			ForcePathStyle: cfg.MirrorS3ForcePathStyle,
			DisableSSL:     cfg.MirrorS3DisableSSL,
		})
		if err != nil {
			return nil, err
		}
		return NewMirroredStorage(fs, mirror), nil
	default:
		return nil, domain.NewStorageError(cfg.Type, "create", "", domain.ErrStorageUnavailable)
	}
}
