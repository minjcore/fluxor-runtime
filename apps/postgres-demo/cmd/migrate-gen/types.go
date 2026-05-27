package main

import (
	"time"
)

// ============================================================================
// TYPE DEFINITIONS
// ============================================================================

// ModelInfo represents metadata about a Go model struct
type ModelInfo struct {
	Name        string          // Struct name (e.g., "Approval")
	TableName   string          // Database table name (e.g., "approvals")
	Comment     string          // Struct comment/documentation
	Fields      []FieldInfo     // Struct fields
	PrimaryKey  []string        // Primary key column names
	Indexes     []IndexInfo     // Index definitions
	ForeignKeys []ForeignKeyInfo // Foreign key constraints
}

// FieldInfo represents metadata about a struct field
type FieldInfo struct {
	Name         string    // Go field name (e.g., "UserID")
	ColumnName   string    // Database column name (e.g., "user_id")
	GoType       string   // Go type (e.g., "string", "*string", "int", "time.Time")
	SQLType      string   // PostgreSQL type (e.g., "VARCHAR(255)", "INTEGER")
	IsNullable   bool     // Whether column is nullable
	IsPrimaryKey bool    // Whether this field is part of primary key
	DefaultValue string   // Default value (if any)
	Comment      string   // Field comment/documentation
	Tag          string   // Raw struct tag
}

// IndexInfo represents a database index
type IndexInfo struct {
	Name    string   // Index name
	Columns []string // Column names
	Unique  bool     // Whether index is unique
}

// ForeignKeyInfo represents a foreign key constraint
type ForeignKeyInfo struct {
	Column       string // Local column name
	RefTable     string // Referenced table name
	RefColumn    string // Referenced column name
	OnDelete     string // ON DELETE action (e.g., "CASCADE", "RESTRICT")
	OnUpdate     string // ON UPDATE action
}

// TypeMapping maps Go types to PostgreSQL types
type TypeMapping struct {
	GoType      string
	SQLType     string
	NullableSQL string // SQL type when nullable
}

// ParserContext holds context for parsing Go files
type ParserContext struct {
	PackageName string
	Imports     map[string]string // import path -> alias
	Models      []ModelInfo
}

// GeneratorConfig holds configuration for migration generation
type GeneratorConfig struct {
	ModelsDir      string   // Directory containing model files
	OutputDir      string   // Output directory for migrations
	DryRun         bool     // If true, don't write files
	ModelNames     []string // Specific models to generate (empty = all)
	Timestamp      time.Time // Timestamp for migration file naming
	PackageName    string   // Go package name
}

// Default type mappings
var defaultTypeMappings = []TypeMapping{
	{"int", "SERIAL", "INTEGER"},
	{"*int", "INTEGER", "INTEGER"},
	{"int64", "BIGINT", "BIGINT"},
	{"*int64", "BIGINT", "BIGINT"},
	{"string", "VARCHAR(255)", "VARCHAR(255)"},
	{"*string", "VARCHAR(255)", "VARCHAR(255)"},
	{"float64", "DECIMAL(15,2)", "DECIMAL(15,2)"},
	{"*float64", "DECIMAL(15,2)", "DECIMAL(15,2)"},
	{"bool", "BOOLEAN", "BOOLEAN"},
	{"*bool", "BOOLEAN", "BOOLEAN"},
	{"time.Time", "TIMESTAMP", "TIMESTAMP"},
	{"*time.Time", "TIMESTAMP", "TIMESTAMP"},
}
