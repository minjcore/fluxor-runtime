package sheets

import "context"

// Client is the main Google Sheets client interface
type Client interface {
	// Read reads values from a range in the spreadsheet
	Read(ctx context.Context, range_ string) ([][]interface{}, error)

	// Write writes values to a range in the spreadsheet
	Write(ctx context.Context, range_ string, values [][]interface{}) error

	// Update updates values in a range (overwrites existing values)
	Update(ctx context.Context, range_ string, values [][]interface{}) error

	// Append appends values to the end of a range
	Append(ctx context.Context, range_ string, values [][]interface{}) error

	// Clear clears values in a range
	Clear(ctx context.Context, range_ string) error

	// BatchRead reads multiple ranges in a single request
	BatchRead(ctx context.Context, ranges []string) (map[string][][]interface{}, error)

	// BatchWrite writes to multiple ranges in a single request
	BatchWrite(ctx context.Context, updates []BatchUpdate) error

	// GetSpreadsheetInfo returns metadata about the spreadsheet
	GetSpreadsheetInfo(ctx context.Context) (*SpreadsheetInfo, error)

	// GetSheetInfo returns metadata about a specific sheet
	GetSheetInfo(ctx context.Context, sheetName string) (*SheetInfo, error)
}

// BatchUpdate represents a batch write operation
type BatchUpdate struct {
	Range_ string            `json:"range"`
	Values [][]interface{}   `json:"values"`
}

// SpreadsheetInfo contains metadata about a spreadsheet
type SpreadsheetInfo struct {
	ID        string      `json:"id"`
	Title     string      `json:"title"`
	URL       string      `json:"url"`
	Sheets    []SheetInfo `json:"sheets"`
	TimeZone  string      `json:"timeZone"`
	Locale    string      `json:"locale"`
}

// SheetInfo contains metadata about a sheet
type SheetInfo struct {
	ID           int    `json:"sheetId"`
	Title        string `json:"title"`
	Index        int    `json:"index"`
	RowCount     int    `json:"rowCount"`
	ColumnCount  int    `json:"columnCount"`
	GridProperties *GridProperties `json:"gridProperties,omitempty"`
}

// GridProperties contains grid properties of a sheet
type GridProperties struct {
	RowCount    int `json:"rowCount"`
	ColumnCount int `json:"columnCount"`
	FrozenRowCount int `json:"frozenRowCount,omitempty"`
	FrozenColumnCount int `json:"frozenColumnCount,omitempty"`
}

// ValueRange represents a range of values
type ValueRange struct {
	Range_ string          `json:"range"`
	Values [][]interface{} `json:"values"`
}

// BatchReadResponse represents the response from a batch read operation
type BatchReadResponse struct {
	ValueRanges []ValueRange `json:"valueRanges"`
}

// ConfigError represents a configuration error
type ConfigError struct {
	Code    string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}

// SheetsError represents an error from the Google Sheets API
type SheetsError struct {
	Code    string
	Message string
	Details interface{}
}

func (e *SheetsError) Error() string {
	return e.Message
}
