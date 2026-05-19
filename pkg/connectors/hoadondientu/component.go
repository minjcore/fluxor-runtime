package hoadondientu

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// Component provides Vietnam e-invoice (Hóa đơn điện tử) integration with Fluxor.
// Compatible with MISA meInvoice (https://doc.meinvoice.vn) and configurable for
// other providers (VNPT, eHoaDon, etc.) via BaseURL.
type Component struct {
	*core.BaseComponent
	config  Config
	client  Client
	mu      sync.RWMutex
	started bool
}

// NewComponent creates a new Hóa đơn điện tử connector component.
func NewComponent(config Config) *Component {
	return &Component{
		BaseComponent: core.NewBaseComponent("hoadondientu"),
		config:        config,
	}
}

// Start starts the connector.
func (c *Component) Start(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return &core.EventBusError{Code: "ALREADY_STARTED", Message: "hoadondientu connector already started"}
	}

	if err := c.doStart(ctx); err != nil {
		return err
	}
	c.started = true
	return nil
}

// Stop stops the connector.
func (c *Component) Stop(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil
	}
	if err := c.doStop(ctx); err != nil {
		return err
	}
	c.started = false
	return nil
}

// IsStarted returns whether the connector is started.
func (c *Component) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

func (c *Component) doStart(ctx core.FluxorContext) error {
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}
	if err := c.config.Validate(); err != nil {
		return &core.EventBusError{Code: "CONFIG_VALIDATION_ERROR", Message: err.Error()}
	}
	client, err := NewClient(c.config)
	if err != nil {
		return &core.EventBusError{Code: "HOADONDIENTU_CLIENT_ERROR", Message: err.Error()}
	}
	c.client = client
	if eventBus := c.EventBus(); eventBus != nil {
		_ = eventBus.Publish("hoadondientu.ready", map[string]interface{}{"component": "hoadondientu"})
	}
	return nil
}

func (c *Component) doStop(ctx core.FluxorContext) error {
	c.client = nil
	if eventBus := c.EventBus(); eventBus != nil {
		_ = eventBus.Publish("hoadondientu.stopped", map[string]interface{}{"component": "hoadondientu"})
	}
	return nil
}

// Client returns the e-invoice API client (valid only when started).
func (c *Component) Client() (Client, error) {
	if !c.IsStarted() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "hoadondientu component is not started"}
	}
	if c.client == nil {
		return nil, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "hoadondientu client is not initialized"}
	}
	return c.client, nil
}

// Invoices returns the invoices client.
func (c *Component) Invoices() (InvoicesClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Invoices(), nil
}

// Name implements connectors.Connector.
func (c *Component) Name() string { return "hoadondientu" }

// Type implements connectors.Connector.
func (c *Component) Type() connectors.ConnectorType { return connectors.TypeAPI }

// Version implements connectors.Connector.
func (c *Component) Version() string { return "1.0.0" }

// IsHealthy implements connectors.Connector.
func (c *Component) IsHealthy(ctx context.Context) (bool, error) {
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "hoadondientu component is not started"}
	}
	if c.client == nil {
		return false, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "hoadondientu client is not initialized"}
	}
	// Light health: ensure token can be obtained (or list templates with limit)
	if err := c.client.EnsureToken(ctx); err != nil {
		return false, err
	}
	return true, nil
}

// GetMetadata implements connectors.Connector.
func (c *Component) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "hoadondientu",
		DisplayName: "Hóa đơn điện tử",
		Description: "Vietnam e-invoice connector: create, publish, list, download PDF/XML, send email. Compatible with MISA meInvoice and configurable for VNPT, eHoaDon, etc.",
		Version:     "1.0.0",
		Type:        connectors.TypeAPI,
		Author:      "Fluxor Team",
		DocsURL:     "https://doc.meinvoice.vn/",
		Capabilities: []connectors.Capability{
			{Name: "invoices", Description: "Create, get, list, update, delete e-invoices", Enabled: true},
			{Name: "publish", Description: "Publish invoice to tax authority", Enabled: true},
			{Name: "download", Description: "Download PDF/XML", Enabled: true},
			{Name: "send_email", Description: "Send invoice by email", Enabled: true},
			{Name: "templates", Description: "List invoice templates (mẫu số, ký hiệu)", Enabled: true},
		},
		AuthMethods: []string{"bearer_token", "username_password"},
		Tags:        []string{"vietnam", "e-invoice", "hoadondientu", "tax", "misa", "vnpt"},
	}
}
