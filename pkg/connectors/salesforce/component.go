package salesforce

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// SalesforceComponent provides a minimal Salesforce connector implementation scaffold.
type SalesforceComponent struct {
	*core.BaseComponent
	config  Config
	mu      sync.RWMutex
	started bool
}

// NewSalesforceComponent creates a new Salesforce connector component.
func NewSalesforceComponent(config Config) *SalesforceComponent {
	return &SalesforceComponent{
		BaseComponent: core.NewBaseComponent("salesforce"),
		config:        config,
	}
}

func (c *SalesforceComponent) Start(ctx core.FluxorContext) error {
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

func (c *SalesforceComponent) Stop(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.started {
		return nil
	}
	c.started = false
	return nil
}

func (c *SalesforceComponent) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

func (c *SalesforceComponent) Name() string { return "salesforce" }

func (c *SalesforceComponent) Type() connectors.ConnectorType { return connectors.TypeCRM }

func (c *SalesforceComponent) Version() string { return "0.1.0" }

func (c *SalesforceComponent) IsHealthy(_ context.Context) (bool, error) {
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "Salesforce component is not started"}
	}
	return true, nil
}

func (c *SalesforceComponent) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "salesforce",
		DisplayName: "Salesforce",
		Description: "Salesforce connector scaffold (sObjects, SOQL, CRM automation)",
		Version:     c.Version(),
		Type:        connectors.TypeCRM,
		Author:      "Fluxor Team",
		DocsURL:     "https://github.com/fluxorio/fluxor/tree/main/pkg/connectors/salesforce",
		Capabilities: []connectors.Capability{
			{Name: "read", Description: "Read sObjects via REST/SOQL", Enabled: true},
			{Name: "write", Description: "Create/update sObjects", Enabled: false},
		},
		AuthMethods: []string{"oauth_refresh_token"},
		Tags:        []string{"crm", "sales"},
	}
}

