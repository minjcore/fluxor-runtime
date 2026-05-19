package aimodule

import (
	"context"
	"fmt"
	"time"

	"github.com/sashabaranov/go-openai"
)

// OpenAIClient implements LLMClient for OpenAI
type OpenAIClient struct {
	client  *openai.Client
	config  Config
	cache   *Cache
	limiter *RateLimiter
}

func init() {
	RegisterClientFactory(ProviderOpenAI, NewOpenAIClient)
	RegisterClientFactory(ProviderGrok, NewOpenAIClient)   // Grok uses OpenAI-compatible API
	RegisterClientFactory(ProviderGemini, NewOpenAIClient) // Gemini uses OpenAI-compatible API
	RegisterClientFactory(ProviderCursor, NewOpenAIClient) // Cursor uses OpenAI-compatible API
}

// NewOpenAIClient creates a new OpenAI client (also supports Grok and Gemini via OpenAI-compatible API)
func NewOpenAIClient(config Config) (LLMClient, error) {
	// Set defaults based on provider
	if config.DefaultModel == "" {
		switch config.Provider {
		case ProviderGrok:
			config.DefaultModel = "grok-beta"
		case ProviderGemini:
			config.DefaultModel = "gemini-1.5-flash"
		default:
			config.DefaultModel = "gpt-3.5-turbo"
		}
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	// Create OpenAI client
	openaiConfig := openai.DefaultConfig(config.APIKey)

	// Set BaseURL based on provider
	if config.BaseURL != "" {
		openaiConfig.BaseURL = config.BaseURL
	} else {
		switch config.Provider {
		case ProviderGrok:
			openaiConfig.BaseURL = "https://api.x.ai/v1"
		case ProviderGemini:
			openaiConfig.BaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
		}
	}

	client := openai.NewClientWithConfig(openaiConfig)

	// Initialize cache if enabled
	var cache *Cache
	if config.Cache != nil && config.Cache.Enabled {
		ttl := config.Cache.TTL
		if ttl == 0 {
			ttl = 5 * time.Minute // Default TTL
		}
		cache = NewCache(ttl)
	}

	// Initialize rate limiter if configured
	var limiter *RateLimiter
	if config.RateLimit != nil {
		limiter = NewRateLimiter(
			config.RateLimit.RequestsPerMinute,
			config.RateLimit.RequestsPerDay,
		)
	}

	return &OpenAIClient{
		client:  client,
		config:  config,
		cache:   cache,
		limiter: limiter,
	}, nil
}

// Provider returns the provider type
func (c *OpenAIClient) Provider() Provider {
	return ProviderOpenAI
}

// Chat sends a chat completion request
func (c *OpenAIClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Check rate limit
	if c.limiter != nil {
		if !c.limiter.Allow() {
			return nil, fmt.Errorf("rate limit exceeded")
		}
	}

	// Check cache
	if c.cache != nil {
		cacheKey, err := GenerateCacheKey(req)
		if err == nil {
			if cached, found := c.cache.Get(cacheKey); found {
				return cached, nil
			}
		}
	}

	// Set default model if not specified
	if req.Model == "" {
		req.Model = c.config.DefaultModel
	}

	// Convert messages
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			Name:       msg.Name,
			ToolCallID: msg.ToolCallID,
		}

		// Convert tool calls
		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]openai.ToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				toolCalls[j] = openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolType(tc.Type),
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
			messages[i].ToolCalls = toolCalls
		}
	}

	// Build request
	openaiReq := openai.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: float32(req.Temperature),
		MaxTokens:   req.MaxTokens,
		TopP:        float32(req.TopP),
		Stream:      req.Stream,
	}

	// Convert tools if provided
	if len(req.Tools) > 0 {
		openaiReq.Tools = make([]openai.Tool, len(req.Tools))
		for i, tool := range req.Tools {
			openaiReq.Tools[i] = openai.Tool{
				Type: openai.ToolType(tool.Type),
				Function: &openai.FunctionDefinition{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			}
		}

		// Set tool choice (ToolChoice is any type, can be string or ToolChoice struct)
		if req.ToolChoice != "" {
			if req.ToolChoice == "none" {
				openaiReq.ToolChoice = "none"
			} else if req.ToolChoice == "auto" || req.ToolChoice == "required" {
				openaiReq.ToolChoice = "auto"
			} else {
				// Specific function
				openaiReq.ToolChoice = openai.ToolChoice{
					Type: openai.ToolTypeFunction,
					Function: openai.ToolFunction{
						Name: req.ToolChoice,
					},
				}
			}
		}
	}

	// Apply timeout
	reqCtx := ctx
	if c.config.Timeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, c.config.Timeout)
		defer cancel()
	}

	// Retry logic
	var resp openai.ChatCompletionResponse
	var err error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		resp, err = c.client.CreateChatCompletion(reqCtx, openaiReq)
		if err == nil {
			break
		}

		// Don't retry on context cancellation
		if reqCtx.Err() != nil {
			return nil, err
		}

		// Wait before retry (exponential backoff)
		if attempt < c.config.MaxRetries {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("openai chat completion failed: %w", err)
	}

	// Convert response
	chatResp := &ChatResponse{
		ID:      resp.ID,
		Model:   resp.Model,
		Choices: make([]Choice, len(resp.Choices)),
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	for i, choice := range resp.Choices {
		msg := Message{
			Role:    choice.Message.Role,
			Content: choice.Message.Content,
			Name:    choice.Message.Name,
		}

		// Convert tool calls
		if len(choice.Message.ToolCalls) > 0 {
			msg.ToolCalls = make([]ToolCall, len(choice.Message.ToolCalls))
			for j, tc := range choice.Message.ToolCalls {
				msg.ToolCalls[j] = ToolCall{
					ID:   tc.ID,
					Type: string(tc.Type),
					Function: ToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		chatResp.Choices[i] = Choice{
			Index:        choice.Index,
			Message:      msg,
			FinishReason: string(choice.FinishReason),
		}
	}

	// Cache response
	if c.cache != nil {
		cacheKey, err := GenerateCacheKey(req)
		if err == nil {
			c.cache.Set(cacheKey, chatResp)
		}
	}

	return chatResp, nil
}

// Embed creates embeddings for text
func (c *OpenAIClient) Embed(ctx context.Context, req EmbedRequest) (*EmbedResponse, error) {
	// Check rate limit
	if c.limiter != nil {
		if !c.limiter.Allow() {
			return nil, fmt.Errorf("rate limit exceeded")
		}
	}

	// Set default model if not specified
	if req.Model == "" {
		req.Model = "text-embedding-ada-002"
	}

	// Apply timeout
	reqCtx := ctx
	if c.config.Timeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, c.config.Timeout)
		defer cancel()
	}

	// Retry logic
	var resp openai.EmbeddingResponse
	var err error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		openaiReq := openai.EmbeddingRequest{
			Model: openai.EmbeddingModel(req.Model),
			Input: req.Input,
		}

		resp, err = c.client.CreateEmbeddings(reqCtx, openaiReq)
		if err == nil {
			break
		}

		// Don't retry on context cancellation
		if reqCtx.Err() != nil {
			return nil, err
		}

		// Wait before retry
		if attempt < c.config.MaxRetries {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("openai embedding failed: %w", err)
	}

	// Convert response
	embedResp := &EmbedResponse{
		Model: string(resp.Model),
		Data:  make([]EmbeddingData, len(resp.Data)),
		Usage: EmbedUsage{
			PromptTokens: resp.Usage.PromptTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}

	for i, data := range resp.Data {
		// Convert float32 to float64
		embedding := make([]float64, len(data.Embedding))
		for j, v := range data.Embedding {
			embedding[j] = float64(v)
		}

		embedResp.Data[i] = EmbeddingData{
			Index:     data.Index,
			Embedding: embedding,
			Object:    data.Object,
		}
	}

	return embedResp, nil
}

// ChatSimple sends a simple chat message and returns the response text
func (c *OpenAIClient) ChatSimple(ctx context.Context, prompt string) (string, error) {
	req := ChatRequest{
		Model: c.config.DefaultModel,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 1.0,
		MaxTokens:   1000,
	}

	resp, err := c.Chat(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return resp.Choices[0].Message.Content, nil
}
