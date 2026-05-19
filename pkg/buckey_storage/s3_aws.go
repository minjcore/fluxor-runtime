package buckey_storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/fluxorio/fluxor/pkg/cloud/aws"
)

// s3OpsAdapter adapts aws.S3Client to buckey_storage.S3Ops (fixed contentType).
type s3OpsAdapter struct {
	client aws.S3Client
}

func (a *s3OpsAdapter) PutObject(ctx context.Context, bucket, key string, data []byte) error {
	return a.client.PutObject(ctx, bucket, key, data, "application/octet-stream")
}

func (a *s3OpsAdapter) GetObject(ctx context.Context, bucket, key string) ([]byte, error) {
	data, err := a.client.GetObject(ctx, bucket, key)
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

func (a *s3OpsAdapter) DeleteObject(ctx context.Context, bucket, key string) error {
	return a.client.DeleteObject(ctx, bucket, key)
}

func (a *s3OpsAdapter) ListObjects(ctx context.Context, bucket, prefix string) ([]string, error) {
	return a.client.ListObjects(ctx, bucket, prefix)
}

// NewS3FromConfig creates S3 BlobStorage from storage config using pkg/cloud/aws.
// Config must have Backend "s3", Bucket set; Region/AccessKeyID/SecretAccessKey optional (env used).
func NewS3FromConfig(cfg *Config) (BlobStorage, error) {
	if cfg == nil || cfg.Bucket == "" {
		return nil, errS3Config
	}
	awsCfg := aws.DefaultConfig()
	awsCfg.Region = cfg.Region
	if awsCfg.Region == "" {
		awsCfg.Region = "us-east-1"
	}
	awsCfg.AccessKeyID = cfg.AccessKeyID
	awsCfg.SecretAccessKey = cfg.SecretAccessKey
	client, err := aws.NewClient(awsCfg)
	if err != nil {
		return nil, err
	}
	adapter := &s3OpsAdapter{client: client.S3()}
	return NewS3(adapter, cfg.Bucket, cfg.Prefix)
}

var errS3Config = fmt.Errorf("buckey_storage: S3 backend requires Config.Bucket")
