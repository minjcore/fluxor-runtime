package aimodule

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// NodeInput represents workflow node input (to avoid import cycle)
type NodeInput struct {
	Data        interface{}            `json:"data"`
	Context     interface{}            `json:"context"`
	Config      map[string]interface{} `json:"config"`
	TriggerData interface{}            `json:"triggerData"`
}

// NodeOutput represents workflow node output (to avoid import cycle)
type NodeOutput struct {
	Data      interface{} `json:"data"`
	Error     error       `json:"error,omitempty"`
	NextNodes []string    `json:"nextNodes,omitempty"`
	Stop      bool        `json:"stop,omitempty"`
}

// AIChatNodeHandler handles AI chat completion nodes
func AIChatNodeHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	// Get provider (default: openai)
	provider := ProviderOpenAI
	if p, ok := input.Config["provider"].(string); ok && p != "" {
		provider = Provider(p)
	}

	// Get or create client
	client, err := getOrCreateClient(provider, input.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI client: %w", err)
	}

	// Get model
	model := ""
	if m, ok := input.Config["model"].(string); ok && m != "" {
		model = m
	}

	// Build messages
	var messages []Message

	// Check if messages are provided in config
	if msgs, ok := input.Config["messages"].([]interface{}); ok && len(msgs) > 0 {
		messages = make([]Message, 0, len(msgs))
		for _, msg := range msgs {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				role, _ := msgMap["role"].(string)
				content, _ := msgMap["content"].(string)
				if content != "" {
					// Process template
					content = processTemplate(content, input.Data)
				}
				messages = append(messages, Message{
					Role:    role,
					Content: content,
				})
			}
		}
	} else {
		// Use prompt template
		var promptText string
		if prompt, ok := input.Config["prompt"]; ok {
			switch p := prompt.(type) {
			case string:
				promptText = processTemplate(p, input.Data)
			case map[string]interface{}:
				if text, ok := p["text"].(string); ok {
					promptText = processTemplate(text, input.Data)
				} else {
					promptText = fmt.Sprintf("%v", p)
				}
			default:
				promptText = fmt.Sprintf("%v", prompt)
			}
		} else {
			// Use input data as prompt
			if data, ok := input.Data.(map[string]interface{}); ok {
				if text, ok := data["text"].(string); ok {
					promptText = text
				} else if text, ok := data["prompt"].(string); ok {
					promptText = text
				} else {
					promptText = fmt.Sprintf("%v", input.Data)
				}
			} else {
				promptText = fmt.Sprintf("%v", input.Data)
			}
		}

		messages = []Message{
			{
				Role:    "user",
				Content: promptText,
			},
		}
	}

	// Build chat request
	chatReq := ChatRequest{
		Model:    model,
		Messages: messages,
	}

	// Set temperature
	if temp, ok := input.Config["temperature"].(float64); ok {
		chatReq.Temperature = temp
	} else {
		chatReq.Temperature = 1.0
	}

	// Set max tokens
	if maxTokens, ok := input.Config["maxTokens"].(float64); ok {
		chatReq.MaxTokens = int(maxTokens)
	} else if maxTokens, ok := input.Config["maxTokens"].(int); ok {
		chatReq.MaxTokens = maxTokens
	}

	// Set top P
	if topP, ok := input.Config["topP"].(float64); ok {
		chatReq.TopP = topP
	}

	// Parse tools if provided
	if toolsConfig, ok := input.Config["tools"].([]interface{}); ok {
		tools := make([]Tool, 0, len(toolsConfig))
		for _, toolConfig := range toolsConfig {
			if toolMap, ok := toolConfig.(map[string]interface{}); ok {
				tool := Tool{
					Type: "function",
				}
				if fn, ok := toolMap["function"].(map[string]interface{}); ok {
					tool.Function = FunctionDefinition{
						Name:        getString(fn, "name"),
						Description: getString(fn, "description"),
						Parameters:  getMap(fn, "parameters"),
					}
				}
				tools = append(tools, tool)
			}
		}
		chatReq.Tools = tools
	}

	// Set tool choice
	if toolChoice, ok := input.Config["toolChoice"].(string); ok {
		chatReq.ToolChoice = toolChoice
	}

	// Call chat
	resp, err := client.Chat(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("AI chat failed: %w", err)
	}

	// Build output
	output := make(map[string]interface{})
	if data, ok := input.Data.(map[string]interface{}); ok {
		for k, v := range data {
			output[k] = v
		}
	}

	// Use parse handler to extract response
	handler := DefaultParseHandler()

	// Configure handler from config
	if rf, ok := input.Config["responseField"].(string); ok && rf != "" {
		handler.ResponseField = rf
	}
	if et, ok := input.Config["extractText"].(bool); ok {
		handler.ExtractText = et
	}
	if ej, ok := input.Config["extractJSON"].(bool); ok {
		handler.ExtractJSON = ej
	}
	if im, ok := input.Config["includeMetadata"].(bool); ok {
		handler.IncludeMetadata = im
	}
	if ifr, ok := input.Config["includeFullResponse"].(bool); ok {
		handler.IncludeFullResponse = ifr
	}

	// Parse response
	parsed, err := handler.ParseResponse(resp, client.Provider())
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Merge parsed data into output
	for k, v := range parsed {
		output[k] = v
	}

	return &NodeOutput{Data: output}, nil
}

// AIEmbedNodeHandler handles AI embedding nodes
func AIEmbedNodeHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	// Get provider (default: openai)
	provider := ProviderOpenAI
	if p, ok := input.Config["provider"].(string); ok && p != "" {
		provider = Provider(p)
	}

	// Get or create client
	client, err := getOrCreateClient(provider, input.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI client: %w", err)
	}

	// Get model
	model := "text-embedding-ada-002"
	if m, ok := input.Config["model"].(string); ok && m != "" {
		model = m
	}

	// Get input texts
	var texts []string
	if inputTexts, ok := input.Config["input"].([]interface{}); ok {
		texts = make([]string, 0, len(inputTexts))
		for _, t := range inputTexts {
			if str, ok := t.(string); ok {
				texts = append(texts, str)
			}
		}
	} else if inputText, ok := input.Config["input"].(string); ok {
		texts = []string{inputText}
	} else {
		// Try to get from input data
		if data, ok := input.Data.(map[string]interface{}); ok {
			if text, ok := data["text"].(string); ok {
				texts = []string{text}
			} else if textArray, ok := data["texts"].([]interface{}); ok {
				texts = make([]string, 0, len(textArray))
				for _, t := range textArray {
					if str, ok := t.(string); ok {
						texts = append(texts, str)
					}
				}
			}
		}
	}

	if len(texts) == 0 {
		return nil, fmt.Errorf("no input texts provided for embedding")
	}

	// Build embed request
	embedReq := EmbedRequest{
		Model: model,
		Input: texts,
	}

	// Call embed
	resp, err := client.Embed(ctx, embedReq)
	if err != nil {
		return nil, fmt.Errorf("AI embed failed: %w", err)
	}

	// Build output
	output := make(map[string]interface{})
	if data, ok := input.Data.(map[string]interface{}); ok {
		for k, v := range data {
			output[k] = v
		}
	}

	// Extract embeddings
	embeddings := make([][]float64, len(resp.Data))
	for i, data := range resp.Data {
		embeddings[i] = data.Embedding
	}

	outputField := "embeddings"
	if of, ok := input.Config["outputField"].(string); ok && of != "" {
		outputField = of
	}

	output[outputField] = embeddings
	if len(embeddings) == 1 {
		output["embedding"] = embeddings[0]
	}

	// Include usage and metadata
	output["_ai_usage"] = resp.Usage
	output["_ai_model"] = resp.Model
	output["_ai_provider"] = string(client.Provider())

	return &NodeOutput{Data: output}, nil
}

// Helper functions

func getOrCreateClient(provider Provider, config map[string]interface{}) (LLMClient, error) {
	// Build config from node config
	aimoduleConfig := Config{
		Provider: provider,
	}

	if apiKey, ok := config["apiKey"].(string); ok && apiKey != "" {
		aimoduleConfig.APIKey = apiKey
	}

	if baseURL, ok := config["baseURL"].(string); ok && baseURL != "" {
		aimoduleConfig.BaseURL = baseURL
	}

	if model, ok := config["model"].(string); ok && model != "" {
		aimoduleConfig.DefaultModel = model
	}

	if timeout, ok := config["timeout"].(string); ok {
		if d, err := time.ParseDuration(timeout); err == nil {
			aimoduleConfig.Timeout = d
		}
	}

	// Check for rate limit config
	if rateLimit, ok := config["rateLimit"].(map[string]interface{}); ok {
		aimoduleConfig.RateLimit = &RateLimitConfig{}
		if rpm, ok := rateLimit["requestsPerMinute"].(float64); ok {
			aimoduleConfig.RateLimit.RequestsPerMinute = int(rpm)
		}
		if rpd, ok := rateLimit["requestsPerDay"].(float64); ok {
			aimoduleConfig.RateLimit.RequestsPerDay = int(rpd)
		}
	}

	// Check for cache config
	if cache, ok := config["cache"].(map[string]interface{}); ok {
		aimoduleConfig.Cache = &CacheConfig{Enabled: true}
		if ttl, ok := cache["ttl"].(string); ok {
			if d, err := time.ParseDuration(ttl); err == nil {
				aimoduleConfig.Cache.TTL = d
			}
		}
	}

	return NewClient(aimoduleConfig)
}

func processTemplate(template string, data interface{}) string {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return template
	}

	result := template

	// Support {{ $.input.field }} syntax
	for key, value := range dataMap {
		// {{ $.input.key }}
		placeholder := fmt.Sprintf("{{ $.input.%s }}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))

		// {{ $.key }}
		placeholder2 := fmt.Sprintf("{{ $.%s }}", key)
		result = strings.ReplaceAll(result, placeholder2, fmt.Sprintf("%v", value))

		// {{ key }}
		placeholder3 := fmt.Sprintf("{{ %s }}", key)
		result = strings.ReplaceAll(result, placeholder3, fmt.Sprintf("%v", value))

		// Support nested access {{ $.input.nested.field }}
		if nested, ok := value.(map[string]interface{}); ok {
			for nestedKey, nestedValue := range nested {
				nestedPlaceholder := fmt.Sprintf("{{ $.input.%s.%s }}", key, nestedKey)
				result = strings.ReplaceAll(result, nestedPlaceholder, fmt.Sprintf("%v", nestedValue))

				nestedPlaceholder2 := fmt.Sprintf("{{ $.%s.%s }}", key, nestedKey)
				result = strings.ReplaceAll(result, nestedPlaceholder2, fmt.Sprintf("%v", nestedValue))
			}
		}
	}

	return result
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key]; ok {
		if m2, ok := v.(map[string]interface{}); ok {
			return m2
		}
	}
	return make(map[string]interface{})
}
