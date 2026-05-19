package telegram

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

// telegramClient implements the Client interface
type telegramClient struct {
	config       Config
	httpClient   *http.Client
	rateLimiter  *rate.Limiter
	messagesImpl *messagesClient
	chatsImpl    *chatsClient
	usersImpl    *usersClient
}

// NewClient creates a new Telegram client
func NewClient(config Config) (Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout: config.GetTimeout(),
	}

	limiter := rate.NewLimiter(rate.Limit(config.RateLimit), 5)

	client := &telegramClient{
		config:      config,
		httpClient:  httpClient,
		rateLimiter: limiter,
	}

	client.messagesImpl = &messagesClient{client: client}
	client.chatsImpl = &chatsClient{client: client}
	client.usersImpl = &usersClient{client: client}

	return client, nil
}

func (c *telegramClient) Messages() MessagesClient { return c.messagesImpl }
func (c *telegramClient) Chats() ChatsClient       { return c.chatsImpl }
func (c *telegramClient) Users() UsersClient       { return c.usersImpl }

func (c *telegramClient) doRequest(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	var reqBody io.Reader
	if params != nil {
		jsonData, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	url := fmt.Sprintf("%s/bot%s/%s", c.config.BaseURL, c.config.BotToken, method)

	req, err := http.NewRequestWithContext(ctx, "POST", url, reqBody)
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

		var apiResp APIResponse
		if err := json.Unmarshal(respBody, &apiResp); err != nil {
			lastErr = fmt.Errorf("failed to parse response: %w", err)
			continue
		}

		if !apiResp.OK {
			if apiResp.ErrorCode == 429 {
				lastErr = fmt.Errorf("rate limited")
				continue
			}
			return nil, &TelegramError{Code: apiResp.ErrorCode, Message: apiResp.Description}
		}

		result, _ := json.Marshal(apiResp.Result)
		return result, nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", c.config.MaxRetries, lastErr)
}

// TelegramError represents an error response from Telegram API
type TelegramError struct {
	Code    int
	Message string
}

func (e *TelegramError) Error() string {
	return fmt.Sprintf("telegram error (%d): %s", e.Code, e.Message)
}
