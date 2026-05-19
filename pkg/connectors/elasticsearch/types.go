package elasticsearch

import "context"

// Client is the main Elasticsearch client interface
type Client interface {
	// Indices returns the indices client
	Indices() IndicesClient

	// Documents returns the documents client
	Documents() DocumentsClient

	// Search returns the search client
	Search() SearchClient
}

// IndicesClient provides operations for Elasticsearch indices
type IndicesClient interface {
	// Create creates a new index
	Create(ctx context.Context, index string, settings map[string]interface{}) error

	// Delete deletes an index
	Delete(ctx context.Context, index string) error

	// Exists checks if an index exists
	Exists(ctx context.Context, index string) (bool, error)

	// GetSettings retrieves index settings
	GetSettings(ctx context.Context, index string) (map[string]interface{}, error)

	// PutMapping updates index mapping
	PutMapping(ctx context.Context, index string, mapping map[string]interface{}) error

	// List lists all indices
	List(ctx context.Context) ([]string, error)
}

// DocumentsClient provides operations for Elasticsearch documents
type DocumentsClient interface {
	// Index indexes a document (creates or updates)
	Index(ctx context.Context, index, id string, document interface{}) error

	// Get retrieves a document by ID
	Get(ctx context.Context, index, id string) (map[string]interface{}, error)

	// Delete deletes a document by ID
	Delete(ctx context.Context, index, id string) error

	// Exists checks if a document exists
	Exists(ctx context.Context, index, id string) (bool, error)

	// Update updates a document
	Update(ctx context.Context, index, id string, doc map[string]interface{}) error

	// Bulk performs bulk operations
	Bulk(ctx context.Context, operations []BulkOperation) (*BulkResponse, error)
}

// SearchClient provides search operations
type SearchClient interface {
	// Search performs a search query
	Search(ctx context.Context, index string, query map[string]interface{}) (*SearchResponse, error)

	// SearchAll searches across all indices
	SearchAll(ctx context.Context, query map[string]interface{}) (*SearchResponse, error)

	// Count counts documents matching a query
	Count(ctx context.Context, index string, query map[string]interface{}) (int64, error)
}

// BulkOperation represents a single bulk operation
type BulkOperation struct {
	// Operation type: index, create, update, delete
	Operation string

	// Index name
	Index string

	// Document ID (optional for index/create)
	ID string

	// Document data (for index, create, update)
	Document interface{}

	// Update script (for update operations)
	Script map[string]interface{}
}

// BulkResponse represents the response from a bulk operation
type BulkResponse struct {
	Took      int64                    `json:"took"`
	Errors    bool                     `json:"errors"`
	Items     []map[string]interface{} `json:"items"`
}

// SearchResponse represents a search response
type SearchResponse struct {
	Took     int64                    `json:"took"`
	TimedOut bool                     `json:"timed_out"`
	Hits     SearchHits               `json:"hits"`
	Aggregations map[string]interface{} `json:"aggregations,omitempty"`
}

// SearchHits represents search hits
type SearchHits struct {
	Total    SearchTotal              `json:"total"`
	MaxScore float64                  `json:"max_score"`
	Hits     []SearchHit              `json:"hits"`
}

// SearchTotal represents total hits
type SearchTotal struct {
	Value    int64  `json:"value"`
	Relation string `json:"relation"`
}

// SearchHit represents a single search hit
type SearchHit struct {
	Index  string                 `json:"_index"`
	ID     string                 `json:"_id"`
	Score  float64                `json:"_score"`
	Source map[string]interface{} `json:"_source"`
}
