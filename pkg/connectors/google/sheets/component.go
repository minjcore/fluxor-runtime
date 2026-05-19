package sheets

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// SheetComponent provides Google Sheets integration with Fluxor
// Similar to AirtableComponent, this component manages Google Sheets client lifecycle
type SheetComponent struct {
	*core.BaseComponent
	config  Config
	client  Client
	mu      sync.RWMutex
	started bool
}

// NewSheetComponent creates a new Google Sheets component
// Fail-fast: Validates configuration
func NewSheetComponent(config Config) *SheetComponent {
	return &SheetComponent{
		BaseComponent: core.NewBaseComponent("google-sheets"),
		config:        config,
	}
}

// Start initializes the component (overrides BaseComponent.Start to call our doStart)
func (c *SheetComponent) Start(ctx core.FluxorContext) error {
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
func (c *SheetComponent) Stop(ctx core.FluxorContext) error {
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
func (c *SheetComponent) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

// doStart initializes the Google Sheets client
// Fail-fast: Validates state and configuration before starting
func (c *SheetComponent) doStart(ctx core.FluxorContext) error {
	// Fail-fast: Validate context
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}

	// Validate configuration
	if err := c.config.Validate(); err != nil {
		return &core.EventBusError{Code: "CONFIG_VALIDATION_ERROR", Message: err.Error()}
	}

	// Create Google Sheets client
	client, err := NewClient(c.config)
	if err != nil {
		return &core.EventBusError{Code: "SHEETS_CLIENT_ERROR", Message: err.Error()}
	}

	c.client = client

	// Notify via EventBus (Premium Pattern integration)
	eventBus := c.EventBus()
	if eventBus != nil {
		if err := eventBus.Publish("google.sheets.ready", map[string]interface{}{
			"component":     "google-sheets",
			"spreadsheetID": c.config.SpreadsheetID,
		}); err != nil {
			// Best-effort notification; ignore on error.
		}
	}

	return nil
}

// doStop stops the Google Sheets component
func (c *SheetComponent) doStop(ctx core.FluxorContext) error {
	// Google Sheets client doesn't need explicit cleanup (HTTP client handles it)
	c.client = nil

	// Notify via EventBus
	eventBus := c.EventBus()
	if eventBus != nil {
		if err := eventBus.Publish("google.sheets.stopped", map[string]interface{}{
			"component": "google-sheets",
		}); err != nil {
			// Best-effort notification; ignore on error.
		}
	}

	return nil
}

// Client returns the Google Sheets client
// Fail-fast: Returns error if component is not started or client is nil
func (c *SheetComponent) Client() (Client, error) {
	if !c.IsStarted() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "Google Sheets component is not started"}
	}
	if c.client == nil {
		return nil, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "Google Sheets client is not initialized"}
	}
	return c.client, nil
}

// Name returns the connector name (implements connectors.Connector)
func (c *SheetComponent) Name() string {
	return "google-sheets"
}

// Type returns the connector type (implements connectors.Connector)
func (c *SheetComponent) Type() connectors.ConnectorType {
	return connectors.TypeProductivity
}

// Version returns the connector version (implements connectors.Connector)
func (c *SheetComponent) Version() string {
	return "1.0.0"
}

// IsHealthy checks if the connector is healthy (implements connectors.Connector)
func (c *SheetComponent) IsHealthy(ctx context.Context) (bool, error) {
	// Check if component is started
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "Google Sheets component is not started"}
	}

	// Check if client is initialized
	if c.client == nil {
		return false, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "Google Sheets client is not initialized"}
	}

	// Try to get spreadsheet info as a health check
	_, err := c.client.GetSpreadsheetInfo(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}

// GetMetadata returns connector metadata (implements connectors.Connector)
func (c *SheetComponent) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "google-sheets",
		DisplayName: "Google Sheets",
		Description: "Google Sheets connector for reading and writing spreadsheet data",
		Version:     "1.0.0",
		Type:        connectors.TypeProductivity,
		Author:      "Fluxor Team",
		DocsURL:     "https://github.com/fluxorio/fluxor/tree/main/pkg/connectors/google/sheets",
		Capabilities: []connectors.Capability{
			{
				Name:        "read",
				Description: "Read values from Google Sheets",
				Enabled:     true,
			},
			{
				Name:        "write",
				Description: "Write values to Google Sheets",
				Enabled:     true,
			},
			{
				Name:        "update",
				Description: "Update values in Google Sheets",
				Enabled:     true,
			},
			{
				Name:        "append",
				Description: "Append values to Google Sheets",
				Enabled:     true,
			},
			{
				Name:        "clear",
				Description: "Clear values in Google Sheets",
				Enabled:     true,
			},
			{
				Name:        "batch",
				Description: "Batch read and write operations",
				Enabled:     true,
			},
			{
				Name:        "metadata",
				Description: "Access spreadsheet and sheet metadata",
				Enabled:     true,
			},
		},
		RateLimits: &connectors.RateLimitInfo{
			RequestsPerSecond: c.config.RateLimit,
		},
		AuthMethods: []string{"oauth2", "service_account"},
		Tags:        []string{"productivity", "spreadsheet", "google", "collaboration"},
	}
}
