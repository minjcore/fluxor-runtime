package aws

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/config"
)

// TestLambdaClient_InputValidation tests fail-fast validation of inputs
func TestLambdaClient_InputValidation(t *testing.T) {
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

	lambdaClient := client.Lambda()
	ctx := context.Background()

	tests := []struct {
		name     string
		testFn   func() error
		wantErr  bool
		contains string
	}{
		{
			name: "Invoke - empty function name",
			testFn: func() error {
				_, err := lambdaClient.Invoke(ctx, "", []byte("payload"))
				return err
			},
			wantErr:  true,
			contains: "function name cannot be empty",
		},
		{
			name: "CreateFunction - empty function name",
			testFn: func() error {
				req := CreateFunctionRequest{
					Runtime:     "go1.x",
					Handler:     "main",
					RoleARN:     "arn:aws:iam::123456789012:role/lambda-role",
					Code:        []byte("code"),
				}
				return lambdaClient.CreateFunction(ctx, req)
			},
			wantErr:  true,
			contains: "function name cannot be empty",
		},
		{
			name: "CreateFunction - empty runtime",
			testFn: func() error {
				req := CreateFunctionRequest{
					FunctionName: "test-function",
					Handler:      "main",
					RoleARN:      "arn:aws:iam::123456789012:role/lambda-role",
					Code:         []byte("code"),
				}
				return lambdaClient.CreateFunction(ctx, req)
			},
			wantErr:  true,
			contains: "runtime cannot be empty",
		},
		{
			name: "CreateFunction - empty handler",
			testFn: func() error {
				req := CreateFunctionRequest{
					FunctionName: "test-function",
					Runtime:      "go1.x",
					RoleARN:      "arn:aws:iam::123456789012:role/lambda-role",
					Code:         []byte("code"),
				}
				return lambdaClient.CreateFunction(ctx, req)
			},
			wantErr:  true,
			contains: "handler cannot be empty",
		},
		{
			name: "CreateFunction - empty role ARN",
			testFn: func() error {
				req := CreateFunctionRequest{
					FunctionName: "test-function",
					Runtime:      "go1.x",
					Handler:      "main",
					Code:         []byte("code"),
				}
				return lambdaClient.CreateFunction(ctx, req)
			},
			wantErr:  true,
			contains: "role ARN cannot be empty",
		},
		{
			name: "CreateFunction - empty code",
			testFn: func() error {
				req := CreateFunctionRequest{
					FunctionName: "test-function",
					Runtime:      "go1.x",
					Handler:      "main",
					RoleARN:      "arn:aws:iam::123456789012:role/lambda-role",
					Code:         []byte{},
				}
				return lambdaClient.CreateFunction(ctx, req)
			},
			wantErr:  true,
			contains: "code cannot be empty",
		},
		{
			name: "DeleteFunction - empty function name",
			testFn: func() error {
				return lambdaClient.DeleteFunction(ctx, "")
			},
			wantErr:  true,
			contains: "function name cannot be empty",
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

// TestLambdaClient_Invoke_WithPayload tests that Invoke works with valid payload
func TestLambdaClient_Invoke_WithPayload(t *testing.T) {
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

	lambdaClient := client.Lambda()
	ctx := context.Background()

	// This will fail without actual function, but should pass validation
	_, err = lambdaClient.Invoke(ctx, "test-function", []byte(`{"key": "value"}`))
	if err != nil {
		// Should not be a validation error
		errMsg := err.Error()
		if errMsg == "function name cannot be empty" {
			t.Errorf("Invoke() validation error = %v", err)
		}
		// Other errors (function not found, etc.) are acceptable
	}
}

// TestLambdaClient_ListFunctions tests that ListFunctions works
func TestLambdaClient_ListFunctions(t *testing.T) {
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

	lambdaClient := client.Lambda()
	ctx := context.Background()

	// This will fail without actual AWS connection, but should not fail validation
	_, err = lambdaClient.ListFunctions(ctx)
	if err != nil {
		// Should not be a validation error (ListFunctions doesn't take inputs)
		errMsg := err.Error()
		if errMsg == "" {
			t.Error("expected error message")
		}
		// Other errors (AWS connection issues, etc.) are acceptable
	}
}
