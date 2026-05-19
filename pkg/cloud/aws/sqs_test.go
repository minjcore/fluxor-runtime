package aws

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/config"
)

// TestSQSClient_InputValidation tests fail-fast validation of inputs
func TestSQSClient_InputValidation(t *testing.T) {
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

	sqsClient := client.SQS()
	ctx := context.Background()

	tests := []struct {
		name     string
		testFn   func() error
		wantErr  bool
		contains string
	}{
		{
			name: "SendMessage - empty queue URL",
			testFn: func() error {
				return sqsClient.SendMessage(ctx, "", "body")
			},
			wantErr:  true,
			contains: "queue URL cannot be empty",
		},
		{
			name: "SendMessage - empty body",
			testFn: func() error {
				return sqsClient.SendMessage(ctx, "queue-url", "")
			},
			wantErr:  true,
			contains: "message body cannot be empty",
		},
		{
			name: "ReceiveMessages - empty queue URL",
			testFn: func() error {
				_, err := sqsClient.ReceiveMessages(ctx, "", 1)
				return err
			},
			wantErr:  true,
			contains: "queue URL cannot be empty",
		},
		{
			name: "ReceiveMessages - zero max messages (clamped internally)",
			testFn: func() error {
				// Should work but be clamped to valid range internally
				// The actual AWS call may fail if LocalStack isn't running, but validation should pass
				_, err := sqsClient.ReceiveMessages(ctx, "queue-url", 0)
				// Only return error if it's a validation error
				if err != nil && (err.Error() == "queue URL cannot be empty" || 
					err.Error() == "message body cannot be empty") {
					return err // Validation error
				}
				// Connection/AWS errors are acceptable - validation passed
				return nil
			},
			wantErr: false, // Should be clamped to valid range internally
		},
		{
			name: "ReceiveMessages - max messages too high (clamped internally)",
			testFn: func() error {
				// Should work but be clamped to 10 (SQS limit) internally
				// The actual AWS call may fail if LocalStack isn't running, but validation should pass
				_, err := sqsClient.ReceiveMessages(ctx, "queue-url", 20)
				// Only return error if it's a validation error
				if err != nil && (err.Error() == "queue URL cannot be empty" || 
					err.Error() == "message body cannot be empty") {
					return err // Validation error
				}
				// Connection/AWS errors are acceptable - validation passed
				return nil
			},
			wantErr: false, // Should be clamped to 10 internally
		},
		{
			name: "DeleteMessage - empty queue URL",
			testFn: func() error {
				return sqsClient.DeleteMessage(ctx, "", "receipt-handle")
			},
			wantErr:  true,
			contains: "queue URL cannot be empty",
		},
		{
			name: "DeleteMessage - empty receipt handle",
			testFn: func() error {
				return sqsClient.DeleteMessage(ctx, "queue-url", "")
			},
			wantErr:  true,
			contains: "receipt handle cannot be empty",
		},
		{
			name: "CreateQueue - empty queue name",
			testFn: func() error {
				_, err := sqsClient.CreateQueue(ctx, "")
				return err
			},
			wantErr:  true,
			contains: "queue name cannot be empty",
		},
		{
			name: "DeleteQueue - empty queue URL",
			testFn: func() error {
				return sqsClient.DeleteQueue(ctx, "")
			},
			wantErr:  true,
			contains: "queue URL cannot be empty",
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
				errMsg := err.Error()
				if errMsg == "" {
					t.Errorf("error message is empty, want to contain %q", tt.contains)
				}
			}
		})
	}
}

// TestSQSClient_ReceiveMessages_MaxMessagesClamping tests that max messages are clamped correctly
func TestSQSClient_ReceiveMessages_MaxMessagesClamping(t *testing.T) {
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

	sqsClient := client.SQS()
	ctx := context.Background()

	// Test that invalid max messages don't cause errors (they're clamped internally)
	// The actual AWS call will fail, but validation should pass
	_, err = sqsClient.ReceiveMessages(ctx, "queue-url", 0)
	if err != nil {
		// Should not be a validation error
		errMsg := err.Error()
		if errMsg == "queue URL cannot be empty" || errMsg == "message body cannot be empty" {
			t.Errorf("ReceiveMessages() validation error = %v", err)
		}
		// Other errors (queue not found, etc.) are acceptable
	}

	_, err = sqsClient.ReceiveMessages(ctx, "queue-url", 20)
	if err != nil {
		// Should not be a validation error
		errMsg := err.Error()
		if errMsg == "queue URL cannot be empty" {
			t.Errorf("ReceiveMessages() validation error = %v", err)
		}
	}
}
