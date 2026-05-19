package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// pullRequestsClient implements the PullRequestsClient interface
type pullRequestsClient struct {
	client *githubClient
}

// List returns pull requests for a repository
func (p *pullRequestsClient) List(ctx context.Context, owner, repo string, params ListPRParams) ([]PullRequest, error) {
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
	path := fmt.Sprintf("/repos/%s/%s/pulls", owner, repo)

	// Build query parameters
	query := url.Values{}
	if params.State != "" {
		query.Set("state", params.State)
	}
	if params.Head != "" {
		query.Set("head", params.Head)
	}
	if params.Base != "" {
		query.Set("base", params.Base)
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
	respBody, err := p.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests: %w", err)
	}

	// Parse response
	var prs []PullRequest
	if err := json.Unmarshal(respBody, &prs); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return prs, nil
}

// Get returns a specific pull request
func (p *pullRequestsClient) Get(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
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
		return nil, fmt.Errorf("pull request number must be positive")
	}

	// Build path
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number)

	// Perform request
	respBody, err := p.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request: %w", err)
	}

	// Parse response
	var pr PullRequest
	if err := json.Unmarshal(respBody, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &pr, nil
}

// Create creates a new pull request
func (p *pullRequestsClient) Create(ctx context.Context, owner, repo string, req *CreatePRRequest) (*PullRequest, error) {
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
		return nil, fmt.Errorf("pull request title cannot be empty")
	}
	if req.Head == "" {
		return nil, fmt.Errorf("head branch cannot be empty")
	}
	if req.Base == "" {
		return nil, fmt.Errorf("base branch cannot be empty")
	}

	// Build path
	path := fmt.Sprintf("/repos/%s/%s/pulls", owner, repo)

	// Perform request
	respBody, err := p.client.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	// Parse response
	var pr PullRequest
	if err := json.Unmarshal(respBody, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &pr, nil
}

// Merge merges a pull request
func (p *pullRequestsClient) Merge(ctx context.Context, owner, repo string, number int, opts *MergeOptions) (*MergeResult, error) {
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
		return nil, fmt.Errorf("pull request number must be positive")
	}

	// Build path
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/merge", owner, repo, number)

	// Prepare request body
	var reqBody interface{}
	if opts != nil {
		reqBody = opts
	} else {
		reqBody = map[string]string{}
	}

	// Perform request
	respBody, err := p.client.doRequest(ctx, "PUT", path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to merge pull request: %w", err)
	}

	// Parse response
	var result MergeResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ListReviews lists reviews for a pull request
func (p *pullRequestsClient) ListReviews(ctx context.Context, owner, repo string, number int) ([]Review, error) {
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
		return nil, fmt.Errorf("pull request number must be positive")
	}

	// Build path
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", owner, repo, number)

	// Perform request
	respBody, err := p.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list reviews: %w", err)
	}

	// Parse response
	var reviews []Review
	if err := json.Unmarshal(respBody, &reviews); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return reviews, nil
}

// CreateReview creates a review for a pull request
func (p *pullRequestsClient) CreateReview(ctx context.Context, owner, repo string, number int, req *CreateReviewRequest) (*Review, error) {
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
		return nil, fmt.Errorf("pull request number must be positive")
	}
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.Event == "" {
		return nil, fmt.Errorf("review event cannot be empty")
	}

	// Build path
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", owner, repo, number)

	// Perform request
	respBody, err := p.client.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create review: %w", err)
	}

	// Parse response
	var review Review
	if err := json.Unmarshal(respBody, &review); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &review, nil
}
