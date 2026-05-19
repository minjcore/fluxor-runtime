package s3

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/cloud/aws"
	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// S3Component provides S3 integration with Fluxor
// Similar to AirtableComponent, this component manages S3 client lifecycle
type S3Component struct {
	*core.BaseComponent
	config     Config
	awsClient  aws.AWSClient
	s3Client   aws.S3Client
	mu         sync.RWMutex
	started    bool
}

// NewS3Component creates a new S3 component
// Fail-fast: Validates configuration
func NewS3Component(config Config) *S3Component {
	return &S3Component{
		BaseComponent: core.NewBaseComponent("s3"),
		config:        config,
	}
}

// Start initializes the component (overrides BaseComponent.Start to call our doStart)
func (c *S3Component) Start(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return &core.EventBusError{Code: "ALREADY_STARTED", Message: "component already started"}
	}

	// Call our custom doStart
	if err := c.doStart(ctx); err != nil {
		return err
	}

	c.started = true
	return nil
}

// Stop stops the component (overrides BaseComponent.Stop to call our doStop)
func (c *S3Component) Stop(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil
	}

	// Call our custom doStop
	if err := c.doStop(ctx); err != nil {
		return err
	}

	c.started = false
	return nil
}

// IsStarted returns whether the component is started (overrides BaseComponent.IsStarted)
func (c *S3Component) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

// doStart initializes the AWS and S3 clients
// Fail-fast: Validates state and configuration before starting
func (c *S3Component) doStart(ctx core.FluxorContext) error {
	// Fail-fast: Validate context
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}

	// Validate configuration
	if err := c.config.Validate(); err != nil {
		return &core.EventBusError{Code: "CONFIG_VALIDATION_ERROR", Message: err.Error()}
	}

	// Create AWS config from S3 config
	awsConfig := aws.Config{
		BaseConfig:      c.config.BaseConfig,
		Region:           c.config.Region,
		AccessKeyID:     c.config.AccessKeyID,
		SecretAccessKey: c.config.SecretAccessKey,
		SessionToken:    c.config.SessionToken,
		Endpoint:        c.config.Endpoint,
		Timeout:         c.config.Timeout,
		MaxRetries:      c.config.MaxRetries,
		Debug:           c.config.Debug,
	}

	// Create AWS client
	awsClient, err := aws.NewClient(awsConfig)
	if err != nil {
		return &core.EventBusError{Code: "AWS_CLIENT_ERROR", Message: err.Error()}
	}

	c.awsClient = awsClient
	c.s3Client = awsClient.S3()

	// Notify via EventBus (Premium Pattern integration)
	eventBus := c.EventBus()
	if eventBus != nil {
		if err := eventBus.Publish("s3.ready", map[string]interface{}{
			"component": "s3",
			"region":    c.config.Region,
			"bucket":    c.config.Bucket,
		}); err != nil {
			// Best-effort notification; ignore on error.
		}
	}

	return nil
}

// doStop stops the S3 component
func (c *S3Component) doStop(ctx core.FluxorContext) error {
	// AWS client doesn't need explicit cleanup (HTTP client handles it)
	c.s3Client = nil
	c.awsClient = nil

	// Notify via EventBus
	eventBus := c.EventBus()
	if eventBus != nil {
		if err := eventBus.Publish("s3.stopped", map[string]interface{}{
			"component": "s3",
		}); err != nil {
			// Best-effort notification; ignore on error.
		}
	}

	return nil
}

// Client returns the S3 client
// Fail-fast: Returns error if component is not started or client is nil
func (c *S3Component) Client() (aws.S3Client, error) {
	if !c.IsStarted() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "S3 component is not started"}
	}
	if c.s3Client == nil {
		return nil, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "S3 client is not initialized"}
	}
	return c.s3Client, nil
}

// AWSClient returns the underlying AWS client
func (c *S3Component) AWSClient() (aws.AWSClient, error) {
	if !c.IsStarted() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "S3 component is not started"}
	}
	if c.awsClient == nil {
		return nil, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "AWS client is not initialized"}
	}
	return c.awsClient, nil
}

// Config returns the component configuration
func (c *S3Component) Config() Config {
	return c.config
}

// S3Client returns a high-level S3 client wrapper
func (c *S3Component) S3Client() (*Client, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return &Client{
		s3Client: client,
		config:   c.config,
	}, nil
}

// Name returns the connector name (implements connectors.Connector)
func (c *S3Component) Name() string {
	return "s3"
}

// Type returns the connector type (implements connectors.Connector)
func (c *S3Component) Type() connectors.ConnectorType {
	return connectors.TypeStorage
}

// Version returns the connector version (implements connectors.Connector)
func (c *S3Component) Version() string {
	return "1.0.0"
}

// IsHealthy checks if the connector is healthy (implements connectors.Connector)
func (c *S3Component) IsHealthy(ctx context.Context) (bool, error) {
	// Check if component is started
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "S3 component is not started"}
	}

	// Check if client is initialized
	if c.s3Client == nil {
		return false, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "S3 client is not initialized"}
	}

	// Try to list objects as a health check (use default bucket if configured)
	// If no bucket is configured, we can't do a real health check, so just verify client exists
	if c.config.Bucket != "" {
		_, err := c.s3Client.ListObjects(ctx, c.config.Bucket, "")
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

// GetMetadata returns connector metadata (implements connectors.Connector)
func (c *S3Component) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "s3",
		DisplayName: "Amazon S3",
		Description: "Amazon S3 connector for object storage operations (upload, download, delete, list)",
		Version:     "1.0.0",
		Type:        connectors.TypeStorage,
		Author:      "Fluxor Team",
		DocsURL:     "https://github.com/fluxorio/fluxor/tree/main/pkg/connectors/s3",
		Capabilities: []connectors.Capability{
			{
				Name:        "read",
				Description: "Read/download objects from S3",
				Enabled:     true,
			},
			{
				Name:        "write",
				Description: "Upload objects to S3",
				Enabled:     true,
			},
			{
				Name:        "delete",
				Description: "Delete objects from S3",
				Enabled:     true,
			},
			{
				Name:        "list",
				Description: "List objects in S3 buckets",
				Enabled:     true,
			},
			{
				Name:        "exists",
				Description: "Check if objects exist in S3",
				Enabled:     true,
			},
		},
		AuthMethods: []string{"aws_credentials", "iam_role", "access_key"},
		Tags:        []string{"storage", "aws", "cloud", "object-storage", "s3"},
	}
}
