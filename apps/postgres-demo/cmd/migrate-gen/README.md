# Migration Generator

A tool that generates SQL migration files from Go model structs, ensuring models are the single source of truth for database schema.

## Overview

The migration generator parses Go struct definitions in the `models/` directory and generates PostgreSQL CREATE TABLE statements that match the model definitions. This ensures that:

- Models are the single source of truth
- Database schema stays in sync with code
- Migrations are generated automatically from model changes

## Usage

### Basic Usage

```bash
# Generate migrations from all models
go run cmd/migrate-gen/main.go

# Specify output directory
go run cmd/migrate-gen/main.go -output migrations/

# Dry run (preview without writing)
go run cmd/migrate-gen/main.go -dry-run

# Generate for specific models only
go run cmd/migrate-gen/main.go -models-list "Approval,SystemSetting"

# Verbose output
go run cmd/migrate-gen/main.go -verbose
```

### Using Makefile

```bash
# Generate migrations
make generate-migrations

# Dry run
make migrate-gen-dry

# Specific models
make migrate-gen-models MODELS='Approval,SystemSetting'
```

## Command Line Options

- `-models <path>`: Directory containing model files (default: auto-detect)
- `-output <path>`: Output directory for migration files (default: `migrations`)
- `-dry-run`: Preview changes without writing files
- `-models-list <list>`: Comma-separated list of specific models to generate (empty = all)
- `-verbose`: Enable verbose output

## How It Works

1. **Parsing**: Uses Go AST to parse struct definitions from model files
2. **Extraction**: Extracts table names, column names, types, and constraints
3. **Type Mapping**: Maps Go types to PostgreSQL types
4. **Generation**: Generates CREATE TABLE statements with indexes and constraints
5. **Output**: Writes timestamped migration files

## Model Requirements

For a struct to be processed:

1. Must have `Table: table_name` comment, OR be a known model type
2. Must have `db` tags on fields
3. Fields with `db:"-"` are skipped

## Type Mapping

| Go Type | PostgreSQL Type |
|---------|----------------|
| `int` | `SERIAL` (if ID field) or `INTEGER` |
| `*int` | `INTEGER` (nullable) |
| `string` | `VARCHAR(255)` (default) or `TEXT` |
| `*string` | `VARCHAR(255)` (nullable) |
| `float64` | `DECIMAL(15,2)` (money) or `DECIMAL(10,2)` |
| `*float64` | `DECIMAL(15,2)` (nullable) |
| `bool` | `BOOLEAN` |
| `*bool` | `BOOLEAN` (nullable) |
| `time.Time` | `TIMESTAMP` |
| `*time.Time` | `TIMESTAMP` (nullable) |
| Custom types | `VARCHAR(50)` |

## Generated Output

The generator creates migration files with:

- CREATE TABLE statements
- Column definitions with types and constraints
- Primary key constraints
- Indexes for foreign keys and timestamps
- Foreign key constraints
- Table and column comments

## Example

Input model:

```go
// Approval represents a two-person approval request
// Table: approvals
type Approval struct {
    ID              int             `db:"id" json:"id"`
    ApprovalType    ApprovalType    `db:"approval_type" json:"approval_type"`
    Status          ApprovalStatus  `db:"status" json:"status"`
    CreatedAt       time.Time       `db:"created_at" json:"created_at"`
}
```

Generated SQL:

```sql
CREATE TABLE IF NOT EXISTS approvals (
    id SERIAL PRIMARY KEY,
    approval_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_approvals_created_at ON approvals(created_at);
```

## Limitations

- Currently generates CREATE TABLE only (no ALTER TABLE for changes)
- Foreign key inference is basic (based on `_id` suffix)
- Composite primary keys must be specified in comments
- Custom constraints need to be added manually to migrations

## Future Enhancements

- Detect schema drift (compare models vs existing migrations)
- Generate ALTER TABLE for model changes
- Support for custom type mappings via config
- Integration with migration versioning system
