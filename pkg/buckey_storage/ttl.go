package buckey_storage

import (
	"context"
	"encoding/binary"
	"time"
)

const ttlHeaderSize = 8

// TTLStorage wraps a BlobStorage to add TTL: values expire after DefaultTTL (0 = no expiry).
// Stored value format: 8-byte little-endian expiry Unix nano, then raw data.
type TTLStorage struct {
	BlobStorage
	DefaultTTL time.Duration
}

// NewTTLStorage wraps s with optional default TTL. If DefaultTTL is 0, Put behaves like the underlying store (no expiry).
func NewTTLStorage(s BlobStorage, defaultTTL time.Duration) *TTLStorage {
	return &TTLStorage{BlobStorage: s, DefaultTTL: defaultTTL}
}

// Put stores data with default TTL (if DefaultTTL > 0).
func (t *TTLStorage) Put(ctx context.Context, key string, data []byte) error {
	return t.PutWithTTL(ctx, key, data, t.DefaultTTL)
}

// PutWithTTL stores data that expires after ttl. ttl 0 means no expiry (store raw).
func (t *TTLStorage) PutWithTTL(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return err
	}
	if data == nil {
		data = []byte{}
	}
	if ttl <= 0 {
		return t.BlobStorage.Put(ctx, key, data)
	}
	expiry := time.Now().Add(ttl).UnixNano()
	buf := make([]byte, ttlHeaderSize+len(data))
	binary.LittleEndian.PutUint64(buf[:ttlHeaderSize], uint64(expiry))
	copy(buf[ttlHeaderSize:], data)
	return t.BlobStorage.Put(ctx, key, buf)
}

// Get returns data only if not expired; otherwise ErrNotFound.
func (t *TTLStorage) Get(ctx context.Context, key string) ([]byte, error) {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return nil, err
	}
	data, err := t.BlobStorage.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if len(data) < ttlHeaderSize {
		return data, nil
	}
	expiry := int64(binary.LittleEndian.Uint64(data[:ttlHeaderSize]))
	if time.Now().UnixNano() >= expiry {
		_ = t.BlobStorage.Delete(ctx, key)
		return nil, ErrNotFound
	}
	return data[ttlHeaderSize:], nil
}

// List returns keys; expired keys may still appear until next Get. Prefer listing then Get each to filter.
func (t *TTLStorage) List(ctx context.Context, prefix string) ([]string, error) {
	return t.BlobStorage.List(ctx, prefix)
}
