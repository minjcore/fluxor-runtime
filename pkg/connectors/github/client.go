package github

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

// githubClient implements the Client interface
type githubClient struct {
	config          Config
	httpClient      *http.Client
	rateLimiter     *rate.Limiter
	repositoriesImpl *repositoriesClient
	issuesImpl      *issuesClient
	pullRequestsImpl *pullRequestsClient
	usersImpl       *usersClient
}

// NewClient creates a new GitHub client with the given configuration
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

	// Create rate limiter (requests per hour -> per second)
	limiter := rate.NewLimiter(rate.Limit(float64(config.RateLimit)/3600.0), 10)

	client := &githubClient{
		config:      config,
		httpClient:  httpClient,
		rateLimiter: limiter,
	}

	// Initialize service clients
	client.repositoriesImpl = &repositoriesClient{client: client}
	client.issuesImpl = &issuesClient{client: client}
	client.pullRequestsImpl = &pullRequestsClient{client: client}
	client.usersImpl = &usersClient{client: client}

	return client, nil
}

// Repositories returns the RepositoriesClient
func (c *githubClient) Repositories() RepositoriesClient {
	return c.repositoriesImpl
}

// Issues returns the IssuesClient
func (c *githubClient) Issues() IssuesClient {
	return c.issuesImpl
}

// PullRequests returns the PullRequestsClient
func (c *githubClient) PullRequests() PullRequestsClient {
	return c.pullRequestsImpl
}

// Users returns the UsersClient
func (c *githubClient) Users() UsersClient {
	return c.usersImpl
}

// doRequest performs an HTTP request with rate limiting and retries
func (c *githubClient) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
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
	url := c.config.BaseURL
	if !strings.HasPrefix(path, "/") {
		url += "/"
	}
	url += path

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.config.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

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
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, nil
		}

		// Handle rate limiting
		if resp.StatusCode == 403 || resp.StatusCode == 429 {
			lastErr = fmt.Errorf("rate limited")
			continue
		}

		// Handle server errors
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		// Parse error response
		var apiError APIResponse
		if err := json.Unmarshal(respBody, &apiError); err == nil && apiError.Message != "" {
			return nil, &GitHubError{
				StatusCode: resp.StatusCode,
				Message:    apiError.Message,
				DocURL:     apiError.DocumentationURL,
			}
		}

		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", c.config.MaxRetries, lastErr)
}

// GitHubError represents an error response from the GitHub API
type GitHubError struct {
	StatusCode int
	Message    string
	DocURL     string
}

func (e *GitHubError) Error() string {
	return fmt.Sprintf("github error (%d): %s", e.StatusCode, e.Message)
}
