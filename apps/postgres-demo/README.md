# PostgreSQL Demo Application

A simple demo application that demonstrates PostgreSQL database connection and query execution using Fluxor's `dbruntime` package.

## Features

- ✅ Reads configuration from `application.properties`
- ✅ Creates PostgreSQL connection pool using `dbruntime`
- ✅ Executes test query: `SELECT 1`
- ✅ Uses **OS Thread Pool** (`osthread`) for database operations
- ✅ Displays pool statistics (database pool + OS thread pool)
- ✅ **Web UI** - HTTP interface for monitoring and querying (port 8080)
- ✅ **Login System** - JWT-based authentication
- ✅ **Purchase Transactions** - ACID-compliant purchase system with:
  - Product catalog management
  - Shopping cart functionality
  - Order processing with Serializable transaction isolation
  - Stock management with concurrent purchase protection
  - Order history tracking
- ✅ Graceful shutdown

## Configuration

The application reads configuration from `application.properties`:

```properties
# PostgreSQL Database Configuration
# Format: postgres://[user]:[password]@[host]:[port]/[database name]
DSN=postgres://postgres:postgres@localhost:5432/fluxor_db?sslmode=disable

# Connection Pool Configuration
MaxOpenConns=25
MaxIdleConns=5
ConnMaxLifetime=5m
ConnMaxIdleTime=10m
```

### Configuration Properties

- **DSN**: PostgreSQL connection string (Data Source Name)
  - Format: `postgres://user:password@host:port/database?sslmode=disable`
- **MaxOpenConns**: Maximum number of open connections (default: 25)
- **MaxIdleConns**: Maximum number of idle connections (default: 5)
- **ConnMaxLifetime**: Maximum connection lifetime (e.g., "5m", "10m")
- **ConnMaxIdleTime**: Maximum idle time before connection is closed (e.g., "10m")

## Architecture

This demo uses **OS Thread Pool** (`osthread`) for database operations:

- **Database Pool** (`dbruntime`): Manages PostgreSQL connections (connection pooling)
- **OS Thread Pool** (`osthread`): Executes database queries on pinned OS threads

### Why OS Thread Pool?

While database operations are IO-bound, using OS thread pool provides:
- **Thread affinity**: Better cache locality
- **Controlled concurrency**: Fixed number of OS threads (typically = CPU cores)
- **Consistent performance**: Predictable thread scheduling

**Note**: For general IO-bound work, Go's goroutine scheduler is usually sufficient. OS thread pool is used here to demonstrate the pattern.

## Prerequisites

1. **PostgreSQL Database**: Running PostgreSQL server
2. **Go**: Go 1.21 or later
3. **PostgreSQL Driver**: `github.com/lib/pq`

## Database Migrations

This application includes a **Model-Based Migration Generator** that generates SQL migration files from Go model structs, ensuring models are the single source of truth for database schema.

### Generating Migrations from Models

The migration generator automatically:
- Parses Go struct definitions with `db` tags
- Extracts table names from comments (`Table: table_name`)
- Maps Go types to PostgreSQL types
- Generates CREATE TABLE statements with indexes and constraints
- Outputs timestamped migration files

#### Using Makefile

```bash
# Generate migrations from all models
make generate-migrations

# Preview migrations without writing (dry run)
make migrate-gen-dry

# Generate migrations for specific models
make migrate-gen-models MODELS='Approval,SystemSetting'
```

#### Using Command Line

```bash
# Generate migrations (auto-detects models directory)
go run cmd/migrate-gen/main.go

# Specify output directory
go run cmd/migrate-gen/main.go -output migrations/

# Dry run (preview without writing)
go run cmd/migrate-gen/main.go -dry-run

# Generate for specific models only
go run cmd/migrate-gen/main.go -models-list "Approval,SystemSetting" -verbose
```

### Migration Generator Features

- **Automatic Type Mapping**: Maps Go types (`int`, `string`, `float64`, `bool`, `time.Time`) to PostgreSQL types
- **Nullable Detection**: Automatically detects nullable fields from pointer types (`*string`, `*int`)
- **Primary Key Detection**: Identifies primary keys from field names (`ID`, `id`) or comments
- **Index Generation**: Automatically creates indexes for foreign keys and timestamp fields
- **Comment Preservation**: Preserves model comments as SQL comments
- **Composite Keys**: Supports composite primary keys from comments

### Model Requirements

For a struct to be processed as a model:

1. Must have a `Table: table_name` comment, OR
2. Must be a known model type (Approval, SystemSetting, Wallet, etc.)
3. Must have `db` tags on fields (fields with `db:"-"` are skipped)

Example model:

```go
// Approval represents a two-person approval request
// Table: approvals
type Approval struct {
    ID              int             `db:"id" json:"id"`
    ApprovalType    ApprovalType    `db:"approval_type" json:"approval_type"`
    Status          ApprovalStatus  `db:"status" json:"status"`
    CreatedAt       time.Time       `db:"created_at" json:"created_at"`
    UpdatedAt       time.Time       `db:"updated_at" json:"updated_at"`
}
```

### Running Migrations

After generating migrations, run them using the migration scripts:

```bash
# Run all migrations
./migrate.sh

# Run with empty database (drops all tables first)
./migrate_empty.sh

# Run with fresh database (drops and recreates database)
./migrate_fresh_db.sh
```

## Setup

1. **Install PostgreSQL Driver**:
```bash
go get github.com/lib/pq
```

2. **Update DSN in `application.properties`**:
```properties
DSN=postgres://your_user:your_password@localhost:5432/your_database?sslmode=disable
```

3. **Run the application**:
```bash
# Option 1: Using run script
./run.sh

# Option 2: Direct execution
go run main.go

# Option 3: Build and run
go build -o postgres-demo main.go
./postgres-demo
```

## Usage

### Basic Usage

```bash
# Run with default configuration
go run main.go
```

The application will start:
- Database connection pool
- HTTP web UI on http://localhost:8080
- Health check monitoring

### Web UI

Once the application is running, open your browser and navigate to:

```
http://localhost:8080
```

You will be redirected to the login page. Use the default credentials:
- **Username**: `admin`
- **Password**: `admin123`

**Note**: In production, change these credentials and use environment variables or a database for user management.

The web UI provides:
- **Login System**: JWT-based authentication
- **Database Status**: Real-time connection pool statistics
- **Purchase System**: 
  - Browse available products
  - Add items to shopping cart
  - Complete purchases with ACID transaction guarantees
  - View order history
  - Automatic stock management
- **Query Executor**: Execute SQL queries directly from the browser
- **Auto-refresh**: Statistics update every 2 seconds
- **Protected Routes**: All database operations require authentication

### Purchase Transaction Features

The purchase system demonstrates advanced PostgreSQL transaction capabilities:

- **Serializable Isolation Level**: Prevents phantom reads and ensures data consistency
- **Row-Level Locking**: Uses `SELECT ... FOR UPDATE` to lock products during purchase
- **ACID Guarantees**: All-or-nothing transaction processing
- **Stock Validation**: Checks and updates inventory atomically
- **Concurrent Purchase Protection**: Multiple users can purchase simultaneously without data corruption

### With Custom Configuration

Update `application.properties` with your database credentials and run:

```bash
go run main.go
```

### Running Persistence Feature Tests

The application includes comprehensive tests for the new persistence features:

```bash
# Run all persistence feature tests
go run main.go --test
```

This will test:
- ✅ **Transaction Isolation Levels**: Tests `BeginTransactionWithOptions` with different isolation levels (Default, Serializable, Read-Only)
- ✅ **Savepoint Support**: Tests savepoint creation, rollback, and release functionality
- ✅ **Error Sanitization**: Tests that sensitive information is removed from error messages

**Note**: Tests require a running PostgreSQL database with the connection configured in `application.properties`.

## Expected Output

```
Starting PostgreSQL Demo Application...
Loaded configuration from application.properties
DSN: postgres://postgres:***@localhost:5432/fluxor_db?sslmode=disable
MaxOpenConns: 25
MaxIdleConns: 5
Database component started successfully
✅ Test query successful! Result: 1

[Web UI] Starting HTTP server on :8080...
✅ Web UI available at http://localhost:8080

Pool Statistics:
  Open Connections: 1
  In Use: 0
  Idle: 1
  Wait Count: 0
  Wait Duration: 0s
✅ PostgreSQL Demo Application started successfully!
Press Ctrl+C to stop...
```

### Web UI Screenshots

The web UI provides a modern, responsive interface with:
- Real-time database connection statistics
- SQL query executor with syntax highlighting
- Results displayed in a formatted table
- Auto-refreshing status indicators

## Architecture

### Component Flow

```
PostgresDemoVerticle
    ↓
DatabaseComponent (dbruntime)
    ↓
Pool (dbruntime)
    ↓
database/sql.DB
    ↓
PostgreSQL Database
```

### Key Features

1. **Configuration Loading**: Uses `pkg/config` to load `application.properties`
2. **Database Component**: Uses `dbruntime.DatabaseComponent` for lifecycle management
3. **WorkerPool**: Uses `ExecuteBlocking` for IO-bound database operations
4. **Connection Pooling**: Efficient connection reuse
5. **Statistics**: Displays pool statistics after query

## Code Structure

```go
// Load configuration
config.LoadProperties("application.properties", &appConfig)

// Create database component
db := dbruntime.NewDatabaseComponent(poolConfig)
db.SetParent(verticle)
db.Start(ctx)

// Execute query using WorkerPool (IO-bound)
result, err := ctx.GoCMD().ExecuteBlocking(func() (interface{}, error) {
    return executeTestQuery(ctx)
})

// Display statistics
stats := db.Stats()
```

## Best Practices Demonstrated

1. ✅ **Configuration Management**: Externalized configuration in properties file
2. ✅ **WorkerPool Usage**: IO-bound operations use `ExecuteBlocking`
3. ✅ **Lifecycle Management**: Proper component start/stop
4. ✅ **Error Handling**: Comprehensive error handling
5. ✅ **Resource Cleanup**: Graceful shutdown
6. ✅ **Security**: DSN password masking in logs

## Troubleshooting

### Connection Failed

**Error**: `failed to start database component: dial tcp: connection refused`

**Solution**: 
- Verify PostgreSQL is running
- Check DSN in `application.properties`
- Verify host, port, and credentials

### Configuration Not Found

**Error**: `failed to load config from application.properties: file not found`

**Solution**:
- Ensure `application.properties` exists in the same directory as `main.go`
- Check file permissions

### Invalid Duration Format

**Warning**: `Invalid ConnMaxLifetime, using default`

**Solution**:
- Use Go duration format: `5m`, `10m`, `1h`, etc.
- Examples: `5m`, `10m`, `30s`, `1h`

## Example DSN Formats

### Local PostgreSQL
```
DSN=postgres://postgres:postgres@localhost:5432/fluxor_db?sslmode=disable
```

### Remote PostgreSQL
```
DSN=postgres://user:password@remote-host:5432/database?sslmode=require
```

### With SSL
```
DSN=postgres://user:password@host:5432/db?sslmode=require
```

## Next Steps

- Add more complex queries
- Implement CRUD operations
- Add transaction examples
- Integrate with persistence package
- Add error retry logic

## Related Documentation

- [dbruntime Architecture](../../docs/postgres/ARCHITECTURE.md)
- [Fluxor Core Documentation](../../pkg/core/README.md)
- [Configuration Package](../../pkg/config/)
