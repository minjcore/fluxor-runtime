package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

type blocksClient struct {
	client *notionClient
}

func (b *blocksClient) Get(ctx context.Context, blockID string) (*Block, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if blockID == "" {
		return nil, fmt.Errorf("blockID cannot be empty")
	}

	path := fmt.Sprintf("/blocks/%s", blockID)

	respBody, err := b.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	var block Block
	if err := json.Unmarshal(respBody, &block); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &block, nil
}

func (b *blocksClient) GetChildren(ctx context.Context, blockID string, cursor string) (*BlockChildrenResponse, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if blockID == "" {
		return nil, fmt.Errorf("blockID cannot be empty")
	}

	path := fmt.Sprintf("/blocks/%s/children", blockID)

	if cursor != "" {
		query := url.Values{}
		query.Set("start_cursor", cursor)
		path += "?" + query.Encode()
	}

	respBody, err := b.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get block children: %w", err)
	}

	var response BlockChildrenResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

func (b *blocksClient) AppendChildren(ctx context.Context, blockID string, children []Block) (*BlockChildrenResponse, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if blockID == "" {
		return nil, fmt.Errorf("blockID cannot be empty")
	}
	if len(children) == 0 {
		return nil, fmt.Errorf("children cannot be empty")
	}

	path := fmt.Sprintf("/blocks/%s/children", blockID)
	reqBody := map[string]interface{}{
		"children": children,
	}

	respBody, err := b.client.doRequest(ctx, "PATCH", path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to append children: %w", err)
	}

	var response BlockChildrenResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

func (b *blocksClient) Update(ctx context.Context, blockID string, block *Block) (*Block, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if blockID == "" {
		return nil, fmt.Errorf("blockID cannot be empty")
	}
	if block == nil {
		return nil, fmt.Errorf("block cannot be nil")
	}

	path := fmt.Sprintf("/blocks/%s", blockID)

	respBody, err := b.client.doRequest(ctx, "PATCH", path, block)
	if err != nil {
		return nil, fmt.Errorf("failed to update block: %w", err)
	}

	var updatedBlock Block
	if err := json.Unmarshal(respBody, &updatedBlock); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &updatedBlock, nil
}

func (b *blocksClient) Delete(ctx context.Context, blockID string) (*Block, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if blockID == "" {
		return nil, fmt.Errorf("blockID cannot be empty")
	}

	path := fmt.Sprintf("/blocks/%s", blockID)

	respBody, err := b.client.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to delete block: %w", err)
	}

	var block Block
	if err := json.Unmarshal(respBody, &block); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &block, nil
}
