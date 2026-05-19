package hubspot

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// HubSpotComponent provides a minimal HubSpot connector implementation scaffold.
type HubSpotComponent struct {
	*core.BaseComponent
	config  Config
	mu      sync.RWMutex
	started bool
}

// NewHubSpotComponent creates a new HubSpot connector component.
func NewHubSpotComponent(config Config) *HubSpotComponent {
	return &HubSpotComponent{
		BaseComponent: core.NewBaseComponent("hubspot"),
		config:        config,
	}
}

func (c *HubSpotComponent) Start(ctx core.FluxorContext) error {
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

func (c *HubSpotComponent) Stop(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.started {
		return nil
	}
	c.started = false
	return nil
}

func (c *HubSpotComponent) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

func (c *HubSpotComponent) Name() string { return "hubspot" }

func (c *HubSpotComponent) Type() connectors.ConnectorType { return connectors.TypeCRM }

func (c *HubSpotComponent) Version() string { return "0.1.0" }

func (c *HubSpotComponent) IsHealthy(_ context.Context) (bool, error) {
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "HubSpot component is not started"}
	}
	return true, nil
}

func (c *HubSpotComponent) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "hubspot",
		DisplayName: "HubSpot",
		Description: "HubSpot connector scaffold (CRM objects, engagements, pipelines)",
		Version:     c.Version(),
		Type:        connectors.TypeCRM,
		Author:      "Fluxor Team",
		DocsURL:     "https://github.com/fluxorio/fluxor/tree/main/pkg/connectors/hubspot",
		Capabilities: []connectors.Capability{
			{Name: "read", Description: "Read CRM objects (contacts, companies, deals)", Enabled: true},
			{Name: "write", Description: "Create/update CRM objects", Enabled: false},
		},
		AuthMethods: []string{"private_app_token"},
		Tags:        []string{"crm", "sales", "marketing"},
	}
}

