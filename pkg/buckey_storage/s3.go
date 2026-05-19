package buckey_storage

import (
	"context"
	"fmt"
	"strings"
)

// S3Ops is the minimal S3 API needed for BlobStorage. Implementations can wrap AWS SDK or MinIO.
type S3Ops interface {
	PutObject(ctx context.Context, bucket, key string, data []byte) error
	GetObject(ctx context.Context, bucket, key string) ([]byte, error)
	DeleteObject(ctx context.Context, bucket, key string) error
	ListObjects(ctx context.Context, bucket, prefix string) ([]string, error)
}

// S3Storage implements BlobStorage using S3Ops (e.g. AWS S3 or MinIO).
type S3Storage struct {
	ops    S3Ops
	bucket string
	prefix string
}

// NewS3 creates a BlobStorage that uses the given S3Ops, bucket, and key prefix.
func NewS3(ops S3Ops, bucket, prefix string) (*S3Storage, error) {
	if bucket == "" {
		return nil, fmt.Errorf("buckey_storage: S3 bucket cannot be empty")
	}
	return &S3Storage{ops: ops, bucket: bucket, prefix: strings.TrimPrefix(prefix, "/")}, nil
}

func (s *S3Storage) fullKey(key string) string {
	if s.prefix == "" {
		return key
	}
	return s.prefix + key
}

// Put stores data under key.
func (s *S3Storage) Put(ctx context.Context, key string, data []byte) error {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return err
	}
	if data == nil {
		data = []byte{}
	}
	return s.ops.PutObject(ctx, s.bucket, s.fullKey(key), data)
}

// Get retrieves data by key.
func (s *S3Storage) Get(ctx context.Context, key string) ([]byte, error) {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return nil, err
	}
	data, err := s.ops.GetObject(ctx, s.bucket, s.fullKey(key))
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Delete removes the blob at key.
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return err
	}
	return s.ops.DeleteObject(ctx, s.bucket, s.fullKey(key))
}

// List returns keys with the given prefix (without the config prefix).
func (s *S3Storage) List(ctx context.Context, prefix string) ([]string, error) {
	ValidateContext(ctx)
	fullPrefix := s.fullKey(prefix)
	keys, err := s.ops.ListObjects(ctx, s.bucket, fullPrefix)
	if err != nil {
		return nil, err
	}
	if s.prefix == "" {
		return keys, nil
	}
	var out []string
	for _, k := range keys {
		if strings.HasPrefix(k, s.prefix) {
			out = append(out, k[len(s.prefix):])
		}
	}
	return out, nil
}
