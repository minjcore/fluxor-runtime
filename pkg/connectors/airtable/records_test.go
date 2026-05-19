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

func TestRecordsClient_Create(t *testing.T) {
	mockRecord := Record{
		ID: "recABC123",
		Fields: map[string]interface{}{
			"Name":   "Test Task",
			"Status": "In Progress",
		},
		CreatedTime: "2024-01-01T00:00:00.000Z",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockRecord)
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

	recordsClient := &recordsClient{client: client}

	ctx := context.Background()
	newRecord := &Record{
		Fields: map[string]interface{}{
			"Name":   "Test Task",
			"Status": "In Progress",
		},
	}

	result, err := recordsClient.Create(ctx, "tblTasks", newRecord)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	if result.ID != "recABC123" {
		t.Errorf("Create() record.ID = %s, want 'recABC123'", result.ID)
	}
	if result.Fields["Name"] != "Test Task" {
		t.Errorf("Create() record.Fields[Name] = %v, want 'Test Task'", result.Fields["Name"])
	}
}

func TestRecordsClient_Get(t *testing.T) {
	mockRecord := Record{
		ID: "recABC123",
		Fields: map[string]interface{}{
			"Name": "Test Task",
		},
		CreatedTime: "2024-01-01T00:00:00.000Z",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockRecord)
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

	recordsClient := &recordsClient{client: client}

	ctx := context.Background()
	result, err := recordsClient.Get(ctx, "tblTasks", "recABC123")
	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}

	if result.ID != "recABC123" {
		t.Errorf("Get() record.ID = %s, want 'recABC123'", result.ID)
	}
}

func TestRecordsClient_Update(t *testing.T) {
	mockRecord := Record{
		ID: "recABC123",
		Fields: map[string]interface{}{
			"Name":   "Updated Task",
			"Status": "Done",
		},
		CreatedTime: "2024-01-01T00:00:00.000Z",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("Expected PATCH request, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockRecord)
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

	recordsClient := &recordsClient{client: client}

	ctx := context.Background()
	updateRecord := &Record{
		Fields: map[string]interface{}{
			"Status": "Done",
		},
	}

	result, err := recordsClient.Update(ctx, "tblTasks", "recABC123", updateRecord)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}

	if result.Fields["Status"] != "Done" {
		t.Errorf("Update() record.Fields[Status] = %v, want 'Done'", result.Fields["Status"])
	}
}

func TestRecordsClient_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE request, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"deleted": true, "id": "recABC123"}`))
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

	recordsClient := &recordsClient{client: client}

	ctx := context.Background()
	err := recordsClient.Delete(ctx, "tblTasks", "recABC123")
	if err != nil {
		t.Fatalf("Delete() unexpected error: %v", err)
	}
}

func TestRecordsClient_List(t *testing.T) {
	mockResponse := ListRecordsResponse{
		Records: []Record{
			{
				ID:     "rec1",
				Fields: map[string]interface{}{"Name": "Task 1"},
			},
			{
				ID:     "rec2",
				Fields: map[string]interface{}{"Name": "Task 2"},
			},
		},
		Offset: "nextPageToken",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		// Check query parameters
		query := r.URL.Query()
		if query.Get("maxRecords") != "10" {
			t.Errorf("Expected maxRecords=10, got %s", query.Get("maxRecords"))
		}

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

	recordsClient := &recordsClient{client: client}

	ctx := context.Background()
	params := ListParams{
		MaxRecords: 10,
	}

	records, err := recordsClient.List(ctx, "tblTasks", params)
	if err != nil {
		t.Fatalf("List() unexpected error: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("List() returned %d records, want 2", len(records))
	}
}

func TestRecordsClient_List_WithSort(t *testing.T) {
	mockResponse := ListRecordsResponse{
		Records: []Record{
			{ID: "rec1", Fields: map[string]interface{}{"Name": "Task 1"}},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("sort[0][field]") != "Name" {
			t.Errorf("Expected sort field Name, got %s", query.Get("sort[0][field]"))
		}
		if query.Get("sort[0][direction]") != "asc" {
			t.Errorf("Expected sort direction asc, got %s", query.Get("sort[0][direction]"))
		}

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

	recordsClient := &recordsClient{client: client}

	ctx := context.Background()
	params := ListParams{
		Sort: []SortParam{
			{Field: "Name", Direction: "asc"},
		},
	}

	_, err := recordsClient.List(ctx, "tblTasks", params)
	if err != nil {
		t.Fatalf("List() unexpected error: %v", err)
	}
}

func TestRecordsClient_ValidationErrors(t *testing.T) {
	client := &airtableClient{}
	recordsClient := &recordsClient{client: client}

	ctx := context.Background()

	// Test Create validation
	_, err := recordsClient.Create(nil, "tbl", &Record{})
	if err == nil {
		t.Error("Create() expected error with nil context")
	}

	_, err = recordsClient.Create(ctx, "", &Record{})
	if err == nil {
		t.Error("Create() expected error with empty tableID")
	}

	_, err = recordsClient.Create(ctx, "tbl", nil)
	if err == nil {
		t.Error("Create() expected error with nil record")
	}

	// Test Get validation
	_, err = recordsClient.Get(ctx, "", "rec")
	if err == nil {
		t.Error("Get() expected error with empty tableID")
	}

	_, err = recordsClient.Get(ctx, "tbl", "")
	if err == nil {
		t.Error("Get() expected error with empty recordID")
	}

	// Test Update validation
	_, err = recordsClient.Update(ctx, "", "rec", &Record{})
	if err == nil {
		t.Error("Update() expected error with empty tableID")
	}

	// Test Delete validation
	err = recordsClient.Delete(ctx, "", "rec")
	if err == nil {
		t.Error("Delete() expected error with empty tableID")
	}

	// Test List validation
	_, err = recordsClient.List(ctx, "", ListParams{})
	if err == nil {
		t.Error("List() expected error with empty tableID")
	}
}
