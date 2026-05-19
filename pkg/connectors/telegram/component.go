package telegram

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// TelegramComponent provides Telegram integration with Fluxor
type TelegramComponent struct {
	*core.BaseComponent
	config  Config
	client  Client
	mu      sync.RWMutex
	started bool
}

// NewTelegramComponent creates a new Telegram component
func NewTelegramComponent(config Config) *TelegramComponent {
	return &TelegramComponent{
		BaseComponent: core.NewBaseComponent("telegram"),
		config:        config,
	}
}

func (c *TelegramComponent) Start(ctx core.FluxorContext) error {
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

func (c *TelegramComponent) Stop(ctx core.FluxorContext) error {
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

func (c *TelegramComponent) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

func (c *TelegramComponent) doStart(ctx core.FluxorContext) error {
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}

	if err := c.config.Validate(); err != nil {
		return &core.EventBusError{Code: "CONFIG_VALIDATION_ERROR", Message: err.Error()}
	}

	client, err := NewClient(c.config)
	if err != nil {
		return &core.EventBusError{Code: "TELEGRAM_CLIENT_ERROR", Message: err.Error()}
	}

	c.client = client

	if eventBus := c.EventBus(); eventBus != nil {
		_ = eventBus.Publish("telegram.ready", map[string]interface{}{"component": "telegram"})
	}

	return nil
}

func (c *TelegramComponent) doStop(ctx core.FluxorContext) error {
	c.client = nil

	if eventBus := c.EventBus(); eventBus != nil {
		_ = eventBus.Publish("telegram.stopped", map[string]interface{}{"component": "telegram"})
	}

	return nil
}

func (c *TelegramComponent) Client() (Client, error) {
	if !c.IsStarted() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "Telegram component is not started"}
	}
	if c.client == nil {
		return nil, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "Telegram client is not initialized"}
	}
	return c.client, nil
}

func (c *TelegramComponent) Messages() (MessagesClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Messages(), nil
}

func (c *TelegramComponent) Chats() (ChatsClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Chats(), nil
}

func (c *TelegramComponent) Users() (UsersClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Users(), nil
}

func (c *TelegramComponent) Name() string {
	return "telegram"
}

func (c *TelegramComponent) Type() connectors.ConnectorType {
	return connectors.TypeMessaging
}

func (c *TelegramComponent) Version() string {
	return "1.0.0"
}

func (c *TelegramComponent) IsHealthy(ctx context.Context) (bool, error) {
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "Telegram component is not started"}
	}
	if c.client == nil {
		return false, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "Telegram client is not initialized"}
	}

	users := c.client.Users()
	_, err := users.GetMe(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (c *TelegramComponent) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "telegram",
		DisplayName: "Telegram",
		Description: "Telegram connector for messages, chats, and users",
		Version:     "1.0.0",
		Type:        connectors.TypeMessaging,
		Author:      "Fluxor Team",
		DocsURL:     "https://github.com/fluxorio/fluxor/tree/main/pkg/connectors/telegram",
		Capabilities: []connectors.Capability{
			{Name: "messages", Description: "Send, edit, and delete messages", Enabled: true},
			{Name: "chats", Description: "Access chat information", Enabled: true},
			{Name: "users", Description: "Access user data", Enabled: true},
			{Name: "media", Description: "Send photos, documents, and files", Enabled: true},
		},
		RateLimits: &connectors.RateLimitInfo{
			RequestsPerSecond: c.config.RateLimit,
		},
		AuthMethods: []string{"bot_token"},
		Tags:        []string{"messaging", "chat", "bot", "automation"},
	}
}
