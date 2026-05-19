package trello

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// TrelloComponent provides Trello integration with Fluxor
type TrelloComponent struct {
	*core.BaseComponent
	config  Config
	client  Client
	mu      sync.RWMutex
	started bool
}

// NewTrelloComponent creates a new Trello component
func NewTrelloComponent(config Config) *TrelloComponent {
	return &TrelloComponent{
		BaseComponent: core.NewBaseComponent("trello"),
		config:        config,
	}
}

func (c *TrelloComponent) Start(ctx core.FluxorContext) error {
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

func (c *TrelloComponent) Stop(ctx core.FluxorContext) error {
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

func (c *TrelloComponent) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

func (c *TrelloComponent) doStart(ctx core.FluxorContext) error {
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}

	if err := c.config.Validate(); err != nil {
		return &core.EventBusError{Code: "CONFIG_VALIDATION_ERROR", Message: err.Error()}
	}

	client, err := NewClient(c.config)
	if err != nil {
		return &core.EventBusError{Code: "TRELLO_CLIENT_ERROR", Message: err.Error()}
	}

	c.client = client

	if eventBus := c.EventBus(); eventBus != nil {
		_ = eventBus.Publish("trello.ready", map[string]interface{}{"component": "trello"})
	}

	return nil
}

func (c *TrelloComponent) doStop(ctx core.FluxorContext) error {
	c.client = nil

	if eventBus := c.EventBus(); eventBus != nil {
		_ = eventBus.Publish("trello.stopped", map[string]interface{}{"component": "trello"})
	}

	return nil
}

func (c *TrelloComponent) Client() (Client, error) {
	if !c.IsStarted() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "Trello component is not started"}
	}
	if c.client == nil {
		return nil, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "Trello client is not initialized"}
	}
	return c.client, nil
}

func (c *TrelloComponent) Boards() (BoardsClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Boards(), nil
}

func (c *TrelloComponent) Lists() (ListsClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Lists(), nil
}

func (c *TrelloComponent) Cards() (CardsClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Cards(), nil
}

func (c *TrelloComponent) Members() (MembersClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Members(), nil
}

func (c *TrelloComponent) Name() string {
	return "trello"
}

func (c *TrelloComponent) Type() connectors.ConnectorType {
	return connectors.TypeProductivity
}

func (c *TrelloComponent) Version() string {
	return "1.0.0"
}

func (c *TrelloComponent) IsHealthy(ctx context.Context) (bool, error) {
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "Trello component is not started"}
	}
	if c.client == nil {
		return false, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "Trello client is not initialized"}
	}

	members := c.client.Members()
	_, err := members.GetMe(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (c *TrelloComponent) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "trello",
		DisplayName: "Trello",
		Description: "Trello connector for boards, lists, and cards",
		Version:     "1.0.0",
		Type:        connectors.TypeProductivity,
		Author:      "Fluxor Team",
		DocsURL:     "https://github.com/fluxorio/fluxor/tree/main/pkg/connectors/trello",
		Capabilities: []connectors.Capability{
			{Name: "boards", Description: "Manage boards", Enabled: true},
			{Name: "lists", Description: "Manage lists", Enabled: true},
			{Name: "cards", Description: "Manage cards, labels, and checklists", Enabled: true},
			{Name: "members", Description: "Access member data", Enabled: true},
		},
		RateLimits: &connectors.RateLimitInfo{
			RequestsPerSecond: c.config.RateLimit / 10, // Per 10 seconds to per second
		},
		AuthMethods: []string{"api_key", "token"},
		Tags:        []string{"productivity", "project-management", "kanban", "collaboration"},
	}
}
