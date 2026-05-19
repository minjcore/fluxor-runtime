package sheets

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

var (
	baseURL = "https://sheets.googleapis.com/v4/spreadsheets"
)

// sheetsClient implements the Client interface
type sheetsClient struct {
	config      Config
	httpClient  *http.Client
	rateLimiter *rate.Limiter
	spreadsheetID string
	token       string
}

// NewClient creates a new Google Sheets client with the given configuration
// Fail-fast: Returns error if configuration is invalid
func NewClient(config Config) (Client, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.GetTimeout(),
	}

	// Create rate limiter (requests per second)
	limiter := rate.NewLimiter(rate.Limit(config.RateLimit), 1)

	client := &sheetsClient{
		config:        config,
		httpClient:    httpClient,
		rateLimiter:   limiter,
		spreadsheetID: config.SpreadsheetID,
	}

	// Get authentication token
	// For now, we'll use OAuth2 token if provided, otherwise expect service account
	// In a full implementation, you would use google.golang.org/api/sheets/v4
	// with proper OAuth2 or service account authentication
	if config.OAuth2Token != "" {
		client.token = config.OAuth2Token
	} else if config.CredentialsPath != "" {
		// In a full implementation, load service account credentials
		// and get access token using google.golang.org/api/option
		// For now, we'll require OAuth2 token
		return nil, &ConfigError{
			Code:    "SERVICE_ACCOUNT_NOT_IMPLEMENTED",
			Message: "Service account authentication requires google.golang.org/api/sheets/v4. Please use OAuth2 token for now.",
		}
	}

	return client, nil
}

// Read reads values from a range in the spreadsheet
func (c *sheetsClient) Read(ctx context.Context, range_ string) ([][]interface{}, error) {
	url := fmt.Sprintf("%s/%s/values/%s", baseURL, c.spreadsheetID, range_)
	
	data, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Values [][]interface{} `json:"values"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, &SheetsError{
			Code:    "PARSE_ERROR",
			Message: fmt.Sprintf("failed to parse response: %v", err),
		}
	}

	return response.Values, nil
}

// Write writes values to a range in the spreadsheet
func (c *sheetsClient) Write(ctx context.Context, range_ string, values [][]interface{}) error {
	return c.Update(ctx, range_, values)
}

// Update updates values in a range (overwrites existing values)
func (c *sheetsClient) Update(ctx context.Context, range_ string, values [][]interface{}) error {
	url := fmt.Sprintf("%s/%s/values/%s?valueInputOption=RAW", baseURL, c.spreadsheetID, range_)
	
	body := map[string]interface{}{
		"values": values,
	}

	_, err := c.doRequest(ctx, "PUT", url, body)
	return err
}

// Append appends values to the end of a range
func (c *sheetsClient) Append(ctx context.Context, range_ string, values [][]interface{}) error {
	url := fmt.Sprintf("%s/%s/values/%s:append?valueInputOption=RAW", baseURL, c.spreadsheetID, range_)
	
	body := map[string]interface{}{
		"values": values,
	}

	_, err := c.doRequest(ctx, "POST", url, body)
	return err
}

// Clear clears values in a range
func (c *sheetsClient) Clear(ctx context.Context, range_ string) error {
	url := fmt.Sprintf("%s/%s/values/%s:clear", baseURL, c.spreadsheetID, range_)
	
	_, err := c.doRequest(ctx, "POST", url, nil)
	return err
}

// BatchRead reads multiple ranges in a single request
func (c *sheetsClient) BatchRead(ctx context.Context, ranges []string) (map[string][][]interface{}, error) {
	url := fmt.Sprintf("%s/%s/values:batchGet", baseURL, c.spreadsheetID)
	
	// Build query string with ranges
	query := ""
	for i, r := range ranges {
		if i > 0 {
			query += "&"
		}
		query += fmt.Sprintf("ranges=%s", r)
	}
	url += "?" + query

	data, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var response BatchReadResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, &SheetsError{
			Code:    "PARSE_ERROR",
			Message: fmt.Sprintf("failed to parse batch response: %v", err),
		}
	}

	result := make(map[string][][]interface{})
	for _, vr := range response.ValueRanges {
		result[vr.Range_] = vr.Values
	}

	return result, nil
}

// BatchWrite writes to multiple ranges in a single request
func (c *sheetsClient) BatchWrite(ctx context.Context, updates []BatchUpdate) error {
	url := fmt.Sprintf("%s/%s/values:batchUpdate", baseURL, c.spreadsheetID)
	
	data := map[string]interface{}{
		"valueInputOption": "RAW",
		"data":            updates,
	}

	_, err := c.doRequest(ctx, "POST", url, data)
	return err
}

// GetSpreadsheetInfo returns metadata about the spreadsheet
func (c *sheetsClient) GetSpreadsheetInfo(ctx context.Context) (*SpreadsheetInfo, error) {
	url := fmt.Sprintf("%s/%s", baseURL, c.spreadsheetID)
	
	data, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		SpreadsheetID string      `json:"spreadsheetId"`
		Properties    struct {
			Title    string `json:"title"`
			TimeZone string `json:"timeZone"`
			Locale   string `json:"locale"`
		} `json:"properties"`
		Sheets []struct {
			Properties struct {
				SheetID int    `json:"sheetId"`
				Title   string `json:"title"`
				Index   int    `json:"index"`
				GridProperties struct {
					RowCount    int `json:"rowCount"`
					ColumnCount int `json:"columnCount"`
				} `json:"gridProperties"`
			} `json:"properties"`
		} `json:"sheets"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, &SheetsError{
			Code:    "PARSE_ERROR",
			Message: fmt.Sprintf("failed to parse spreadsheet info: %v", err),
		}
	}

	info := &SpreadsheetInfo{
		ID:       response.SpreadsheetID,
		Title:    response.Properties.Title,
		TimeZone: response.Properties.TimeZone,
		Locale:   response.Properties.Locale,
		URL:      fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s", response.SpreadsheetID),
		Sheets:   make([]SheetInfo, 0, len(response.Sheets)),
	}

	for _, s := range response.Sheets {
		info.Sheets = append(info.Sheets, SheetInfo{
			ID:          s.Properties.SheetID,
			Title:       s.Properties.Title,
			Index:       s.Properties.Index,
			RowCount:    s.Properties.GridProperties.RowCount,
			ColumnCount: s.Properties.GridProperties.ColumnCount,
			GridProperties: &GridProperties{
				RowCount:    s.Properties.GridProperties.RowCount,
				ColumnCount: s.Properties.GridProperties.ColumnCount,
			},
		})
	}

	return info, nil
}

// GetSheetInfo returns metadata about a specific sheet
func (c *sheetsClient) GetSheetInfo(ctx context.Context, sheetName string) (*SheetInfo, error) {
	info, err := c.GetSpreadsheetInfo(ctx)
	if err != nil {
		return nil, err
	}

	for _, sheet := range info.Sheets {
		if sheet.Title == sheetName {
			return &sheet, nil
		}
	}

	return nil, &SheetsError{
		Code:    "SHEET_NOT_FOUND",
		Message: fmt.Sprintf("sheet '%s' not found", sheetName),
	}
}

// doRequest performs an HTTP request with rate limiting and retries
func (c *sheetsClient) doRequest(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Prepare request body
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Content-Type", "application/json")

	// Perform request with retries
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		// Check status code
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, nil
		}

		// Handle error responses
		var apiError struct {
			Error struct {
				Code    string      `json:"code"`
				Message string      `json:"message"`
				Details interface{} `json:"details"`
			} `json:"error"`
		}
		if err := json.Unmarshal(respBody, &apiError); err == nil && apiError.Error.Code != "" {
			// Retry on rate limit or server errors
			if resp.StatusCode == 429 || resp.StatusCode >= 500 {
				lastErr = &SheetsError{
					Code:    apiError.Error.Code,
					Message: apiError.Error.Message,
					Details: apiError.Error.Details,
				}
				continue
			}
			// Don't retry on client errors
			return nil, &SheetsError{
				Code:    apiError.Error.Code,
				Message: apiError.Error.Message,
				Details: apiError.Error.Details,
			}
		}

		// Fallback error
		lastErr = fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
		if resp.StatusCode >= 500 {
			continue
		}
		return nil, lastErr
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", c.config.MaxRetries, lastErr)
}
