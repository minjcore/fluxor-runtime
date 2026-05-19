package smtp

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// MailConnector implements connectors.Connector for SMTP email.
type MailConnector struct {
	*core.BaseComponent
	config  Config
	client  *Client
	mu      sync.RWMutex
	started bool
}

// NewMailConnector creates a new mail (SMTP) connector.
func NewMailConnector(config Config) *MailConnector {
	return &MailConnector{
		BaseComponent: core.NewBaseComponent("smtp"),
		config:        config,
	}
}

// Start starts the connector and initializes the SMTP client.
func (c *MailConnector) Start(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.started {
		return &core.EventBusError{Code: "ALREADY_STARTED", Message: "mail connector already started"}
	}
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}
	if err := c.config.Validate(); err != nil {
		return &core.EventBusError{Code: "CONFIG_VALIDATION_ERROR", Message: err.Error()}
	}
	c.client = NewClient(c.config)
	c.started = true
	return nil
}

// Stop stops the connector.
func (c *MailConnector) Stop(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.started {
		return nil
	}
	c.client = nil
	c.started = false
	return nil
}

// IsStarted returns whether the connector is started.
func (c *MailConnector) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

// Client returns the SMTP client. Call after Start().
func (c *MailConnector) Client() (*Client, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.started {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "mail connector is not started"}
	}
	if c.client == nil {
		return nil, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "mail client is not initialized"}
	}
	return c.client, nil
}

// Send sends an email using the connector's client.
func (c *MailConnector) Send(ctx context.Context, in SendInput) SendResult {
	client, err := c.Client()
	if err != nil {
		return SendResult{Success: false, Error: err.Error()}
	}
	return client.Send(ctx, in)
}

// Name implements connectors.Connector.
func (c *MailConnector) Name() string { return "smtp" }

// Type implements connectors.Connector.
func (c *MailConnector) Type() connectors.ConnectorType { return connectors.TypeMessaging }

// Version implements connectors.Connector.
func (c *MailConnector) Version() string { return "0.1.0" }

// IsHealthy implements connectors.Connector. For SMTP we consider healthy if config has host and we are started.
func (c *MailConnector) IsHealthy(ctx context.Context) (bool, error) {
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "mail connector is not started"}
	}
	c.mu.RLock()
	host := c.config.Host
	c.mu.RUnlock()
	if host == "" {
		return false, &core.EventBusError{Code: "INVALID_CONFIG", Message: "smtp host not configured"}
	}
	return true, nil
}

// GetMetadata implements connectors.Connector.
func (c *MailConnector) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "smtp",
		DisplayName: "SMTP (Mail)",
		Description: "SMTP connector for sending email. Supports TLS, plain auth, and standard SMTP servers (Gmail, SendGrid, etc.).",
		Version:     c.Version(),
		Type:        connectors.TypeMessaging,
		Author:      "Fluxor Team",
		DocsURL:     "https://datatracker.ietf.org/doc/html/rfc5321",
		Capabilities: []connectors.Capability{
			{Name: "send", Description: "Send email via SMTP", Enabled: true},
		},
		AuthMethods: []string{"plain_auth"},
		Tags:        []string{"email", "smtp", "messaging", "notifications"},
	}
}
