package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Repository provides a generic interface for data persistence operations
// This interface abstracts database operations and can be implemented
// by various backends (SQL, NoSQL, etc.)
type Repository interface {
	// FindByID retrieves an entity by its ID
	FindByID(ctx context.Context, id interface{}) (interface{}, error)

	// FindAll retrieves all entities matching the query
	FindAll(ctx context.Context, query *Query) ([]interface{}, error)

	// FindOne retrieves a single entity matching the query
	FindOne(ctx context.Context, query *Query) (interface{}, error)

	// Create persists a new entity
	Create(ctx context.Context, entity interface{}) error

	// Update updates an existing entity
	Update(ctx context.Context, id interface{}, entity interface{}) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id interface{}) error

	// Count returns the number of entities matching the query
	Count(ctx context.Context, query *Query) (int64, error)

	// Exists checks if an entity with the given ID exists
	Exists(ctx context.Context, id interface{}) (bool, error)

	// BatchCreate persists multiple entities in a single operation
	// More efficient than multiple Create() calls
	BatchCreate(ctx context.Context, entities []interface{}) error

	// BatchUpdate updates multiple entities in a single operation
	// entities should be a slice of maps with "id" field and update fields
	// More efficient than multiple Update() calls
	BatchUpdate(ctx context.Context, entities []interface{}) error

	// BatchDelete removes multiple entities by IDs in a single operation
	// More efficient than multiple Delete() calls
	BatchDelete(ctx context.Context, ids []interface{}) error

	// BeginTransaction starts a new transaction
	BeginTransaction(ctx context.Context) (Transaction, error)

	// Close closes the repository and releases resources
	Close() error
}

// Transaction represents a database transaction
type Transaction interface {
	// Commit commits the transaction
	Commit() error

	// Rollback rolls back the transaction
	Rollback() error

	// Repository returns a repository bound to this transaction
	Repository() Repository

	// Savepoint creates a savepoint (for nested transaction-like behavior)
	// Returns the savepoint name that can be used with RollbackToSavepoint
	Savepoint(ctx context.Context, name string) error

	// RollbackToSavepoint rolls back to a previously created savepoint
	// The savepoint remains active after rollback
	RollbackToSavepoint(ctx context.Context, name string) error

	// ReleaseSavepoint releases a savepoint
	// After release, the savepoint cannot be rolled back to
	ReleaseSavepoint(ctx context.Context, name string) error
}

// Query represents a database query with filters, sorting, and pagination
type Query struct {
	// Filters are key-value pairs for WHERE clauses
	Filters map[string]interface{}

	// OrderBy specifies sorting (field -> direction: "ASC" or "DESC")
	OrderBy map[string]string

	// Limit limits the number of results
	Limit int

	// Offset specifies the offset for pagination
	Offset int

	// SelectFields specifies which fields to select (empty = all)
	SelectFields []string
}

// NewQuery creates a new query builder
func NewQuery() *Query {
	return &Query{
		Filters:      make(map[string]interface{}),
		OrderBy:      make(map[string]string),
		SelectFields: []string{},
	}
}

// WithFilter adds a filter condition
func (q *Query) WithFilter(field string, value interface{}) *Query {
	q.Filters[field] = value
	return q
}

// WithOrderBy adds an ordering clause
func (q *Query) WithOrderBy(field string, direction string) *Query {
	if direction != "ASC" && direction != "DESC" {
		direction = "ASC"
	}
	q.OrderBy[field] = direction
	return q
}

// WithLimit sets the limit
func (q *Query) WithLimit(limit int) *Query {
	q.Limit = limit
	return q
}

// WithOffset sets the offset
func (q *Query) WithOffset(offset int) *Query {
	q.Offset = offset
	return q
}

// WithSelectFields sets the fields to select
func (q *Query) WithSelectFields(fields ...string) *Query {
	q.SelectFields = fields
	return q
}

// Config configures a persistence repository
type Config struct {
	// TableName is the database table name
	TableName string

	// IDField is the name of the ID field (default: "id")
	IDField string

	// ConnectionPool is the database connection pool
	ConnectionPool *sql.DB

	// MaxRetries is the maximum number of retries for failed operations
	MaxRetries int

	// RetryDelay is the delay between retries
	RetryDelay time.Duration

	// QueryTimeout is the timeout for queries
	QueryTimeout time.Duration
}

// DefaultConfig returns a default configuration
func DefaultConfig(tableName string, pool *sql.DB) Config {
	return Config{
		TableName:      tableName,
		IDField:        "id",
		ConnectionPool: pool,
		MaxRetries:     3,
		RetryDelay:     100 * time.Millisecond,
		QueryTimeout:   30 * time.Second,
	}
}

// Validate validates the configuration
// Fail-fast: Returns error if configuration is invalid
func (c *Config) Validate() error {
	if c.TableName == "" {
		return &Error{Code: "INVALID_CONFIG", Message: "TableName cannot be empty"}
	}
	// Validate table name to prevent SQL injection
	if err := ValidateTableName(c.TableName); err != nil {
		return &Error{Code: "INVALID_CONFIG", Message: fmt.Sprintf("TableName is invalid: %v", err)}
	}
	if c.ConnectionPool == nil {
		return &Error{Code: "INVALID_CONFIG", Message: "ConnectionPool cannot be nil"}
	}
	if c.IDField == "" {
		return &Error{Code: "INVALID_CONFIG", Message: "IDField cannot be empty"}
	}
	// Validate ID field name to prevent SQL injection
	if err := ValidateFieldName(c.IDField); err != nil {
		return &Error{Code: "INVALID_CONFIG", Message: fmt.Sprintf("IDField is invalid: %v", err)}
	}
	if c.MaxRetries < 0 {
		return &Error{Code: "INVALID_CONFIG", Message: "MaxRetries cannot be negative"}
	}
	if c.RetryDelay < 0 {
		return &Error{Code: "INVALID_CONFIG", Message: "RetryDelay cannot be negative"}
	}
	if c.QueryTimeout <= 0 {
		return &Error{Code: "INVALID_CONFIG", Message: "QueryTimeout must be positive"}
	}
	return nil
}
