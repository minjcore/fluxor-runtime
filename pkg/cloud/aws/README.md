# AWS Cloud Integration

The `pkg/cloud/aws` package provides comprehensive AWS cloud integration for Fluxor, following Fluxor's reactive patterns and fail-fast principles.

## Features

- ✅ **Full AWS SDK v2 Integration** - Modern AWS SDK with support for all major services
- ✅ **EventBus Integration** - AWS events published to Fluxor EventBus
- ✅ **Component Lifecycle** - Proper startup/shutdown with BaseComponent pattern
- ✅ **Multiple Services** - S3, SQS, SNS, Lambda, EC2 support
- ✅ **LocalStack Support** - Test locally with LocalStack
- ✅ **Fail-Fast Validation** - Immediate error detection
- ✅ **IAM Role Support** - Works with IAM roles, credentials, or environment variables

## Quick Start

### Installation

Add AWS SDK v2 to your `go.mod`:

```bash
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/s3
go get github.com/aws/aws-sdk-go-v2/service/sqs
go get github.com/aws/aws-sdk-go-v2/service/sns
go get github.com/aws/aws-sdk-go-v2/service/lambda
go get github.com/aws/aws-sdk-go-v2/service/ec2
go get github.com/aws/aws-sdk-go-v2/credentials
```

### Basic Usage

```go
package main

import (
    "context"
    "github.com/fluxorio/fluxor/pkg/cloud/aws"
    "github.com/fluxorio/fluxor/pkg/core"
)

type MyVerticle struct {
    *core.BaseVerticle
    aws *aws.AWSComponent
}

func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    // Create AWS component
    config := aws.DefaultConfig()
    config.Region = "us-east-1"
    v.aws = aws.NewAWSComponent(config)
    
    // Start component
    if err := v.aws.Start(ctx); err != nil {
        return err
    }
    
    // Use S3
    s3, err := v.aws.S3()
    if err != nil {
        return err
    }
    
    // Upload file
    err = s3.PutObject(ctx.Context(), "my-bucket", "key", []byte("data"), "text/plain")
    if err != nil {
        return err
    }
    
    return nil
}

func (v *MyVerticle) Stop(ctx core.FluxorContext) error {
    if v.aws != nil {
        return v.aws.Stop(ctx)
    }
    return nil
}
```

## Configuration

### Environment Variables

The package supports standard AWS environment variables:

- `AWS_REGION` - AWS region (default: us-east-1)
- `AWS_ACCESS_KEY_ID` - AWS access key ID
- `AWS_SECRET_ACCESS_KEY` - AWS secret access key
- `AWS_SESSION_TOKEN` - AWS session token (for temporary credentials)

### Programmatic Configuration

```go
config := aws.Config{
    Region:          "us-west-2",
    AccessKeyID:     "your-access-key",
    SecretAccessKey: "your-secret-key",
    Timeout:         "30s",
    MaxRetries:      3,
    Debug:           false,
}

// For LocalStack
config.Endpoint = "http://localhost:4566"
```

### IAM Role Support

When running on EC2 or ECS, the package automatically uses IAM roles if credentials are not provided:

```go
config := aws.DefaultConfig()
config.Region = "us-east-1"
// No credentials needed - uses IAM role
```

## Services

### S3 (Simple Storage Service)

```go
s3, _ := awsComponent.S3()

// Upload object
err := s3.PutObject(ctx, "my-bucket", "path/to/file.txt", []byte("content"), "text/plain")

// Download object
data, err := s3.GetObject(ctx, "my-bucket", "path/to/file.txt")

// Delete object
err := s3.DeleteObject(ctx, "my-bucket", "path/to/file.txt")

// List objects
keys, err := s3.ListObjects(ctx, "my-bucket", "prefix/")

// Check if object exists
exists, err := s3.ObjectExists(ctx, "my-bucket", "path/to/file.txt")
```

### SQS (Simple Queue Service)

```go
sqs, _ := awsComponent.SQS()

// Create queue
queueURL, err := sqs.CreateQueue(ctx, "my-queue")

// Send message
err := sqs.SendMessage(ctx, queueURL, "message body")

// Receive messages
messages, err := sqs.ReceiveMessages(ctx, queueURL, 10)

// Delete message
err := sqs.DeleteMessage(ctx, queueURL, receiptHandle)
```

### SNS (Simple Notification Service)

```go
sns, _ := awsComponent.SNS()

// Create topic
topicARN, err := sns.CreateTopic(ctx, "my-topic")

// Publish message
err := sns.Publish(ctx, topicARN, "message")

// Subscribe
subscriptionARN, err := sns.Subscribe(ctx, topicARN, "email", "user@example.com")

// Unsubscribe
err := sns.Unsubscribe(ctx, subscriptionARN)
```

### Lambda

```go
lambda, _ := awsComponent.Lambda()

// Invoke function
payload := []byte(`{"key": "value"}`)
result, err := lambda.Invoke(ctx, "my-function", payload)

// Create function
req := aws.CreateFunctionRequest{
    FunctionName: "my-function",
    Runtime:      "go1.x",
    Handler:      "main",
    RoleARN:      "arn:aws:iam::123456789012:role/lambda-role",
    Code:         zipFileBytes,
}
err := lambda.CreateFunction(ctx, req)

// List functions
functions, err := lambda.ListFunctions(ctx)
```

### EC2

```go
ec2, _ := awsComponent.EC2()

// Describe instances
instances, err := ec2.DescribeInstances(ctx, []string{"i-1234567890abcdef0"})

// Start instance
err := ec2.StartInstance(ctx, "i-1234567890abcdef0")

// Stop instance
err := ec2.StopInstance(ctx, "i-1234567890abcdef0")

// Terminate instance
err := ec2.TerminateInstance(ctx, "i-1234567890abcdef0")
```

## EventBus Integration

The AWS component publishes events to Fluxor's EventBus:

### Events

- `aws.ready` - Published when AWS component is ready
  ```json
  {
    "component": "aws",
    "region": "us-east-1"
  }
  ```

- `aws.stopped` - Published when AWS component is stopped
  ```json
  {
    "component": "aws"
  }
  ```

### Listening to Events

```go
eventBus := gocmd.EventBus()

// Listen for AWS ready event
eventBus.Consumer("aws.ready").Handler(func(ctx core.FluxorContext, msg core.Message) error {
    var data map[string]interface{}
    if err := msg.Decode(&data); err != nil {
        return err
    }
    
    region := data["region"].(string)
    log.Printf("AWS component ready in region: %s", region)
    return nil
})
```

## Local Development with LocalStack

For local development, use LocalStack:

```go
config := aws.Config{
    Region:   "us-east-1",
    Endpoint: "http://localhost:4566", // LocalStack endpoint
}

awsComponent := aws.NewAWSComponent(config)
```

Start LocalStack:

```bash
docker run -d -p 4566:4566 localstack/localstack
```

## Error Handling

All methods follow Fluxor's fail-fast pattern:

```go
s3, err := awsComponent.S3()
if err != nil {
    // Component not started or invalid
    return err
}

data, err := s3.GetObject(ctx, "bucket", "key")
if err != nil {
    // Handle AWS error
    return err
}
```

## Multi-Cloud Detection

The package includes utilities to detect if code is running on Google Cloud Platform, useful for multi-cloud deployments:

### Google Cloud Detection

```go
import "github.com/fluxorio/fluxor/pkg/cloud/aws"

// Check if running on Google Cloud Platform
if aws.IsGoogleCloud() {
    // Running on GCP
    project := aws.GetGoogleCloudProject()
    log.Printf("Running on GCP project: %s", project)
}

// Get GCP project ID
projectID := aws.GetGoogleCloudProject()
if projectID != "" {
    log.Printf("GCP Project: %s", projectID)
}
```

The `IsGoogleCloud()` function checks:
- Environment variables: `GOOGLE_CLOUD_PROJECT`, `GCP_PROJECT`, `GCE_METADATA_HOST`, `GCE_METADATA_ROOT`
- GCE metadata server availability (169.254.169.254)

The `GetGoogleCloudProject()` function returns the GCP project ID from environment variables.

## Best Practices

1. **Always check errors** - All methods return errors that should be handled
2. **Use context** - Pass context for cancellation and timeouts
3. **Reuse component** - Create component once, reuse across verticles
4. **EventBus integration** - Listen to AWS events for reactive patterns
5. **IAM roles** - Use IAM roles in production instead of credentials
6. **LocalStack** - Use LocalStack for local testing
7. **Multi-cloud detection** - Use `IsGoogleCloud()` to detect cloud provider at runtime

## Integration with Workflows

AWS services can be used in Fluxor workflows:

```json
{
  "id": "s3-upload-workflow",
  "nodes": [
    {
      "id": "trigger",
      "type": "webhook"
    },
    {
      "id": "upload",
      "type": "function",
      "config": {
        "code": "s3.PutObject(ctx, 'bucket', 'key', data, 'text/plain')"
      }
    }
  ]
}
```

## Roadmap

Future enhancements:

- [ ] DynamoDB support
- [ ] CloudWatch integration
- [ ] Step Functions support
- [ ] EventBridge integration
- [ ] S3 event notifications via EventBus
- [ ] SQS message processing with EventBus
- [ ] SNS subscription handling with EventBus

## License

Part of Fluxor framework - see main LICENSE file.
