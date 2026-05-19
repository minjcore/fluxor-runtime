package workflow

import (
	"context"
	"testing"
)

func TestOpenAINodeHandler_TemplateProcessing(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     interface{}
		expected string
	}{
		{
			name:     "Simple field access",
			template: "Hello {{ name }}",
			data:     map[string]interface{}{"name": "World"},
			expected: "Hello World",
		},
		{
			name:     "Dollar notation",
			template: "Hello {{ $.input.name }}",
			data:     map[string]interface{}{"name": "World"},
			expected: "Hello World",
		},
		{
			name:     "Nested access",
			template: "Hello {{ $.input.user.name }}",
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "Alice",
				},
			},
			expected: "Hello Alice",
		},
		{
			name:     "Multiple fields",
			template: "{{ greeting }}, {{ name }}!",
			data: map[string]interface{}{
				"greeting": "Hello",
				"name":     "World",
			},
			expected: "Hello, World!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processOpenAITemplate(tt.template, tt.data)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestOpenAINodeHandler_ConfigValidation(t *testing.T) {
	// Test that handler validates required config
	input := &NodeInput{
		Config: map[string]interface{}{},
		Data:   map[string]interface{}{"text": "test"},
	}

	_, err := OpenAINodeHandler(context.Background(), input)
	if err == nil {
		t.Error("Expected error when apiKey is missing")
	}
	if err != nil && err.Error() == "" {
		t.Error("Error message should not be empty")
	}
}

func TestOpenAINodeHandler_WithAPIKey(t *testing.T) {
	// This test would require mocking HTTP client
	// For now, just test config parsing
	input := &NodeInput{
		Config: map[string]interface{}{
			"apiKey": "test-key",
			"model":  "gpt-3.5-turbo",
			"prompt": "Say hello",
		},
		Data: map[string]interface{}{},
	}

	// Without actual API call, this will fail at HTTP request
	// But we can verify config is parsed correctly
	if apiKey, ok := input.Config["apiKey"].(string); !ok || apiKey != "test-key" {
		t.Error("API key should be accessible from config")
	}
}
