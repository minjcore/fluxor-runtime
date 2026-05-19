package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// AINodeHandler handles generic AI API requests (OpenAI-compatible, Cursor, Anthropic, etc.)
func AINodeHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	// Config:
	// - "provider": "openai", "cursor", "anthropic", "custom" (default: "openai")
	// - "apiKey": API key (or use $API_KEY env var)
	// - "baseURL": Base URL (provider-specific defaults)
	// - "model": Model name
	// - "prompt": Prompt template (supports {{ $.input.text }} syntax)
	// - "messages": Chat messages array
	// - "temperature": 0-2 (default: 1.0)
	// - "maxTokens": Max tokens (default: 1000)
	// - "timeout": Request timeout (default: 60s)
	// - "responseField": Field name for response (default: "response")
	// - "extractText": Extract text from response (default: true)

	provider := "openai"
	if p, ok := input.Config["provider"].(string); ok && p != "" {
		provider = strings.ToLower(p)
	}

	// Get API key
	apiKey, _ := input.Config["apiKey"].(string)
	if apiKey == "" {
		// Try provider-specific env vars
		envVar := getProviderEnvVar(provider)
		apiKey = os.Getenv(envVar)
		if apiKey == "" {
			return nil, fmt.Errorf("ai node requires 'apiKey' config or %s env var", envVar)
		}
	}

	// Get base URL based on provider
	baseURL := getProviderBaseURL(provider)
	if url, ok := input.Config["baseURL"].(string); ok && url != "" {
		baseURL = url
	}

	// Get model
	model := getProviderDefaultModel(provider)
	if m, ok := input.Config["model"].(string); ok && m != "" {
		model = m
	}

	// Get timeout
	timeout := 60 * time.Second
	if t, ok := input.Config["timeout"].(string); ok {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}

	// Determine endpoint
	endpoint := getProviderEndpoint(provider, model)

	// Build request payload - provider-specific format
	var requestBody map[string]interface{}

	// Handle Gemini separately due to different API format
	if provider == "gemini" {
		requestBody = buildGeminiRequest(input, model)
	} else {
		requestBody = buildOpenAICompatibleRequest(input, model, provider, endpoint)
	}

	// Marshal request body
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	requestURL := baseURL + endpoint

	// For Gemini, add API key as query parameter (properly URL encoded)
	if provider == "gemini" {
		separator := "?"
		if strings.Contains(requestURL, "?") {
			separator = "&"
		}
		requestURL += separator + "key=" + url.QueryEscape(apiKey)
	}

	req, err := http.NewRequestWithContext(reqCtx, "POST", requestURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers based on provider (Gemini doesn't use Authorization header)
	req.Header.Set("Content-Type", "application/json")
	if provider != "gemini" {
		setProviderHeaders(req, provider, apiKey)
	}

	// Execute request
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ai request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			if errorMsg, ok := errorResp["error"].(map[string]interface{}); ok {
				if message, ok := errorMsg["message"].(string); ok {
					return nil, fmt.Errorf("ai API error: %s", message)
				}
			}
		}
		return nil, fmt.Errorf("ai API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var responseData map[string]interface{}
	if err := json.Unmarshal(respBody, &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text from response (provider-specific)
	extractText := true
	if et, ok := input.Config["extractText"].(bool); ok {
		extractText = et
	}

	responseField := "response"
	if rf, ok := input.Config["responseField"].(string); ok && rf != "" {
		responseField = rf
	}

	output := make(map[string]interface{})
	if data, ok := input.Data.(map[string]interface{}); ok {
		for k, v := range data {
			output[k] = v
		}
	}

	if extractText {
		if provider == "gemini" {
			// Gemini response format: candidates[0].content.parts[0].text
			if candidates, ok := responseData["candidates"].([]interface{}); ok && len(candidates) > 0 {
				if candidate, ok := candidates[0].(map[string]interface{}); ok {
					if content, ok := candidate["content"].(map[string]interface{}); ok {
						if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
							if part, ok := parts[0].(map[string]interface{}); ok {
								if text, ok := part["text"].(string); ok {
									output[responseField] = text
								}
							}
						}
					}
				}
			}
		} else {
			// OpenAI-compatible format: choices[0].message.content or choices[0].text
			if choices, ok := responseData["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					if message, ok := choice["message"].(map[string]interface{}); ok {
						if content, ok := message["content"].(string); ok {
							output[responseField] = content
						}
					} else if text, ok := choice["text"].(string); ok {
						output[responseField] = text
					}
				}
			}
		}
	}

	// Include full response
	output["_ai_response"] = responseData
	if usage, ok := responseData["usage"]; ok {
		output["_ai_usage"] = usage
	}
	output["_ai_provider"] = provider

	return &NodeOutput{Data: output}, nil
}

// buildOpenAICompatibleRequest builds request for OpenAI-compatible providers
func buildOpenAICompatibleRequest(input *NodeInput, model, provider, endpoint string) map[string]interface{} {
	var requestBody map[string]interface{}

	if messages, ok := input.Config["messages"].([]interface{}); ok && len(messages) > 0 {
		// Chat completions format
		requestBody = map[string]interface{}{
			"model": model,
		}

		// Process messages with templating
		processedMessages := make([]map[string]interface{}, 0, len(messages))
		for _, msg := range messages {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				processedMsg := make(map[string]interface{})
				for k, v := range msgMap {
					if str, ok := v.(string); ok {
						processedMsg[k] = processOpenAITemplate(str, input.Data)
					} else {
						processedMsg[k] = v
					}
				}
				processedMessages = append(processedMessages, processedMsg)
			}
		}
		requestBody["messages"] = processedMessages
	} else {
		// Get prompt
		var promptText string
		if prompt, ok := input.Config["prompt"]; ok {
			switch p := prompt.(type) {
			case string:
				promptText = processOpenAITemplate(p, input.Data)
			case map[string]interface{}:
				if text, ok := p["text"].(string); ok {
					promptText = processOpenAITemplate(text, input.Data)
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
				} else if message, ok := data["message"].(string); ok {
					promptText = message
				} else {
					promptText = fmt.Sprintf("%v", input.Data)
				}
			} else {
				promptText = fmt.Sprintf("%v", input.Data)
			}
		}

		requestBody = map[string]interface{}{
			"model": model,
		}

		if endpoint == "/chat/completions" || provider == "cursor" {
			// Chat format
			requestBody["messages"] = []map[string]interface{}{
				{
					"role":    "user",
					"content": promptText,
				},
			}
		} else {
			// Text completion format
			requestBody["prompt"] = promptText
		}
	}

	// Add optional parameters
	if temp, ok := input.Config["temperature"].(float64); ok {
		requestBody["temperature"] = temp
	} else {
		requestBody["temperature"] = 1.0
	}

	if maxTokens, ok := input.Config["maxTokens"].(float64); ok {
		requestBody["max_tokens"] = int(maxTokens)
	} else if maxTokens, ok := input.Config["maxTokens"].(int); ok {
		requestBody["max_tokens"] = maxTokens
	} else {
		requestBody["max_tokens"] = 1000
	}

	if topP, ok := input.Config["topP"].(float64); ok {
		requestBody["top_p"] = topP
	} else {
		requestBody["top_p"] = 1.0
	}

	return requestBody
}

// buildGeminiRequest builds request for Gemini API
func buildGeminiRequest(input *NodeInput, model string) map[string]interface{} {
	requestBody := map[string]interface{}{}

	// Get prompt text
	var promptText string
	if prompt, ok := input.Config["prompt"]; ok {
		switch p := prompt.(type) {
		case string:
			promptText = processOpenAITemplate(p, input.Data)
		case map[string]interface{}:
			if text, ok := p["text"].(string); ok {
				promptText = processOpenAITemplate(text, input.Data)
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
			} else if message, ok := data["message"].(string); ok {
				promptText = message
			} else {
				promptText = fmt.Sprintf("%v", input.Data)
			}
		} else {
			promptText = fmt.Sprintf("%v", input.Data)
		}
	}

	// Gemini uses contents array with parts
	requestBody["contents"] = []map[string]interface{}{
		{
			"parts": []map[string]interface{}{
				{
					"text": promptText,
				},
			},
		},
	}

	// Add generation config (Gemini-specific parameters)
	generationConfig := map[string]interface{}{}

	if temp, ok := input.Config["temperature"].(float64); ok {
		generationConfig["temperature"] = temp
	} else {
		generationConfig["temperature"] = 1.0
	}

	if maxTokens, ok := input.Config["maxTokens"].(float64); ok {
		generationConfig["maxOutputTokens"] = int(maxTokens)
	} else if maxTokens, ok := input.Config["maxTokens"].(int); ok {
		generationConfig["maxOutputTokens"] = maxTokens
	} else {
		generationConfig["maxOutputTokens"] = 1000
	}

	if topP, ok := input.Config["topP"].(float64); ok {
		generationConfig["topP"] = topP
	} else {
		generationConfig["topP"] = 1.0
	}

	requestBody["generationConfig"] = generationConfig

	return requestBody
}

// getProviderBaseURL returns the base URL for a provider
func getProviderBaseURL(provider string) string {
	switch provider {
	case "openai":
		return "https://api.openai.com/v1"
	case "cursor":
		// Cursor may use OpenAI-compatible API or custom endpoint
		return "https://api.openai.com/v1" // Default, can be overridden
	case "anthropic":
		return "https://api.anthropic.com/v1"
	case "gemini":
		return "https://generativelanguage.googleapis.com/v1"
	case "grok":
		return "https://api.x.ai/v1"
	case "custom":
		return "" // Must be provided in config
	default:
		return "https://api.openai.com/v1"
	}
}

// getProviderEndpoint returns the endpoint for a provider and model
func getProviderEndpoint(provider, model string) string {
	switch provider {
	case "openai", "cursor", "grok":
		if strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "o1-") || strings.HasPrefix(model, "grok-") {
			return "/chat/completions"
		}
		return "/completions"
	case "anthropic":
		return "/messages"
	case "gemini":
		// Gemini uses model name in the endpoint path
		return fmt.Sprintf("/models/%s:generateContent", model)
	default:
		return "/chat/completions"
	}
}

// getProviderDefaultModel returns the default model for a provider
func getProviderDefaultModel(provider string) string {
	switch provider {
	case "openai":
		return "gpt-3.5-turbo"
	case "cursor":
		return "gpt-4" // Cursor typically uses GPT-4
	case "anthropic":
		return "claude-3-sonnet-20240229"
	case "gemini":
		return "gemini-1.5-flash"
	case "grok":
		return "grok-beta"
	default:
		return "gpt-3.5-turbo"
	}
}

// getProviderEnvVar returns the environment variable name for a provider's API key
func getProviderEnvVar(provider string) string {
	switch provider {
	case "openai":
		return "OPENAI_API_KEY"
	case "cursor":
		return "CURSOR_API_KEY" // or OPENAI_API_KEY if using OpenAI-compatible
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "gemini":
		return "GEMINI_API_KEY"
	case "grok":
		return "GROK_API_KEY"
	default:
		return "AI_API_KEY"
	}
}

// setProviderHeaders sets provider-specific headers
func setProviderHeaders(req *http.Request, provider, apiKey string) {
	switch provider {
	case "openai", "cursor", "grok":
		req.Header.Set("Authorization", "Bearer "+apiKey)
	case "anthropic":
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	default:
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
}
