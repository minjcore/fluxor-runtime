package buckey_storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// ErrNotFound is returned when a key does not exist.
var ErrNotFound = errors.New("buckey_storage: key not found")

// BlobStorage provides key-based blob storage operations.
type BlobStorage interface {
	// Put stores data under key. Overwrites if key exists.
	Put(ctx context.Context, key string, data []byte) error

	// Get retrieves data by key. Returns error if key not found.
	Get(ctx context.Context, key string) ([]byte, error)

	// Delete removes the blob at key. No error if key does not exist.
	Delete(ctx context.Context, key string) error

	// List returns keys with the given prefix (empty prefix = all keys).
	List(ctx context.Context, prefix string) ([]string, error)
}

// BlobStorageStreaming extends BlobStorage with streaming Put/Get for large blobs.
// Implementations may buffer in memory (e.g. BlobStorageStream) or stream natively (S3/FS).
type BlobStorageStreaming interface {
	BlobStorage
	// PutStream stores data from r under key.
	PutStream(ctx context.Context, key string, r io.Reader) error
	// GetStream returns a reader for the blob; caller must Close it.
	GetStream(ctx context.Context, key string) (io.ReadCloser, error)
}

// ValidateKey validates a blob storage key.
// Fail-fast: Returns error if key is invalid.
func ValidateKey(key string) error {
	if key == "" {
		return fmt.Errorf("fail-fast: buckey_storage key cannot be empty")
	}
	if len(key) > 1024 {
		return fmt.Errorf("fail-fast: buckey_storage key too long (max 1024 characters), got %d", len(key))
	}
	return nil
}

// ValidateContext validates context. Panics if nil.
func ValidateContext(ctx context.Context) {
	failfast.NotNil(ctx, "context")
}
