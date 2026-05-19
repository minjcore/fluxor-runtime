package buckey_storage

import (
	"bytes"
	"context"
	"io"
)

// BlobStorageStream wraps any BlobStorage to implement BlobStorageStreaming
// by buffering streamed data in memory (suitable for moderate-sized blobs).
type BlobStorageStream struct {
	BlobStorage
}

// NewBlobStorageStream returns a BlobStorageStreaming that uses the given store;
// PutStream/GetStream buffer in memory.
func NewBlobStorageStream(s BlobStorage) *BlobStorageStream {
	return &BlobStorageStream{BlobStorage: s}
}

// PutStream reads r until EOF and stores the data under key.
func (b *BlobStorageStream) PutStream(ctx context.Context, key string, r io.Reader) error {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return err
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return b.Put(ctx, key, data)
}

// GetStream returns a ReadCloser for the blob; caller must Close it.
func (b *BlobStorageStream) GetStream(ctx context.Context, key string) (io.ReadCloser, error) {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return nil, err
	}
	data, err := b.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

// Ensure BlobStorageStream implements BlobStorageStreaming.
var _ BlobStorageStreaming = (*BlobStorageStream)(nil)
