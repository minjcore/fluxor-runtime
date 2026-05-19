package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
)

type searchClient struct {
	client *elasticsearchClient
}

func (s *searchClient) Search(ctx context.Context, index string, query map[string]interface{}) (*SearchResponse, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if index == "" {
		return nil, fmt.Errorf("index name cannot be empty")
	}

	path := fmt.Sprintf("/%s/_search", index)
	respBody, err := s.client.doRequest(ctx, "POST", path, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	var result SearchResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	return &result, nil
}

func (s *searchClient) SearchAll(ctx context.Context, query map[string]interface{}) (*SearchResponse, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	path := "/_search"
	respBody, err := s.client.doRequest(ctx, "POST", path, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	var result SearchResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	return &result, nil
}

func (s *searchClient) Count(ctx context.Context, index string, query map[string]interface{}) (int64, error) {
	if ctx == nil {
		return 0, fmt.Errorf("context cannot be nil")
	}
	if index == "" {
		return 0, fmt.Errorf("index name cannot be empty")
	}

	path := fmt.Sprintf("/%s/_count", index)
	respBody, err := s.client.doRequest(ctx, "POST", path, query)
	if err != nil {
		return 0, fmt.Errorf("failed to count: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("failed to parse count response: %w", err)
	}

	if count, ok := result["count"].(float64); ok {
		return int64(count), nil
	}

	return 0, fmt.Errorf("invalid count response format")
}
