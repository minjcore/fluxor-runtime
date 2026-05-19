package aws

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/config"
)

// TestSNSClient_InputValidation tests fail-fast validation of inputs
func TestSNSClient_InputValidation(t *testing.T) {
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

	snsClient := client.SNS()
	ctx := context.Background()

	tests := []struct {
		name     string
		testFn   func() error
		wantErr  bool
		contains string
	}{
		{
			name: "Publish - empty topic ARN",
			testFn: func() error {
				return snsClient.Publish(ctx, "", "message")
			},
			wantErr:  true,
			contains: "topic ARN cannot be empty",
		},
		{
			name: "Publish - empty message",
			testFn: func() error {
				return snsClient.Publish(ctx, "topic-arn", "")
			},
			wantErr:  true,
			contains: "message cannot be empty",
		},
		{
			name: "Subscribe - empty topic ARN",
			testFn: func() error {
				_, err := snsClient.Subscribe(ctx, "", "http", "endpoint")
				return err
			},
			wantErr:  true,
			contains: "topic ARN cannot be empty",
		},
		{
			name: "Subscribe - empty protocol",
			testFn: func() error {
				_, err := snsClient.Subscribe(ctx, "topic-arn", "", "endpoint")
				return err
			},
			wantErr:  true,
			contains: "protocol cannot be empty",
		},
		{
			name: "Subscribe - empty endpoint",
			testFn: func() error {
				_, err := snsClient.Subscribe(ctx, "topic-arn", "http", "")
				return err
			},
			wantErr:  true,
			contains: "endpoint cannot be empty",
		},
		{
			name: "Unsubscribe - empty subscription ARN",
			testFn: func() error {
				return snsClient.Unsubscribe(ctx, "")
			},
			wantErr:  true,
			contains: "subscription ARN cannot be empty",
		},
		{
			name: "CreateTopic - empty topic name",
			testFn: func() error {
				_, err := snsClient.CreateTopic(ctx, "")
				return err
			},
			wantErr:  true,
			contains: "topic name cannot be empty",
		},
		{
			name: "DeleteTopic - empty topic ARN",
			testFn: func() error {
				return snsClient.DeleteTopic(ctx, "")
			},
			wantErr:  true,
			contains: "topic ARN cannot be empty",
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
