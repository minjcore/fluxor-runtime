package buckey_storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultRepoSkipDirs are directory names skipped when storing a repo (e.g. .git, node_modules).
var DefaultRepoSkipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true,
	"__pycache__": true, ".venv": true, "venv": true,
	"dist": true, "build": true, ".next": true, ".nuxt": true,
	".idea": true, ".vscode": true, ".cursor": true,
}

// StoreRepoOptions configures how a local repo is stored.
type StoreRepoOptions struct {
	// KeyPrefix is prepended to every key (e.g. "repos/myproject/").
	KeyPrefix string
	// SkipDirs: dir names to skip (nil = use DefaultRepoSkipDirs).
	SkipDirs map[string]bool
	// TextOnly: if true, only store files that look like text (UTF-8, not binary).
	TextOnly bool
	// MaxFileSize is the max size in bytes to store per file (0 = no limit).
	MaxFileSize int64
	// IncludeSuffixes: if non-empty, only store files whose name ends with one of these (e.g. []string{".md"}).
	IncludeSuffixes []string
}

// StoreRepoToStorage walks rootDir (e.g. a local GitHub clone) and stores each file under
// key = KeyPrefix + relative path (slash-separated). Skips .git and other common dirs.
// Returns the number of files stored and the first error, if any.
func StoreRepoToStorage(ctx context.Context, s BlobStorage, rootDir string, opts StoreRepoOptions) (stored int, err error) {
	ValidateContext(ctx)
	rootDir, err = filepath.Abs(rootDir)
	if err != nil {
		return 0, fmt.Errorf("buckey_storage: repo root: %w", err)
	}
	info, err := os.Stat(rootDir)
	if err != nil {
		return 0, fmt.Errorf("buckey_storage: repo root: %w", err)
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("buckey_storage: repo root is not a directory: %s", rootDir)
	}

	skipDirs := opts.SkipDirs
	if skipDirs == nil {
		skipDirs = DefaultRepoSkipDirs
	}
	prefix := strings.TrimSuffix(opts.KeyPrefix, "/")
	if prefix != "" {
		prefix += "/"
	}

	err = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := d.Name()
			if skipDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		key := prefix + rel
		if err := ValidateKey(key); err != nil {
			return err
		}
		if len(opts.IncludeSuffixes) > 0 && !hasAnySuffix(d.Name(), opts.IncludeSuffixes) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		if opts.MaxFileSize > 0 && info.Size() > opts.MaxFileSize {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if opts.TextOnly && !isIndexableText(data) {
			return nil
		}

		if err := s.Put(ctx, key, data); err != nil {
			return err
		}
		stored++
		return nil
	})

	return stored, err
}

func hasAnySuffix(name string, suffixes []string) bool {
	for _, suf := range suffixes {
		if strings.HasSuffix(strings.ToLower(name), strings.ToLower(suf)) {
			return true
		}
	}
	return false
}

// StoreRepoToStorageWithIndex stores the repo into BlobStorage and indexes each stored file
// into the FullTextEngine so you can full-text search the source code.
func StoreRepoToStorageWithIndex(ctx context.Context, engine *FullTextEngine, rootDir string, opts StoreRepoOptions) (stored int, err error) {
	ValidateContext(ctx)
	if engine == nil {
		return 0, fmt.Errorf("buckey_storage: engine is nil")
	}
	rootDir, err = filepath.Abs(rootDir)
	if err != nil {
		return 0, fmt.Errorf("buckey_storage: repo root: %w", err)
	}
	info, err := os.Stat(rootDir)
	if err != nil {
		return 0, fmt.Errorf("buckey_storage: repo root: %w", err)
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("buckey_storage: repo root is not a directory: %s", rootDir)
	}

	skipDirs := opts.SkipDirs
	if skipDirs == nil {
		skipDirs = DefaultRepoSkipDirs
	}
	prefix := strings.TrimSuffix(opts.KeyPrefix, "/")
	if prefix != "" {
		prefix += "/"
	}

	err = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		key := prefix + rel
		if err := ValidateKey(key); err != nil {
			return err
		}
		if len(opts.IncludeSuffixes) > 0 && !hasAnySuffix(d.Name(), opts.IncludeSuffixes) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		if opts.MaxFileSize > 0 && info.Size() > opts.MaxFileSize {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if opts.TextOnly && !isIndexableText(data) {
			return nil
		}

		if err := engine.storage.Put(ctx, key, data); err != nil {
			return err
		}
		if err := engine.IndexBlob(ctx, key, data); err != nil {
			return err
		}
		stored++
		return nil
	})

	return stored, err
}
