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
	"time"
)

// ZaloZNSNodeHandler handles sending Zalo ZNS messages via Zalo OpenAPI.
//
// Config:
// - "accessToken": ZNS access token (or env ZALO_ZNS_ACCESS_TOKEN)
// - "baseURL": base URL (default env ZALO_ZNS_BASE_URL or https://business.openapi.zalo.me)
// - "endpoint": API path (default: /message/template)
// - "payload": object payload (default: input.Data)
// - "timeout": request timeout (default: 30s)
// - "dryRun": if true, do not send; return built request instead
func ZaloZNSNodeHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	accessToken, _ := input.Config["accessToken"].(string)
	if accessToken == "" {
		accessToken = os.Getenv("ZALO_ZNS_ACCESS_TOKEN")
	}
	if accessToken == "" {
		return nil, fmt.Errorf("zalo zns node requires 'accessToken' config or ZALO_ZNS_ACCESS_TOKEN env var")
	}

	baseURL := "https://business.openapi.zalo.me"
	if b, ok := input.Config["baseURL"].(string); ok && b != "" {
		baseURL = b
	} else if env := os.Getenv("ZALO_ZNS_BASE_URL"); env != "" {
		baseURL = env
	}

	endpoint := "/message/template"
	if ep, ok := input.Config["endpoint"].(string); ok && ep != "" {
		endpoint = ep
	}

	timeout := 30 * time.Second
	if t, ok := input.Config["timeout"].(string); ok && t != "" {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}

	dryRun, _ := input.Config["dryRun"].(bool)

	// Prepare payload (apply simple {{field}} templating when it's a map)
	var payload interface{} = input.Data
	if p := input.Config["payload"]; p != nil {
		if pm, ok := p.(map[string]interface{}); ok {
			payload = processTemplateMap(pm, input.Data)
		} else if ps, ok := p.(string); ok {
			payload = processTemplate(ps, input.Data)
		} else {
			payload = p
		}
	} else {
		if dm, ok := input.Data.(map[string]interface{}); ok {
			payload = processTemplateMap(dm, input.Data)
		}
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal zalo zns payload: %w", err)
	}

	// Build request URL with access_token as query parameter (Zalo OpenAPI style).
	rawURL := baseURL + endpoint
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid zalo zns url: %w", err)
	}
	q := parsed.Query()
	q.Set("access_token", accessToken)
	parsed.RawQuery = q.Encode()

	if dryRun {
		return &NodeOutput{
			Data: map[string]interface{}{
				"request": map[string]interface{}{
					"method": "POST",
					"url":    parsed.String(),
					"body":   payload,
				},
				"_input": input.Data,
			},
		}, nil
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, parsed.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create zalo zns request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zalo zns request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read zalo zns response: %w", err)
	}

	var responseData interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &responseData); err != nil {
			responseData = string(respBody)
		}
	}

	out := map[string]interface{}{
		"statusCode": resp.StatusCode,
		"body":       responseData,
		"_input":     input.Data,
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Return as error but still surface body for debugging.
		return &NodeOutput{Data: out}, fmt.Errorf("zalo zns API error: status %d", resp.StatusCode)
	}

	return &NodeOutput{Data: out}, nil
}

