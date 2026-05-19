package s3

import (
	"context"

	"github.com/fluxorio/fluxor/pkg/cloud/aws"
)

// Client provides a high-level interface for S3 operations
// It wraps the AWS S3 client with convenience methods
type Client struct {
	s3Client aws.S3Client
	config   Config
}

// NewClient creates a new S3 client from a component
func NewClient(component *S3Component) (*Client, error) {
	s3Client, err := component.Client()
	if err != nil {
		return nil, err
	}

	return &Client{
		s3Client: s3Client,
		config:   component.Config(),
	}, nil
}

// PutObject uploads an object to S3
// Uses default bucket from config if bucket is empty
func (c *Client) PutObject(ctx context.Context, bucket, key string, data []byte, contentType string) error {
	// Use default bucket if not specified
	if bucket == "" {
		bucket = c.config.Bucket
	}
	if bucket == "" {
		return &ClientError{
			Code:    "MISSING_BUCKET",
			Message: "bucket name is required",
		}
	}

	// Use default content type if not specified
	if contentType == "" {
		contentType = c.config.DefaultContentType
	}

	return c.s3Client.PutObject(ctx, bucket, key, data, contentType)
}

// GetObject downloads an object from S3
// Uses default bucket from config if bucket is empty
func (c *Client) GetObject(ctx context.Context, bucket, key string) ([]byte, error) {
	// Use default bucket if not specified
	if bucket == "" {
		bucket = c.config.Bucket
	}
	if bucket == "" {
		return nil, &ClientError{
			Code:    "MISSING_BUCKET",
			Message: "bucket name is required",
		}
	}

	return c.s3Client.GetObject(ctx, bucket, key)
}

// DeleteObject deletes an object from S3
// Uses default bucket from config if bucket is empty
func (c *Client) DeleteObject(ctx context.Context, bucket, key string) error {
	// Use default bucket if not specified
	if bucket == "" {
		bucket = c.config.Bucket
	}
	if bucket == "" {
		return &ClientError{
			Code:    "MISSING_BUCKET",
			Message: "bucket name is required",
		}
	}

	return c.s3Client.DeleteObject(ctx, bucket, key)
}

// ListObjects lists objects in a bucket
// Uses default bucket from config if bucket is empty
func (c *Client) ListObjects(ctx context.Context, bucket, prefix string) ([]string, error) {
	// Use default bucket if not specified
	if bucket == "" {
		bucket = c.config.Bucket
	}
	if bucket == "" {
		return nil, &ClientError{
			Code:    "MISSING_BUCKET",
			Message: "bucket name is required",
		}
	}

	return c.s3Client.ListObjects(ctx, bucket, prefix)
}

// ObjectExists checks if an object exists in S3
// Uses default bucket from config if bucket is empty
func (c *Client) ObjectExists(ctx context.Context, bucket, key string) (bool, error) {
	// Use default bucket if not specified
	if bucket == "" {
		bucket = c.config.Bucket
	}
	if bucket == "" {
		return false, &ClientError{
			Code:    "MISSING_BUCKET",
			Message: "bucket name is required",
		}
	}

	return c.s3Client.ObjectExists(ctx, bucket, key)
}

// ClientError represents a client error
type ClientError struct {
	Code    string
	Message string
}

func (e *ClientError) Error() string {
	return e.Message
}
