package telegram

import (
	"context"
	"encoding/json"
	"fmt"
)

type usersClient struct {
	client *telegramClient
}

func (u *usersClient) GetMe(ctx context.Context) (*User, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	result, err := u.client.doRequest(ctx, "getMe", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get bot user: %w", err)
	}

	var user User
	if err := json.Unmarshal(result, &user); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &user, nil
}

func (u *usersClient) GetUserProfilePhotos(ctx context.Context, userID int64, offset, limit int) (*UserProfilePhotos, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	params := map[string]interface{}{
		"user_id": userID,
	}

	if offset > 0 {
		params["offset"] = offset
	}
	if limit > 0 {
		params["limit"] = limit
	}

	result, err := u.client.doRequest(ctx, "getUserProfilePhotos", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile photos: %w", err)
	}

	var photos UserProfilePhotos
	if err := json.Unmarshal(result, &photos); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &photos, nil
}
