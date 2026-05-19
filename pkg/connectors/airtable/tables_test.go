package airtable

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fluxorio/fluxor/pkg/config"
	"golang.org/x/time/rate"
)

func TestTablesClient_List(t *testing.T) {
	// Create mock server
	mockResponse := ListTablesResponse{
		Tables: []Table{
			{
				ID:           "tblABC123",
				Name:         "Tasks",
				PrimaryField: "fldPrimary",
				Fields: []Field{
					{ID: "fld1", Name: "Name", Type: "singleLineText"},
					{ID: "fld2", Name: "Status", Type: "singleSelect"},
				},
			},
			{
				ID:           "tblXYZ456",
				Name:         "Projects",
				PrimaryField: "fldPrimary2",
				Fields: []Field{
					{ID: "fld3", Name: "Title", Type: "singleLineText"},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer keyTEST123" {
			t.Errorf("Invalid authorization header: %s", r.Header.Get("Authorization"))
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Create client with mock server
	client := &airtableClient{
		config: Config{
			BaseConfig: *config.NewBaseConfig(),
			APIKey:     "keyTEST123",
			BaseID:     "appTEST456",
		},
		httpClient:  server.Client(),
		rateLimiter: rate.NewLimiter(rate.Inf, 1),
	}

	// Override base URL for testing
	originalBaseURL := baseURL
	baseURL = server.URL
	defer func() { baseURL = originalBaseURL }()

	tablesClient := &tablesClient{client: client}

	// Test List
	ctx := context.Background()
	tables, err := tablesClient.List(ctx)
	if err != nil {
		t.Fatalf("List() unexpected error: %v", err)
	}

	if len(tables) != 2 {
		t.Errorf("List() returned %d tables, want 2", len(tables))
	}

	if tables[0].Name != "Tasks" {
		t.Errorf("List() tables[0].Name = %s, want 'Tasks'", tables[0].Name)
	}
	if tables[1].Name != "Projects" {
		t.Errorf("List() tables[1].Name = %s, want 'Projects'", tables[1].Name)
	}
}

func TestTablesClient_List_NilContext(t *testing.T) {
	client := &airtableClient{}
	tablesClient := &tablesClient{client: client}

	_, err := tablesClient.List(nil)
	if err == nil {
		t.Error("List() expected error with nil context, got nil")
	}
}

func TestTablesClient_Get(t *testing.T) {
	// Create mock server
	mockResponse := ListTablesResponse{
		Tables: []Table{
			{
				ID:           "tblABC123",
				Name:         "Tasks",
				PrimaryField: "fldPrimary",
				Fields: []Field{
					{ID: "fld1", Name: "Name", Type: "singleLineText"},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Create client with mock server
	client := &airtableClient{
		config: Config{
			BaseConfig: *config.NewBaseConfig(),
			APIKey:     "keyTEST123",
			BaseID:     "appTEST456",
		},
		httpClient:  server.Client(),
		rateLimiter: rate.NewLimiter(rate.Inf, 1),
	}

	// Override base URL for testing
	originalBaseURL := baseURL
	baseURL = server.URL
	defer func() { baseURL = originalBaseURL }()

	tablesClient := &tablesClient{client: client}

	// Test Get by ID
	ctx := context.Background()
	table, err := tablesClient.Get(ctx, "tblABC123")
	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}

	if table.Name != "Tasks" {
		t.Errorf("Get() table.Name = %s, want 'Tasks'", table.Name)
	}
	if table.ID != "tblABC123" {
		t.Errorf("Get() table.ID = %s, want 'tblABC123'", table.ID)
	}
}

func TestTablesClient_Get_ByName(t *testing.T) {
	// Create mock server
	mockResponse := ListTablesResponse{
		Tables: []Table{
			{
				ID:           "tblABC123",
				Name:         "Tasks",
				PrimaryField: "fldPrimary",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Create client
	client := &airtableClient{
		config: Config{
			BaseConfig: *config.NewBaseConfig(),
			APIKey:     "keyTEST123",
			BaseID:     "appTEST456",
		},
		httpClient:  server.Client(),
		rateLimiter: rate.NewLimiter(rate.Inf, 1),
	}

	originalBaseURL := baseURL
	baseURL = server.URL
	defer func() { baseURL = originalBaseURL }()

	tablesClient := &tablesClient{client: client}

	// Test Get by name
	ctx := context.Background()
	table, err := tablesClient.Get(ctx, "Tasks")
	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}

	if table.Name != "Tasks" {
		t.Errorf("Get() table.Name = %s, want 'Tasks'", table.Name)
	}
}

func TestTablesClient_Get_NotFound(t *testing.T) {
	// Create mock server
	mockResponse := ListTablesResponse{
		Tables: []Table{
			{ID: "tblABC123", Name: "Tasks"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	client := &airtableClient{
		config: Config{
			BaseConfig: *config.NewBaseConfig(),
			APIKey:     "keyTEST123",
			BaseID:     "appTEST456",
		},
		httpClient:  server.Client(),
		rateLimiter: rate.NewLimiter(rate.Inf, 1),
	}

	originalBaseURL := baseURL
	baseURL = server.URL
	defer func() { baseURL = originalBaseURL }()

	tablesClient := &tablesClient{client: client}

	// Test Get with non-existent table
	ctx := context.Background()
	_, err := tablesClient.Get(ctx, "NonExistent")
	if err == nil {
		t.Error("Get() expected error for non-existent table, got nil")
	}
}

func TestTablesClient_Get_EmptyTableID(t *testing.T) {
	client := &airtableClient{}
	tablesClient := &tablesClient{client: client}

	ctx := context.Background()
	_, err := tablesClient.Get(ctx, "")
	if err == nil {
		t.Error("Get() expected error with empty tableID, got nil")
	}
}
