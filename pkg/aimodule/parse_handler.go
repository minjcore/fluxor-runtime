package aimodule

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseHandler handles parsing of model responses with various extraction strategies
type ParseHandler struct {
	// ExtractText extracts plain text from response (default: true)
	ExtractText bool
	// ExtractJSON extracts and parses JSON from response
	ExtractJSON bool
	// ExtractStructured extracts structured data based on schema
	ExtractStructured bool
	// ResponseField is the field name to store the parsed result (default: "response")
	ResponseField string
	// IncludeMetadata includes usage, model, and provider info
	IncludeMetadata bool
	// IncludeFullResponse includes the full response object
	IncludeFullResponse bool
	// CustomExtractor is a custom function to extract data
	CustomExtractor func(*ChatResponse) (interface{}, error)
}

// DefaultParseHandler returns a parse handler with sensible defaults
func DefaultParseHandler() *ParseHandler {
	return &ParseHandler{
		ExtractText:         true,
		ExtractJSON:         false,
		ExtractStructured:   false,
		ResponseField:       "response",
		IncludeMetadata:     true,
		IncludeFullResponse: false,
	}
}

// ParseResponse parses a ChatResponse and returns a map with extracted data
func (h *ParseHandler) ParseResponse(resp *ChatResponse, provider Provider) (map[string]interface{}, error) {
	if resp == nil {
		return nil, fmt.Errorf("response cannot be nil")
	}

	result := make(map[string]interface{})

	// Extract text content
	if h.ExtractText {
		text, err := h.extractText(resp)
		if err != nil {
			return nil, fmt.Errorf("failed to extract text: %w", err)
		}
		result[h.ResponseField] = text
	}

	// Extract JSON if requested
	if h.ExtractJSON {
		jsonData, err := h.extractJSON(resp)
		if err == nil && jsonData != nil {
			result[h.ResponseField+"_json"] = jsonData
		} else {
			// If JSON extraction fails, don't fail the whole parse
			// Just log that JSON extraction was attempted but failed
			result[h.ResponseField+"_json_error"] = err.Error()
		}
	}

	// Use custom extractor if provided
	if h.CustomExtractor != nil {
		customData, err := h.CustomExtractor(resp)
		if err == nil && customData != nil {
			result[h.ResponseField+"_custom"] = customData
		}
	}

	// Include metadata
	if h.IncludeMetadata {
		result["_ai_usage"] = resp.Usage
		result["_ai_model"] = resp.Model
		result["_ai_provider"] = string(provider)
		result["_ai_id"] = resp.ID
	}

	// Include full response
	if h.IncludeFullResponse {
		result["_ai_response"] = resp
	}

	// Include tool calls if present
	if len(resp.Choices) > 0 && len(resp.Choices[0].Message.ToolCalls) > 0 {
		result["tool_calls"] = resp.Choices[0].Message.ToolCalls
		result["has_tool_calls"] = true
	}

	return result, nil
}

// extractText extracts plain text from the response
func (h *ParseHandler) extractText(resp *ChatResponse) (string, error) {
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	choice := resp.Choices[0]
	content := choice.Message.Content

	// If content is empty, try to extract from tool calls
	if content == "" && len(choice.Message.ToolCalls) > 0 {
		// Return a summary of tool calls
		toolNames := make([]string, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			toolNames[i] = tc.Function.Name
		}
		return fmt.Sprintf("Tool calls: %s", strings.Join(toolNames, ", ")), nil
	}

	return content, nil
}

// extractJSON attempts to extract and parse JSON from the response
func (h *ParseHandler) extractJSON(resp *ChatResponse) (interface{}, error) {
	text, err := h.extractText(resp)
	if err != nil {
		return nil, err
	}

	// Try to find JSON in the text (could be wrapped in markdown code blocks)
	jsonText := text

	// Remove markdown code blocks if present
	if strings.Contains(jsonText, "```json") {
		start := strings.Index(jsonText, "```json")
		remaining := jsonText[start+7:]
		end := strings.Index(remaining, "```")
		if end > 0 {
			jsonText = remaining[:end]
		} else {
			// No closing ```, try to extract anyway
			jsonText = remaining
		}
	} else if strings.Contains(jsonText, "```") {
		start := strings.Index(jsonText, "```")
		remaining := jsonText[start+3:]
		end := strings.Index(remaining, "```")
		if end > 0 {
			jsonText = remaining[:end]
		} else {
			// No closing ```, try to extract anyway
			jsonText = remaining
		}
	}

	// Trim whitespace
	jsonText = strings.TrimSpace(jsonText)

	// If empty after trimming, return error
	if jsonText == "" {
		return nil, fmt.Errorf("no JSON content found")
	}

	// Try to parse as JSON
	var jsonData interface{}
	if err := json.Unmarshal([]byte(jsonText), &jsonData); err != nil {
		return nil, fmt.Errorf("not valid JSON: %w", err)
	}

	return jsonData, nil
}

// ParseResponseSimple is a convenience function that parses a response and returns just the text
func ParseResponseSimple(resp *ChatResponse) (string, error) {
	handler := DefaultParseHandler()
	result, err := handler.ParseResponse(resp, ProviderOpenAI)
	if err != nil {
		return "", err
	}

	if text, ok := result["response"].(string); ok {
		return text, nil
	}

	return "", fmt.Errorf("no text content in response")
}

// ParseResponseWithOptions parses a response with custom options
func ParseResponseWithOptions(resp *ChatResponse, provider Provider, options ...ParseOption) (map[string]interface{}, error) {
	handler := DefaultParseHandler()

	// Apply options
	for _, opt := range options {
		opt(handler)
	}

	return handler.ParseResponse(resp, provider)
}

// ParseOption is a function that modifies a ParseHandler
type ParseOption func(*ParseHandler)

// WithExtractText sets whether to extract text
func WithExtractText(extract bool) ParseOption {
	return func(h *ParseHandler) {
		h.ExtractText = extract
	}
}

// WithExtractJSON sets whether to extract JSON
func WithExtractJSON(extract bool) ParseOption {
	return func(h *ParseHandler) {
		h.ExtractJSON = extract
	}
}

// WithResponseField sets the response field name
func WithResponseField(field string) ParseOption {
	return func(h *ParseHandler) {
		h.ResponseField = field
	}
}

// WithIncludeMetadata sets whether to include metadata
func WithIncludeMetadata(include bool) ParseOption {
	return func(h *ParseHandler) {
		h.IncludeMetadata = include
	}
}

// WithIncludeFullResponse sets whether to include full response
func WithIncludeFullResponse(include bool) ParseOption {
	return func(h *ParseHandler) {
		h.IncludeFullResponse = include
	}
}

// WithCustomExtractor sets a custom extractor function
func WithCustomExtractor(extractor func(*ChatResponse) (interface{}, error)) ParseOption {
	return func(h *ParseHandler) {
		h.CustomExtractor = extractor
	}
}

// ParseToolCalls extracts and parses tool calls from the response
func ParseToolCalls(resp *ChatResponse) ([]ParsedToolCall, error) {
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := resp.Choices[0]
	if len(choice.Message.ToolCalls) == 0 {
		return nil, fmt.Errorf("no tool calls in response")
	}

	parsed := make([]ParsedToolCall, len(choice.Message.ToolCalls))
	for i, tc := range choice.Message.ToolCalls {
		// Parse arguments JSON
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			// If not valid JSON, store as string
			args = map[string]interface{}{
				"_raw": tc.Function.Arguments,
			}
		}

		parsed[i] = ParsedToolCall{
			ID:       tc.ID,
			Name:     tc.Function.Name,
			Arguments: args,
		}
	}

	return parsed, nil
}

// ParsedToolCall represents a parsed tool call with extracted arguments
type ParsedToolCall struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ParseUsage extracts usage information in a structured format
func ParseUsage(resp *ChatResponse) map[string]interface{} {
	return map[string]interface{}{
		"prompt_tokens":     resp.Usage.PromptTokens,
		"completion_tokens": resp.Usage.CompletionTokens,
		"total_tokens":      resp.Usage.TotalTokens,
		"estimated_cost":   estimateCost(resp.Model, resp.Usage),
	}
}

// estimateCost provides a rough cost estimate (in USD) based on model and usage
func estimateCost(model string, usage Usage) float64 {
	// Rough pricing estimates (as of 2024, may vary)
	var promptCost, completionCost float64

	if strings.Contains(model, "gpt-4") {
		promptCost = 0.03 / 1000    // $0.03 per 1K tokens
		completionCost = 0.06 / 1000 // $0.06 per 1K tokens
	} else if strings.Contains(model, "gpt-3.5") {
		promptCost = 0.0015 / 1000  // $0.0015 per 1K tokens
		completionCost = 0.002 / 1000 // $0.002 per 1K tokens
	} else if strings.Contains(model, "gpt-4o") {
		promptCost = 0.005 / 1000   // $0.005 per 1K tokens
		completionCost = 0.015 / 1000 // $0.015 per 1K tokens
	} else {
		// Default estimate
		promptCost = 0.001 / 1000
		completionCost = 0.002 / 1000
	}

	return (float64(usage.PromptTokens)*promptCost + float64(usage.CompletionTokens)*completionCost)
}
