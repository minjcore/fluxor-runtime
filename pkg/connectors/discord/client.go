package discord

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

const baseURL = "https://discord.com/api/v10"

// discordClient implements the Client interface
type discordClient struct {
	config       Config
	httpClient   *http.Client
	rateLimiter  *rate.Limiter
	messagesImpl *messagesClient
	channelsImpl *channelsClient
	guildsImpl   *guildsClient
	usersImpl    *usersClient
}

// NewClient creates a new Discord client
func NewClient(config Config) (Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout: config.GetTimeout(),
	}

	limiter := rate.NewLimiter(rate.Limit(config.RateLimit), 5)

	client := &discordClient{
		config:      config,
		httpClient:  httpClient,
		rateLimiter: limiter,
	}

	client.messagesImpl = &messagesClient{client: client}
	client.channelsImpl = &channelsClient{client: client}
	client.guildsImpl = &guildsClient{client: client}
	client.usersImpl = &usersClient{client: client}

	return client, nil
}

func (c *discordClient) Messages() MessagesClient { return c.messagesImpl }
func (c *discordClient) Channels() ChannelsClient { return c.channelsImpl }
func (c *discordClient) Guilds() GuildsClient     { return c.guildsImpl }
func (c *discordClient) Users() UsersClient       { return c.usersImpl }

func (c *discordClient) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
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

	url := baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+c.config.BotToken)
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

		var apiError APIResponse
		if err := json.Unmarshal(respBody, &apiError); err == nil && apiError.Message != "" {
			return nil, &DiscordError{Code: apiError.Code, Message: apiError.Message}
		}

		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", c.config.MaxRetries, lastErr)
}

// DiscordError represents an error response from Discord API
type DiscordError struct {
	Code    int
	Message string
}

func (e *DiscordError) Error() string {
	return fmt.Sprintf("discord error (%d): %s", e.Code, e.Message)
}
