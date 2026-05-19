package github

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// GitHubComponent provides GitHub integration with Fluxor
type GitHubComponent struct {
	*core.BaseComponent
	config  Config
	client  Client
	mu      sync.RWMutex
	started bool
}

// NewGitHubComponent creates a new GitHub component
func NewGitHubComponent(config Config) *GitHubComponent {
	return &GitHubComponent{
		BaseComponent: core.NewBaseComponent("github"),
		config:        config,
	}
}

// Start initializes the component
func (c *GitHubComponent) Start(ctx core.FluxorContext) error {
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

// Stop stops the component
func (c *GitHubComponent) Stop(ctx core.FluxorContext) error {
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

// IsStarted returns whether the component is started
func (c *GitHubComponent) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

// doStart initializes the GitHub client
func (c *GitHubComponent) doStart(ctx core.FluxorContext) error {
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}

	if err := c.config.Validate(); err != nil {
		return &core.EventBusError{Code: "CONFIG_VALIDATION_ERROR", Message: err.Error()}
	}

	client, err := NewClient(c.config)
	if err != nil {
		return &core.EventBusError{Code: "GITHUB_CLIENT_ERROR", Message: err.Error()}
	}

	c.client = client

	eventBus := c.EventBus()
	if eventBus != nil {
		_ = eventBus.Publish("github.ready", map[string]interface{}{
			"component": "github",
		})
	}

	return nil
}

// doStop stops the GitHub component
func (c *GitHubComponent) doStop(ctx core.FluxorContext) error {
	c.client = nil

	eventBus := c.EventBus()
	if eventBus != nil {
		_ = eventBus.Publish("github.stopped", map[string]interface{}{
			"component": "github",
		})
	}

	return nil
}

// Client returns the GitHub client
func (c *GitHubComponent) Client() (Client, error) {
	if !c.IsStarted() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "GitHub component is not started"}
	}
	if c.client == nil {
		return nil, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "GitHub client is not initialized"}
	}
	return c.client, nil
}

// Repositories returns the Repositories client
func (c *GitHubComponent) Repositories() (RepositoriesClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Repositories(), nil
}

// Issues returns the Issues client
func (c *GitHubComponent) Issues() (IssuesClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Issues(), nil
}

// PullRequests returns the PullRequests client
func (c *GitHubComponent) PullRequests() (PullRequestsClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.PullRequests(), nil
}

// Users returns the Users client
func (c *GitHubComponent) Users() (UsersClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Users(), nil
}

// Name returns the connector name
func (c *GitHubComponent) Name() string {
	return "github"
}

// Type returns the connector type
func (c *GitHubComponent) Type() connectors.ConnectorType {
	return connectors.TypeAPI
}

// Version returns the connector version
func (c *GitHubComponent) Version() string {
	return "1.0.0"
}

// IsHealthy checks if the connector is healthy
func (c *GitHubComponent) IsHealthy(ctx context.Context) (bool, error) {
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "GitHub component is not started"}
	}

	if c.client == nil {
		return false, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "GitHub client is not initialized"}
	}

	// Try to get the authenticated user as a health check
	users := c.client.Users()
	_, err := users.Get(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}

// GetMetadata returns connector metadata
func (c *GitHubComponent) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "github",
		DisplayName: "GitHub",
		Description: "GitHub connector for repositories, issues, and pull requests",
		Version:     "1.0.0",
		Type:        connectors.TypeAPI,
		Author:      "Fluxor Team",
		DocsURL:     "https://github.com/fluxorio/fluxor/tree/main/pkg/connectors/github",
		Capabilities: []connectors.Capability{
			{Name: "repositories", Description: "Manage repositories", Enabled: true},
			{Name: "issues", Description: "Manage issues and comments", Enabled: true},
			{Name: "pull_requests", Description: "Manage pull requests and reviews", Enabled: true},
			{Name: "users", Description: "Access user and organization data", Enabled: true},
		},
		RateLimits: &connectors.RateLimitInfo{
			RequestsPerHour: c.config.RateLimit,
		},
		AuthMethods: []string{"personal_access_token", "oauth", "github_app"},
		Tags:        []string{"api", "git", "version-control", "collaboration"},
	}
}
