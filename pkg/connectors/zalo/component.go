package zalo

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// ZaloConnector implements connectors.Connector for Zalo (OAuth + ZNS).
// Reference: apps/fluxor-mail ZNS client and zalo_zns utils.
type ZaloConnector struct {
	*core.BaseComponent
	config  Config
	client  *Client
	mu      sync.RWMutex
	started bool
}

// NewZaloConnector creates a new Zalo connector.
func NewZaloConnector(config Config) *ZaloConnector {
	return &ZaloConnector{
		BaseComponent: core.NewBaseComponent("zalo"),
		config:        config,
	}
}

// Start starts the connector and initializes the Zalo client.
func (c *ZaloConnector) Start(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.started {
		return &core.EventBusError{Code: "ALREADY_STARTED", Message: "zalo connector already started"}
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
func (c *ZaloConnector) Stop(ctx core.FluxorContext) error {
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
func (c *ZaloConnector) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

// Client returns the Zalo client. Call after Start().
func (c *ZaloConnector) Client() (*Client, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.started {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "zalo connector is not started"}
	}
	if c.client == nil {
		return nil, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "zalo client is not initialized"}
	}
	return c.client, nil
}

// Name implements connectors.Connector.
func (c *ZaloConnector) Name() string { return "zalo" }

// Type implements connectors.Connector.
func (c *ZaloConnector) Type() connectors.ConnectorType { return connectors.TypeMessaging }

// Version implements connectors.Connector.
func (c *ZaloConnector) Version() string { return "0.1.0" }

// IsHealthy implements connectors.Connector (uses GetQuota as a lightweight check when token is available).
func (c *ZaloConnector) IsHealthy(ctx context.Context) (bool, error) {
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "zalo connector is not started"}
	}
	client, err := c.Client()
	if err != nil {
		return false, err
	}
	_, err = client.GetQuota(ctx)
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetMetadata implements connectors.Connector.
func (c *ZaloConnector) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "zalo",
		DisplayName: "Zalo",
		Description: "Zalo Official Account connector: OAuth and ZNS (Zalo Notification Service) for templated messages. ZNS: gửi tin template tới SĐT (xác nhận đơn, nhắc lịch, OTP). Docs: https://developers.zalo.me/docs/zalo-notification-service/gui-tin-zns/gui-zns",
		Version:     c.Version(),
		Type:        connectors.TypeMessaging,
		Author:      "Fluxor Team",
		DocsURL:     "https://developers.zalo.me/docs/zalo-notification-service/gui-tin-zns/gui-zns",
		Capabilities: []connectors.Capability{
			{Name: "oauth", Description: "Exchange code for token, refresh access token", Enabled: true},
			{Name: "zns_send", Description: "Send ZNS templated notification messages", Enabled: true},
			{Name: "zns_template_info", Description: "Get ZNS template information", Enabled: true},
			{Name: "zns_quota", Description: "Get ZNS message quota", Enabled: true},
		},
		AuthMethods: []string{"access_token", "app_id+app_secret", "oauth"},
		Tags:        []string{"messaging", "notifications", "zalo", "zns"},
	}
}
