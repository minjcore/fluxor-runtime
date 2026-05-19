package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// issuesClient implements the IssuesClient interface
type issuesClient struct {
	client *githubClient
}

// List returns issues for a repository
func (i *issuesClient) List(ctx context.Context, owner, repo string, params ListIssuesParams) ([]Issue, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if owner == "" {
		return nil, fmt.Errorf("owner cannot be empty")
	}
	if repo == "" {
		return nil, fmt.Errorf("repo cannot be empty")
	}

	// Build path
	path := fmt.Sprintf("/repos/%s/%s/issues", owner, repo)

	// Build query parameters
	query := url.Values{}
	if params.State != "" {
		query.Set("state", params.State)
	}
	if len(params.Labels) > 0 {
		query.Set("labels", strings.Join(params.Labels, ","))
	}
	if params.Sort != "" {
		query.Set("sort", params.Sort)
	}
	if params.Direction != "" {
		query.Set("direction", params.Direction)
	}
	if params.Since != "" {
		query.Set("since", params.Since)
	}
	if params.PerPage > 0 {
		query.Set("per_page", strconv.Itoa(params.PerPage))
	}
	if params.Page > 0 {
		query.Set("page", strconv.Itoa(params.Page))
	}

	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	// Perform request
	respBody, err := i.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	// Parse response
	var issues []Issue
	if err := json.Unmarshal(respBody, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return issues, nil
}

// Get returns a specific issue
func (i *issuesClient) Get(ctx context.Context, owner, repo string, number int) (*Issue, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if owner == "" {
		return nil, fmt.Errorf("owner cannot be empty")
	}
	if repo == "" {
		return nil, fmt.Errorf("repo cannot be empty")
	}
	if number <= 0 {
		return nil, fmt.Errorf("issue number must be positive")
	}

	// Build path
	path := fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, number)

	// Perform request
	respBody, err := i.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	// Parse response
	var issue Issue
	if err := json.Unmarshal(respBody, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &issue, nil
}

// Create creates a new issue
func (i *issuesClient) Create(ctx context.Context, owner, repo string, req *CreateIssueRequest) (*Issue, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if owner == "" {
		return nil, fmt.Errorf("owner cannot be empty")
	}
	if repo == "" {
		return nil, fmt.Errorf("repo cannot be empty")
	}
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.Title == "" {
		return nil, fmt.Errorf("issue title cannot be empty")
	}

	// Build path
	path := fmt.Sprintf("/repos/%s/%s/issues", owner, repo)

	// Perform request
	respBody, err := i.client.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	// Parse response
	var issue Issue
	if err := json.Unmarshal(respBody, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &issue, nil
}

// Update updates an existing issue
func (i *issuesClient) Update(ctx context.Context, owner, repo string, number int, req *UpdateIssueRequest) (*Issue, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if owner == "" {
		return nil, fmt.Errorf("owner cannot be empty")
	}
	if repo == "" {
		return nil, fmt.Errorf("repo cannot be empty")
	}
	if number <= 0 {
		return nil, fmt.Errorf("issue number must be positive")
	}
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	// Build path
	path := fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, number)

	// Perform request
	respBody, err := i.client.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update issue: %w", err)
	}

	// Parse response
	var issue Issue
	if err := json.Unmarshal(respBody, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &issue, nil
}

// Close closes an issue
func (i *issuesClient) Close(ctx context.Context, owner, repo string, number int) (*Issue, error) {
	return i.Update(ctx, owner, repo, number, &UpdateIssueRequest{State: "closed"})
}

// CreateComment creates a comment on an issue
func (i *issuesClient) CreateComment(ctx context.Context, owner, repo string, number int, body string) (*Comment, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if owner == "" {
		return nil, fmt.Errorf("owner cannot be empty")
	}
	if repo == "" {
		return nil, fmt.Errorf("repo cannot be empty")
	}
	if number <= 0 {
		return nil, fmt.Errorf("issue number must be positive")
	}
	if body == "" {
		return nil, fmt.Errorf("body cannot be empty")
	}

	// Build path
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, number)

	// Perform request
	respBody, err := i.client.doRequest(ctx, "POST", path, map[string]string{"body": body})
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	// Parse response
	var comment Comment
	if err := json.Unmarshal(respBody, &comment); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &comment, nil
}

// ListComments lists comments on an issue
func (i *issuesClient) ListComments(ctx context.Context, owner, repo string, number int) ([]Comment, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if owner == "" {
		return nil, fmt.Errorf("owner cannot be empty")
	}
	if repo == "" {
		return nil, fmt.Errorf("repo cannot be empty")
	}
	if number <= 0 {
		return nil, fmt.Errorf("issue number must be positive")
	}

	// Build path
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, number)

	// Perform request
	respBody, err := i.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list comments: %w", err)
	}

	// Parse response
	var comments []Comment
	if err := json.Unmarshal(respBody, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return comments, nil
}
