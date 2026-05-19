package airtable

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// recordsClient implements the RecordsClient interface
type recordsClient struct {
	client *airtableClient
}

// Create creates a new record in the specified table
func (r *recordsClient) Create(ctx context.Context, tableID string, record *Record) (*Record, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if tableID == "" {
		return nil, fmt.Errorf("tableID cannot be empty")
	}
	if record == nil {
		return nil, fmt.Errorf("record cannot be nil")
	}

	// Build path
	path := fmt.Sprintf("/%s/%s", r.client.config.BaseID, tableID)

	// Prepare request body
	requestBody := map[string]interface{}{
		"fields": record.Fields,
	}

	// Perform request
	respBody, err := r.client.doRequest(ctx, "POST", path, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}

	// Parse response
	var createdRecord Record
	if err := json.Unmarshal(respBody, &createdRecord); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &createdRecord, nil
}

// Get retrieves a single record by ID
func (r *recordsClient) Get(ctx context.Context, tableID, recordID string) (*Record, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if tableID == "" {
		return nil, fmt.Errorf("tableID cannot be empty")
	}
	if recordID == "" {
		return nil, fmt.Errorf("recordID cannot be empty")
	}

	// Build path
	path := fmt.Sprintf("/%s/%s/%s", r.client.config.BaseID, tableID, recordID)

	// Perform request
	respBody, err := r.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get record: %w", err)
	}

	// Parse response
	var record Record
	if err := json.Unmarshal(respBody, &record); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &record, nil
}

// Update updates an existing record
func (r *recordsClient) Update(ctx context.Context, tableID, recordID string, record *Record) (*Record, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if tableID == "" {
		return nil, fmt.Errorf("tableID cannot be empty")
	}
	if recordID == "" {
		return nil, fmt.Errorf("recordID cannot be empty")
	}
	if record == nil {
		return nil, fmt.Errorf("record cannot be nil")
	}

	// Build path
	path := fmt.Sprintf("/%s/%s/%s", r.client.config.BaseID, tableID, recordID)

	// Prepare request body
	requestBody := map[string]interface{}{
		"fields": record.Fields,
	}

	// Perform request (PATCH for partial update)
	respBody, err := r.client.doRequest(ctx, "PATCH", path, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to update record: %w", err)
	}

	// Parse response
	var updatedRecord Record
	if err := json.Unmarshal(respBody, &updatedRecord); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &updatedRecord, nil
}

// Delete deletes a record by ID
func (r *recordsClient) Delete(ctx context.Context, tableID, recordID string) error {
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}
	if tableID == "" {
		return fmt.Errorf("tableID cannot be empty")
	}
	if recordID == "" {
		return fmt.Errorf("recordID cannot be empty")
	}

	// Build path
	path := fmt.Sprintf("/%s/%s/%s", r.client.config.BaseID, tableID, recordID)

	// Perform request
	_, err := r.client.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	return nil
}

// List retrieves multiple records with optional filtering
func (r *recordsClient) List(ctx context.Context, tableID string, params ListParams) ([]Record, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if tableID == "" {
		return nil, fmt.Errorf("tableID cannot be empty")
	}

	// Build path with query parameters
	path := fmt.Sprintf("/%s/%s", r.client.config.BaseID, tableID)

	// Build query parameters
	query := url.Values{}
	if params.MaxRecords > 0 {
		query.Set("maxRecords", strconv.Itoa(params.MaxRecords))
	}
	if params.PageSize > 0 {
		query.Set("pageSize", strconv.Itoa(params.PageSize))
	}
	if params.View != "" {
		query.Set("view", params.View)
	}
	if params.FilterByFormula != "" {
		query.Set("filterByFormula", params.FilterByFormula)
	}
	if params.Offset != "" {
		query.Set("offset", params.Offset)
	}
	if len(params.Sort) > 0 {
		for i, sort := range params.Sort {
			query.Set(fmt.Sprintf("sort[%d][field]", i), sort.Field)
			query.Set(fmt.Sprintf("sort[%d][direction]", i), sort.Direction)
		}
	}

	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	// Perform request
	respBody, err := r.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}

	// Parse response
	var response ListRecordsResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Records, nil
}
