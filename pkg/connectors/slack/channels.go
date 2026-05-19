package slack

import (
	"context"
	"encoding/json"
	"fmt"
)

// channelsClient implements the ChannelsClient interface
type channelsClient struct {
	client *slackClient
}

// List returns all channels
func (c *channelsClient) List(ctx context.Context, params ListChannelsParams) ([]Channel, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	// Prepare request body
	requestBody := map[string]interface{}{}

	if params.Limit > 0 {
		requestBody["limit"] = params.Limit
	}
	if params.Cursor != "" {
		requestBody["cursor"] = params.Cursor
	}
	if params.ExcludeArchived {
		requestBody["exclude_archived"] = params.ExcludeArchived
	}
	if params.Types != "" {
		requestBody["types"] = params.Types
	}

	// Perform request
	respBody, err := c.client.doRequest(ctx, "POST", "conversations.list", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to list channels: %w", err)
	}

	// Parse response
	var response struct {
		APIResponse
		Channels []Channel `json:"channels"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !response.OK {
		return nil, &SlackError{Code: response.Error, Message: response.Error}
	}

	return response.Channels, nil
}

// Get returns a specific channel
func (c *channelsClient) Get(ctx context.Context, channelID string) (*Channel, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if channelID == "" {
		return nil, fmt.Errorf("channelID cannot be empty")
	}

	// Prepare request body
	requestBody := map[string]interface{}{
		"channel": channelID,
	}

	// Perform request
	respBody, err := c.client.doRequest(ctx, "POST", "conversations.info", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	// Parse response
	var response struct {
		APIResponse
		Channel Channel `json:"channel"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !response.OK {
		return nil, &SlackError{Code: response.Error, Message: response.Error}
	}

	return &response.Channel, nil
}

// Create creates a new channel
func (c *channelsClient) Create(ctx context.Context, name string, isPrivate bool) (*Channel, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	// Prepare request body
	requestBody := map[string]interface{}{
		"name":       name,
		"is_private": isPrivate,
	}

	// Perform request
	respBody, err := c.client.doRequest(ctx, "POST", "conversations.create", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	// Parse response
	var response struct {
		APIResponse
		Channel Channel `json:"channel"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !response.OK {
		return nil, &SlackError{Code: response.Error, Message: response.Error}
	}

	return &response.Channel, nil
}

// Archive archives a channel
func (c *channelsClient) Archive(ctx context.Context, channelID string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if channelID == "" {
		return fmt.Errorf("channelID cannot be empty")
	}

	// Prepare request body
	requestBody := map[string]interface{}{
		"channel": channelID,
	}

	// Perform request
	respBody, err := c.client.doRequest(ctx, "POST", "conversations.archive", requestBody)
	if err != nil {
		return fmt.Errorf("failed to archive channel: %w", err)
	}

	// Parse response
	var response APIResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !response.OK {
		return &SlackError{Code: response.Error, Message: response.Error}
	}

	return nil
}

// Invite invites a user to a channel
func (c *channelsClient) Invite(ctx context.Context, channelID, userID string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if channelID == "" {
		return fmt.Errorf("channelID cannot be empty")
	}
	if userID == "" {
		return fmt.Errorf("userID cannot be empty")
	}

	// Prepare request body
	requestBody := map[string]interface{}{
		"channel": channelID,
		"users":   userID,
	}

	// Perform request
	respBody, err := c.client.doRequest(ctx, "POST", "conversations.invite", requestBody)
	if err != nil {
		return fmt.Errorf("failed to invite user: %w", err)
	}

	// Parse response
	var response APIResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !response.OK {
		return &SlackError{Code: response.Error, Message: response.Error}
	}

	return nil
}
