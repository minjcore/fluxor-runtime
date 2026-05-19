package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// repositoriesClient implements the RepositoriesClient interface
type repositoriesClient struct {
	client *githubClient
}

// List returns repositories for the authenticated user or organization
func (r *repositoriesClient) List(ctx context.Context, owner string, params ListReposParams) ([]Repository, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	// Build path
	var path string
	if owner == "" {
		path = "/user/repos"
	} else {
		path = fmt.Sprintf("/users/%s/repos", owner)
	}

	// Build query parameters
	query := url.Values{}
	if params.Type != "" {
		query.Set("type", params.Type)
	}
	if params.Sort != "" {
		query.Set("sort", params.Sort)
	}
	if params.Direction != "" {
		query.Set("direction", params.Direction)
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
	respBody, err := r.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	// Parse response
	var repos []Repository
	if err := json.Unmarshal(respBody, &repos); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return repos, nil
}

// Get returns a specific repository
func (r *repositoriesClient) Get(ctx context.Context, owner, repo string) (*Repository, error) {
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
	path := fmt.Sprintf("/repos/%s/%s", owner, repo)

	// Perform request
	respBody, err := r.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	// Parse response
	var repository Repository
	if err := json.Unmarshal(respBody, &repository); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &repository, nil
}

// Create creates a new repository
func (r *repositoriesClient) Create(ctx context.Context, req *CreateRepoRequest) (*Repository, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("repository name cannot be empty")
	}

	// Perform request
	respBody, err := r.client.doRequest(ctx, "POST", "/user/repos", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	// Parse response
	var repository Repository
	if err := json.Unmarshal(respBody, &repository); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &repository, nil
}

// Delete deletes a repository
func (r *repositoriesClient) Delete(ctx context.Context, owner, repo string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if owner == "" {
		return fmt.Errorf("owner cannot be empty")
	}
	if repo == "" {
		return fmt.Errorf("repo cannot be empty")
	}

	// Build path
	path := fmt.Sprintf("/repos/%s/%s", owner, repo)

	// Perform request
	_, err := r.client.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
	}

	return nil
}

// ListBranches lists branches for a repository
func (r *repositoriesClient) ListBranches(ctx context.Context, owner, repo string) ([]Branch, error) {
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
	path := fmt.Sprintf("/repos/%s/%s/branches", owner, repo)

	// Perform request
	respBody, err := r.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	// Parse response
	var branches []Branch
	if err := json.Unmarshal(respBody, &branches); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return branches, nil
}

// GetBranch gets a specific branch
func (r *repositoriesClient) GetBranch(ctx context.Context, owner, repo, branch string) (*Branch, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if owner == "" {
		return nil, fmt.Errorf("owner cannot be empty")
	}
	if repo == "" {
		return nil, fmt.Errorf("repo cannot be empty")
	}
	if branch == "" {
		return nil, fmt.Errorf("branch cannot be empty")
	}

	// Build path
	path := fmt.Sprintf("/repos/%s/%s/branches/%s", owner, repo, branch)

	// Perform request
	respBody, err := r.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch: %w", err)
	}

	// Parse response
	var b Branch
	if err := json.Unmarshal(respBody, &b); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &b, nil
}
