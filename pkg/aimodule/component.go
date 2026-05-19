package aimodule

import (
	"github.com/fluxorio/fluxor/pkg/core"
)

// AIComponent provides AI client integration with Fluxor
// Similar to DatabaseComponent, this component manages AI client lifecycle
type AIComponent struct {
	*core.BaseComponent
	config Config
	client LLMClient
}

// NewAIComponent creates a new AI component
// Fail-fast: Validates configuration
func NewAIComponent(config Config) *AIComponent {
	// Fail-fast: Validate provider
	if config.Provider == "" {
		config.Provider = ProviderOpenAI
	}

	// Handle Model field (backward compatibility with pkg/ai)
	if config.Model != "" && config.DefaultModel == "" {
		config.DefaultModel = config.Model
	}

	return &AIComponent{
		BaseComponent: core.NewBaseComponent("ai"),
		config:        config,
	}
}

// doStart initializes the AI client
// Fail-fast: Validates state and configuration before starting
func (c *AIComponent) doStart(ctx core.FluxorContext) error {
	// Fail-fast: Validate context
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}

	// Create AI client
	client, err := NewClient(c.config)
	if err != nil {
		return &core.EventBusError{Code: "AI_CLIENT_ERROR", Message: err.Error()}
	}

	c.client = client

	// Notify via EventBus (Premium Pattern integration)
	eventBus := c.EventBus()
	if eventBus != nil {
		model := c.config.DefaultModel
		if model == "" {
			model = c.config.Model
		}
		if err := eventBus.Publish("ai.ready", map[string]interface{}{
			"component": string(c.config.Provider),
			"model":     model,
		}); err != nil {
			// Best-effort notification; ignore on error.
		}
	}

	return nil
}

// doStop stops the AI component
func (c *AIComponent) doStop(ctx core.FluxorContext) error {
	// AI client doesn't need explicit cleanup (HTTP client is stateless)
	c.client = nil

	// Notify via EventBus
	eventBus := c.EventBus()
	if eventBus != nil {
		if err := eventBus.Publish("ai.stopped", map[string]interface{}{
			"component": string(c.config.Provider),
		}); err != nil {
			// Best-effort notification; ignore on error.
		}
	}

	return nil
}

// Client returns the AI client
// Fail-fast: Returns error if component is not started
func (c *AIComponent) Client() (LLMClient, error) {
	if !c.IsStarted() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "AI component is not started"}
	}
	return c.client, nil
}

// Chat sends a chat completion request
func (c *AIComponent) Chat(ctx core.FluxorContext, req ChatRequest) (*ChatResponse, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Chat(ctx.Context(), req)
}

// ChatSimple sends a simple chat message and returns the response text
func (c *AIComponent) ChatSimple(ctx core.FluxorContext, prompt string) (string, error) {
	client, err := c.Client()
	if err != nil {
		return "", err
	}
	return client.ChatSimple(ctx.Context(), prompt)
}
