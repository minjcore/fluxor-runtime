package aws

import (
	"github.com/fluxorio/fluxor/pkg/core"
)

// AWSComponent provides AWS cloud integration with Fluxor
// Similar to AIComponent, this component manages AWS client lifecycle
type AWSComponent struct {
	*core.BaseComponent
	config Config
	client AWSClient
}

// NewAWSComponent creates a new AWS component
// Fail-fast: Validates configuration
func NewAWSComponent(config Config) *AWSComponent {
	return &AWSComponent{
		BaseComponent: core.NewBaseComponent("aws"),
		config:        config,
	}
}

// doStart initializes the AWS client
// Fail-fast: Validates state and configuration before starting
func (c *AWSComponent) doStart(ctx core.FluxorContext) error {
	// Fail-fast: Validate context
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}

	// Create AWS client
	client, err := NewClient(c.config)
	if err != nil {
		return &core.EventBusError{Code: "AWS_CLIENT_ERROR", Message: err.Error()}
	}

	c.client = client

	// Notify via EventBus (Premium Pattern integration)
	eventBus := c.EventBus()
	if eventBus != nil {
		if err := eventBus.Publish("aws.ready", map[string]interface{}{
			"component": "aws",
			"region":    c.config.Region,
		}); err != nil {
			// Best-effort notification; ignore on error.
		}
	}

	return nil
}

// doStop stops the AWS component
func (c *AWSComponent) doStop(ctx core.FluxorContext) error {
	// AWS client doesn't need explicit cleanup (SDK handles it)
	c.client = nil

	// Notify via EventBus
	eventBus := c.EventBus()
	if eventBus != nil {
		if err := eventBus.Publish("aws.stopped", map[string]interface{}{
			"component": "aws",
		}); err != nil {
			// Best-effort notification; ignore on error.
		}
	}

	return nil
}

// Client returns the AWS client
// Fail-fast: Returns error if component is not started or client is nil
func (c *AWSComponent) Client() (AWSClient, error) {
	if !c.IsStarted() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "AWS component is not started"}
	}
	if c.client == nil {
		return nil, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "AWS client is not initialized"}
	}
	return c.client, nil
}

// S3 returns the S3 client
func (c *AWSComponent) S3() (S3Client, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.S3(), nil
}

// SQS returns the SQS client
func (c *AWSComponent) SQS() (SQSClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.SQS(), nil
}

// SNS returns the SNS client
func (c *AWSComponent) SNS() (SNSClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.SNS(), nil
}

// Lambda returns the Lambda client
func (c *AWSComponent) Lambda() (LambdaClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Lambda(), nil
}

// EC2 returns the EC2 client
func (c *AWSComponent) EC2() (EC2Client, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.EC2(), nil
}
