package github

import (
	"context"
	"encoding/json"
	"fmt"
)

// usersClient implements the UsersClient interface
type usersClient struct {
	client *githubClient
}

// Get returns the authenticated user
func (u *usersClient) Get(ctx context.Context) (*User, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	// Perform request
	respBody, err := u.client.doRequest(ctx, "GET", "/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Parse response
	var user User
	if err := json.Unmarshal(respBody, &user); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &user, nil
}

// GetByUsername returns a user by username
func (u *usersClient) GetByUsername(ctx context.Context, username string) (*User, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	// Build path
	path := fmt.Sprintf("/users/%s", username)

	// Perform request
	respBody, err := u.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Parse response
	var user User
	if err := json.Unmarshal(respBody, &user); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &user, nil
}

// ListOrgs lists organizations for the authenticated user
func (u *usersClient) ListOrgs(ctx context.Context) ([]Organization, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	// Perform request
	respBody, err := u.client.doRequest(ctx, "GET", "/user/orgs", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}

	// Parse response
	var orgs []Organization
	if err := json.Unmarshal(respBody, &orgs); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return orgs, nil
}
