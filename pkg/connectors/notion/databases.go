package notion

import (
	"context"
	"encoding/json"
	"fmt"
)

type databasesClient struct {
	client *notionClient
}

func (d *databasesClient) Create(ctx context.Context, req *CreateDatabaseRequest) (*Database, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	respBody, err := d.client.doRequest(ctx, "POST", "/databases", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	var database Database
	if err := json.Unmarshal(respBody, &database); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &database, nil
}

func (d *databasesClient) Get(ctx context.Context, databaseID string) (*Database, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if databaseID == "" {
		return nil, fmt.Errorf("databaseID cannot be empty")
	}

	path := fmt.Sprintf("/databases/%s", databaseID)

	respBody, err := d.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	var database Database
	if err := json.Unmarshal(respBody, &database); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &database, nil
}

func (d *databasesClient) Query(ctx context.Context, databaseID string, query *DatabaseQuery) (*QueryResult, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if databaseID == "" {
		return nil, fmt.Errorf("databaseID cannot be empty")
	}

	path := fmt.Sprintf("/databases/%s/query", databaseID)

	var reqBody interface{}
	if query != nil {
		reqBody = query
	} else {
		reqBody = map[string]interface{}{}
	}

	respBody, err := d.client.doRequest(ctx, "POST", path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %w", err)
	}

	var result QueryResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func (d *databasesClient) Update(ctx context.Context, databaseID string, req *UpdateDatabaseRequest) (*Database, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if databaseID == "" {
		return nil, fmt.Errorf("databaseID cannot be empty")
	}
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	path := fmt.Sprintf("/databases/%s", databaseID)

	respBody, err := d.client.doRequest(ctx, "PATCH", path, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update database: %w", err)
	}

	var database Database
	if err := json.Unmarshal(respBody, &database); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &database, nil
}
