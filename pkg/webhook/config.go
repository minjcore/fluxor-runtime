package webhook

import (
	"fmt"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// EndpointConfig configures a webhook endpoint
type EndpointConfig struct {
	// Path is the webhook path (e.g., "/webhooks/github")
	Path string
	// Secret is the webhook secret for signature validation
	Secret string
	// Algorithm is the signature algorithm (default: hmac-sha256)
	Algorithm SignatureAlgorithm
	// SignatureHeader is the header name containing the signature (default: "X-Webhook-Signature")
	SignatureHeader string
	// SignaturePrefix is the signature prefix (e.g., "sha256=")
	SignaturePrefix string
	// EventBusAddress is the EventBus address to publish webhook events
	EventBusAddress string
	// CustomValidator is an optional custom signature validator
	CustomValidator SignatureValidator
	// SkipValidation skips signature validation (not recommended for production)
	SkipValidation bool
}

// Validate validates the endpoint configuration
func (c *EndpointConfig) Validate() error {
	if c.Path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	if !c.SkipValidation {
		if c.CustomValidator == nil {
			if c.Secret == "" {
				return fmt.Errorf("secret or CustomValidator must be provided")
			}
		}
	}

	if c.EventBusAddress == "" {
		return fmt.Errorf("eventBusAddress cannot be empty")
	}

	return nil
}

// ReceiverConfig configures the webhook receiver
type ReceiverConfig struct {
	// Prefix is the URL prefix for all webhooks (e.g., "/webhooks")
	Prefix string
	// Endpoints is the list of webhook endpoints
	Endpoints []EndpointConfig
	// OnError is an optional error handler
	OnError func(path string, err error) error
}

// DefaultReceiverConfig returns a default receiver configuration
func DefaultReceiverConfig() *ReceiverConfig {
	return &ReceiverConfig{
		Prefix:   "/webhooks",
		Endpoints: make([]EndpointConfig, 0),
	}
}

// Validate validates the receiver configuration
func (c *ReceiverConfig) Validate() error {
	if c.Prefix == "" {
		return fmt.Errorf("prefix cannot be empty")
	}

	if len(c.Endpoints) == 0 {
		return fmt.Errorf("at least one endpoint must be configured")
	}

	for i, endpoint := range c.Endpoints {
		if err := endpoint.Validate(); err != nil {
			return fmt.Errorf("endpoint[%d]: %w", i, err)
		}
	}

	return nil
}

// SetDefaults sets default values for the configuration
func (c *EndpointConfig) SetDefaults() {
	if c.Algorithm == "" {
		c.Algorithm = SignatureAlgorithmHMACSHA256
	}

	if c.SignatureHeader == "" {
		c.SignatureHeader = "X-Webhook-Signature"
	}

	if c.SignaturePrefix == "" {
		c.SignaturePrefix = "sha256="
	}
}

// GetValidator returns the signature validator for this endpoint
func (c *EndpointConfig) GetValidator() SignatureValidator {
	if c.CustomValidator != nil {
		return c.CustomValidator
	}

	if c.SkipValidation {
		return nil
	}

	failfast.If(c.Secret != "", "secret cannot be empty when CustomValidator is nil")
	
	return NewHMACSignatureValidator(c.Secret, c.Algorithm, c.SignatureHeader, c.SignaturePrefix)
}