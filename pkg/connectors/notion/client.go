package notion

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

type notionClient struct {
	config        Config
	httpClient    *http.Client
	rateLimiter   *rate.Limiter
	pagesImpl     *pagesClient
	databasesImpl *databasesClient
	blocksImpl    *blocksClient
	usersImpl     *usersClient
	searchImpl    *searchClient
}

func NewClient(config Config) (Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout: config.GetTimeout(),
	}

	limiter := rate.NewLimiter(rate.Limit(config.RateLimit), 1)

	client := &notionClient{
		config:      config,
		httpClient:  httpClient,
		rateLimiter: limiter,
	}

	client.pagesImpl = &pagesClient{client: client}
	client.databasesImpl = &databasesClient{client: client}
	client.blocksImpl = &blocksClient{client: client}
	client.usersImpl = &usersClient{client: client}
	client.searchImpl = &searchClient{client: client}

	return client, nil
}

func (c *notionClient) Pages() PagesClient         { return c.pagesImpl }
func (c *notionClient) Databases() DatabasesClient { return c.databasesImpl }
func (c *notionClient) Blocks() BlocksClient       { return c.blocksImpl }
func (c *notionClient) Users() UsersClient         { return c.usersImpl }
func (c *notionClient) Search() SearchClient       { return c.searchImpl }

func (c *notionClient) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	url := c.config.BaseURL + "/v1" + path

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", c.config.Version)

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

		if resp.StatusCode == 429 {
			lastErr = fmt.Errorf("rate limited")
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		var apiError APIResponse
		if err := json.Unmarshal(respBody, &apiError); err == nil && apiError.Code != "" {
			return nil, &NotionError{Code: apiError.Code, Message: apiError.Message, Status: apiError.Status}
		}

		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", c.config.MaxRetries, lastErr)
}

// NotionError represents an error response from Notion API
type NotionError struct {
	Status  int
	Code    string
	Message string
}

func (e *NotionError) Error() string {
	return fmt.Sprintf("notion error (%s): %s", e.Code, e.Message)
}
