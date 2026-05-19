package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

var (
	baseURL = "https://slack.com/api"
)

// slackClient implements the Client interface
type slackClient struct {
	config       Config
	httpClient   *http.Client
	rateLimiter  *rate.Limiter
	messagesImpl *messagesClient
	channelsImpl *channelsClient
	usersImpl    *usersClient
}

// NewClient creates a new Slack client with the given configuration
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

	// Create rate limiter (requests per minute -> per second)
	limiter := rate.NewLimiter(rate.Limit(float64(config.RateLimit)/60.0), 1)

	client := &slackClient{
		config:      config,
		httpClient:  httpClient,
		rateLimiter: limiter,
	}

	// Initialize service clients
	client.messagesImpl = &messagesClient{client: client}
	client.channelsImpl = &channelsClient{client: client}
	client.usersImpl = &usersClient{client: client}

	return client, nil
}

// Messages returns the MessagesClient
func (c *slackClient) Messages() MessagesClient {
	return c.messagesImpl
}

// Channels returns the ChannelsClient
func (c *slackClient) Channels() ChannelsClient {
	return c.channelsImpl
}

// Users returns the UsersClient
func (c *slackClient) Users() UsersClient {
	return c.usersImpl
}

// doRequest performs an HTTP request with rate limiting and retries
func (c *slackClient) doRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Prepare request body
	var reqBody io.Reader
	contentType := "application/json; charset=utf-8"

	if body != nil {
		switch v := body.(type) {
		case url.Values:
			reqBody = strings.NewReader(v.Encode())
			contentType = "application/x-www-form-urlencoded"
		default:
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			reqBody = bytes.NewReader(jsonData)
		}
	}

	// Build URL
	reqURL := baseURL + "/" + endpoint

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.config.BotToken)
	req.Header.Set("Content-Type", contentType)

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

		// Check HTTP status code
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		if resp.StatusCode == 429 {
			// Rate limited, retry after backoff
			lastErr = fmt.Errorf("rate limited")
			continue
		}

		// Parse response to check for Slack API errors
		var apiResp APIResponse
		if err := json.Unmarshal(respBody, &apiResp); err == nil && !apiResp.OK {
			// Check if it's a retryable error
			if isRetryableError(apiResp.Error) {
				lastErr = &SlackError{Code: apiResp.Error, Message: apiResp.Error}
				continue
			}
			return nil, &SlackError{Code: apiResp.Error, Message: apiResp.Error}
		}

		return respBody, nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", c.config.MaxRetries, lastErr)
}

// isRetryableError checks if an error is retryable
func isRetryableError(errCode string) bool {
	retryable := []string{
		"internal_error",
		"service_unavailable",
		"ratelimited",
	}
	for _, r := range retryable {
		if errCode == r {
			return true
		}
	}
	return false
}

// SlackError represents an error response from the Slack API
type SlackError struct {
	Code    string
	Message string
}

func (e *SlackError) Error() string {
	return fmt.Sprintf("slack error: %s", e.Message)
}
