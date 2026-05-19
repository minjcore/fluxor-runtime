package notion

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// NotionComponent provides Notion integration with Fluxor
type NotionComponent struct {
	*core.BaseComponent
	config  Config
	client  Client
	mu      sync.RWMutex
	started bool
}

// NewNotionComponent creates a new Notion component
func NewNotionComponent(config Config) *NotionComponent {
	return &NotionComponent{
		BaseComponent: core.NewBaseComponent("notion"),
		config:        config,
	}
}

func (c *NotionComponent) Start(ctx core.FluxorContext) error {
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

func (c *NotionComponent) Stop(ctx core.FluxorContext) error {
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

func (c *NotionComponent) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

func (c *NotionComponent) doStart(ctx core.FluxorContext) error {
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}

	if err := c.config.Validate(); err != nil {
		return &core.EventBusError{Code: "CONFIG_VALIDATION_ERROR", Message: err.Error()}
	}

	client, err := NewClient(c.config)
	if err != nil {
		return &core.EventBusError{Code: "NOTION_CLIENT_ERROR", Message: err.Error()}
	}

	c.client = client

	if eventBus := c.EventBus(); eventBus != nil {
		_ = eventBus.Publish("notion.ready", map[string]interface{}{"component": "notion"})
	}

	return nil
}

func (c *NotionComponent) doStop(ctx core.FluxorContext) error {
	c.client = nil

	if eventBus := c.EventBus(); eventBus != nil {
		_ = eventBus.Publish("notion.stopped", map[string]interface{}{"component": "notion"})
	}

	return nil
}

func (c *NotionComponent) Client() (Client, error) {
	if !c.IsStarted() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "Notion component is not started"}
	}
	if c.client == nil {
		return nil, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "Notion client is not initialized"}
	}
	return c.client, nil
}

func (c *NotionComponent) Pages() (PagesClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Pages(), nil
}

func (c *NotionComponent) Databases() (DatabasesClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Databases(), nil
}

func (c *NotionComponent) Blocks() (BlocksClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Blocks(), nil
}

func (c *NotionComponent) Users() (UsersClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Users(), nil
}

func (c *NotionComponent) Search() (SearchClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Search(), nil
}

func (c *NotionComponent) Name() string {
	return "notion"
}

func (c *NotionComponent) Type() connectors.ConnectorType {
	return connectors.TypeProductivity
}

func (c *NotionComponent) Version() string {
	return "1.0.0"
}

func (c *NotionComponent) IsHealthy(ctx context.Context) (bool, error) {
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "Notion component is not started"}
	}
	if c.client == nil {
		return false, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "Notion client is not initialized"}
	}

	users := c.client.Users()
	_, err := users.GetMe(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (c *NotionComponent) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "notion",
		DisplayName: "Notion",
		Description: "Notion connector for pages, databases, and blocks",
		Version:     "1.0.0",
		Type:        connectors.TypeProductivity,
		Author:      "Fluxor Team",
		DocsURL:     "https://github.com/fluxorio/fluxor/tree/main/pkg/connectors/notion",
		Capabilities: []connectors.Capability{
			{Name: "pages", Description: "Create, read, update pages", Enabled: true},
			{Name: "databases", Description: "Create, query, update databases", Enabled: true},
			{Name: "blocks", Description: "Manage content blocks", Enabled: true},
			{Name: "search", Description: "Search pages and databases", Enabled: true},
		},
		RateLimits: &connectors.RateLimitInfo{
			RequestsPerSecond: c.config.RateLimit,
		},
		AuthMethods: []string{"integration_token"},
		Tags:        []string{"productivity", "database", "wiki", "collaboration"},
	}
}
