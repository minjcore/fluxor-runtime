package workflow

import (
	"context"
	"testing"
)

func TestAINodeHandler_ProviderConfig(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		expected string
	}{
		{"OpenAI default", "openai", "https://api.openai.com/v1"},
		{"Cursor provider", "cursor", "https://api.openai.com/v1"},
		{"Anthropic provider", "anthropic", "https://api.anthropic.com/v1"},
		{"Gemini provider", "gemini", "https://generativelanguage.googleapis.com/v1"},
		{"Grok provider", "grok", "https://api.x.ai/v1"},
		{"Unknown provider", "unknown", "https://api.openai.com/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseURL := getProviderBaseURL(tt.provider)
			if baseURL != tt.expected {
				t.Errorf("Expected baseURL %q, got %q", tt.expected, baseURL)
			}
		})
	}
}

func TestAINodeHandler_DefaultModels(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		expected string
	}{
		{"OpenAI default model", "openai", "gpt-3.5-turbo"},
		{"Cursor default model", "cursor", "gpt-4"},
		{"Anthropic default model", "anthropic", "claude-3-sonnet-20240229"},
		{"Gemini default model", "gemini", "gemini-1.5-flash"},
		{"Grok default model", "grok", "grok-beta"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := getProviderDefaultModel(tt.provider)
			if model != tt.expected {
				t.Errorf("Expected model %q, got %q", tt.expected, model)
			}
		})
	}
}

func TestAINodeHandler_EnvVars(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		expected string
	}{
		{"OpenAI env var", "openai", "OPENAI_API_KEY"},
		{"Cursor env var", "cursor", "CURSOR_API_KEY"},
		{"Anthropic env var", "anthropic", "ANTHROPIC_API_KEY"},
		{"Gemini env var", "gemini", "GEMINI_API_KEY"},
		{"Grok env var", "grok", "GROK_API_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envVar := getProviderEnvVar(tt.provider)
			if envVar != tt.expected {
				t.Errorf("Expected env var %q, got %q", tt.expected, envVar)
			}
		})
	}
}

func TestAINodeHandler_ConfigValidation(t *testing.T) {
	// Test that handler validates required config
	input := &NodeInput{
		Config: map[string]interface{}{
			"provider": "cursor",
		},
		Data: map[string]interface{}{"prompt": "test"},
	}

	_, err := AINodeHandler(context.Background(), input)
	if err == nil {
		t.Error("Expected error when apiKey is missing")
	}
}

func TestAINodeHandler_EndpointSelection(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
		expected string
	}{
		{"OpenAI chat model", "openai", "gpt-4", "/chat/completions"},
		{"OpenAI text model", "openai", "text-davinci-003", "/completions"},
		{"Cursor chat model", "cursor", "gpt-4", "/chat/completions"},
		{"Anthropic messages", "anthropic", "claude-3", "/messages"},
		{"Gemini generateContent", "gemini", "gemini-1.5-flash", "/models/gemini-1.5-flash:generateContent"},
		{"Gemini pro model", "gemini", "gemini-pro", "/models/gemini-pro:generateContent"},
		{"Grok chat model", "grok", "grok-beta", "/chat/completions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := getProviderEndpoint(tt.provider, tt.model)
			if endpoint != tt.expected {
				t.Errorf("Expected endpoint %q, got %q", tt.expected, endpoint)
			}
		})
	}
}

func TestBuildGeminiRequest(t *testing.T) {
	tests := []struct {
		name     string
		input    *NodeInput
		model    string
		validate func(t *testing.T, requestBody map[string]interface{})
	}{
		{
			name: "Simple prompt from config",
			input: &NodeInput{
				Config: map[string]interface{}{
					"prompt": "Hello, how are you?",
				},
				Data: map[string]interface{}{},
			},
			model: "gemini-1.5-flash",
			validate: func(t *testing.T, requestBody map[string]interface{}) {
				contents, ok := requestBody["contents"].([]map[string]interface{})
				if !ok || len(contents) == 0 {
					t.Fatal("Expected contents array")
				}
				content := contents[0]
				parts, ok := content["parts"].([]map[string]interface{})
				if !ok || len(parts) == 0 {
					t.Fatal("Expected parts array")
				}
				part := parts[0]
				text, ok := part["text"].(string)
				if !ok || text != "Hello, how are you?" {
					t.Errorf("Expected text 'Hello, how are you?', got %q", text)
				}

				genConfig, ok := requestBody["generationConfig"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected generationConfig")
				}
				if temp, ok := genConfig["temperature"]; !ok || temp.(float64) != 1.0 {
					t.Errorf("Expected default temperature 1.0, got %v", temp)
				}
				if maxTokens, ok := genConfig["maxOutputTokens"]; !ok || maxTokens.(int) != 1000 {
					t.Errorf("Expected default maxOutputTokens 1000, got %v", maxTokens)
				}
			},
		},
		{
			name: "Prompt from input data message field",
			input: &NodeInput{
				Config: map[string]interface{}{},
				Data: map[string]interface{}{
					"message": "Test message",
				},
			},
			model: "gemini-pro",
			validate: func(t *testing.T, requestBody map[string]interface{}) {
				contents, ok := requestBody["contents"].([]map[string]interface{})
				if !ok || len(contents) == 0 {
					t.Fatal("Expected contents array")
				}
				content := contents[0]
				parts, ok := content["parts"].([]map[string]interface{})
				if !ok || len(parts) == 0 {
					t.Fatal("Expected parts array")
				}
				part := parts[0]
				text, ok := part["text"].(string)
				if !ok || text != "Test message" {
					t.Errorf("Expected text 'Test message', got %q", text)
				}
			},
		},
		{
			name: "Custom temperature and maxTokens",
			input: &NodeInput{
				Config: map[string]interface{}{
					"prompt":      "Custom prompt",
					"temperature": 0.7,
					"maxTokens":   500,
				},
				Data: map[string]interface{}{},
			},
			model: "gemini-1.5-flash",
			validate: func(t *testing.T, requestBody map[string]interface{}) {
				genConfig := requestBody["generationConfig"].(map[string]interface{})
				if temp := genConfig["temperature"].(float64); temp != 0.7 {
					t.Errorf("Expected temperature 0.7, got %v", temp)
				}
				if maxTokens := genConfig["maxOutputTokens"].(int); maxTokens != 500 {
					t.Errorf("Expected maxOutputTokens 500, got %v", maxTokens)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody := buildGeminiRequest(tt.input, tt.model)
			tt.validate(t, requestBody)
		})
	}
}
