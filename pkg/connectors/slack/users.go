package slack

import (
	"context"
	"encoding/json"
	"fmt"
)

// usersClient implements the UsersClient interface
type usersClient struct {
	client *slackClient
}

// List returns all users
func (u *usersClient) List(ctx context.Context) ([]User, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	// Perform request
	respBody, err := u.client.doRequest(ctx, "POST", "users.list", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Parse response
	var response struct {
		APIResponse
		Members []User `json:"members"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !response.OK {
		return nil, &SlackError{Code: response.Error, Message: response.Error}
	}

	return response.Members, nil
}

// Get returns a specific user
func (u *usersClient) Get(ctx context.Context, userID string) (*User, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	// Prepare request body
	requestBody := map[string]interface{}{
		"user": userID,
	}

	// Perform request
	respBody, err := u.client.doRequest(ctx, "POST", "users.info", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Parse response
	var response struct {
		APIResponse
		User User `json:"user"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !response.OK {
		return nil, &SlackError{Code: response.Error, Message: response.Error}
	}

	return &response.User, nil
}

// GetByEmail returns a user by email
func (u *usersClient) GetByEmail(ctx context.Context, email string) (*User, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if email == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}

	// Prepare request body
	requestBody := map[string]interface{}{
		"email": email,
	}

	// Perform request
	respBody, err := u.client.doRequest(ctx, "POST", "users.lookupByEmail", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	// Parse response
	var response struct {
		APIResponse
		User User `json:"user"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !response.OK {
		return nil, &SlackError{Code: response.Error, Message: response.Error}
	}

	return &response.User, nil
}
