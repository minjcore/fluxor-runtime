package aimodule

import (
	"context"
	"time"
)

// Provider represents an AI provider type
type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderOllama    Provider = "ollama"
	ProviderGrok      Provider = "grok"
	ProviderGemini    Provider = "gemini"
	ProviderCursor    Provider = "cursor"
	ProviderCustom    Provider = "custom"
)

// LLMClient is the main interface for LLM operations
type LLMClient interface {
	// Chat sends a chat completion request
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// Embed creates embeddings for text
	Embed(ctx context.Context, req EmbedRequest) (*EmbedResponse, error)

	// ChatSimple sends a simple chat message and returns the response text
	ChatSimple(ctx context.Context, prompt string) (string, error)

	// Provider returns the provider type
	Provider() Provider
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	Tools       []Tool    `json:"tools,omitempty"`       // For function calling
	ToolChoice  string    `json:"tool_choice,omitempty"` // "auto", "none", or specific tool
	Stream      bool      `json:"stream,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role       string     `json:"role"`                   // "system", "user", "assistant", "tool"
	Content    string     `json:"content,omitempty"`      // Text content
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // Tool calls from assistant
	ToolCallID string     `json:"tool_call_id,omitempty"` // For tool messages
	Name       string     `json:"name,omitempty"`         // Tool name
}

// Tool represents a function/tool definition for function calling
type Tool struct {
	Type     string             `json:"type"` // "function"
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition defines a function for tool calling
type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON schema
}

// ToolCall represents a tool call from the assistant
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"` // "function"
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction contains the function call details
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	ID      string   `json:"id"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a response choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// EmbedRequest represents an embedding request
type EmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"` // Can be string or array of strings
}

// EmbedResponse represents an embedding response
type EmbedResponse struct {
	Data  []EmbeddingData `json:"data"`
	Model string          `json:"model"`
	Usage EmbedUsage      `json:"usage"`
}

// EmbeddingData contains a single embedding
type EmbeddingData struct {
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
	Object    string    `json:"object"`
}

// EmbedUsage represents embedding usage
type EmbedUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// Config represents AI module configuration
type Config struct {
	Provider     Provider         `json:"provider"`
	APIKey       string           `json:"apiKey,omitempty"`
	BaseURL      string           `json:"baseURL,omitempty"`
	DefaultModel string           `json:"defaultModel,omitempty"`
	Model        string           `json:"model,omitempty"`        // Alias for DefaultModel (for backward compatibility)
	Timeout      time.Duration    `json:"timeout,omitempty"`
	MaxRetries   int              `json:"maxRetries,omitempty"`
	RateLimit    *RateLimitConfig `json:"rateLimit,omitempty"`
	Cache        *CacheConfig     `json:"cache,omitempty"`
}

// RateLimitConfig configures rate limiting
type RateLimitConfig struct {
	RequestsPerMinute int `json:"requestsPerMinute"`
	RequestsPerDay    int `json:"requestsPerDay"`
}

// CacheConfig configures response caching
type CacheConfig struct {
	Enabled bool          `json:"enabled"`
	TTL     time.Duration `json:"ttl,omitempty"`
}
