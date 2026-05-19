package zns

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// ZNSComponent provides a minimal Zalo ZNS connector implementation scaffold.
type ZNSComponent struct {
	*core.BaseComponent
	config  Config
	mu      sync.RWMutex
	started bool
}

// NewZNSComponent creates a new Zalo ZNS connector component.
func NewZNSComponent(config Config) *ZNSComponent {
	return &ZNSComponent{
		BaseComponent: core.NewBaseComponent("zalo-zns"),
		config:        config,
	}
}

func (c *ZNSComponent) Start(ctx core.FluxorContext) error {
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

func (c *ZNSComponent) Stop(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.started {
		return nil
	}
	c.started = false
	return nil
}

func (c *ZNSComponent) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

func (c *ZNSComponent) Name() string { return "zalo-zns" }

func (c *ZNSComponent) Type() connectors.ConnectorType { return connectors.TypeMessaging }

func (c *ZNSComponent) Version() string { return "0.1.0" }

func (c *ZNSComponent) IsHealthy(_ context.Context) (bool, error) {
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "Zalo ZNS component is not started"}
	}
	return true, nil
}

func (c *ZNSComponent) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "zalo-zns",
		DisplayName: "Zalo ZNS",
		Description: "Zalo Notification Service (ZNS) connector scaffold for sending templated notifications",
		Version:     c.Version(),
		Type:        connectors.TypeMessaging,
		Author:      "Fluxor Team",
		DocsURL:     "https://github.com/fluxorio/fluxor/tree/main/pkg/connectors/zalo/zns",
		Capabilities: []connectors.Capability{
			{Name: "send", Description: "Send ZNS templated notification messages", Enabled: false},
			{Name: "status", Description: "Query message delivery status", Enabled: false},
		},
		AuthMethods: []string{"access_token", "app_secret"},
		Tags:        []string{"messaging", "notifications", "zalo", "zns"},
	}
}

