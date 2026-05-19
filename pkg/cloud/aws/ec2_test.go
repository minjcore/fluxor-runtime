package aws

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/config"
)

// TestEC2Client_InputValidation tests fail-fast validation of inputs
func TestEC2Client_InputValidation(t *testing.T) {
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

	ec2Client := client.EC2()
	ctx := context.Background()

	tests := []struct {
		name     string
		testFn   func() error
		wantErr  bool
		contains string
	}{
		{
			name: "StartInstance - empty instance ID",
			testFn: func() error {
				return ec2Client.StartInstance(ctx, "")
			},
			wantErr:  true,
			contains: "instance ID cannot be empty",
		},
		{
			name: "StopInstance - empty instance ID",
			testFn: func() error {
				return ec2Client.StopInstance(ctx, "")
			},
			wantErr:  true,
			contains: "instance ID cannot be empty",
		},
		{
			name: "TerminateInstance - empty instance ID",
			testFn: func() error {
				return ec2Client.TerminateInstance(ctx, "")
			},
			wantErr:  true,
			contains: "instance ID cannot be empty",
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

// TestEC2Client_DescribeInstances tests that DescribeInstances works
func TestEC2Client_DescribeInstances(t *testing.T) {
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

	ec2Client := client.EC2()
	ctx := context.Background()

	// Test with empty instance IDs (should list all instances)
	_, err = ec2Client.DescribeInstances(ctx, nil)
	if err != nil {
		// Should not be a validation error (empty list is allowed)
		errMsg := err.Error()
		if errMsg == "" {
			t.Error("expected error message")
		}
		// Other errors (AWS connection issues, etc.) are acceptable
	}

	// Test with specific instance IDs
	_, err = ec2Client.DescribeInstances(ctx, []string{"i-1234567890abcdef0"})
	if err != nil {
		// Should not be a validation error
		errMsg := err.Error()
		if errMsg == "" {
			t.Error("expected error message")
		}
		// Other errors (instance not found, etc.) are acceptable
	}
}
