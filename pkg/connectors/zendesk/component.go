package zendesk

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// ZendeskComponent provides a minimal Zendesk connector implementation scaffold.
type ZendeskComponent struct {
	*core.BaseComponent
	config  Config
	mu      sync.RWMutex
	started bool
}

// NewZendeskComponent creates a new Zendesk connector component.
func NewZendeskComponent(config Config) *ZendeskComponent {
	return &ZendeskComponent{
		BaseComponent: core.NewBaseComponent("zendesk"),
		config:        config,
	}
}

func (c *ZendeskComponent) Start(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return &core.EventBusError{Code: "ALREADY_STARTED", Message: "component already started"}
	}
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}
	if err := c.config.Validate(); err != nil {
		return &core.EventBusError{Code: "CONFIG_VALIDATION_ERROR", Message: err.Error()}
	}

	c.started = true
	return nil
}

func (c *ZendeskComponent) Stop(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.started {
		return nil
	}
	c.started = false
	return nil
}

func (c *ZendeskComponent) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

func (c *ZendeskComponent) Name() string { return "zendesk" }

func (c *ZendeskComponent) Type() connectors.ConnectorType { return connectors.TypeCRM }

func (c *ZendeskComponent) Version() string { return "0.1.0" }

func (c *ZendeskComponent) IsHealthy(_ context.Context) (bool, error) {
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "Zendesk component is not started"}
	}
	return true, nil
}

func (c *ZendeskComponent) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "zendesk",
		DisplayName: "Zendesk",
		Description: "Zendesk connector scaffold (tickets, users, organizations)",
		Version:     c.Version(),
		Type:        connectors.TypeCRM,
		Author:      "Fluxor Team",
		DocsURL:     "https://github.com/fluxorio/fluxor/tree/main/pkg/connectors/zendesk",
		Capabilities: []connectors.Capability{
			{Name: "read", Description: "Read tickets and related objects", Enabled: true},
			{Name: "write", Description: "Create/update tickets", Enabled: false},
		},
		AuthMethods: []string{"api_token"},
		Tags:        []string{"crm", "support", "helpdesk"},
	}
}

