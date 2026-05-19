package notion

import (
	"context"
	"encoding/json"
	"fmt"
)

type pagesClient struct {
	client *notionClient
}

func (p *pagesClient) Create(ctx context.Context, req *CreatePageRequest) (*Page, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	respBody, err := p.client.doRequest(ctx, "POST", "/pages", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	var page Page
	if err := json.Unmarshal(respBody, &page); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &page, nil
}

func (p *pagesClient) Get(ctx context.Context, pageID string) (*Page, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if pageID == "" {
		return nil, fmt.Errorf("pageID cannot be empty")
	}

	path := fmt.Sprintf("/pages/%s", pageID)

	respBody, err := p.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get page: %w", err)
	}

	var page Page
	if err := json.Unmarshal(respBody, &page); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &page, nil
}

func (p *pagesClient) Update(ctx context.Context, pageID string, properties map[string]Property) (*Page, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if pageID == "" {
		return nil, fmt.Errorf("pageID cannot be empty")
	}
	if properties == nil {
		return nil, fmt.Errorf("properties cannot be nil")
	}

	path := fmt.Sprintf("/pages/%s", pageID)
	reqBody := map[string]interface{}{
		"properties": properties,
	}

	respBody, err := p.client.doRequest(ctx, "PATCH", path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to update page: %w", err)
	}

	var page Page
	if err := json.Unmarshal(respBody, &page); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &page, nil
}

func (p *pagesClient) Archive(ctx context.Context, pageID string) (*Page, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if pageID == "" {
		return nil, fmt.Errorf("pageID cannot be empty")
	}

	path := fmt.Sprintf("/pages/%s", pageID)
	reqBody := map[string]interface{}{
		"archived": true,
	}

	respBody, err := p.client.doRequest(ctx, "PATCH", path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to archive page: %w", err)
	}

	var page Page
	if err := json.Unmarshal(respBody, &page); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &page, nil
}
