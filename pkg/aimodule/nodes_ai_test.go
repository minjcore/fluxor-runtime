package aimodule

import (
	"context"
	"testing"
)

func TestAIChatNodeHandler_WithPrompt(t *testing.T) {
	// Test with simple prompt
	input := &NodeInput{
		Data: map[string]interface{}{
			"query": "Hello, how are you?",
		},
		Config: map[string]interface{}{
			"provider":    "openai",
			"model":       "gpt-3.5-turbo",
			"prompt":      "Answer this question: {{ $.input.query }}",
			"temperature": 0.7,
		},
	}

	ctx := context.Background()

	// This will fail without API key, but we can test the handler setup
	_, err := AIChatNodeHandler(ctx, input)

	// Expect error due to missing API key or invalid client
	if err == nil {
		t.Error("Expected error without API key")
	}
}

func TestAIChatNodeHandler_WithMessages(t *testing.T) {
	input := &NodeInput{
		Data: map[string]interface{}{
			"user_message": "What is AI?",
		},
		Config: map[string]interface{}{
			"provider": "openai",
			"model":    "gpt-3.5-turbo",
			"messages": []interface{}{
				map[string]interface{}{
					"role":    "system",
					"content": "You are a helpful assistant.",
				},
				map[string]interface{}{
					"role":    "user",
					"content": "{{ $.input.user_message }}",
				},
			},
		},
	}

	ctx := context.Background()
	_, err := AIChatNodeHandler(ctx, input)

	// Expect error due to missing API key
	if err == nil {
		t.Error("Expected error without API key")
	}
}

func TestAIChatNodeHandler_DefaultProvider(t *testing.T) {
	input := &NodeInput{
		Data: map[string]interface{}{
			"text": "Hello",
		},
		Config: map[string]interface{}{
			"prompt": "Say hello",
		},
	}

	ctx := context.Background()
	_, err := AIChatNodeHandler(ctx, input)

	// Should default to OpenAI provider
	if err == nil {
		t.Error("Expected error without API key")
	}
}

func TestAIChatNodeHandler_WithTools(t *testing.T) {
	input := &NodeInput{
		Data: map[string]interface{}{
			"query": "What's the weather?",
		},
		Config: map[string]interface{}{
			"provider": "openai",
			"model":    "gpt-3.5-turbo",
			"prompt":   "{{ $.input.query }}",
			"tools": []interface{}{
				map[string]interface{}{
					"type": "function",
					"function": map[string]interface{}{
						"name":        "get_weather",
						"description": "Get weather for location",
						"parameters": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"location": map[string]interface{}{
									"type":        "string",
									"description": "City name",
								},
							},
						},
					},
				},
			},
			"toolChoice": "auto",
		},
	}

	ctx := context.Background()
	_, err := AIChatNodeHandler(ctx, input)

	// Expect error due to missing API key
	if err == nil {
		t.Error("Expected error without API key")
	}
}

func TestAIEmbedNodeHandler_Basic(t *testing.T) {
	input := &NodeInput{
		Data: map[string]interface{}{
			"text": "Hello, world!",
		},
		Config: map[string]interface{}{
			"provider": "openai",
			"model":    "text-embedding-ada-002",
			"input":    "{{ $.input.text }}",
		},
	}

	ctx := context.Background()
	_, err := AIEmbedNodeHandler(ctx, input)

	// Expect error due to missing API key
	if err == nil {
		t.Error("Expected error without API key")
	}
}

func TestAIEmbedNodeHandler_WithArray(t *testing.T) {
	input := &NodeInput{
		Data: map[string]interface{}{},
		Config: map[string]interface{}{
			"provider": "openai",
			"model":    "text-embedding-ada-002",
			"input":    []interface{}{"First text", "Second text"},
		},
	}

	ctx := context.Background()
	_, err := AIEmbedNodeHandler(ctx, input)

	// Expect error due to missing API key
	if err == nil {
		t.Error("Expected error without API key")
	}
}

func TestAIEmbedNodeHandler_FromInputData(t *testing.T) {
	input := &NodeInput{
		Data: map[string]interface{}{
			"texts": []interface{}{"Text 1", "Text 2"},
		},
		Config: map[string]interface{}{
			"provider": "openai",
			"model":    "text-embedding-ada-002",
		},
	}

	ctx := context.Background()
	_, err := AIEmbedNodeHandler(ctx, input)

	// Expect error due to missing API key
	if err == nil {
		t.Error("Expected error without API key")
	}
}

func TestAIEmbedNodeHandler_NoInput(t *testing.T) {
	input := &NodeInput{
		Data: map[string]interface{}{},
		Config: map[string]interface{}{
			"provider": "openai",
			"model":    "text-embedding-ada-002",
		},
	}

	ctx := context.Background()
	_, err := AIEmbedNodeHandler(ctx, input)

	// Should fail because no input provided
	if err == nil {
		t.Error("Expected error when no input provided")
	}
}

func TestProcessTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     interface{}
		expected string
	}{
		{
			name:     "simple placeholder",
			template: "Hello {{ $.input.name }}",
			data: map[string]interface{}{
				"name": "World",
			},
			expected: "Hello World",
		},
		{
			name:     "nested placeholder",
			template: "{{ $.input.user.name }}",
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "John",
				},
			},
			expected: "John",
		},
		{
			name:     "multiple placeholders",
			template: "{{ $.input.first }} and {{ $.input.second }}",
			data: map[string]interface{}{
				"first":  "one",
				"second": "two",
			},
			expected: "one and two",
		},
		{
			name:     "no placeholders",
			template: "Hello world",
			data:     map[string]interface{}{},
			expected: "Hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processTemplate(tt.template, tt.data)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestProcessTemplate_InvalidData(t *testing.T) {
	// Test with non-map data
	result := processTemplate("Hello {{ $.input.name }}", "not a map")
	if result != "Hello {{ $.input.name }}" {
		t.Errorf("Expected template to remain unchanged with invalid data, got %q", result)
	}
}

func TestGetOrCreateClient(t *testing.T) {
	config := map[string]interface{}{
		"provider": "openai",
		"apiKey":   "test-key",
		"model":    "gpt-4",
	}

	client, err := getOrCreateClient(ProviderOpenAI, config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.Provider() != ProviderOpenAI {
		t.Errorf("Expected provider %s, got %s", ProviderOpenAI, client.Provider())
	}
}

func TestGetOrCreateClient_WithRateLimit(t *testing.T) {
	config := map[string]interface{}{
		"provider": "openai",
		"apiKey":   "test-key",
		"rateLimit": map[string]interface{}{
			"requestsPerMinute": 60.0,
			"requestsPerDay":    10000.0,
		},
	}

	client, err := getOrCreateClient(ProviderOpenAI, config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.Provider() != ProviderOpenAI {
		t.Errorf("Expected provider %s, got %s", ProviderOpenAI, client.Provider())
	}
}

func TestGetOrCreateClient_WithCache(t *testing.T) {
	config := map[string]interface{}{
		"provider": "openai",
		"apiKey":   "test-key",
		"cache": map[string]interface{}{
			"enabled": true,
			"ttl":     "5m",
		},
	}

	client, err := getOrCreateClient(ProviderOpenAI, config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.Provider() != ProviderOpenAI {
		t.Errorf("Expected provider %s, got %s", ProviderOpenAI, client.Provider())
	}
}

func TestGetString(t *testing.T) {
	m := map[string]interface{}{
		"stringKey": "value",
		"intKey":    123,
		"nilKey":    nil,
	}

	if getString(m, "stringKey") != "value" {
		t.Error("Expected 'value', got different value")
	}

	if getString(m, "intKey") != "" {
		t.Error("Expected empty string for non-string value")
	}

	if getString(m, "nilKey") != "" {
		t.Error("Expected empty string for nil value")
	}

	if getString(m, "missing") != "" {
		t.Error("Expected empty string for missing key")
	}
}

func TestGetMap(t *testing.T) {
	m := map[string]interface{}{
		"mapKey": map[string]interface{}{
			"nested": "value",
		},
		"stringKey": "not a map",
		"nilKey":    nil,
	}

	result := getMap(m, "mapKey")
	if len(result) == 0 {
		t.Error("Expected non-empty map")
	}

	result = getMap(m, "stringKey")
	if len(result) != 0 {
		t.Error("Expected empty map for non-map value")
	}

	result = getMap(m, "nilKey")
	if len(result) != 0 {
		t.Error("Expected empty map for nil value")
	}

	result = getMap(m, "missing")
	if len(result) != 0 {
		t.Error("Expected empty map for missing key")
	}
}
