package slack

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// SlackComponent provides Slack integration with Fluxor
type SlackComponent struct {
	*core.BaseComponent
	config  Config
	client  Client
	mu      sync.RWMutex
	started bool
}

// NewSlackComponent creates a new Slack component
// Fail-fast: Validates configuration
func NewSlackComponent(config Config) *SlackComponent {
	return &SlackComponent{
		BaseComponent: core.NewBaseComponent("slack"),
		config:        config,
	}
}

// Start initializes the component
func (c *SlackComponent) Start(ctx core.FluxorContext) error {
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

// Stop stops the component
func (c *SlackComponent) Stop(ctx core.FluxorContext) error {
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

// IsStarted returns whether the component is started
func (c *SlackComponent) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

// doStart initializes the Slack client
func (c *SlackComponent) doStart(ctx core.FluxorContext) error {
	// Fail-fast: Validate context
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}

	// Validate configuration
	if err := c.config.Validate(); err != nil {
		return &core.EventBusError{Code: "CONFIG_VALIDATION_ERROR", Message: err.Error()}
	}

	// Create Slack client
	client, err := NewClient(c.config)
	if err != nil {
		return &core.EventBusError{Code: "SLACK_CLIENT_ERROR", Message: err.Error()}
	}

	c.client = client

	// Notify via EventBus
	eventBus := c.EventBus()
	if eventBus != nil {
		if err := eventBus.Publish("slack.ready", map[string]interface{}{
			"component": "slack",
		}); err != nil {
			// Best-effort notification
		}
	}

	return nil
}

// doStop stops the Slack component
func (c *SlackComponent) doStop(ctx core.FluxorContext) error {
	c.client = nil

	// Notify via EventBus
	eventBus := c.EventBus()
	if eventBus != nil {
		if err := eventBus.Publish("slack.stopped", map[string]interface{}{
			"component": "slack",
		}); err != nil {
			// Best-effort notification
		}
	}

	return nil
}

// Client returns the Slack client
func (c *SlackComponent) Client() (Client, error) {
	if !c.IsStarted() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "Slack component is not started"}
	}
	if c.client == nil {
		return nil, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "Slack client is not initialized"}
	}
	return c.client, nil
}

// Messages returns the Messages client
func (c *SlackComponent) Messages() (MessagesClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Messages(), nil
}

// Channels returns the Channels client
func (c *SlackComponent) Channels() (ChannelsClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Channels(), nil
}

// Users returns the Users client
func (c *SlackComponent) Users() (UsersClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Users(), nil
}

// Name returns the connector name (implements connectors.Connector)
func (c *SlackComponent) Name() string {
	return "slack"
}

// Type returns the connector type (implements connectors.Connector)
func (c *SlackComponent) Type() connectors.ConnectorType {
	return connectors.TypeMessaging
}

// Version returns the connector version (implements connectors.Connector)
func (c *SlackComponent) Version() string {
	return "1.0.0"
}

// IsHealthy checks if the connector is healthy (implements connectors.Connector)
func (c *SlackComponent) IsHealthy(ctx context.Context) (bool, error) {
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "Slack component is not started"}
	}

	if c.client == nil {
		return false, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "Slack client is not initialized"}
	}

	// Try to list users as a health check (auth.test would be better but this works)
	users := c.client.Users()
	_, err := users.List(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}

// GetMetadata returns connector metadata (implements connectors.Connector)
func (c *SlackComponent) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "slack",
		DisplayName: "Slack",
		Description: "Slack connector for messaging, channels, and users",
		Version:     "1.0.0",
		Type:        connectors.TypeMessaging,
		Author:      "Fluxor Team",
		DocsURL:     "https://github.com/fluxorio/fluxor/tree/main/pkg/connectors/slack",
		Capabilities: []connectors.Capability{
			{
				Name:        "messages",
				Description: "Send, update, and delete messages",
				Enabled:     true,
			},
			{
				Name:        "channels",
				Description: "Manage channels (create, archive, invite)",
				Enabled:     true,
			},
			{
				Name:        "users",
				Description: "List and lookup users",
				Enabled:     true,
			},
		},
		RateLimits: &connectors.RateLimitInfo{
			RequestsPerMinute: c.config.RateLimit,
		},
		AuthMethods: []string{"oauth", "bot_token"},
		Tags:        []string{"messaging", "chat", "collaboration", "team"},
	}
}
