package airtable

import (
	"context"
	"encoding/json"
	"fmt"
)

// tablesClient implements the TablesClient interface
type tablesClient struct {
	client *airtableClient
}

// List returns all tables in the base
func (t *tablesClient) List(ctx context.Context) ([]Table, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	// Airtable Meta API endpoint
	path := fmt.Sprintf("/meta/bases/%s/tables", t.client.config.BaseID)

	// Perform request
	respBody, err := t.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	// Parse response
	var response ListTablesResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Tables, nil
}

// Get returns metadata for a specific table
func (t *tablesClient) Get(ctx context.Context, tableID string) (*Table, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if tableID == "" {
		return nil, fmt.Errorf("tableID cannot be empty")
	}

	// List all tables and find the matching one
	tables, err := t.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get table: %w", err)
	}

	// Find table by ID
	for _, table := range tables {
		if table.ID == tableID || table.Name == tableID {
			return &table, nil
		}
	}

	return nil, fmt.Errorf("table not found: %s", tableID)
}
