package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

type usersClient struct {
	client *notionClient
}

func (u *usersClient) Get(ctx context.Context, userID string) (*User, error) {
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

func (u *usersClient) List(ctx context.Context, cursor string) (*UsersResponse, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	path := "/users"

	if cursor != "" {
		query := url.Values{}
		query.Set("start_cursor", cursor)
		path += "?" + query.Encode()
	}

	respBody, err := u.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	var response UsersResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

func (u *usersClient) GetMe(ctx context.Context) (*User, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	respBody, err := u.client.doRequest(ctx, "GET", "/users/me", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get bot user: %w", err)
	}

	var user User
	if err := json.Unmarshal(respBody, &user); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &user, nil
}
