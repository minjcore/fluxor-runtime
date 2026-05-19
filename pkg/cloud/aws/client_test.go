package aws

import (
	"context"
	"os"
	"testing"

	"github.com/fluxorio/fluxor/pkg/config"
)

func TestNewClient(t *testing.T) {
	// Save original env vars
	originalRegion := os.Getenv("AWS_REGION")
	originalEndpoint := os.Getenv("AWS_ENDPOINT")

	defer func() {
		if originalRegion != "" {
			os.Setenv("AWS_REGION", originalRegion)
		} else {
			os.Unsetenv("AWS_REGION")
		}
		if originalEndpoint != "" {
			os.Setenv("AWS_ENDPOINT", originalEndpoint)
		} else {
			os.Unsetenv("AWS_ENDPOINT")
		}
	}()

	tests := []struct {
		name    string
		config  Config
		wantErr bool
		check   func(*testing.T, AWSClient, error)
	}{
		{
			name: "valid config with region",
			config: Config{
				BaseConfig:      *config.NewBaseConfig(),
				Region:          "us-east-1",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
			},
			wantErr: false,
			check: func(t *testing.T, client AWSClient, err error) {
				if err != nil {
					t.Errorf("NewClient() error = %v, want nil", err)
					return
				}
				if client == nil {
					t.Error("NewClient() returned nil client")
					return
				}
				if client.Region() != "us-east-1" {
					t.Errorf("NewClient() Region() = %v, want us-east-1", client.Region())
				}
				// Verify service clients are initialized
				if client.S3() == nil {
					t.Error("NewClient() S3() returned nil")
				}
				if client.SQS() == nil {
					t.Error("NewClient() SQS() returned nil")
				}
				if client.SNS() == nil {
					t.Error("NewClient() SNS() returned nil")
				}
				if client.Lambda() == nil {
					t.Error("NewClient() Lambda() returned nil")
				}
				if client.EC2() == nil {
					t.Error("NewClient() EC2() returned nil")
				}
			},
		},
		{
			name: "invalid config - missing region",
			config: Config{
				BaseConfig: *config.NewBaseConfig(),
			},
			wantErr: true,
			check: func(t *testing.T, client AWSClient, err error) {
				if err == nil {
					t.Error("NewClient() expected error, got nil")
				}
				if client != nil {
					t.Error("NewClient() expected nil client on error")
				}
			},
		},
		{
			name: "config with endpoint for LocalStack",
			config: Config{
				BaseConfig:      *config.NewBaseConfig(),
				Region:          "us-east-1",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				Endpoint:        "http://localhost:4566",
			},
			wantErr: false,
			check: func(t *testing.T, client AWSClient, err error) {
				if err != nil {
					t.Errorf("NewClient() error = %v, want nil", err)
					return
				}
				if client == nil {
					t.Error("NewClient() returned nil client")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil {
				tt.check(t, client, err)
			}
		})
	}
}

func TestAWSClient_ServiceClients(t *testing.T) {
	// This test verifies that service clients are properly initialized
	// We'll use a minimal valid config (actual AWS calls would require real credentials)
	os.Setenv("AWS_REGION", "us-east-1")
	defer os.Unsetenv("AWS_REGION")

	cfg := Config{
		BaseConfig:      *config.NewBaseConfig(),
		Region:          "us-east-1",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		Endpoint:        "http://localhost:4566", // Use LocalStack endpoint to avoid real AWS calls
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Test that all service clients are initialized
	if client.S3() == nil {
		t.Error("S3() returned nil")
	}
	if client.SQS() == nil {
		t.Error("SQS() returned nil")
	}
	if client.SNS() == nil {
		t.Error("SNS() returned nil")
	}
	if client.Lambda() == nil {
		t.Error("Lambda() returned nil")
	}
	if client.EC2() == nil {
		t.Error("EC2() returned nil")
	}

	// Test that Region() returns the correct value
	if client.Region() != "us-east-1" {
		t.Errorf("Region() = %v, want us-east-1", client.Region())
	}
}

func TestAWSClient_Region(t *testing.T) {
	cfg := Config{
		BaseConfig:      *config.NewBaseConfig(),
		Region:          "eu-west-1",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		Endpoint:        "http://localhost:4566",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	region := client.Region()
	if region != "eu-west-1" {
		t.Errorf("Region() = %v, want eu-west-1", region)
	}
}

// TestAWSClient_LoadConfigWithIAMRole tests that client can be created
// without explicit credentials (using IAM role)
func TestAWSClient_LoadConfigWithIAMRole(t *testing.T) {
	// Save original env
	originalRegion := os.Getenv("AWS_REGION")
	originalAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	originalSecret := os.Getenv("AWS_SECRET_ACCESS_KEY")

	defer func() {
		if originalRegion != "" {
			os.Setenv("AWS_REGION", originalRegion)
		} else {
			os.Unsetenv("AWS_REGION")
		}
		if originalAccessKey != "" {
			os.Setenv("AWS_ACCESS_KEY_ID", originalAccessKey)
		} else {
			os.Unsetenv("AWS_ACCESS_KEY_ID")
		}
		if originalSecret != "" {
			os.Setenv("AWS_SECRET_ACCESS_KEY", originalSecret)
		} else {
			os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		}
	}()

	// Clear credentials to simulate IAM role usage
	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	os.Setenv("AWS_REGION", "us-east-1")

	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		Region:     "us-east-1",
		// No credentials - should use IAM role or default credential chain
	}

	// This will fail if credentials are required, but structure should be valid
	// In real AWS environment, this would work with IAM roles
	_, err := NewClient(cfg)
	// We expect this might fail in test environment without real AWS credentials
	// But the config validation should pass
	if err != nil {
		// Check if error is from config validation or AWS SDK loading
		if _, ok := err.(*ConfigError); ok {
			t.Errorf("NewClient() config error = %v", err)
		}
		// AWS SDK loading errors are acceptable in test environment
		t.Logf("NewClient() AWS SDK loading error (expected in test env): %v", err)
	}
}

func TestLoadAWSConfig(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "config with region and credentials",
			config: Config{
				Region:          "us-east-1",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
			},
			wantErr: false,
		},
		{
			name: "config with custom endpoint",
			config: Config{
				Region:          "us-east-1",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				Endpoint:        "http://localhost:4566",
			},
			wantErr: false,
		},
		{
			name: "config with session token",
			config: Config{
				Region:          "us-east-1",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				SessionToken:    "test-token",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := loadAWSConfig(ctx, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadAWSConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && cfg.Region != tt.config.Region {
				t.Errorf("loadAWSConfig() Region = %v, want %v", cfg.Region, tt.config.Region)
			}
		})
	}
}

func TestIsGoogleCloud(t *testing.T) {
	// Save original env vars
	originalGoogleProject := os.Getenv("GOOGLE_CLOUD_PROJECT")
	originalGCPProject := os.Getenv("GCP_PROJECT")
	originalMetadataHost := os.Getenv("GCE_METADATA_HOST")
	originalMetadataRoot := os.Getenv("GCE_METADATA_ROOT")

	defer func() {
		if originalGoogleProject != "" {
			os.Setenv("GOOGLE_CLOUD_PROJECT", originalGoogleProject)
		} else {
			os.Unsetenv("GOOGLE_CLOUD_PROJECT")
		}
		if originalGCPProject != "" {
			os.Setenv("GCP_PROJECT", originalGCPProject)
		} else {
			os.Unsetenv("GCP_PROJECT")
		}
		if originalMetadataHost != "" {
			os.Setenv("GCE_METADATA_HOST", originalMetadataHost)
		} else {
			os.Unsetenv("GCE_METADATA_HOST")
		}
		if originalMetadataRoot != "" {
			os.Setenv("GCE_METADATA_ROOT", originalMetadataRoot)
		} else {
			os.Unsetenv("GCE_METADATA_ROOT")
		}
	}()

	tests := []struct {
		name           string
		setup          func()
		expectedResult bool
	}{
		{
			name: "detected via GOOGLE_CLOUD_PROJECT",
			setup: func() {
				os.Setenv("GOOGLE_CLOUD_PROJECT", "my-gcp-project")
				os.Unsetenv("GCP_PROJECT")
				os.Unsetenv("GCE_METADATA_HOST")
				os.Unsetenv("GCE_METADATA_ROOT")
			},
			expectedResult: true,
		},
		{
			name: "detected via GCP_PROJECT",
			setup: func() {
				os.Unsetenv("GOOGLE_CLOUD_PROJECT")
				os.Setenv("GCP_PROJECT", "my-gcp-project")
				os.Unsetenv("GCE_METADATA_HOST")
				os.Unsetenv("GCE_METADATA_ROOT")
			},
			expectedResult: true,
		},
		{
			name: "detected via GCE_METADATA_HOST",
			setup: func() {
				os.Unsetenv("GOOGLE_CLOUD_PROJECT")
				os.Unsetenv("GCP_PROJECT")
				os.Setenv("GCE_METADATA_HOST", "169.254.169.254")
				os.Unsetenv("GCE_METADATA_ROOT")
			},
			expectedResult: true,
		},
		{
			name: "detected via GCE_METADATA_ROOT",
			setup: func() {
				os.Unsetenv("GOOGLE_CLOUD_PROJECT")
				os.Unsetenv("GCP_PROJECT")
				os.Unsetenv("GCE_METADATA_HOST")
				os.Setenv("GCE_METADATA_ROOT", "http://169.254.169.254")
			},
			expectedResult: true,
		},
		{
			name: "not detected - no env vars",
			setup: func() {
				os.Unsetenv("GOOGLE_CLOUD_PROJECT")
				os.Unsetenv("GCP_PROJECT")
				os.Unsetenv("GCE_METADATA_HOST")
				os.Unsetenv("GCE_METADATA_ROOT")
			},
			expectedResult: false, // Will check metadata server, likely false in test env
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result := IsGoogleCloud()
			// Note: If no env vars are set, it tries to check metadata server
			// which will likely fail in test environment, but we can't be certain
			// So we only assert for cases where env vars are set
			if tt.name != "not detected - no env vars" {
				if result != tt.expectedResult {
					t.Errorf("IsGoogleCloud() = %v, want %v", result, tt.expectedResult)
				}
			} else {
				// For the metadata server check case, we just verify it doesn't panic
				// and returns a boolean
				_ = result
			}
		})
	}
}

func TestGetGoogleCloudProject(t *testing.T) {
	// Save original env vars
	originalGoogleProject := os.Getenv("GOOGLE_CLOUD_PROJECT")
	originalGCPProject := os.Getenv("GCP_PROJECT")

	defer func() {
		if originalGoogleProject != "" {
			os.Setenv("GOOGLE_CLOUD_PROJECT", originalGoogleProject)
		} else {
			os.Unsetenv("GOOGLE_CLOUD_PROJECT")
		}
		if originalGCPProject != "" {
			os.Setenv("GCP_PROJECT", originalGCPProject)
		} else {
			os.Unsetenv("GCP_PROJECT")
		}
	}()

	tests := []struct {
		name           string
		setup          func()
		expectedResult string
	}{
		{
			name: "returns GOOGLE_CLOUD_PROJECT",
			setup: func() {
				os.Setenv("GOOGLE_CLOUD_PROJECT", "my-gcp-project")
				os.Unsetenv("GCP_PROJECT")
			},
			expectedResult: "my-gcp-project",
		},
		{
			name: "returns GCP_PROJECT when GOOGLE_CLOUD_PROJECT not set",
			setup: func() {
				os.Unsetenv("GOOGLE_CLOUD_PROJECT")
				os.Setenv("GCP_PROJECT", "my-gcp-project-alt")
			},
			expectedResult: "my-gcp-project-alt",
		},
		{
			name: "GOOGLE_CLOUD_PROJECT takes precedence over GCP_PROJECT",
			setup: func() {
				os.Setenv("GOOGLE_CLOUD_PROJECT", "primary-project")
				os.Setenv("GCP_PROJECT", "secondary-project")
			},
			expectedResult: "primary-project",
		},
		{
			name: "returns empty string when no env vars set",
			setup: func() {
				os.Unsetenv("GOOGLE_CLOUD_PROJECT")
				os.Unsetenv("GCP_PROJECT")
			},
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result := GetGoogleCloudProject()
			if result != tt.expectedResult {
				t.Errorf("GetGoogleCloudProject() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}
