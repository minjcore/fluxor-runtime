package telegram

import (
	"context"
	"encoding/json"
	"fmt"
)

type chatsClient struct {
	client *telegramClient
}

func (c *chatsClient) Get(ctx context.Context, chatID int64) (*Chat, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	params := map[string]interface{}{
		"chat_id": chatID,
	}

	result, err := c.client.doRequest(ctx, "getChat", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	var chat Chat
	if err := json.Unmarshal(result, &chat); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &chat, nil
}

func (c *chatsClient) GetAdministrators(ctx context.Context, chatID int64) ([]ChatMember, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	params := map[string]interface{}{
		"chat_id": chatID,
	}

	result, err := c.client.doRequest(ctx, "getChatAdministrators", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat administrators: %w", err)
	}

	var members []ChatMember
	if err := json.Unmarshal(result, &members); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return members, nil
}

func (c *chatsClient) GetMemberCount(ctx context.Context, chatID int64) (int, error) {
	if ctx == nil {
		return 0, fmt.Errorf("context cannot be nil")
	}

	params := map[string]interface{}{
		"chat_id": chatID,
	}

	result, err := c.client.doRequest(ctx, "getChatMemberCount", params)
	if err != nil {
		return 0, fmt.Errorf("failed to get chat member count: %w", err)
	}

	var count int
	if err := json.Unmarshal(result, &count); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return count, nil
}

func (c *chatsClient) GetMember(ctx context.Context, chatID, userID int64) (*ChatMember, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	params := map[string]interface{}{
		"chat_id": chatID,
		"user_id": userID,
	}

	result, err := c.client.doRequest(ctx, "getChatMember", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat member: %w", err)
	}

	var member ChatMember
	if err := json.Unmarshal(result, &member); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &member, nil
}

func (c *chatsClient) Leave(ctx context.Context, chatID int64) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}

	params := map[string]interface{}{
		"chat_id": chatID,
	}

	_, err := c.client.doRequest(ctx, "leaveChat", params)
	if err != nil {
		return fmt.Errorf("failed to leave chat: %w", err)
	}

	return nil
}
