package notion

import (
	"context"
	"encoding/json"
	"fmt"
)

type searchClient struct {
	client *notionClient
}

func (s *searchClient) Search(ctx context.Context, query *SearchRequest) (*SearchResponse, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	var reqBody interface{}
	if query != nil {
		reqBody = query
	} else {
		reqBody = map[string]interface{}{}
	}

	respBody, err := s.client.doRequest(ctx, "POST", "/search", reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	var response SearchResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}
