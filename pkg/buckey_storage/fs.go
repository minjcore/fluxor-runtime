package buckey_storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FSStorage implements BlobStorage using the local filesystem.
type FSStorage struct {
	root  string
	prefix string
}

// NewFS creates a BlobStorage that stores blobs as files under root.
// Keys are sanitized: path separators and ".." are replaced so keys cannot escape root.
func NewFS(root, prefix string) (*FSStorage, error) {
	if root == "" {
		return nil, fmt.Errorf("buckey_storage: FS root path cannot be empty")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return nil, err
	}
	return &FSStorage{root: abs, prefix: strings.TrimPrefix(prefix, "/")}, nil
}

// safeKey converts key to a filesystem-safe path (no "..", no absolute paths).
func safeKey(key string) string {
	key = filepath.Clean(key)
	key = strings.TrimPrefix(key, "/")
	key = strings.TrimPrefix(key, "..")
	key = filepath.Clean(key)
	if key == "" || key == "." {
		key = "_"
	}
	return key
}

func (f *FSStorage) fullPath(key string) string {
	safe := safeKey(key)
	if f.prefix == "" {
		return filepath.Join(f.root, safe)
	}
	return filepath.Join(f.root, f.prefix, safe)
}

// Put stores data under key.
func (f *FSStorage) Put(ctx context.Context, key string, data []byte) error {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return err
	}
	if data == nil {
		data = []byte{}
	}
	p := f.fullPath(key)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

// Get retrieves data by key.
func (f *FSStorage) Get(ctx context.Context, key string) ([]byte, error) {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(f.fullPath(key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

// Delete removes the blob at key.
func (f *FSStorage) Delete(ctx context.Context, key string) error {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return err
	}
	p := f.fullPath(key)
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// List returns keys with the given prefix.
func (f *FSStorage) List(ctx context.Context, prefix string) ([]string, error) {
	ValidateContext(ctx)
	base := filepath.Join(f.root, f.prefix)
	if prefix != "" {
		base = filepath.Join(base, safeKey(prefix))
	}
	var out []string
	err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(filepath.Join(f.root, f.prefix), path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		out = append(out, rel)
		return nil
	})
	return out, err
}
