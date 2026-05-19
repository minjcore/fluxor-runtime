package http

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// HTTPComponent provides HTTP client integration with Fluxor
type HTTPComponent struct {
	*core.BaseComponent
	config  Config
	client  Client
	mu      sync.RWMutex
	started bool
}

// NewHTTPComponent creates a new HTTP component
func NewHTTPComponent(config Config) *HTTPComponent {
	return &HTTPComponent{
		BaseComponent: core.NewBaseComponent("http"),
		config:        config,
	}
}

func (c *HTTPComponent) Start(ctx core.FluxorContext) error {
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

func (c *HTTPComponent) Stop(ctx core.FluxorContext) error {
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

func (c *HTTPComponent) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

func (c *HTTPComponent) doStart(ctx core.FluxorContext) error {
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}

	if err := c.config.Validate(); err != nil {
		return &core.EventBusError{Code: "CONFIG_VALIDATION_ERROR", Message: err.Error()}
	}

	client, err := NewClient(c.config)
	if err != nil {
		return &core.EventBusError{Code: "HTTP_CLIENT_ERROR", Message: err.Error()}
	}

	c.client = client

	if eventBus := c.EventBus(); eventBus != nil {
		_ = eventBus.Publish("http.ready", map[string]interface{}{
			"component": "http",
			"baseURL":   c.config.BaseURL,
		})
	}

	return nil
}

func (c *HTTPComponent) doStop(ctx core.FluxorContext) error {
	c.client = nil

	if eventBus := c.EventBus(); eventBus != nil {
		_ = eventBus.Publish("http.stopped", map[string]interface{}{
			"component": "http",
		})
	}

	return nil
}

func (c *HTTPComponent) Client() (Client, error) {
	if !c.IsStarted() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "HTTP component is not started"}
	}
	if c.client == nil {
		return nil, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "HTTP client is not initialized"}
	}
	return c.client, nil
}

func (c *HTTPComponent) Name() string {
	return "http"
}

func (c *HTTPComponent) Type() connectors.ConnectorType {
	return connectors.TypeAPI
}

func (c *HTTPComponent) Version() string {
	return "1.0.0"
}

func (c *HTTPComponent) IsHealthy(ctx context.Context) (bool, error) {
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "HTTP component is not started"}
	}
	if c.client == nil {
		return false, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "HTTP client is not initialized"}
	}

	// If BaseURL is set, try a HEAD request as health check
	if c.config.BaseURL != "" {
		_, err := c.client.Get(ctx, c.config.BaseURL+"/health", nil)
		if err != nil {
			// Health check failed, but component is still functional
			// Return true if it's just a 404/500, false if connection error
			return false, err
		}
	}

	return true, nil
}

func (c *HTTPComponent) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "http",
		DisplayName: "HTTP Client",
		Description: "Generic HTTP client connector for making HTTP requests to any API",
		Version:     "1.0.0",
		Type:        connectors.TypeAPI,
		Author:      "Fluxor Team",
		DocsURL:     "https://github.com/fluxorio/fluxor/tree/main/pkg/connectors/http",
		Capabilities: []connectors.Capability{
			{Name: "get", Description: "Perform GET requests", Enabled: true},
			{Name: "post", Description: "Perform POST requests", Enabled: true},
			{Name: "put", Description: "Perform PUT requests", Enabled: true},
			{Name: "patch", Description: "Perform PATCH requests", Enabled: true},
			{Name: "delete", Description: "Perform DELETE requests", Enabled: true},
			{Name: "custom", Description: "Perform custom HTTP requests", Enabled: true},
			{Name: "auth", Description: "Support multiple authentication methods", Enabled: true},
			{Name: "retry", Description: "Automatic retry with exponential backoff", Enabled: true},
			{Name: "rate_limit", Description: "Rate limiting support", Enabled: true},
		},
		AuthMethods: []string{"bearer", "basic", "apikey", "custom", "none"},
		Tags:        []string{"http", "rest", "api", "client", "integration"},
	}
}
