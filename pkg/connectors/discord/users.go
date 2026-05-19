package discord

import (
	"context"
	"encoding/json"
	"fmt"
)

type usersClient struct {
	client *discordClient
}

func (u *usersClient) GetCurrentUser(ctx context.Context) (*User, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	respBody, err := u.client.doRequest(ctx, "GET", "/users/@me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	var user User
	if err := json.Unmarshal(respBody, &user); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &user, nil
}

func (u *usersClient) GetUser(ctx context.Context, userID string) (*User, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	path := fmt.Sprintf("/users/%s", userID)

	respBody, err := u.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var user User
	if err := json.Unmarshal(respBody, &user); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &user, nil
}

func (u *usersClient) CreateDM(ctx context.Context, userID string) (*Channel, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	reqBody := map[string]string{
		"recipient_id": userID,
	}

	respBody, err := u.client.doRequest(ctx, "POST", "/users/@me/channels", reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create DM: %w", err)
	}

	var channel Channel
	if err := json.Unmarshal(respBody, &channel); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &channel, nil
}
