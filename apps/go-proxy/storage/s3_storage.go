// Copyright (c) 2026 Fluxor Framework
// SPDX-License-Identifier: MIT

package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/fluxorio/goproxy/domain"
)

// S3Config holds S3-specific configuration
type S3Config struct {
	Endpoint       string
	Bucket         string
	Region         string
	AccessKey      string
	SecretKey      string
	ForcePathStyle bool
	DisableSSL     bool
}

// S3Storage implements Storage using S3-compatible object storage
type S3Storage struct {
	client *s3.Client
	bucket string
	mu     sync.RWMutex

	// In-memory metadata index for fast searches
	// In production, this could be backed by a database
	moduleIndex map[string]*domain.Module
}

// S3 object key layout:
// modules/{escaped_module}/@v/list          - version list (text, one per line)
// modules/{escaped_module}/@v/{version}.info - version info (JSON)
// modules/{escaped_module}/@v/{version}.mod  - go.mod file (text)
// modules/{escaped_module}/@v/{version}.zip  - source archive (binary)
// metadata/modules.json                       - module index for search

const (
	modulesPrefix   = "modules/"
	metadataPrefix  = "metadata/"
	moduleIndexFile = "metadata/modules.json"
)

// NewS3Storage creates a new S3 storage backend
func NewS3Storage(cfg S3Config) (*S3Storage, error) {
	// Build AWS config
	var opts []func(*config.LoadOptions) error

	opts = append(opts, config.WithRegion(cfg.Region))

	// Use static credentials if provided
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, domain.NewStorageError("s3", "init", "", err)
	}

	// Create S3 client with custom endpoint if specified
	var s3Opts []func(*s3.Options)

	if cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = cfg.ForcePathStyle
		})
	}

	client := s3.NewFromConfig(awsCfg, s3Opts...)

	storage := &S3Storage{
		client:      client,
		bucket:      cfg.Bucket,
		moduleIndex: make(map[string]*domain.Module),
	}

	// Load existing module index
	if err := storage.loadModuleIndex(context.Background()); err != nil {
		// Index doesn't exist yet, that's OK
		if !domain.IsNotFound(err) {
			// Log warning but continue
			fmt.Printf("Warning: failed to load module index: %v\n", err)
		}
	}

	return storage, nil
}

// moduleKey returns the S3 key for a module's directory
func moduleKey(module string) string {
	return modulesPrefix + domain.EscapeModulePath(module) + "/"
}

// versionListKey returns the S3 key for the version list
func versionListKey(module string) string {
	return moduleKey(module) + "@v/list"
}

// versionInfoKey returns the S3 key for version info
func versionInfoKey(module, version string) string {
	return moduleKey(module) + "@v/" + version + ".info"
}

// versionModKey returns the S3 key for the go.mod file
func versionModKey(module, version string) string {
	return moduleKey(module) + "@v/" + version + ".mod"
}

// versionZipKey returns the S3 key for the zip archive
func versionZipKey(module, version string) string {
	return moduleKey(module) + "@v/" + version + ".zip"
}

// ListVersions returns all versions for a module
func (s *S3Storage) ListVersions(ctx context.Context, module string) ([]string, error) {
	key := versionListKey(module)

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFoundErr(err) {
			return nil, domain.NewModuleError("list", module, "", domain.ErrModuleNotFound)
		}
		return nil, domain.NewStorageError("s3", "get", key, err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, domain.NewStorageError("s3", "read", key, err)
	}

	// Parse version list (one version per line)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	versions := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			versions = append(versions, line)
		}
	}

	return versions, nil
}

// GetVersionInfo returns metadata for a specific version
func (s *S3Storage) GetVersionInfo(ctx context.Context, module, version string) (*domain.VersionInfo, error) {
	key := versionInfoKey(module, version)

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFoundErr(err) {
			return nil, domain.NewModuleError("info", module, version, domain.ErrVersionNotFound)
		}
		return nil, domain.NewStorageError("s3", "get", key, err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, domain.NewStorageError("s3", "read", key, err)
	}

	info, err := domain.ParseVersionInfo(data)
	if err != nil {
		return nil, domain.NewStorageError("s3", "parse", key, err)
	}

	return info, nil
}

// GetModFile returns the go.mod file for a version
func (s *S3Storage) GetModFile(ctx context.Context, module, version string) ([]byte, error) {
	key := versionModKey(module, version)

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFoundErr(err) {
			return nil, domain.NewModuleError("mod", module, version, domain.ErrVersionNotFound)
		}
		return nil, domain.NewStorageError("s3", "get", key, err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, domain.NewStorageError("s3", "read", key, err)
	}

	return data, nil
}

// GetZip returns the source archive for a version
func (s *S3Storage) GetZip(ctx context.Context, module, version string) (io.ReadCloser, error) {
	key := versionZipKey(module, version)

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFoundErr(err) {
			return nil, domain.NewModuleError("zip", module, version, domain.ErrVersionNotFound)
		}
		return nil, domain.NewStorageError("s3", "get", key, err)
	}

	return result.Body, nil
}

// GetLatest returns the latest version info for a module
func (s *S3Storage) GetLatest(ctx context.Context, module string) (*domain.VersionInfo, error) {
	versions, err := s.ListVersions(ctx, module)
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, domain.NewModuleError("latest", module, "", domain.ErrVersionNotFound)
	}

	// Sort versions and get the latest
	sort.Strings(versions)
	latestVersion := versions[len(versions)-1]

	return s.GetVersionInfo(ctx, module, latestVersion)
}

// PutModule stores a new module version
func (s *S3Storage) PutModule(ctx context.Context, module, version string, zipReader io.Reader, modContent []byte) error {
	if !domain.IsValidVersion(version) {
		return domain.NewModuleError("put", module, version, domain.ErrInvalidVersion)
	}

	// Read zip into memory (needed for S3 upload)
	zipData, err := io.ReadAll(zipReader)
	if err != nil {
		return domain.NewStorageError("s3", "read-zip", "", err)
	}

	// Create version info
	info := domain.NewVersionInfo(version)
	infoData, err := info.ToJSON()
	if err != nil {
		return domain.NewStorageError("s3", "marshal-info", "", err)
	}

	// Upload all files
	uploads := []struct {
		key         string
		data        []byte
		contentType string
	}{
		{versionInfoKey(module, version), infoData, "application/json"},
		{versionModKey(module, version), modContent, "text/plain; charset=utf-8"},
		{versionZipKey(module, version), zipData, "application/zip"},
	}

	for _, u := range uploads {
		_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(s.bucket),
			Key:         aws.String(u.key),
			Body:        bytes.NewReader(u.data),
			ContentType: aws.String(u.contentType),
		})
		if err != nil {
			return domain.NewStorageError("s3", "put", u.key, err)
		}
	}

	// Update version list
	if err := s.updateVersionList(ctx, module, version, true); err != nil {
		return err
	}

	// Update module index
	s.updateModuleIndex(module, version)

	return nil
}

// DeleteVersion removes a specific version
func (s *S3Storage) DeleteVersion(ctx context.Context, module, version string) error {
	// Delete all version files
	keys := []string{
		versionInfoKey(module, version),
		versionModKey(module, version),
		versionZipKey(module, version),
	}

	objects := make([]types.ObjectIdentifier, len(keys))
	for i, key := range keys {
		objects[i] = types.ObjectIdentifier{Key: aws.String(key)}
	}

	_, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(s.bucket),
		Delete: &types.Delete{Objects: objects},
	})
	if err != nil {
		return domain.NewStorageError("s3", "delete", module+"@"+version, err)
	}

	// Update version list
	if err := s.updateVersionList(ctx, module, version, false); err != nil {
		return err
	}

	return nil
}

// SearchModules searches for modules matching a query
func (s *S3Storage) SearchModules(ctx context.Context, query string, limit int) ([]domain.ModuleSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(query)
	results := make([]domain.ModuleSummary, 0, limit)

	for _, mod := range s.moduleIndex {
		if strings.Contains(strings.ToLower(mod.Path), query) {
			results = append(results, mod.ToSummary())
			if len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

// ListModules lists all modules with an optional prefix
func (s *S3Storage) ListModules(ctx context.Context, prefix string, limit int) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]string, 0, limit)

	for modulePath := range s.moduleIndex {
		if prefix == "" || strings.HasPrefix(modulePath, prefix) {
			results = append(results, modulePath)
			if len(results) >= limit {
				break
			}
		}
	}

	sort.Strings(results)
	return results, nil
}

// GetModuleStats returns statistics for a module
func (s *S3Storage) GetModuleStats(ctx context.Context, module string) (*ModuleStats, error) {
	s.mu.RLock()
	mod, ok := s.moduleIndex[module]
	s.mu.RUnlock()

	if !ok {
		return nil, domain.NewModuleError("stats", module, "", domain.ErrModuleNotFound)
	}

	return &ModuleStats{
		Module:         module,
		VersionCount:   len(mod.Versions),
		TotalDownloads: mod.TotalDownloads,
	}, nil
}

// IncrementDownloads increments the download counter
func (s *S3Storage) IncrementDownloads(ctx context.Context, module, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if mod, ok := s.moduleIndex[module]; ok {
		mod.TotalDownloads++
	}

	return nil
}

// Close closes the storage backend
func (s *S3Storage) Close() error {
	// Save module index before closing
	return s.saveModuleIndex(context.Background())
}

// updateVersionList updates the version list file
func (s *S3Storage) updateVersionList(ctx context.Context, module, version string, add bool) error {
	key := versionListKey(module)

	// Get existing versions
	versions, err := s.ListVersions(ctx, module)
	if err != nil && !domain.IsNotFound(err) {
		return err
	}

	if add {
		// Add version if not exists
		found := false
		for _, v := range versions {
			if v == version {
				found = true
				break
			}
		}
		if !found {
			versions = append(versions, version)
		}
	} else {
		// Remove version
		for i, v := range versions {
			if v == version {
				versions = append(versions[:i], versions[i+1:]...)
				break
			}
		}
	}

	// Sort and write back
	sort.Strings(versions)
	content := strings.Join(versions, "\n")

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        strings.NewReader(content),
		ContentType: aws.String("text/plain; charset=utf-8"),
	})
	if err != nil {
		return domain.NewStorageError("s3", "put", key, err)
	}

	return nil
}

// updateModuleIndex updates the in-memory module index
func (s *S3Storage) updateModuleIndex(modulePath, version string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	mod, ok := s.moduleIndex[modulePath]
	if !ok {
		mod = domain.NewModule(modulePath)
		s.moduleIndex[modulePath] = mod
	}
	mod.AddVersion(version)
}

// loadModuleIndex loads the module index from S3
func (s *S3Storage) loadModuleIndex(ctx context.Context) error {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(moduleIndexFile),
	})
	if err != nil {
		if isNotFoundErr(err) {
			return domain.ErrModuleNotFound
		}
		return domain.NewStorageError("s3", "get", moduleIndexFile, err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return domain.NewStorageError("s3", "read", moduleIndexFile, err)
	}

	var modules []*domain.Module
	if err := json.Unmarshal(data, &modules); err != nil {
		return domain.NewStorageError("s3", "parse", moduleIndexFile, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, mod := range modules {
		s.moduleIndex[mod.Path] = mod
	}

	return nil
}

// saveModuleIndex saves the module index to S3
func (s *S3Storage) saveModuleIndex(ctx context.Context) error {
	s.mu.RLock()
	modules := make([]*domain.Module, 0, len(s.moduleIndex))
	for _, mod := range s.moduleIndex {
		modules = append(modules, mod)
	}
	s.mu.RUnlock()

	data, err := json.Marshal(modules)
	if err != nil {
		return domain.NewStorageError("s3", "marshal", moduleIndexFile, err)
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(moduleIndexFile),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return domain.NewStorageError("s3", "put", moduleIndexFile, err)
	}

	return nil
}

// isNotFoundErr checks if an error is a "not found" error from S3
func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	// Check for S3 NoSuchKey error
	return strings.Contains(err.Error(), "NoSuchKey") ||
		strings.Contains(err.Error(), "NotFound") ||
		strings.Contains(err.Error(), "404")
}

// scanModulesFromS3 scans the S3 bucket to build the module index
// This is useful for initial startup or recovery
func (s *S3Storage) scanModulesFromS3(ctx context.Context) error {
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(modulesPrefix),
	})

	modules := make(map[string]*domain.Module)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return domain.NewStorageError("s3", "list", modulesPrefix, err)
		}

		for _, obj := range page.Contents {
			key := aws.ToString(obj.Key)
			
			// Extract module path from key
			// Format: modules/{escaped_module}/@v/{version}.info
			if !strings.HasSuffix(key, ".info") {
				continue
			}

			// Remove prefix and suffix
			key = strings.TrimPrefix(key, modulesPrefix)
			parts := strings.Split(key, "/@v/")
			if len(parts) != 2 {
				continue
			}

			escapedModule := parts[0]
			modulePath := domain.UnescapeModulePath(escapedModule)
			version := strings.TrimSuffix(parts[1], ".info")

			mod, ok := modules[modulePath]
			if !ok {
				mod = domain.NewModule(modulePath)
				mod.CreatedAt = *obj.LastModified
				modules[modulePath] = mod
			}
			mod.AddVersion(version)
			if obj.LastModified.After(mod.UpdatedAt) {
				mod.UpdatedAt = *obj.LastModified
			}
		}
	}

	s.mu.Lock()
	s.moduleIndex = modules
	s.mu.Unlock()

	return s.saveModuleIndex(ctx)
}

// Ensure S3Storage implements Storage interface
var _ Storage = (*S3Storage)(nil)

// Helper to get current time in UTC
func nowUTC() time.Time {
	return time.Now().UTC()
}

// Helper to build S3 key path
func buildKey(parts ...string) string {
	return path.Join(parts...)
}
