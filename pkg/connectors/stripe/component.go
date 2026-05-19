package stripe

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// StripeComponent provides Stripe integration with Fluxor
type StripeComponent struct {
	*core.BaseComponent
	config  Config
	client  Client
	mu      sync.RWMutex
	started bool
}

// NewStripeComponent creates a new Stripe component
func NewStripeComponent(config Config) *StripeComponent {
	return &StripeComponent{
		BaseComponent: core.NewBaseComponent("stripe"),
		config:        config,
	}
}

func (c *StripeComponent) Start(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return &core.EventBusError{Code: "ALREADY_STARTED", Message: "component already started"}
	}

	if err := c.doStart(ctx); err != nil {
		return err
	}

	c.started = true
	return nil
}

func (c *StripeComponent) Stop(ctx core.FluxorContext) error {
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

func (c *StripeComponent) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

func (c *StripeComponent) doStart(ctx core.FluxorContext) error {
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}

	if err := c.config.Validate(); err != nil {
		return &core.EventBusError{Code: "CONFIG_VALIDATION_ERROR", Message: err.Error()}
	}

	client, err := NewClient(c.config)
	if err != nil {
		return &core.EventBusError{Code: "STRIPE_CLIENT_ERROR", Message: err.Error()}
	}

	c.client = client

	if eventBus := c.EventBus(); eventBus != nil {
		_ = eventBus.Publish("stripe.ready", map[string]interface{}{"component": "stripe"})
	}

	return nil
}

func (c *StripeComponent) doStop(ctx core.FluxorContext) error {
	c.client = nil

	if eventBus := c.EventBus(); eventBus != nil {
		_ = eventBus.Publish("stripe.stopped", map[string]interface{}{"component": "stripe"})
	}

	return nil
}

func (c *StripeComponent) Client() (Client, error) {
	if !c.IsStarted() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "Stripe component is not started"}
	}
	if c.client == nil {
		return nil, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "Stripe client is not initialized"}
	}
	return c.client, nil
}

func (c *StripeComponent) Customers() (CustomersClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Customers(), nil
}

func (c *StripeComponent) PaymentIntents() (PaymentIntentsClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.PaymentIntents(), nil
}

func (c *StripeComponent) Subscriptions() (SubscriptionsClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Subscriptions(), nil
}

func (c *StripeComponent) Name() string {
	return "stripe"
}

func (c *StripeComponent) Type() connectors.ConnectorType {
	return connectors.TypeAPI
}

func (c *StripeComponent) Version() string {
	return "1.0.0"
}

func (c *StripeComponent) IsHealthy(ctx context.Context) (bool, error) {
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "Stripe component is not started"}
	}
	if c.client == nil {
		return false, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "Stripe client is not initialized"}
	}

	// Try to list customers with limit 1 as health check
	customers := c.client.Customers()
	_, err := customers.List(ctx, &ListParams{Limit: 1})
	if err != nil {
		return false, err
	}

	return true, nil
}

func (c *StripeComponent) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "stripe",
		DisplayName: "Stripe",
		Description: "Stripe connector for payments, subscriptions, and billing",
		Version:     "1.0.0",
		Type:        connectors.TypeAPI,
		Author:      "Fluxor Team",
		DocsURL:     "https://github.com/fluxorio/fluxor/tree/main/pkg/connectors/stripe",
		Capabilities: []connectors.Capability{
			{Name: "customers", Description: "Manage customers", Enabled: true},
			{Name: "payments", Description: "Process payments", Enabled: true},
			{Name: "subscriptions", Description: "Manage subscriptions", Enabled: true},
			{Name: "invoices", Description: "Manage invoices", Enabled: true},
			{Name: "products", Description: "Manage products and prices", Enabled: true},
		},
		RateLimits: &connectors.RateLimitInfo{
			RequestsPerSecond: c.config.RateLimit,
		},
		AuthMethods: []string{"secret_key"},
		Tags:        []string{"payment", "billing", "subscription", "commerce"},
	}
}
