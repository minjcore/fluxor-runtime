// Copyright (c) 2026 Fluxor Framework
// SPDX-License-Identifier: MIT

package storage

import (
	"bytes"
	"context"
	"io"
	"log"

	"github.com/fluxorio/goproxy/domain"
)

func newByteReader(b []byte) io.Reader { return bytes.NewReader(b) }

// MirroredStorage writes to both a primary and a mirror backend.
// Reads are served from primary; if primary returns not-found, the mirror
// is tried as a fallback (useful after a restore from OSS).
// Mirror writes happen asynchronously so they never block the caller.
type MirroredStorage struct {
	primary Storage
	mirror  Storage
}

// NewMirroredStorage creates a mirrored storage with primary and mirror backends.
func NewMirroredStorage(primary, mirror Storage) *MirroredStorage {
	return &MirroredStorage{primary: primary, mirror: mirror}
}

func (m *MirroredStorage) ListVersions(ctx context.Context, module string) ([]string, error) {
	versions, err := m.primary.ListVersions(ctx, module)
	if err != nil && domain.IsNotFound(err) {
		return m.mirror.ListVersions(ctx, module)
	}
	return versions, err
}

func (m *MirroredStorage) GetVersionInfo(ctx context.Context, module, version string) (*domain.VersionInfo, error) {
	info, err := m.primary.GetVersionInfo(ctx, module, version)
	if err != nil && domain.IsNotFound(err) {
		return m.mirror.GetVersionInfo(ctx, module, version)
	}
	return info, err
}

func (m *MirroredStorage) GetModFile(ctx context.Context, module, version string) ([]byte, error) {
	data, err := m.primary.GetModFile(ctx, module, version)
	if err != nil && domain.IsNotFound(err) {
		return m.mirror.GetModFile(ctx, module, version)
	}
	return data, err
}

func (m *MirroredStorage) GetZip(ctx context.Context, module, version string) (io.ReadCloser, error) {
	rc, err := m.primary.GetZip(ctx, module, version)
	if err != nil && domain.IsNotFound(err) {
		return m.mirror.GetZip(ctx, module, version)
	}
	return rc, err
}

func (m *MirroredStorage) GetLatest(ctx context.Context, module string) (*domain.VersionInfo, error) {
	info, err := m.primary.GetLatest(ctx, module)
	if err != nil && domain.IsNotFound(err) {
		return m.mirror.GetLatest(ctx, module)
	}
	return info, err
}

func (m *MirroredStorage) PutModule(ctx context.Context, module, version string, zip io.Reader, mod []byte) error {
	// Read zip into memory so we can send it to both backends.
	zipData, err := io.ReadAll(zip)
	if err != nil {
		return domain.NewStorageError("mirrored", "read-zip", "", err)
	}

	if err := m.primary.PutModule(ctx, module, version, newByteReader(zipData), mod); err != nil {
		return err
	}

	// Mirror write is async — never block the upload on OSS latency.
	modCopy := make([]byte, len(mod))
	copy(modCopy, mod)
	zipCopy := make([]byte, len(zipData))
	copy(zipCopy, zipData)
	go func() {
		if err := m.mirror.PutModule(context.Background(), module, version, newByteReader(zipCopy), modCopy); err != nil {
			log.Printf("mirror: PutModule %s@%s failed: %v", module, version, err)
		} else {
			log.Printf("mirror: backed up %s@%s to OSS", module, version)
		}
	}()

	return nil
}

func (m *MirroredStorage) DeleteVersion(ctx context.Context, module, version string) error {
	if err := m.primary.DeleteVersion(ctx, module, version); err != nil {
		return err
	}
	go func() {
		if err := m.mirror.DeleteVersion(context.Background(), module, version); err != nil {
			log.Printf("mirror: DeleteVersion %s@%s failed: %v", module, version, err)
		}
	}()
	return nil
}

func (m *MirroredStorage) SearchModules(ctx context.Context, query string, limit int) ([]domain.ModuleSummary, error) {
	return m.primary.SearchModules(ctx, query, limit)
}

func (m *MirroredStorage) ListModules(ctx context.Context, prefix string, limit int) ([]string, error) {
	return m.primary.ListModules(ctx, prefix, limit)
}

func (m *MirroredStorage) GetModuleStats(ctx context.Context, module string) (*ModuleStats, error) {
	return m.primary.GetModuleStats(ctx, module)
}

func (m *MirroredStorage) IncrementDownloads(ctx context.Context, module, version string) error {
	return m.primary.IncrementDownloads(ctx, module, version)
}

func (m *MirroredStorage) Close() error {
	primaryErr := m.primary.Close()
	if err := m.mirror.Close(); err != nil {
		log.Printf("mirror: Close failed: %v", err)
	}
	return primaryErr
}

var _ Storage = (*MirroredStorage)(nil)
