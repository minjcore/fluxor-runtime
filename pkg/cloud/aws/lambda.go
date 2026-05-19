package aws

import (
	"context"
	"fmt"
	"math"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

// lambdaClient implements LambdaClient
type lambdaClient struct {
	client *lambda.Client
}

// newLambdaClient creates a new Lambda client
func newLambdaClient(cfg aws.Config) LambdaClient {
	return &lambdaClient{
		client: lambda.NewFromConfig(cfg),
	}
}

// Invoke invokes a Lambda function
func (c *lambdaClient) Invoke(ctx context.Context, functionName string, payload []byte) ([]byte, error) {
	// Fail-fast: Validate inputs
	if functionName == "" {
		return nil, fmt.Errorf("function name cannot be empty")
	}

	input := &lambda.InvokeInput{
		FunctionName: aws.String(functionName),
		Payload:      payload,
	}

	output, err := c.client.Invoke(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke Lambda function: %w", err)
	}

	// Check for function error
	if output.FunctionError != nil {
		return nil, fmt.Errorf("lambda function error: %s", aws.ToString(output.FunctionError))
	}

	return output.Payload, nil
}

// CreateFunction creates a new Lambda function
func (c *lambdaClient) CreateFunction(ctx context.Context, req CreateFunctionRequest) error {
	// Fail-fast: Validate inputs
	if req.FunctionName == "" {
		return fmt.Errorf("function name cannot be empty")
	}
	if req.Runtime == "" {
		return fmt.Errorf("runtime cannot be empty")
	}
	if req.Handler == "" {
		return fmt.Errorf("handler cannot be empty")
	}
	if req.RoleARN == "" {
		return fmt.Errorf("role ARN cannot be empty")
	}
	if len(req.Code) == 0 {
		return fmt.Errorf("code cannot be empty")
	}

	code := &types.FunctionCode{
		ZipFile: req.Code,
	}

	env := &types.Environment{}
	if len(req.Environment) > 0 {
		env.Variables = req.Environment
	}

	input := &lambda.CreateFunctionInput{
		FunctionName: aws.String(req.FunctionName),
		Runtime:      types.Runtime(req.Runtime),
		Handler:      aws.String(req.Handler),
		Role:         aws.String(req.RoleARN),
		Code:         code,
		Environment:  env,
	}

	if req.Timeout > 0 {
		if req.Timeout > math.MaxInt32 {
			return fmt.Errorf("timeout value %d exceeds maximum allowed value %d", req.Timeout, math.MaxInt32)
		}
		input.Timeout = aws.Int32(int32(req.Timeout))
	}

	if req.MemorySize > 0 {
		if req.MemorySize > math.MaxInt32 {
			return fmt.Errorf("memory size value %d exceeds maximum allowed value %d", req.MemorySize, math.MaxInt32)
		}
		input.MemorySize = aws.Int32(int32(req.MemorySize))
	}

	_, err := c.client.CreateFunction(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create Lambda function: %w", err)
	}

	return nil
}

// DeleteFunction deletes a Lambda function
func (c *lambdaClient) DeleteFunction(ctx context.Context, functionName string) error {
	// Fail-fast: Validate inputs
	if functionName == "" {
		return fmt.Errorf("function name cannot be empty")
	}

	input := &lambda.DeleteFunctionInput{
		FunctionName: aws.String(functionName),
	}

	_, err := c.client.DeleteFunction(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete Lambda function: %w", err)
	}

	return nil
}

// ListFunctions lists all Lambda functions
func (c *lambdaClient) ListFunctions(ctx context.Context) ([]string, error) {
	input := &lambda.ListFunctionsInput{}

	var functionNames []string
	paginator := lambda.NewListFunctionsPaginator(c.client, input)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list Lambda functions: %w", err)
		}

		for _, fn := range output.Functions {
			if fn.FunctionName != nil {
				functionNames = append(functionNames, *fn.FunctionName)
			}
		}
	}

	return functionNames, nil
}
