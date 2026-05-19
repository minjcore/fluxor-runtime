package aws

import (
	"context"
)

// AWSClient is the main interface for AWS services
type AWSClient interface {
	// S3 returns the S3 client
	S3() S3Client

	// SQS returns the SQS client
	SQS() SQSClient

	// SNS returns the SNS client
	SNS() SNSClient

	// Lambda returns the Lambda client
	Lambda() LambdaClient

	// EC2 returns the EC2 client
	EC2() EC2Client

	// Region returns the AWS region
	Region() string
}

// S3Client provides S3 operations
type S3Client interface {
	// PutObject uploads an object to S3
	PutObject(ctx context.Context, bucket, key string, data []byte, contentType string) error

	// GetObject downloads an object from S3
	GetObject(ctx context.Context, bucket, key string) ([]byte, error)

	// DeleteObject deletes an object from S3
	DeleteObject(ctx context.Context, bucket, key string) error

	// ListObjects lists objects in a bucket
	ListObjects(ctx context.Context, bucket, prefix string) ([]string, error)

	// ObjectExists checks if an object exists
	ObjectExists(ctx context.Context, bucket, key string) (bool, error)
}

// SQSClient provides SQS operations
type SQSClient interface {
	// SendMessage sends a message to a queue
	SendMessage(ctx context.Context, queueURL string, body string) error

	// ReceiveMessages receives messages from a queue
	ReceiveMessages(ctx context.Context, queueURL string, maxMessages int) ([]SQSMessage, error)

	// DeleteMessage deletes a message from a queue
	DeleteMessage(ctx context.Context, queueURL, receiptHandle string) error

	// CreateQueue creates a new queue
	CreateQueue(ctx context.Context, queueName string) (string, error)

	// DeleteQueue deletes a queue
	DeleteQueue(ctx context.Context, queueURL string) error
}

// SQSMessage represents an SQS message
type SQSMessage struct {
	MessageID     string
	ReceiptHandle string
	Body          string
	Attributes    map[string]string
}

// SNSClient provides SNS operations
type SNSClient interface {
	// Publish publishes a message to a topic
	Publish(ctx context.Context, topicARN string, message string) error

	// Subscribe subscribes to a topic
	Subscribe(ctx context.Context, topicARN, protocol, endpoint string) (string, error)

	// Unsubscribe unsubscribes from a topic
	Unsubscribe(ctx context.Context, subscriptionARN string) error

	// CreateTopic creates a new topic
	CreateTopic(ctx context.Context, topicName string) (string, error)

	// DeleteTopic deletes a topic
	DeleteTopic(ctx context.Context, topicARN string) error
}

// LambdaClient provides Lambda operations
type LambdaClient interface {
	// Invoke invokes a Lambda function
	Invoke(ctx context.Context, functionName string, payload []byte) ([]byte, error)

	// CreateFunction creates a new Lambda function
	CreateFunction(ctx context.Context, req CreateFunctionRequest) error

	// DeleteFunction deletes a Lambda function
	DeleteFunction(ctx context.Context, functionName string) error

	// ListFunctions lists all Lambda functions
	ListFunctions(ctx context.Context) ([]string, error)
}

// CreateFunctionRequest represents a request to create a Lambda function
type CreateFunctionRequest struct {
	FunctionName string
	Runtime      string
	Handler      string
	RoleARN      string
	Code         []byte
	Environment  map[string]string
	Timeout      int
	MemorySize   int
}

// EC2Client provides EC2 operations
type EC2Client interface {
	// DescribeInstances describes EC2 instances
	DescribeInstances(ctx context.Context, instanceIDs []string) ([]EC2Instance, error)

	// StartInstance starts an EC2 instance
	StartInstance(ctx context.Context, instanceID string) error

	// StopInstance stops an EC2 instance
	StopInstance(ctx context.Context, instanceID string) error

	// TerminateInstance terminates an EC2 instance
	TerminateInstance(ctx context.Context, instanceID string) error
}

// EC2Instance represents an EC2 instance
type EC2Instance struct {
	InstanceID   string
	State       string
	InstanceType string
	PrivateIP   string
	PublicIP    string
	Tags        map[string]string
}
