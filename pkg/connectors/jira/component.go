package jira

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// JiraComponent provides a minimal Jira connector implementation scaffold.
type JiraComponent struct {
	*core.BaseComponent
	config  Config
	mu      sync.RWMutex
	started bool
}

// NewJiraComponent creates a new Jira connector component.
func NewJiraComponent(config Config) *JiraComponent {
	return &JiraComponent{
		BaseComponent: core.NewBaseComponent("jira"),
		config:        config,
	}
}

func (c *JiraComponent) Start(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return &core.EventBusError{Code: "ALREADY_STARTED", Message: "component already started"}
	}
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}
	if err := c.config.Validate(); err != nil {
		return &core.EventBusError{Code: "CONFIG_VALIDATION_ERROR", Message: err.Error()}
	}

	c.started = true
	return nil
}

func (c *JiraComponent) Stop(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.started {
		return nil
	}
	c.started = false
	return nil
}

func (c *JiraComponent) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

func (c *JiraComponent) Name() string { return "jira" }

func (c *JiraComponent) Type() connectors.ConnectorType { return connectors.TypeProductivity }

func (c *JiraComponent) Version() string { return "0.1.0" }

func (c *JiraComponent) IsHealthy(_ context.Context) (bool, error) {
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "Jira component is not started"}
	}
	return true, nil
}

func (c *JiraComponent) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "jira",
		DisplayName: "Jira",
		Description: "Jira connector scaffold (issues, projects, workflows)",
		Version:     c.Version(),
		Type:        connectors.TypeProductivity,
		Author:      "Fluxor Team",
		DocsURL:     "https://github.com/fluxorio/fluxor/tree/main/pkg/connectors/jira",
		Capabilities: []connectors.Capability{
			{Name: "read", Description: "Read issues, projects, boards", Enabled: true},
			{Name: "write", Description: "Create/update issues and comments", Enabled: false},
			{Name: "webhook", Description: "Receive Jira webhook events", Enabled: false},
		},
		AuthMethods: []string{"api_token"},
		Tags:        []string{"productivity", "issues", "atlassian"},
	}
}

