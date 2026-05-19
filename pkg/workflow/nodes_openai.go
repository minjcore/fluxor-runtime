package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// OpenAINodeConfig represents configuration for OpenAI node.
type OpenAINodeConfig struct {
	// API configuration
	APIKey  string `json:"apiKey"`  // OpenAI API key (can use env var: $OPENAI_API_KEY)
	BaseURL string `json:"baseURL"` // Base URL (default: https://api.openai.com/v1)
	Model   string `json:"model"`   // Model to use (default: gpt-3.5-turbo)
	Timeout string `json:"timeout"` // Request timeout (default: 60s)

	// Request configuration
	Prompt      interface{}              `json:"prompt"`      // Prompt template or static string
	Messages    []map[string]interface{} `json:"messages"`    // Chat messages (for chat completions)
	Temperature float64                  `json:"temperature"` // Temperature (0-2, default: 1.0)
	MaxTokens   int                      `json:"maxTokens"`   // Max tokens (default: 1000)
	TopP        float64                  `json:"topP"`        // Top P (default: 1.0)
	Stream      bool                     `json:"stream"`      // Stream response (default: false)

	// Response handling
	ResponseField string `json:"responseField"` // Field name for response in output (default: "response")
	ExtractText   bool   `json:"extractText"`   // Extract text from choices[0].message.content (default: true)
}

// OpenAINodeHandler handles OpenAI API request nodes.
func OpenAINodeHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	// Config:
	// - "apiKey": OpenAI API key (or use $OPENAI_API_KEY env var)
	// - "baseURL": Base URL (default: https://api.openai.com/v1)
	// - "model": Model name (default: gpt-3.5-turbo)
	// - "prompt": Prompt template (supports {{ $.input.text }} syntax)
	// - "messages": Chat messages array (for chat completions)
	// - "temperature": 0-2 (default: 1.0)
	// - "maxTokens": Max tokens (default: 1000)
	// - "timeout": Request timeout (default: 60s)
	// - "responseField": Field name for response (default: "response")
	// - "extractText": Extract text from response (default: true)

	// Get API key
	apiKey, _ := input.Config["apiKey"].(string)
	if apiKey == "" {
		// Try environment variable
		apiKey = getEnv("OPENAI_API_KEY", "")
		if apiKey == "" {
			return nil, fmt.Errorf("openai node requires 'apiKey' config or OPENAI_API_KEY env var")
		}
	}

	// Get base URL
	baseURL := "https://api.openai.com/v1"
	if url, ok := input.Config["baseURL"].(string); ok && url != "" {
		baseURL = url
	}

	// Get model
	model := "gpt-3.5-turbo"
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

	// Determine endpoint based on model
	endpoint := "/chat/completions" // Default to chat completions
	if strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "o1-") {
		endpoint = "/chat/completions"
	} else if strings.HasPrefix(model, "text-") {
		endpoint = "/completions"
	}

	// Build request payload
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
		// Text completions or chat with prompt
		requestBody = map[string]interface{}{
			"model": model,
		}

		// Get prompt
		var promptText string
		if prompt, ok := input.Config["prompt"]; ok {
			switch p := prompt.(type) {
			case string:
				promptText = processOpenAITemplate(p, input.Data)
			case map[string]interface{}:
				// If prompt is a map, try to extract text or use as-is
				if text, ok := p["text"].(string); ok {
					promptText = processOpenAITemplate(text, input.Data)
				} else {
					// Use entire prompt map
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
					// Convert entire input to string
					promptText = fmt.Sprintf("%v", input.Data)
				}
			} else {
				promptText = fmt.Sprintf("%v", input.Data)
			}
		}

		if endpoint == "/chat/completions" {
			// Chat format with single user message
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

	if stream, ok := input.Config["stream"].(bool); ok {
		requestBody["stream"] = stream
	}

	// Marshal request body
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	url := baseURL + endpoint
	req, err := http.NewRequestWithContext(reqCtx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Execute request
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
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
					return nil, fmt.Errorf("openai API error: %s", message)
				}
			}
		}
		return nil, fmt.Errorf("openai API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var responseData map[string]interface{}
	if err := json.Unmarshal(respBody, &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text from response
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
		// Extract text from choices[0].message.content or choices[0].text
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

	// Also include full response
	output["_openai_response"] = responseData
	output["_openai_usage"] = responseData["usage"]

	return &NodeOutput{Data: output}, nil
}

// processOpenAITemplate processes template strings with support for:
// - {{ $.input.field }} - Access input data
// - {{ $.field }} - Access root data
// - {{ field }} - Simple field access
func processOpenAITemplate(template string, data interface{}) string {
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

// getEnv gets environment variable value or returns default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
