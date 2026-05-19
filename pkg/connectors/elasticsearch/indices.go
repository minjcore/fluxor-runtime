package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
)

type indicesClient struct {
	client *elasticsearchClient
}

func (i *indicesClient) Create(ctx context.Context, index string, settings map[string]interface{}) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if index == "" {
		return fmt.Errorf("index name cannot be empty")
	}

	path := fmt.Sprintf("/%s", index)
	_, err := i.client.doRequest(ctx, "PUT", path, settings)
	return err
}

func (i *indicesClient) Delete(ctx context.Context, index string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if index == "" {
		return fmt.Errorf("index name cannot be empty")
	}

	path := fmt.Sprintf("/%s", index)
	_, err := i.client.doRequest(ctx, "DELETE", path, nil)
	return err
}

func (i *indicesClient) Exists(ctx context.Context, index string) (bool, error) {
	if ctx == nil {
		return false, fmt.Errorf("context cannot be nil")
	}
	if index == "" {
		return false, fmt.Errorf("index name cannot be empty")
	}

	path := fmt.Sprintf("/%s", index)
	_, err := i.client.doRequest(ctx, "HEAD", path, nil)
	if err != nil {
		if esErr, ok := err.(*ElasticsearchError); ok && esErr.Status == 404 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (i *indicesClient) GetSettings(ctx context.Context, index string) (map[string]interface{}, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if index == "" {
		return nil, fmt.Errorf("index name cannot be empty")
	}

	path := fmt.Sprintf("/%s/_settings", index)
	respBody, err := i.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get index settings: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract settings for the specific index
	if indexSettings, ok := result[index].(map[string]interface{}); ok {
		if settings, ok := indexSettings["settings"].(map[string]interface{}); ok {
			return settings, nil
		}
	}

	return result, nil
}

func (i *indicesClient) PutMapping(ctx context.Context, index string, mapping map[string]interface{}) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if index == "" {
		return fmt.Errorf("index name cannot be empty")
	}

	path := fmt.Sprintf("/%s/_mapping", index)
	_, err := i.client.doRequest(ctx, "PUT", path, mapping)
	return err
}

func (i *indicesClient) List(ctx context.Context) ([]string, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	respBody, err := i.client.doRequest(ctx, "GET", "/_cat/indices?format=json", nil)
	if err != nil {
		// Fallback to _aliases endpoint
		respBody, err = i.client.doRequest(ctx, "GET", "/_aliases", nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list indices: %w", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		indices := make([]string, 0, len(result))
		for index := range result {
			indices = append(indices, index)
		}
		return indices, nil
	}

	// Parse _cat/indices response
	var indices []map[string]interface{}
	if err := json.Unmarshal(respBody, &indices); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := make([]string, 0, len(indices))
	for _, idx := range indices {
		if name, ok := idx["index"].(string); ok {
			result = append(result, name)
		}
	}
	return result, nil
}
