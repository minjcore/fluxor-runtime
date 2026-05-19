package aimodule

import (
	"testing"
)

func TestParseHandler_ExtractText(t *testing.T) {
	resp := &ChatResponse{
		ID:    "test-id",
		Model: "gpt-3.5-turbo",
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "Hello, world!",
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	handler := DefaultParseHandler()
	result, err := handler.ParseResponse(resp, ProviderOpenAI)
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if text, ok := result["response"].(string); !ok || text != "Hello, world!" {
		t.Errorf("Expected 'Hello, world!', got %v", result["response"])
	}

	if usage, ok := result["_ai_usage"].(Usage); !ok {
		t.Error("Expected usage metadata")
	} else if usage.TotalTokens != 15 {
		t.Errorf("Expected 15 total tokens, got %d", usage.TotalTokens)
	}
}

func TestParseHandler_ExtractJSON(t *testing.T) {
	resp := &ChatResponse{
		ID:    "test-id",
		Model: "gpt-3.5-turbo",
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "```json\n{\"name\": \"John\", \"age\": 30}\n```",
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	handler := DefaultParseHandler()
	handler.ExtractJSON = true
	result, err := handler.ParseResponse(resp, ProviderOpenAI)
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if jsonData, ok := result["response_json"]; !ok {
		t.Error("Expected response_json field")
	} else {
		jsonMap, ok := jsonData.(map[string]interface{})
		if !ok {
			t.Errorf("Expected map, got %T", jsonData)
		} else {
			if name, ok := jsonMap["name"].(string); !ok || name != "John" {
				t.Errorf("Expected name='John', got %v", jsonMap["name"])
			}
		}
	}
}

func TestParseHandler_ToolCalls(t *testing.T) {
	resp := &ChatResponse{
		ID:    "test-id",
		Model: "gpt-4",
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "",
					ToolCalls: []ToolCall{
						{
							ID:   "call_123",
							Type: "function",
							Function: ToolCallFunction{
								Name:      "get_weather",
								Arguments: `{"location": "San Francisco"}`,
							},
						},
					},
				},
				FinishReason: "tool_calls",
			},
		},
		Usage: Usage{
			PromptTokens:     20,
			CompletionTokens: 10,
			TotalTokens:      30,
		},
	}

	handler := DefaultParseHandler()
	result, err := handler.ParseResponse(resp, ProviderOpenAI)
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}

	if hasToolCalls, ok := result["has_tool_calls"].(bool); !ok || !hasToolCalls {
		t.Error("Expected has_tool_calls to be true")
	}

	// Test ParseToolCalls
	parsed, err := ParseToolCalls(resp)
	if err != nil {
		t.Fatalf("ParseToolCalls failed: %v", err)
	}

	if len(parsed) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(parsed))
	}

	if parsed[0].Name != "get_weather" {
		t.Errorf("Expected tool name 'get_weather', got %s", parsed[0].Name)
	}

	if location, ok := parsed[0].Arguments["location"].(string); !ok || location != "San Francisco" {
		t.Errorf("Expected location='San Francisco', got %v", parsed[0].Arguments["location"])
	}
}

func TestParseResponseSimple(t *testing.T) {
	resp := &ChatResponse{
		ID:    "test-id",
		Model: "gpt-3.5-turbo",
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "Simple response",
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	text, err := ParseResponseSimple(resp)
	if err != nil {
		t.Fatalf("ParseResponseSimple failed: %v", err)
	}

	if text != "Simple response" {
		t.Errorf("Expected 'Simple response', got %s", text)
	}
}

func TestParseResponseWithOptions(t *testing.T) {
	resp := &ChatResponse{
		ID:    "test-id",
		Model: "gpt-3.5-turbo",
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "Test response",
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	result, err := ParseResponseWithOptions(
		resp,
		ProviderOpenAI,
		WithResponseField("result"),
		WithIncludeMetadata(false),
		WithIncludeFullResponse(true),
	)
	if err != nil {
		t.Fatalf("ParseResponseWithOptions failed: %v", err)
	}

	if _, ok := result["result"]; !ok {
		t.Error("Expected 'result' field")
	}

	if _, ok := result["_ai_response"]; !ok {
		t.Error("Expected '_ai_response' field")
	}

	if _, ok := result["_ai_usage"]; ok {
		t.Error("Should not include metadata when disabled")
	}
}

func TestParseUsage(t *testing.T) {
	resp := &ChatResponse{
		ID:    "test-id",
		Model: "gpt-4",
		Choices: []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "Test",
				},
			},
		},
		Usage: Usage{
			PromptTokens:     1000,
			CompletionTokens: 500,
			TotalTokens:      1500,
		},
	}

	usage := ParseUsage(resp)
	if usage["total_tokens"] != 1500 {
		t.Errorf("Expected 1500 total tokens, got %v", usage["total_tokens"])
	}

	if cost, ok := usage["estimated_cost"].(float64); !ok || cost <= 0 {
		t.Error("Expected estimated_cost to be a positive number")
	}
}
