// Copyright (c) 2026 Fluxor Framework
// SPDX-License-Identifier: MIT

package storage

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/fluxorio/goproxy/domain"
)

// FilesystemStorage implements Storage using the local filesystem.
//
// Layout mirrors the S3 key structure:
//
//	{basePath}/modules/{escaped_module}/@v/list
//	{basePath}/modules/{escaped_module}/@v/{version}.info
//	{basePath}/modules/{escaped_module}/@v/{version}.mod
//	{basePath}/modules/{escaped_module}/@v/{version}.zip
//	{basePath}/metadata/modules.json
type FilesystemStorage struct {
	basePath string
	mu       sync.RWMutex
	index    map[string]*domain.Module
}

// NewFilesystemStorage creates a new filesystem-backed storage.
func NewFilesystemStorage(basePath string) (*FilesystemStorage, error) {
	for _, sub := range []string{"modules", "metadata"} {
		if err := os.MkdirAll(filepath.Join(basePath, sub), 0o755); err != nil {
			return nil, domain.NewStorageError("filesystem", "init", sub, err)
		}
	}
	s := &FilesystemStorage{
		basePath: basePath,
		index:    make(map[string]*domain.Module),
	}
	if err := s.loadIndex(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *FilesystemStorage) modulePath(module string) string {
	return filepath.Join(s.basePath, "modules", domain.EscapeModulePath(module))
}

func (s *FilesystemStorage) versionDir(module string) string {
	return filepath.Join(s.modulePath(module), "@v")
}

func (s *FilesystemStorage) listFile(module string) string {
	return filepath.Join(s.versionDir(module), "list")
}

func (s *FilesystemStorage) infoFile(module, version string) string {
	return filepath.Join(s.versionDir(module), version+".info")
}

func (s *FilesystemStorage) modFile(module, version string) string {
	return filepath.Join(s.versionDir(module), version+".mod")
}

func (s *FilesystemStorage) zipFile(module, version string) string {
	return filepath.Join(s.versionDir(module), version+".zip")
}

func (s *FilesystemStorage) indexFile() string {
	return filepath.Join(s.basePath, "metadata", "modules.json")
}

func (s *FilesystemStorage) ListVersions(_ context.Context, module string) ([]string, error) {
	data, err := os.ReadFile(s.listFile(module))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, domain.NewModuleError("list", module, "", domain.ErrModuleNotFound)
		}
		return nil, domain.NewStorageError("filesystem", "read", s.listFile(module), err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	versions := make([]string, 0, len(lines))
	for _, l := range lines {
		if l = strings.TrimSpace(l); l != "" {
			versions = append(versions, l)
		}
	}
	return versions, nil
}

func (s *FilesystemStorage) GetVersionInfo(_ context.Context, module, version string) (*domain.VersionInfo, error) {
	data, err := os.ReadFile(s.infoFile(module, version))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, domain.NewModuleError("info", module, version, domain.ErrVersionNotFound)
		}
		return nil, domain.NewStorageError("filesystem", "read", s.infoFile(module, version), err)
	}
	info, err := domain.ParseVersionInfo(data)
	if err != nil {
		return nil, domain.NewStorageError("filesystem", "parse", s.infoFile(module, version), err)
	}
	return info, nil
}

func (s *FilesystemStorage) GetModFile(_ context.Context, module, version string) ([]byte, error) {
	data, err := os.ReadFile(s.modFile(module, version))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, domain.NewModuleError("mod", module, version, domain.ErrVersionNotFound)
		}
		return nil, domain.NewStorageError("filesystem", "read", s.modFile(module, version), err)
	}
	return data, nil
}

func (s *FilesystemStorage) GetZip(_ context.Context, module, version string) (io.ReadCloser, error) {
	f, err := os.Open(s.zipFile(module, version))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, domain.NewModuleError("zip", module, version, domain.ErrVersionNotFound)
		}
		return nil, domain.NewStorageError("filesystem", "open", s.zipFile(module, version), err)
	}
	return f, nil
}

func (s *FilesystemStorage) GetLatest(ctx context.Context, module string) (*domain.VersionInfo, error) {
	versions, err := s.ListVersions(ctx, module)
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		return nil, domain.NewModuleError("latest", module, "", domain.ErrVersionNotFound)
	}
	sort.Strings(versions)
	return s.GetVersionInfo(ctx, module, versions[len(versions)-1])
}

func (s *FilesystemStorage) PutModule(ctx context.Context, module, version string, zipReader io.Reader, modContent []byte) error {
	if !domain.IsValidVersion(version) {
		return domain.NewModuleError("put", module, version, domain.ErrInvalidVersion)
	}
	dir := s.versionDir(module)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return domain.NewStorageError("filesystem", "mkdir", dir, err)
	}

	// Write .info
	info := domain.NewVersionInfo(version)
	infoData, err := info.ToJSON()
	if err != nil {
		return domain.NewStorageError("filesystem", "marshal", "", err)
	}
	if err := os.WriteFile(s.infoFile(module, version), infoData, 0o644); err != nil {
		return domain.NewStorageError("filesystem", "write", s.infoFile(module, version), err)
	}

	// Write .mod
	if err := os.WriteFile(s.modFile(module, version), modContent, 0o644); err != nil {
		return domain.NewStorageError("filesystem", "write", s.modFile(module, version), err)
	}

	// Write .zip
	f, err := os.Create(s.zipFile(module, version))
	if err != nil {
		return domain.NewStorageError("filesystem", "create", s.zipFile(module, version), err)
	}
	if _, err := io.Copy(f, zipReader); err != nil {
		f.Close()
		return domain.NewStorageError("filesystem", "write", s.zipFile(module, version), err)
	}
	f.Close()

	if err := s.updateVersionList(ctx, module, version, true); err != nil {
		return err
	}
	s.updateIndex(module, version)
	return nil
}

func (s *FilesystemStorage) DeleteVersion(ctx context.Context, module, version string) error {
	for _, f := range []string{
		s.infoFile(module, version),
		s.modFile(module, version),
		s.zipFile(module, version),
	} {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			return domain.NewStorageError("filesystem", "delete", f, err)
		}
	}
	return s.updateVersionList(ctx, module, version, false)
}

func (s *FilesystemStorage) SearchModules(_ context.Context, query string, limit int) ([]domain.ModuleSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q := strings.ToLower(query)
	out := make([]domain.ModuleSummary, 0, limit)
	for _, m := range s.index {
		if strings.Contains(strings.ToLower(m.Path), q) {
			out = append(out, m.ToSummary())
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (s *FilesystemStorage) ListModules(_ context.Context, prefix string, limit int) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, limit)
	for p := range s.index {
		if prefix == "" || strings.HasPrefix(p, prefix) {
			out = append(out, p)
			if len(out) >= limit {
				break
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

func (s *FilesystemStorage) GetModuleStats(_ context.Context, module string) (*ModuleStats, error) {
	s.mu.RLock()
	m, ok := s.index[module]
	s.mu.RUnlock()
	if !ok {
		return nil, domain.NewModuleError("stats", module, "", domain.ErrModuleNotFound)
	}
	return &ModuleStats{
		Module:         module,
		VersionCount:   len(m.Versions),
		TotalDownloads: m.TotalDownloads,
	}, nil
}

func (s *FilesystemStorage) IncrementDownloads(_ context.Context, module, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.index[module]; ok {
		m.TotalDownloads++
	}
	return nil
}

func (s *FilesystemStorage) Close() error {
	return s.saveIndex()
}

func (s *FilesystemStorage) updateVersionList(ctx context.Context, module, version string, add bool) error {
	versions, err := s.ListVersions(ctx, module)
	if err != nil && !domain.IsNotFound(err) {
		return err
	}
	if add {
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
		for i, v := range versions {
			if v == version {
				versions = append(versions[:i], versions[i+1:]...)
				break
			}
		}
	}
	sort.Strings(versions)
	content := strings.Join(versions, "\n")
	if err := os.WriteFile(s.listFile(module), []byte(content), 0o644); err != nil {
		return domain.NewStorageError("filesystem", "write", s.listFile(module), err)
	}
	return nil
}

func (s *FilesystemStorage) updateIndex(modulePath, version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.index[modulePath]
	if !ok {
		m = domain.NewModule(modulePath)
		s.index[modulePath] = m
	}
	m.AddVersion(version)
	_ = s.saveIndexLocked()
}

func (s *FilesystemStorage) loadIndex() error {
	data, err := os.ReadFile(s.indexFile())
	if err != nil {
		return err
	}
	var modules []*domain.Module
	if err := json.Unmarshal(data, &modules); err != nil {
		return domain.NewStorageError("filesystem", "parse", s.indexFile(), err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range modules {
		s.index[m.Path] = m
	}
	return nil
}

func (s *FilesystemStorage) saveIndex() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.saveIndexLocked()
}

// saveIndexLocked must be called with s.mu held (at least read).
func (s *FilesystemStorage) saveIndexLocked() error {
	modules := make([]*domain.Module, 0, len(s.index))
	for _, m := range s.index {
		modules = append(modules, m)
	}
	data, err := json.Marshal(modules)
	if err != nil {
		return domain.NewStorageError("filesystem", "marshal", s.indexFile(), err)
	}
	if err := os.WriteFile(s.indexFile(), data, 0o644); err != nil {
		return domain.NewStorageError("filesystem", "write", s.indexFile(), err)
	}
	return nil
}

var _ Storage = (*FilesystemStorage)(nil)
