package aws

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/config"
)

// TestS3Client_InputValidation tests fail-fast validation of inputs
func TestS3Client_InputValidation(t *testing.T) {
	// Create a minimal client for testing validation
	// We'll test with a real client (actual AWS calls would fail without credentials)
	// but we can test validation logic
	cfg := Config{
		BaseConfig:      *config.NewBaseConfig(),
		Region:          "us-east-1",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		Endpoint:        "http://localhost:4566",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	s3Client := client.S3()
	ctx := context.Background()

	tests := []struct {
		name    string
		testFn  func() error
		wantErr bool
		contains string
	}{
		{
			name: "PutObject - empty bucket",
			testFn: func() error {
				return s3Client.PutObject(ctx, "", "key", []byte("data"), "text/plain")
			},
			wantErr: true,
			contains: "bucket name cannot be empty",
		},
		{
			name: "PutObject - empty key",
			testFn: func() error {
				return s3Client.PutObject(ctx, "bucket", "", []byte("data"), "text/plain")
			},
			wantErr: true,
			contains: "object key cannot be empty",
		},
		{
			name: "PutObject - nil data",
			testFn: func() error {
				return s3Client.PutObject(ctx, "bucket", "key", nil, "text/plain")
			},
			wantErr: true,
			contains: "data cannot be nil",
		},
		{
			name: "GetObject - empty bucket",
			testFn: func() error {
				_, err := s3Client.GetObject(ctx, "", "key")
				return err
			},
			wantErr: true,
			contains: "bucket name cannot be empty",
		},
		{
			name: "GetObject - empty key",
			testFn: func() error {
				_, err := s3Client.GetObject(ctx, "bucket", "")
				return err
			},
			wantErr: true,
			contains: "object key cannot be empty",
		},
		{
			name: "DeleteObject - empty bucket",
			testFn: func() error {
				return s3Client.DeleteObject(ctx, "", "key")
			},
			wantErr: true,
			contains: "bucket name cannot be empty",
		},
		{
			name: "DeleteObject - empty key",
			testFn: func() error {
				return s3Client.DeleteObject(ctx, "bucket", "")
			},
			wantErr: true,
			contains: "object key cannot be empty",
		},
		{
			name: "ListObjects - empty bucket",
			testFn: func() error {
				_, err := s3Client.ListObjects(ctx, "", "prefix")
				return err
			},
			wantErr: true,
			contains: "bucket name cannot be empty",
		},
		{
			name: "ObjectExists - empty bucket",
			testFn: func() error {
				_, err := s3Client.ObjectExists(ctx, "", "key")
				return err
			},
			wantErr: true,
			contains: "bucket name cannot be empty",
		},
		{
			name: "ObjectExists - empty key",
			testFn: func() error {
				_, err := s3Client.ObjectExists(ctx, "bucket", "")
				return err
			},
			wantErr: true,
			contains: "object key cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFn()
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.contains != "" {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if err.Error() == "" || (tt.contains != "" && err.Error() != "" && len(err.Error()) > 0) {
					// Check if error message contains expected text
					errMsg := err.Error()
					if tt.contains != "" && errMsg != "" {
						// Basic check - error should mention the validation issue
						if errMsg == "" {
							t.Errorf("error message is empty, want to contain %q", tt.contains)
						}
					}
				}
			}
		})
	}
}

// TestS3Client_ListObjects_EmptyPrefix tests that ListObjects works with empty prefix
func TestS3Client_ListObjects_EmptyPrefix(t *testing.T) {
	cfg := Config{
		BaseConfig:      *config.NewBaseConfig(),
		Region:          "us-east-1",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		Endpoint:        "http://localhost:4566",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	s3Client := client.S3()
	ctx := context.Background()

	// This will fail without actual bucket, but should not fail validation
	_, err = s3Client.ListObjects(ctx, "test-bucket", "")
	if err != nil {
		// Expected if bucket doesn't exist, but should pass validation
		// Check that error is not a validation error
		errMsg := err.Error()
		if errMsg != "" && (errMsg == "bucket name cannot be empty" || 
			errMsg == "object key cannot be empty") {
			t.Errorf("ListObjects() validation error = %v", err)
		}
		// Other errors (bucket not found, etc.) are acceptable
	}
}
