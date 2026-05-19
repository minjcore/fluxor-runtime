package momo

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// Component provides MoMo Payment Gateway integration with Fluxor.
// See https://developers.momo.vn/v3/docs/payment/api/credit/onetime/
type Component struct {
	*core.BaseComponent
	config Config
	client Client
	mu     sync.RWMutex
	started bool
}

// NewComponent creates a new MoMo connector component.
func NewComponent(config Config) *Component {
	return &Component{
		BaseComponent: core.NewBaseComponent("momo"),
		config:        config,
	}
}

// Start starts the connector.
func (c *Component) Start(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return &core.EventBusError{Code: "ALREADY_STARTED", Message: "momo connector already started"}
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
		return &core.EventBusError{Code: "MOMO_CLIENT_ERROR", Message: err.Error()}
	}
	c.client = client
	if eventBus := c.EventBus(); eventBus != nil {
		_ = eventBus.Publish("momo.ready", map[string]interface{}{"component": "momo"})
	}
	return nil
}

func (c *Component) doStop(ctx core.FluxorContext) error {
	c.client = nil
	if eventBus := c.EventBus(); eventBus != nil {
		_ = eventBus.Publish("momo.stopped", map[string]interface{}{"component": "momo"})
	}
	return nil
}

// Client returns the MoMo API client (valid only when started).
func (c *Component) Client() (Client, error) {
	if !c.IsStarted() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "MoMo component is not started"}
	}
	if c.client == nil {
		return nil, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "MoMo client is not initialized"}
	}
	return c.client, nil
}

// Payments returns the payments client for create/confirm.
func (c *Component) Payments() (PaymentsClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Payments(), nil
}

// Name implements connectors.Connector.
func (c *Component) Name() string { return "momo" }

// Type implements connectors.Connector.
func (c *Component) Type() connectors.ConnectorType { return connectors.TypeAPI }

// Version implements connectors.Connector.
func (c *Component) Version() string { return "1.0.0" }

// IsHealthy implements connectors.Connector.
func (c *Component) IsHealthy(ctx context.Context) (bool, error) {
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "MoMo component is not started"}
	}
	if c.client == nil {
		return false, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "MoMo client is not initialized"}
	}
	// Light health: config valid and client exists. Optional: call MoMo status API if available.
	return true, nil
}

// GetMetadata implements connectors.Connector.
func (c *Component) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "momo",
		DisplayName: "MoMo",
		Description: "MoMo Payment Gateway connector (Vietnam e-wallet): create payment, IPN/callback verification",
		Version:     "1.0.0",
		Type:        connectors.TypeAPI,
		Author:      "Fluxor Team",
		DocsURL:     "https://developers.momo.vn/v3/docs/payment/api/credit/onetime/",
		Capabilities: []connectors.Capability{
			{Name: "payments", Description: "Create payment (payUrl), verify callback/IPN", Enabled: true},
		},
		AuthMethods: []string{"partner_code", "access_key", "secret_key"},
		Tags:        []string{"payment", "ewallet", "vietnam", "momo"},
	}
}
