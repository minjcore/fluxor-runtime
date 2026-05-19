package aws_test

import (
	"context"
	"os"
	"testing"

	"github.com/fluxorio/fluxor/pkg/cloud/aws"
	"github.com/fluxorio/fluxor/pkg/config"
	"github.com/fluxorio/fluxor/pkg/core"
)

// ExampleAWSComponent demonstrates how to use AWS component in a verticle
// This example shows how BaseConfig is embedded and can be used
func ExampleAWSComponent() {
	// Create AWS component with default config (uses env vars)
	// DefaultConfig() initializes BaseConfig with defaults
	config := aws.DefaultConfig()
	config.Region = "us-east-1"
	
	// BaseConfig fields are available through embedding
	config.Service.Name = "aws-service"
	config.Server.Addr = ":8080"
	config.Profile = "production"
	config.Environment = "prod"

	awsComponent := aws.NewAWSComponent(config)

	// In a verticle's Start method:
	// Create GoCMD and FluxorContext
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Create FluxorContext (internal function, but shown for example)
	// In real usage, this would be provided by the verticle's Start method
	// fluxorCtx := newFluxorContext(ctx, gocmd) // This is internal
	// For this example, we'll use a mock approach
	_ = gocmd // Use gocmd to create context in real implementation

	// Note: In actual verticle implementation, FluxorContext is provided
	// by the framework in the Start() method

	// Note: Component must be started with a proper FluxorContext
	// This is typically done in a verticle's Start() method
	_ = awsComponent // Component created but not started in this example
	// See README.md for full usage examples
}

// ExampleAWSWithCredentials demonstrates using explicit credentials
// This example shows how to initialize Config with BaseConfig
func ExampleAWSComponent_withCredentials() {
	config := aws.Config{
		// Initialize BaseConfig explicitly
		BaseConfig:      *config.NewBaseConfig(),
		Region:          "us-west-2",
		AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		Timeout:         "30s",
		MaxRetries:      3,
	}
	
	// Access BaseConfig fields
	config.Service.Name = "aws-worker"
	config.Environment = "staging"

	awsComponent := aws.NewAWSComponent(config)
	// Component would be started in a verticle's Start() method
	// with a proper FluxorContext provided by the framework
	_ = awsComponent
}

// ExampleAWSComponent_localStack demonstrates using LocalStack for local development
// This example shows BaseConfig usage with LocalStack
func ExampleAWSComponent_localStack() {
	config := aws.DefaultConfig()
	config.Region = "us-east-1"
	config.Endpoint = "http://localhost:4566" // LocalStack endpoint
	
	// BaseConfig can be customized for local development
	config.Environment = "local"
	config.Profile = "localstack"

	awsComponent := aws.NewAWSComponent(config)
	// Component would be started in a verticle's Start() method
	// All AWS services work the same way with LocalStack
	_ = awsComponent
}

// TestAWSConfigValidation tests configuration validation
func TestAWSConfigValidation(t *testing.T) {
	// Test missing region
	cfg := aws.Config{
		BaseConfig: *config.NewBaseConfig(),
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("expected error for missing region")
	}

	// Test valid config with region from env
	os.Setenv("AWS_REGION", "us-east-1")
	defer os.Unsetenv("AWS_REGION")

	cfg = aws.Config{
		BaseConfig: *config.NewBaseConfig(),
	}
	err = cfg.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test invalid credentials (one without the other)
	cfg = aws.Config{
		BaseConfig:  *config.NewBaseConfig(),
		Region:      "us-east-1",
		AccessKeyID: "test-key",
		// Missing SecretAccessKey
	}
	err = cfg.Validate()
	if err == nil {
		t.Error("expected error for incomplete credentials")
	}
}
