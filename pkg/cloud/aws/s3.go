package aws

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// s3Client implements S3Client
type s3Client struct {
	client *s3.Client
}

// newS3Client creates a new S3 client
func newS3Client(cfg aws.Config) S3Client {
	return &s3Client{
		client: s3.NewFromConfig(cfg),
	}
}

// PutObject uploads an object to S3
func (c *s3Client) PutObject(ctx context.Context, bucket, key string, data []byte, contentType string) error {
	// Fail-fast: Validate inputs
	if bucket == "" {
		return fmt.Errorf("bucket name cannot be empty")
	}
	if key == "" {
		return fmt.Errorf("object key cannot be empty")
	}
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	}

	_, err := c.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put object to S3: %w", err)
	}

	return nil
}

// GetObject downloads an object from S3
func (c *s3Client) GetObject(ctx context.Context, bucket, key string) ([]byte, error) {
	// Fail-fast: Validate inputs
	if bucket == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}
	if key == "" {
		return nil, fmt.Errorf("object key cannot be empty")
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	output, err := c.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer output.Body.Close()

	data, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object body: %w", err)
	}

	return data, nil
}

// DeleteObject deletes an object from S3
func (c *s3Client) DeleteObject(ctx context.Context, bucket, key string) error {
	// Fail-fast: Validate inputs
	if bucket == "" {
		return fmt.Errorf("bucket name cannot be empty")
	}
	if key == "" {
		return fmt.Errorf("object key cannot be empty")
	}

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err := c.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete object from S3: %w", err)
	}

	return nil
}

// ListObjects lists objects in a bucket
func (c *s3Client) ListObjects(ctx context.Context, bucket, prefix string) ([]string, error) {
	// Fail-fast: Validate inputs
	if bucket == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	var keys []string
	paginator := s3.NewListObjectsV2Paginator(c.client, input)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects from S3: %w", err)
		}

		for _, obj := range output.Contents {
			if obj.Key != nil {
				keys = append(keys, *obj.Key)
			}
		}
	}

	return keys, nil
}

// ObjectExists checks if an object exists
func (c *s3Client) ObjectExists(ctx context.Context, bucket, key string) (bool, error) {
	// Fail-fast: Validate inputs
	if bucket == "" {
		return false, fmt.Errorf("bucket name cannot be empty")
	}
	if key == "" {
		return false, fmt.Errorf("object key cannot be empty")
	}

	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err := c.client.HeadObject(ctx, input)
	if err != nil {
		// Check if error is "not found" (NoSuchKey or NotFound)
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return false, nil
		}
		var nf *types.NotFound
		if errors.As(err, &nf) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}
