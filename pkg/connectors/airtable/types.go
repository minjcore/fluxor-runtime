package airtable

import "context"

// Client is the main Airtable client interface
type Client interface {
	Tables() TablesClient
	Records() RecordsClient
}

// TablesClient provides operations for Airtable tables
type TablesClient interface {
	// List returns all tables in the base
	List(ctx context.Context) ([]Table, error)

	// Get returns metadata for a specific table
	Get(ctx context.Context, tableID string) (*Table, error)
}

// RecordsClient provides CRUD operations for Airtable records
type RecordsClient interface {
	// Create creates a new record in the specified table
	Create(ctx context.Context, tableID string, record *Record) (*Record, error)

	// Get retrieves a single record by ID
	Get(ctx context.Context, tableID, recordID string) (*Record, error)

	// Update updates an existing record
	Update(ctx context.Context, tableID, recordID string, record *Record) (*Record, error)

	// Delete deletes a record by ID
	Delete(ctx context.Context, tableID, recordID string) error

	// List retrieves multiple records with optional filtering
	List(ctx context.Context, tableID string, params ListParams) ([]Record, error)
}

// Table represents an Airtable table with its metadata
type Table struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	PrimaryField string  `json:"primaryFieldId"`
	Fields       []Field `json:"fields"`
}

// Field represents a table field definition
type Field struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// Record represents an Airtable record
type Record struct {
	ID          string                 `json:"id,omitempty"`
	Fields      map[string]interface{} `json:"fields"`
	CreatedTime string                 `json:"createdTime,omitempty"`
}

// ListParams contains parameters for listing records
type ListParams struct {
	// MaxRecords limits the number of records returned (max 100)
	MaxRecords int `json:"maxRecords,omitempty"`

	// PageSize specifies the number of records per page (1-100)
	PageSize int `json:"pageSize,omitempty"`

	// View specifies the view to use for filtering
	View string `json:"view,omitempty"`

	// FilterByFormula allows filtering using Airtable formula syntax
	FilterByFormula string `json:"filterByFormula,omitempty"`

	// Sort specifies sort order
	Sort []SortParam `json:"sort,omitempty"`

	// Offset for pagination (returned from previous request)
	Offset string `json:"offset,omitempty"`
}

// SortParam specifies sorting criteria
type SortParam struct {
	Field     string `json:"field"`
	Direction string `json:"direction"` // "asc" or "desc"
}

// ListRecordsResponse represents the response from listing records
type ListRecordsResponse struct {
	Records []Record `json:"records"`
	Offset  string   `json:"offset,omitempty"`
}

// ListTablesResponse represents the response from listing tables
type ListTablesResponse struct {
	Tables []Table `json:"tables"`
}
