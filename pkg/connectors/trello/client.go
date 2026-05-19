package trello

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/time/rate"
)

type trelloClient struct {
	config      Config
	httpClient  *http.Client
	rateLimiter *rate.Limiter
	boardsImpl  *boardsClient
	listsImpl   *listsClient
	cardsImpl   *cardsClient
	membersImpl *membersClient
}

func NewClient(config Config) (Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout: config.GetTimeout(),
	}

	// Rate limit: 100 requests per 10 seconds = 10 per second
	limiter := rate.NewLimiter(rate.Limit(float64(config.RateLimit)/10.0), 10)

	client := &trelloClient{
		config:      config,
		httpClient:  httpClient,
		rateLimiter: limiter,
	}

	client.boardsImpl = &boardsClient{client: client}
	client.listsImpl = &listsClient{client: client}
	client.cardsImpl = &cardsClient{client: client}
	client.membersImpl = &membersClient{client: client}

	return client, nil
}

func (c *trelloClient) Boards() BoardsClient   { return c.boardsImpl }
func (c *trelloClient) Lists() ListsClient     { return c.listsImpl }
func (c *trelloClient) Cards() CardsClient     { return c.cardsImpl }
func (c *trelloClient) Members() MembersClient { return c.membersImpl }

func (c *trelloClient) doRequest(ctx context.Context, method, path string, body interface{}, query url.Values) ([]byte, error) {
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

	// Add auth params
	if query == nil {
		query = url.Values{}
	}
	query.Set("key", c.config.APIKey)
	query.Set("token", c.config.Token)

	reqURL := c.config.BaseURL + "/1" + path + "?" + query.Encode()

	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

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

		var apiError APIError
		if err := json.Unmarshal(respBody, &apiError); err == nil && apiError.Message != "" {
			return nil, &TrelloError{Message: apiError.Message}
		}

		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", c.config.MaxRetries, lastErr)
}

// TrelloError represents an error response from Trello API
type TrelloError struct {
	Message string
}

func (e *TrelloError) Error() string {
	return fmt.Sprintf("trello error: %s", e.Message)
}
