package slack

import (
	"context"
	"encoding/json"
	"fmt"
)

// messagesClient implements the MessagesClient interface
type messagesClient struct {
	client *slackClient
}

// Send sends a message to a channel
func (m *messagesClient) Send(ctx context.Context, channelID string, message *Message) (*Message, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if channelID == "" {
		return nil, fmt.Errorf("channelID cannot be empty")
	}
	if message == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	// Prepare request body
	requestBody := map[string]interface{}{
		"channel": channelID,
		"text":    message.Text,
	}

	if len(message.Attachments) > 0 {
		requestBody["attachments"] = message.Attachments
	}

	if len(message.Blocks) > 0 {
		requestBody["blocks"] = message.Blocks
	}

	if message.ThreadTS != "" {
		requestBody["thread_ts"] = message.ThreadTS
	}

	// Perform request
	respBody, err := m.client.doRequest(ctx, "POST", "chat.postMessage", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Parse response
	var response struct {
		APIResponse
		Message Message `json:"message"`
		Channel string  `json:"channel"`
		TS      string  `json:"ts"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !response.OK {
		return nil, &SlackError{Code: response.Error, Message: response.Error}
	}

	response.Message.Channel = response.Channel
	response.Message.Timestamp = response.TS

	return &response.Message, nil
}

// Update updates an existing message
func (m *messagesClient) Update(ctx context.Context, channelID, timestamp string, message *Message) (*Message, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if channelID == "" {
		return nil, fmt.Errorf("channelID cannot be empty")
	}
	if timestamp == "" {
		return nil, fmt.Errorf("timestamp cannot be empty")
	}
	if message == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	// Prepare request body
	requestBody := map[string]interface{}{
		"channel": channelID,
		"ts":      timestamp,
		"text":    message.Text,
	}

	if len(message.Attachments) > 0 {
		requestBody["attachments"] = message.Attachments
	}

	if len(message.Blocks) > 0 {
		requestBody["blocks"] = message.Blocks
	}

	// Perform request
	respBody, err := m.client.doRequest(ctx, "POST", "chat.update", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to update message: %w", err)
	}

	// Parse response
	var response struct {
		APIResponse
		Message Message `json:"message"`
		Channel string  `json:"channel"`
		TS      string  `json:"ts"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !response.OK {
		return nil, &SlackError{Code: response.Error, Message: response.Error}
	}

	response.Message.Channel = response.Channel
	response.Message.Timestamp = response.TS

	return &response.Message, nil
}

// Delete deletes a message
func (m *messagesClient) Delete(ctx context.Context, channelID, timestamp string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if channelID == "" {
		return fmt.Errorf("channelID cannot be empty")
	}
	if timestamp == "" {
		return fmt.Errorf("timestamp cannot be empty")
	}

	// Prepare request body
	requestBody := map[string]interface{}{
		"channel": channelID,
		"ts":      timestamp,
	}

	// Perform request
	respBody, err := m.client.doRequest(ctx, "POST", "chat.delete", requestBody)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
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

// GetHistory retrieves message history for a channel
func (m *messagesClient) GetHistory(ctx context.Context, channelID string, params HistoryParams) ([]Message, error) {
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

	if params.Limit > 0 {
		requestBody["limit"] = params.Limit
	}
	if params.Cursor != "" {
		requestBody["cursor"] = params.Cursor
	}
	if params.Latest != "" {
		requestBody["latest"] = params.Latest
	}
	if params.Oldest != "" {
		requestBody["oldest"] = params.Oldest
	}
	if params.Inclusive {
		requestBody["inclusive"] = params.Inclusive
	}

	// Perform request
	respBody, err := m.client.doRequest(ctx, "POST", "conversations.history", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to get message history: %w", err)
	}

	// Parse response
	var response struct {
		APIResponse
		Messages []Message `json:"messages"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !response.OK {
		return nil, &SlackError{Code: response.Error, Message: response.Error}
	}

	return response.Messages, nil
}
