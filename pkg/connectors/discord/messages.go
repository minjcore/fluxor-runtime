package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

type messagesClient struct {
	client *discordClient
}

func (m *messagesClient) Send(ctx context.Context, channelID string, message *MessageCreate) (*Message, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if channelID == "" {
		return nil, fmt.Errorf("channelID cannot be empty")
	}
	if message == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	path := fmt.Sprintf("/channels/%s/messages", channelID)

	respBody, err := m.client.doRequest(ctx, "POST", path, message)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	var msg Message
	if err := json.Unmarshal(respBody, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &msg, nil
}

func (m *messagesClient) Edit(ctx context.Context, channelID, messageID string, message *MessageEdit) (*Message, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if channelID == "" {
		return nil, fmt.Errorf("channelID cannot be empty")
	}
	if messageID == "" {
		return nil, fmt.Errorf("messageID cannot be empty")
	}
	if message == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	path := fmt.Sprintf("/channels/%s/messages/%s", channelID, messageID)

	respBody, err := m.client.doRequest(ctx, "PATCH", path, message)
	if err != nil {
		return nil, fmt.Errorf("failed to edit message: %w", err)
	}

	var msg Message
	if err := json.Unmarshal(respBody, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &msg, nil
}

func (m *messagesClient) Delete(ctx context.Context, channelID, messageID string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if channelID == "" {
		return fmt.Errorf("channelID cannot be empty")
	}
	if messageID == "" {
		return fmt.Errorf("messageID cannot be empty")
	}

	path := fmt.Sprintf("/channels/%s/messages/%s", channelID, messageID)

	_, err := m.client.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}

func (m *messagesClient) Get(ctx context.Context, channelID, messageID string) (*Message, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if channelID == "" {
		return nil, fmt.Errorf("channelID cannot be empty")
	}
	if messageID == "" {
		return nil, fmt.Errorf("messageID cannot be empty")
	}

	path := fmt.Sprintf("/channels/%s/messages/%s", channelID, messageID)

	respBody, err := m.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	var msg Message
	if err := json.Unmarshal(respBody, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &msg, nil
}

func (m *messagesClient) GetHistory(ctx context.Context, channelID string, params HistoryParams) ([]Message, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if channelID == "" {
		return nil, fmt.Errorf("channelID cannot be empty")
	}

	path := fmt.Sprintf("/channels/%s/messages", channelID)

	query := url.Values{}
	if params.Limit > 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Before != "" {
		query.Set("before", params.Before)
	}
	if params.After != "" {
		query.Set("after", params.After)
	}
	if params.Around != "" {
		query.Set("around", params.Around)
	}

	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	respBody, err := m.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get message history: %w", err)
	}

	var messages []Message
	if err := json.Unmarshal(respBody, &messages); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return messages, nil
}

func (m *messagesClient) React(ctx context.Context, channelID, messageID, emoji string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if channelID == "" {
		return fmt.Errorf("channelID cannot be empty")
	}
	if messageID == "" {
		return fmt.Errorf("messageID cannot be empty")
	}
	if emoji == "" {
		return fmt.Errorf("emoji cannot be empty")
	}

	path := fmt.Sprintf("/channels/%s/messages/%s/reactions/%s/@me", channelID, messageID, url.PathEscape(emoji))

	_, err := m.client.doRequest(ctx, "PUT", path, nil)
	if err != nil {
		return fmt.Errorf("failed to add reaction: %w", err)
	}

	return nil
}

func (m *messagesClient) DeleteReaction(ctx context.Context, channelID, messageID, emoji string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if channelID == "" {
		return fmt.Errorf("channelID cannot be empty")
	}
	if messageID == "" {
		return fmt.Errorf("messageID cannot be empty")
	}
	if emoji == "" {
		return fmt.Errorf("emoji cannot be empty")
	}

	path := fmt.Sprintf("/channels/%s/messages/%s/reactions/%s/@me", channelID, messageID, url.PathEscape(emoji))

	_, err := m.client.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("failed to delete reaction: %w", err)
	}

	return nil
}
