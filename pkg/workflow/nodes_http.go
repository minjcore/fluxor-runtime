package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func init() {
	// Register HTTP node handler
}

// HTTPNodeHandler handles HTTP request nodes.
func HTTPNodeHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	// Config:
	// - "url": request URL (required)
	// - "method": HTTP method (default: GET)
	// - "headers": map of headers
	// - "body": request body
	// - "timeout": request timeout (default: 30s)
	// - "responseType": "json" (default), "text", "binary"

	url, ok := input.Config["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("http node requires 'url' config")
	}

	// Process URL templates
	url = processTemplate(url, input.Data)

	method := "GET"
	if m, ok := input.Config["method"].(string); ok {
		method = strings.ToUpper(m)
	}

	timeout := 30 * time.Second
	if t, ok := input.Config["timeout"].(string); ok {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}

	// Prepare body
	var bodyReader io.Reader
	if body := input.Config["body"]; body != nil {
		switch b := body.(type) {
		case string:
			bodyReader = strings.NewReader(processTemplate(b, input.Data))
		case map[string]interface{}:
			processedBody := processTemplateMap(b, input.Data)
			jsonBody, err := json.Marshal(processedBody)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonBody)
		}
	} else if method == "POST" || method == "PUT" || method == "PATCH" {
		// Use input data as body
		if input.Data != nil {
			jsonBody, err := json.Marshal(input.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal input data: %w", err)
			}
			bodyReader = bytes.NewReader(jsonBody)
		}
	}

	// Create request
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if headers, ok := input.Config["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			req.Header.Set(k, processTemplate(fmt.Sprintf("%v", v), input.Data))
		}
	}

	// Execute request
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response based on type
	responseType := "json"
	if rt, ok := input.Config["responseType"].(string); ok {
		responseType = rt
	}

	var responseData interface{}
	switch responseType {
	case "json":
		if len(respBody) > 0 {
			if err := json.Unmarshal(respBody, &responseData); err != nil {
				// If JSON parsing fails, return as text
				responseData = string(respBody)
			}
		}
	case "text":
		responseData = string(respBody)
	case "binary":
		responseData = respBody
	}

	return &NodeOutput{
		Data: map[string]interface{}{
			"statusCode": resp.StatusCode,
			"headers":    headerToMap(resp.Header),
			"body":       responseData,
			"_input":     input.Data, // Preserve input for chaining
		},
	}, nil
}

// processTemplate replaces {{field}} placeholders with values from data.
func processTemplate(template string, data interface{}) string {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return template
	}

	result := template
	for key, value := range dataMap {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))

		// Also support dot notation for nested access
		if nested, ok := value.(map[string]interface{}); ok {
			for nestedKey, nestedValue := range nested {
				nestedPlaceholder := fmt.Sprintf("{{%s.%s}}", key, nestedKey)
				result = strings.ReplaceAll(result, nestedPlaceholder, fmt.Sprintf("%v", nestedValue))
			}
		}
	}

	return result
}

func processTemplateMap(m map[string]interface{}, data interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		switch val := v.(type) {
		case string:
			result[k] = processTemplate(val, data)
		case map[string]interface{}:
			result[k] = processTemplateMap(val, data)
		default:
			result[k] = v
		}
	}
	return result
}

func headerToMap(h http.Header) map[string]string {
	result := make(map[string]string)
	for k, v := range h {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result
}
