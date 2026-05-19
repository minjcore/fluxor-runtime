package persistence

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestSQLInjection_FieldNames tests that malicious field names are rejected
func TestSQLInjection_FieldNames(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultConfig("test_table", db)
	repo, err := NewSQLRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// Test malicious field names in filters
	maliciousFields := []string{
		"'; DROP TABLE test_table; --",
		"name'; DELETE FROM test_table; --",
		"name) UNION SELECT * FROM users--",
		"name; INSERT INTO test_table VALUES (1, 'hack')--",
		"name OR 1=1--",
		"name' OR '1'='1",
		"name; UPDATE test_table SET name='hack'--",
		"name\"; DROP TABLE test_table; --",
		"name`; DROP TABLE test_table; --",
		"name\nDELETE FROM test_table",
		"name\tDROP TABLE test_table",
		"name (SELECT * FROM users)",
		"name;--",
		"name/*",
		"name*/",
	}

	for _, maliciousField := range maliciousFields {
		t.Run("filter_"+maliciousField, func(t *testing.T) {
			query := NewQuery().WithFilter(maliciousField, "value")
			_, err := repo.FindAll(ctx, query)
			if err == nil {
				t.Errorf("Expected error for malicious field name: %s", maliciousField)
			}
		})
	}

	// Test malicious field names in order by
	for _, maliciousField := range maliciousFields {
		t.Run("orderby_"+maliciousField, func(t *testing.T) {
			query := NewQuery().WithOrderBy(maliciousField, "ASC")
			_, err := repo.FindAll(ctx, query)
			if err == nil {
				t.Errorf("Expected error for malicious field name in ORDER BY: %s", maliciousField)
			}
		})
	}
}

// TestSQLInjection_TableNames tests that malicious table names are rejected
func TestSQLInjection_TableNames(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	maliciousTableNames := []string{
		"test_table; DROP TABLE users; --",
		"test_table' UNION SELECT * FROM users--",
		"test_table; DELETE FROM users--",
		"test_table (SELECT * FROM users)",
		"test_table\nDROP TABLE users",
	}

	for _, maliciousTable := range maliciousTableNames {
		t.Run("table_"+maliciousTable, func(t *testing.T) {
			config := DefaultConfig(maliciousTable, db)
			_, err := NewSQLRepository(config)
			if err == nil {
				t.Errorf("Expected error for malicious table name: %s", maliciousTable)
			}
		})
	}
}

// TestSQLInjection_SelectFields tests that malicious select fields are rejected
func TestSQLInjection_SelectFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultConfig("test_table", db)
	repo, err := NewSQLRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	maliciousFields := []string{
		"*; DROP TABLE test_table; --",
		"name; DELETE FROM test_table--",
		"name) UNION SELECT password FROM users--",
	}

	for _, maliciousField := range maliciousFields {
		t.Run("select_"+maliciousField, func(t *testing.T) {
			query := NewQuery().WithSelectFields(maliciousField)
			_, err := repo.FindAll(ctx, query)
			if err == nil {
				t.Errorf("Expected error for malicious select field: %s", maliciousField)
			}
		})
	}
}

// TestInputValidation_ValidIdentifiers tests that valid identifiers are accepted
func TestInputValidation_ValidIdentifiers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultConfig("test_table", db)
	repo, err := NewSQLRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// Valid identifiers should work
	validFields := []string{
		"name",
		"user_id",
		"created_at",
		"status",
		"email_address",
		"id",
		"_internal",
		"field123",
	}

	for _, validField := range validFields {
		t.Run("valid_"+validField, func(t *testing.T) {
			query := NewQuery().WithFilter(validField, "value")
			_, err := repo.FindAll(ctx, query)
			// Should not error due to validation (may error due to no data, which is OK)
			if err != nil {
				// Check if it's a validation error
				if e, ok := err.(*Error); ok && e.Code == "INVALID_INPUT" {
					t.Errorf("Valid field name rejected: %s, error: %v", validField, err)
				}
				// Other errors (like no data) are acceptable
			}
		})
	}
}

// TestSQLValidation_ValidateSQLIdentifier tests the validation function directly
func TestSQLValidation_ValidateSQLIdentifier(t *testing.T) {
	tests := []struct {
		name      string
		identifier string
		wantErr   bool
	}{
		{"valid simple", "name", false},
		{"valid with underscore", "user_id", false},
		{"valid with numbers", "field123", false},
		{"valid starts with underscore", "_internal", false},
		{"empty", "", true},
		{"semicolon", "name; DROP TABLE", true},
		{"comment", "name--", true},
		{"block comment", "name/*", true},
		{"single quote", "name'", true},
		{"double quote", "name\"", true},
		{"backtick", "name`", true},
		{"parentheses", "name()", true},
		{"space", "name field", true},
		{"newline", "name\nfield", true},
		{"tab", "name\tfield", true},
		{"starts with number", "123field", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSQLIdentifier(tt.identifier)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSQLIdentifier(%q) error = %v, wantErr %v", tt.identifier, err, tt.wantErr)
			}
		})
	}
}

// TestSQLValidation_QuoteSQLIdentifier tests identifier quoting
func TestSQLValidation_QuoteSQLIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		wantQuoted string
	}{
		{"simple", "name", `"name"`},
		{"with underscore", "user_id", `"user_id"`},
		{"already quoted", `"name"`, `"name"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quoted := QuoteSQLIdentifier(tt.identifier)
			if quoted != tt.wantQuoted {
				t.Errorf("QuoteSQLIdentifier(%q) = %q, want %q", tt.identifier, quoted, tt.wantQuoted)
			}
		})
	}
}

// TestSQLInjection_CreateEntity tests that malicious field names in Create are rejected
func TestSQLInjection_CreateEntity(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultConfig("test_table", db)
	repo, err := NewSQLRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	maliciousFields := []string{
		"name'; DROP TABLE test_table; --",
		"name; DELETE FROM test_table--",
		"name) VALUES (1, 'hack')--",
	}

	for _, maliciousField := range maliciousFields {
		t.Run("create_"+maliciousField, func(t *testing.T) {
			entity := map[string]interface{}{
				maliciousField: "value",
			}
			err := repo.Create(ctx, entity)
			if err == nil {
				t.Errorf("Expected error for malicious field name in Create: %s", maliciousField)
			}
		})
	}
}

// TestSQLInjection_UpdateEntity tests that malicious field names in Update are rejected
func TestSQLInjection_UpdateEntity(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultConfig("test_table", db)
	repo, err := NewSQLRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	maliciousFields := []string{
		"name'; DROP TABLE test_table; --",
		"name; DELETE FROM test_table--",
		"name = 'hack'; UPDATE test_table SET name='hack'--",
	}

	for _, maliciousField := range maliciousFields {
		t.Run("update_"+maliciousField, func(t *testing.T) {
			entity := map[string]interface{}{
				maliciousField: "value",
			}
			err := repo.Update(ctx, 1, entity)
			if err == nil {
				t.Errorf("Expected error for malicious field name in Update: %s", maliciousField)
			}
		})
	}
}
