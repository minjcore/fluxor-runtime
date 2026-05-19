package discord

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// DiscordComponent provides Discord integration with Fluxor
type DiscordComponent struct {
	*core.BaseComponent
	config  Config
	client  Client
	mu      sync.RWMutex
	started bool
}

// NewDiscordComponent creates a new Discord component
func NewDiscordComponent(config Config) *DiscordComponent {
	return &DiscordComponent{
		BaseComponent: core.NewBaseComponent("discord"),
		config:        config,
	}
}

func (c *DiscordComponent) Start(ctx core.FluxorContext) error {
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

func (c *DiscordComponent) Stop(ctx core.FluxorContext) error {
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

func (c *DiscordComponent) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

func (c *DiscordComponent) doStart(ctx core.FluxorContext) error {
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}

	if err := c.config.Validate(); err != nil {
		return &core.EventBusError{Code: "CONFIG_VALIDATION_ERROR", Message: err.Error()}
	}

	client, err := NewClient(c.config)
	if err != nil {
		return &core.EventBusError{Code: "DISCORD_CLIENT_ERROR", Message: err.Error()}
	}

	c.client = client

	if eventBus := c.EventBus(); eventBus != nil {
		_ = eventBus.Publish("discord.ready", map[string]interface{}{"component": "discord"})
	}

	return nil
}

func (c *DiscordComponent) doStop(ctx core.FluxorContext) error {
	c.client = nil

	if eventBus := c.EventBus(); eventBus != nil {
		_ = eventBus.Publish("discord.stopped", map[string]interface{}{"component": "discord"})
	}

	return nil
}

func (c *DiscordComponent) Client() (Client, error) {
	if !c.IsStarted() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "Discord component is not started"}
	}
	if c.client == nil {
		return nil, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "Discord client is not initialized"}
	}
	return c.client, nil
}

func (c *DiscordComponent) Messages() (MessagesClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Messages(), nil
}

func (c *DiscordComponent) Channels() (ChannelsClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Channels(), nil
}

func (c *DiscordComponent) Guilds() (GuildsClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Guilds(), nil
}

func (c *DiscordComponent) Users() (UsersClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Users(), nil
}

func (c *DiscordComponent) Name() string {
	return "discord"
}

func (c *DiscordComponent) Type() connectors.ConnectorType {
	return connectors.TypeMessaging
}

func (c *DiscordComponent) Version() string {
	return "1.0.0"
}

func (c *DiscordComponent) IsHealthy(ctx context.Context) (bool, error) {
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "Discord component is not started"}
	}
	if c.client == nil {
		return false, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "Discord client is not initialized"}
	}

	users := c.client.Users()
	_, err := users.GetCurrentUser(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (c *DiscordComponent) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "discord",
		DisplayName: "Discord",
		Description: "Discord connector for messages, channels, guilds, and users",
		Version:     "1.0.0",
		Type:        connectors.TypeMessaging,
		Author:      "Fluxor Team",
		DocsURL:     "https://github.com/fluxorio/fluxor/tree/main/pkg/connectors/discord",
		Capabilities: []connectors.Capability{
			{Name: "messages", Description: "Send, edit, and delete messages", Enabled: true},
			{Name: "channels", Description: "Manage channels", Enabled: true},
			{Name: "guilds", Description: "Access guild data", Enabled: true},
			{Name: "users", Description: "Access user data", Enabled: true},
		},
		RateLimits: &connectors.RateLimitInfo{
			RequestsPerSecond: c.config.RateLimit,
		},
		AuthMethods: []string{"bot_token"},
		Tags:        []string{"messaging", "chat", "gaming", "community"},
	}
}
