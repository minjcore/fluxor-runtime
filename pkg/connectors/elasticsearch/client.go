package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// elasticsearchClient implements the Client interface
type elasticsearchClient struct {
	config        Config
	httpClient    *http.Client
	rateLimiter   *rate.Limiter
	indicesImpl   *indicesClient
	documentsImpl *documentsClient
	searchImpl    *searchClient
}

// NewClient creates a new Elasticsearch client
func NewClient(config Config) (Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout: config.GetTimeout(),
	}

	// Rate limiter (default: 100 requests per second)
	rateLimit := 100.0
	limiter := rate.NewLimiter(rate.Limit(rateLimit), 1)

	client := &elasticsearchClient{
		config:      config,
		httpClient:  httpClient,
		rateLimiter: limiter,
	}

	// Initialize service clients
	client.indicesImpl = &indicesClient{client: client}
	client.documentsImpl = &documentsClient{client: client}
	client.searchImpl = &searchClient{client: client}

	return client, nil
}

func (c *elasticsearchClient) Indices() IndicesClient {
	return c.indicesImpl
}

func (c *elasticsearchClient) Documents() DocumentsClient {
	return c.documentsImpl
}

func (c *elasticsearchClient) Search() SearchClient {
	return c.searchImpl
}

// doRequest performs an HTTP request to Elasticsearch
func (c *elasticsearchClient) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Build URL (use first address)
	baseURL := c.config.Addresses[0]
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	url := baseURL + strings.TrimPrefix(path, "/")

	// Prepare request body
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "ApiKey "+c.config.APIKey)
	} else if c.config.Username != "" && c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	// Perform request with retries
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, nil
		}

		// Handle specific status codes
		if resp.StatusCode == 404 {
			// Not found - return error with context
			var esError map[string]interface{}
			if err := json.Unmarshal(respBody, &esError); err == nil {
				return nil, &ElasticsearchError{
					Status:  resp.StatusCode,
					Type:    getString(esError, "error", "type"),
					Reason:  getString(esError, "error", "reason"),
					Message: getString(esError, "error", "root_cause", "0", "reason"),
				}
			}
			return nil, fmt.Errorf("not found: %s", string(respBody))
		}

		if resp.StatusCode == 429 {
			lastErr = fmt.Errorf("rate limited")
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		// Parse error response
		var esError map[string]interface{}
		if err := json.Unmarshal(respBody, &esError); err == nil {
			return nil, &ElasticsearchError{
				Status:  resp.StatusCode,
				Type:    getString(esError, "error", "type"),
				Reason:  getString(esError, "error", "reason"),
				Message: getString(esError, "error", "root_cause", "0", "reason"),
			}
		}

		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", c.config.MaxRetries, lastErr)
}

// doBulkRequest performs a bulk request with NDJSON format
func (c *elasticsearchClient) doBulkRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Build URL (use first address)
	baseURL := c.config.Addresses[0]
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	url := baseURL + strings.TrimPrefix(path, "/")

	// Create request with raw body
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-ndjson")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "ApiKey "+c.config.APIKey)
	} else if c.config.Username != "" && c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	// Perform request with retries
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			// Recreate request for retry
			req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
			if err != nil {
				lastErr = fmt.Errorf("failed to create request: %w", err)
				continue
			}
			req.Header.Set("Content-Type", "application/x-ndjson")
			if c.config.APIKey != "" {
				req.Header.Set("Authorization", "ApiKey "+c.config.APIKey)
			} else if c.config.Username != "" && c.config.Password != "" {
				req.SetBasicAuth(c.config.Username, c.config.Password)
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, nil
		}

		if resp.StatusCode == 429 {
			lastErr = fmt.Errorf("rate limited")
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		// Parse error response
		var esError map[string]interface{}
		if err := json.Unmarshal(respBody, &esError); err == nil {
			return nil, &ElasticsearchError{
				Status:  resp.StatusCode,
				Type:    getString(esError, "error", "type"),
				Reason:  getString(esError, "error", "reason"),
				Message: getString(esError, "error", "root_cause", "0", "reason"),
			}
		}

		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", c.config.MaxRetries, lastErr)
}

// getString safely gets a string value from nested map
func getString(m map[string]interface{}, keys ...string) string {
	current := interface{}(m)
	for _, key := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			if val, exists := m[key]; exists {
				current = val
			} else {
				return ""
			}
		} else if arr, ok := current.([]interface{}); ok && len(arr) > 0 {
			if m, ok := arr[0].(map[string]interface{}); ok {
				if val, exists := m[key]; exists {
					current = val
				} else {
					return ""
				}
			} else {
				return ""
			}
		} else {
			return ""
		}
	}
	if str, ok := current.(string); ok {
		return str
	}
	return ""
}

// ElasticsearchError represents an error response from Elasticsearch
type ElasticsearchError struct {
	Status  int
	Type    string
	Reason  string
	Message string
}

func (e *ElasticsearchError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("elasticsearch error (%s): %s", e.Type, e.Message)
	}
	if e.Reason != "" {
		return fmt.Sprintf("elasticsearch error (%s): %s", e.Type, e.Reason)
	}
	return fmt.Sprintf("elasticsearch error: status %d", e.Status)
}
