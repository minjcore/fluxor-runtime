package airtable

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

var (
	baseURL = "https://api.airtable.com/v0"
)

// airtableClient implements the Client interface
type airtableClient struct {
	config      Config
	httpClient  *http.Client
	rateLimiter *rate.Limiter
	tablesImpl  *tablesClient
	recordsImpl *recordsClient
}

// NewClient creates a new Airtable client with the given configuration
// Fail-fast: Returns error if configuration is invalid
func NewClient(config Config) (Client, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.GetTimeout(),
	}

	// Create rate limiter (requests per second)
	limiter := rate.NewLimiter(rate.Limit(config.RateLimit), 1)

	client := &airtableClient{
		config:      config,
		httpClient:  httpClient,
		rateLimiter: limiter,
	}

	// Initialize service clients
	client.tablesImpl = &tablesClient{client: client}
	client.recordsImpl = &recordsClient{client: client}

	return client, nil
}

// Tables returns the TablesClient
func (c *airtableClient) Tables() TablesClient {
	return c.tablesImpl
}

// Records returns the RecordsClient
func (c *airtableClient) Records() RecordsClient {
	return c.recordsImpl
}

// doRequest performs an HTTP request with rate limiting and retries
func (c *airtableClient) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Prepare request body
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	// Build URL
	url := baseURL + path

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	// Perform request with retries
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
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

		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		// Check status code
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, nil
		}

		// Handle error responses
		var apiError AirtableError
		if err := json.Unmarshal(respBody, &apiError); err == nil && apiError.ErrorInfo.Type != "" {
			// Retry on rate limit or server errors
			if resp.StatusCode == 429 || resp.StatusCode >= 500 {
				lastErr = &apiError
				continue
			}
			// Don't retry on client errors
			return nil, &apiError
		}

		// Fallback error
		lastErr = fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
		if resp.StatusCode >= 500 {
			continue
		}
		return nil, lastErr
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", c.config.MaxRetries, lastErr)
}

// AirtableError represents an error response from the Airtable API
type AirtableError struct {
	ErrorInfo struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (e *AirtableError) Error() string {
	return fmt.Sprintf("airtable error (%s): %s", e.ErrorInfo.Type, e.ErrorInfo.Message)
}
