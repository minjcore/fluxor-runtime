package airtable

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// AirtableComponent provides Airtable integration with Fluxor
// Similar to AWSComponent, this component manages Airtable client lifecycle
type AirtableComponent struct {
	*core.BaseComponent
	config  Config
	client  Client
	mu      sync.RWMutex
	started bool
}

// NewAirtableComponent creates a new Airtable component
// Fail-fast: Validates configuration
func NewAirtableComponent(config Config) *AirtableComponent {
	return &AirtableComponent{
		BaseComponent: core.NewBaseComponent("airtable"),
		config:        config,
	}
}

// Start initializes the component (overrides BaseComponent.Start to call our doStart)
func (c *AirtableComponent) Start(ctx core.FluxorContext) error {
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
func (c *AirtableComponent) Stop(ctx core.FluxorContext) error {
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
func (c *AirtableComponent) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

// doStart initializes the Airtable client
// Fail-fast: Validates state and configuration before starting
func (c *AirtableComponent) doStart(ctx core.FluxorContext) error {
	// Fail-fast: Validate context
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}

	// Validate configuration
	if err := c.config.Validate(); err != nil {
		return &core.EventBusError{Code: "CONFIG_VALIDATION_ERROR", Message: err.Error()}
	}

	// Create Airtable client
	client, err := NewClient(c.config)
	if err != nil {
		return &core.EventBusError{Code: "AIRTABLE_CLIENT_ERROR", Message: err.Error()}
	}

	c.client = client

	// Notify via EventBus (Premium Pattern integration)
	eventBus := c.EventBus()
	if eventBus != nil {
		if err := eventBus.Publish("airtable.ready", map[string]interface{}{
			"component": "airtable",
			"baseID":    c.config.BaseID,
		}); err != nil {
			// Best-effort notification; ignore on error.
		}
	}

	return nil
}

// doStop stops the Airtable component
func (c *AirtableComponent) doStop(ctx core.FluxorContext) error {
	// Airtable client doesn't need explicit cleanup (HTTP client handles it)
	c.client = nil

	// Notify via EventBus
	eventBus := c.EventBus()
	if eventBus != nil {
		if err := eventBus.Publish("airtable.stopped", map[string]interface{}{
			"component": "airtable",
		}); err != nil {
			// Best-effort notification; ignore on error.
		}
	}

	return nil
}

// Client returns the Airtable client
// Fail-fast: Returns error if component is not started or client is nil
func (c *AirtableComponent) Client() (Client, error) {
	if !c.IsStarted() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "Airtable component is not started"}
	}
	if c.client == nil {
		return nil, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "Airtable client is not initialized"}
	}
	return c.client, nil
}

// Tables returns the Tables client
func (c *AirtableComponent) Tables() (TablesClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Tables(), nil
}

// Records returns the Records client
func (c *AirtableComponent) Records() (RecordsClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Records(), nil
}

// Name returns the connector name (implements connectors.Connector)
func (c *AirtableComponent) Name() string {
	return "airtable"
}

// Type returns the connector type (implements connectors.Connector)
func (c *AirtableComponent) Type() connectors.ConnectorType {
	return connectors.TypeProductivity
}

// Version returns the connector version (implements connectors.Connector)
func (c *AirtableComponent) Version() string {
	return "1.0.0"
}

// IsHealthy checks if the connector is healthy (implements connectors.Connector)
func (c *AirtableComponent) IsHealthy(ctx context.Context) (bool, error) {
	// Check if component is started
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "Airtable component is not started"}
	}

	// Check if client is initialized
	if c.client == nil {
		return false, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "Airtable client is not initialized"}
	}

	// Try to list tables as a health check
	tables := c.client.Tables()
	_, err := tables.List(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}

// GetMetadata returns connector metadata (implements connectors.Connector)
func (c *AirtableComponent) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "airtable",
		DisplayName: "Airtable",
		Description: "Airtable connector for managing bases, tables, and records",
		Version:     "1.0.0",
		Type:        connectors.TypeProductivity,
		Author:      "Fluxor Team",
		DocsURL:     "https://github.com/fluxorio/fluxor/tree/main/pkg/connectors/airtable",
		Capabilities: []connectors.Capability{
			{
				Name:        "read",
				Description: "Read tables and records from Airtable",
				Enabled:     true,
			},
			{
				Name:        "write",
				Description: "Create and update records in Airtable",
				Enabled:     true,
			},
			{
				Name:        "delete",
				Description: "Delete records from Airtable",
				Enabled:     true,
			},
			{
				Name:        "metadata",
				Description: "Access table and field metadata",
				Enabled:     true,
			},
		},
		RateLimits: &connectors.RateLimitInfo{
			RequestsPerSecond: c.config.RateLimit,
		},
		AuthMethods: []string{"api_key"},
		Tags:        []string{"productivity", "database", "spreadsheet", "collaboration"},
	}
}
