package persistence

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestNewSQLRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultConfig("test_table", db)
	repo, err := NewSQLRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	if repo == nil {
		t.Fatal("Repository is nil")
	}
}

func TestNewSQLRepository_InvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "empty table name",
			config: Config{
				TableName:      "",
				IDField:        "id",
				ConnectionPool: setupTestDB(t),
			},
		},
		{
			name: "nil connection pool",
			config: Config{
				TableName:      "test_table",
				IDField:        "id",
				ConnectionPool: nil,
			},
		},
		{
			name: "empty ID field",
			config: Config{
				TableName:      "test_table",
				IDField:        "",
				ConnectionPool: setupTestDB(t),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.ConnectionPool != nil {
				defer tt.config.ConnectionPool.Close()
			}
			repo, err := NewSQLRepository(tt.config)
			if err == nil {
				if repo != nil {
					repo.Close()
				}
				t.Fatal("Expected error for invalid config")
			}
		})
	}
}

func TestQueryBuilder(t *testing.T) {
	query := NewQuery()
	if query == nil {
		t.Fatal("Query is nil")
	}

	query.
		WithFilter("status", "active").
		WithFilter("age", 25).
		WithOrderBy("created_at", "DESC").
		WithLimit(10).
		WithOffset(0).
		WithSelectFields("id", "name", "email")

	if len(query.Filters) != 2 {
		t.Errorf("Expected 2 filters, got %d", len(query.Filters))
	}

	if len(query.OrderBy) != 1 {
		t.Errorf("Expected 1 order by, got %d", len(query.OrderBy))
	}

	if query.Limit != 10 {
		t.Errorf("Expected limit 10, got %d", query.Limit)
	}

	if query.Offset != 0 {
		t.Errorf("Expected offset 0, got %d", query.Offset)
	}

	if len(query.SelectFields) != 3 {
		t.Errorf("Expected 3 select fields, got %d", len(query.SelectFields))
	}
}

func TestQueryBuilder_InvalidDirection(t *testing.T) {
	query := NewQuery().WithOrderBy("field", "INVALID")
	
	if query.OrderBy["field"] != "ASC" {
		t.Errorf("Expected ASC for invalid direction, got %s", query.OrderBy["field"])
	}
}

func TestConfig_Validate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				TableName:      "test_table",
				IDField:        "id",
				ConnectionPool: db,
				MaxRetries:     3,
				RetryDelay:     100 * time.Millisecond,
				QueryTimeout:   30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty table name",
			config: Config{
				TableName:      "",
				IDField:        "id",
				ConnectionPool: db,
			},
			wantErr: true,
		},
		{
			name: "nil connection pool",
			config: Config{
				TableName:      "test_table",
				IDField:        "id",
				ConnectionPool: nil,
			},
			wantErr: true,
		},
		{
			name: "empty ID field",
			config: Config{
				TableName:      "test_table",
				IDField:        "",
				ConnectionPool: db,
			},
			wantErr: true,
		},
		{
			name: "negative max retries",
			config: Config{
				TableName:      "test_table",
				IDField:        "id",
				ConnectionPool: db,
				MaxRetries:     -1,
			},
			wantErr: true,
		},
		{
			name: "negative retry delay",
			config: Config{
				TableName:      "test_table",
				IDField:        "id",
				ConnectionPool: db,
				RetryDelay:     -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero query timeout",
			config: Config{
				TableName:      "test_table",
				IDField:        "id",
				ConnectionPool: db,
				QueryTimeout:   0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestError_Error(t *testing.T) {
	err := NewError(ErrCodeNotFound, "entity not found", nil)
	if err.Error() == "" {
		t.Fatal("Error message is empty")
	}

	if err.Code != ErrCodeNotFound {
		t.Errorf("Expected code %s, got %s", ErrCodeNotFound, err.Code)
	}
}

func TestError_WithCause(t *testing.T) {
	cause := sql.ErrNoRows
	err := NewError(ErrCodeNotFound, "entity not found", cause)
	
	if err.Unwrap() != cause {
		t.Errorf("Expected cause %v, got %v", cause, err.Unwrap())
	}
}

func TestIsNotFound(t *testing.T) {
	err := NewError(ErrCodeNotFound, "not found", nil)
	if !IsNotFound(err) {
		t.Error("Expected IsNotFound to return true")
	}

	otherErr := NewError(ErrCodeQuery, "query error", nil)
	if IsNotFound(otherErr) {
		t.Error("Expected IsNotFound to return false")
	}
}

func TestIsTransactionError(t *testing.T) {
	err := NewError(ErrCodeTransaction, "transaction error", nil)
	if !IsTransactionError(err) {
		t.Error("Expected IsTransactionError to return true")
	}

	otherErr := NewError(ErrCodeQuery, "query error", nil)
	if IsTransactionError(otherErr) {
		t.Error("Expected IsTransactionError to return false")
	}
}

// Helper function to set up test database
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS test_table (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			email TEXT,
			status TEXT,
			age INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create test table: %v", err)
	}

	return db
}

func TestDefaultConfig(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultConfig("test_table", db)

	if config.TableName != "test_table" {
		t.Errorf("Expected table name 'test_table', got '%s'", config.TableName)
	}

	if config.IDField != "id" {
		t.Errorf("Expected ID field 'id', got '%s'", config.IDField)
	}

	if config.ConnectionPool != db {
		t.Error("Connection pool mismatch")
	}

	if config.MaxRetries != 3 {
		t.Errorf("Expected max retries 3, got %d", config.MaxRetries)
	}

	if config.QueryTimeout != 30*time.Second {
		t.Errorf("Expected query timeout 30s, got %v", config.QueryTimeout)
	}
}

func TestRepository_Close(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultConfig("test_table", db)
	repo, err := NewSQLRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Close should not error
	err = repo.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestRepository_FindByID_NilContext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultConfig("test_table", db)
	repo, err := NewSQLRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Should panic on nil context (fail-fast)
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil context")
		}
	}()

	repo.FindByID(nil, 1)
}

func TestRepository_FindByID_NilID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultConfig("test_table", db)
	repo, err := NewSQLRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Should panic on nil ID (fail-fast)
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil ID")
		}
	}()

	repo.FindByID(context.Background(), nil)
}
