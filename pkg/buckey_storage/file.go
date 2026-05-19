package buckey_storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// CopyFileToStorage reads the file at filePath and stores its contents under key.
// The key can be the filename, a path, or any string you use to retrieve it later.
func CopyFileToStorage(ctx context.Context, s BlobStorage, key, filePath string) error {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return err
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("buckey_storage: read file %q: %w", filePath, err)
	}
	return s.Put(ctx, key, data)
}

// CopyFileToClipboard reads the file at filePath and stores its contents as a clipboard
// entry with the given title (e.g. the filename or "không bac91 buộc").
func CopyFileToClipboard(ctx context.Context, s BlobStorage, title, filePath string) error {
	ValidateContext(ctx)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("buckey_storage: read file %q: %w", filePath, err)
	}
	return StoreClipboard(ctx, s, title, string(data))
}

// CopyFileToStorageAsKey uses the file's base name (or full path) as the storage key
// if key is empty. Otherwise behaves like CopyFileToStorage.
func CopyFileToStorageAsKey(ctx context.Context, s BlobStorage, key, filePath string) error {
	if key == "" {
		key = filepath.Base(filePath)
		if key == "" || key == "." {
			key = filePath
		}
	}
	return CopyFileToStorage(ctx, s, key, filePath)
}
