# Persistence Package

A generic, high-performance data persistence package for the Fluxor framework with repository pattern, transaction support, and query builders.

## 🎯 Overview

The `persistence` package provides a clean abstraction layer for data persistence operations:

- ✅ **Repository Pattern**: Clean separation of data access logic
- ✅ **Transaction Support**: ACID transactions with rollback capability
- ✅ **Query Builder**: Flexible query construction with filters, sorting, and pagination
- ✅ **Fail-Fast Validation**: All operations validate inputs before processing
- ✅ **Context Support**: All operations accept `context.Context` for cancellation
- ✅ **Database Agnostic**: Works with any SQL database via `database/sql`

## Features

- **Generic Interface**: Works with any entity type
- **Transaction Management**: Begin, commit, and rollback transactions
- **Query Builder**: Build complex queries with filters, sorting, and pagination
- **Fail-Fast Validation**: Input validation before processing
- **Thread-Safe**: Safe for concurrent use
- **Connection Pooling**: Integrates with existing connection pools

## Quick Start

### Basic Usage

```go
import (
    "context"
    "database/sql"
    "github.com/fluxorio/fluxor/pkg/persistence"
    _ "github.com/lib/pq" // PostgreSQL driver
)

// Create connection pool
db, err := sql.Open("postgres", "postgres://user:pass@localhost/dbname?sslmode=disable")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Create repository
config := persistence.DefaultConfig("users", db)
repo, err := persistence.NewSQLRepository(config)
if err != nil {
    log.Fatal(err)
}
defer repo.Close()

ctx := context.Background()

// Create a new entity
entity := map[string]interface{}{
    "name":  "John Doe",
    "email": "john@example.com",
}
err = repo.Create(ctx, entity)

// Find by ID
user, err := repo.FindByID(ctx, 1)

// Update entity
updated := map[string]interface{}{
    "name": "Jane Doe",
}
err = repo.Update(ctx, 1, updated)

// Delete entity
err = repo.Delete(ctx, 1)
```

### Query Builder

```go
// Build a query with filters, sorting, and pagination
query := persistence.NewQuery().
    WithFilter("status", "active").
    WithFilter("age", 25).
    WithOrderBy("created_at", "DESC").
    WithLimit(10).
    WithOffset(0)

// Find all matching entities
users, err := repo.FindAll(ctx, query)

// Find one matching entity
user, err := repo.FindOne(ctx, query)

// Count matching entities
count, err := repo.Count(ctx, query)
```

### Transactions

```go
// Begin a transaction
tx, err := repo.BeginTransaction(ctx)
if err != nil {
    log.Fatal(err)
}

// Get repository bound to transaction
txRepo := tx.Repository()

// Perform operations within transaction
err = txRepo.Create(ctx, entity1)
if err != nil {
    tx.Rollback()
    return err
}

err = txRepo.Update(ctx, 2, entity2)
if err != nil {
    tx.Rollback()
    return err
}

// Commit transaction
err = tx.Commit()
if err != nil {
    log.Fatal(err)
}
```

### Integration with dbruntime Pool

```go
import (
    "github.com/fluxorio/fluxor/pkg/dbruntime"
    "github.com/fluxorio/fluxor/pkg/persistence"
)

// Create connection pool using dbruntime
pool, err := dbruntime.NewPool(dbruntime.PoolConfig{
    DSN:        "postgres://user:pass@localhost/dbname",
    DriverName: "postgres",
    MaxOpenConns: 25,
    MaxIdleConns: 5,
})
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

// Get database connection from pool
db := pool.DB()

// Create repository
config := persistence.DefaultConfig("users", db)
repo, err := persistence.NewSQLRepository(config)
```

## API Reference

### Repository Interface

```go
type Repository interface {
    FindByID(ctx context.Context, id interface{}) (interface{}, error)
    FindAll(ctx context.Context, query *Query) ([]interface{}, error)
    FindOne(ctx context.Context, query *Query) (interface{}, error)
    Create(ctx context.Context, entity interface{}) error
    Update(ctx context.Context, id interface{}, entity interface{}) error
    Delete(ctx context.Context, id interface{}) error
    Count(ctx context.Context, query *Query) (int64, error)
    Exists(ctx context.Context, id interface{}) (bool, error)
    BeginTransaction(ctx context.Context) (Transaction, error)
    Close() error
}
```

### Transaction Interface

```go
type Transaction interface {
    Commit() error
    Rollback() error
    Repository() Repository
}
```

### Query Builder

```go
type Query struct {
    Filters      map[string]interface{}
    OrderBy      map[string]string
    Limit        int
    Offset       int
    SelectFields []string
}

// Builder methods
func NewQuery() *Query
func (q *Query) WithFilter(field string, value interface{}) *Query
func (q *Query) WithOrderBy(field string, direction string) *Query
func (q *Query) WithLimit(limit int) *Query
func (q *Query) WithOffset(offset int) *Query
func (q *Query) WithSelectFields(fields ...string) *Query
```

### Configuration

```go
type Config struct {
    TableName      string
    IDField        string
    ConnectionPool *sql.DB
    MaxRetries     int
    RetryDelay     time.Duration
    QueryTimeout   time.Duration
}

func DefaultConfig(tableName string, pool *sql.DB) Config
func (c *Config) Validate() error
```

## Fail-Fast Principles

All persistence operations follow fail-fast principles:

- **Input Validation**: All inputs are validated before processing
- **Nil Checks**: Context and entity parameters are checked for nil
- **Immediate Errors**: Errors are returned immediately with clear messages
- **Configuration Validation**: Configuration is validated on creation

Example:

```go
// This will fail-fast with a clear error
err := repo.Create(nil, entity)
// Error: "fail-fast: context is nil"

// This will fail-fast
err := repo.Create(ctx, nil)
// Error: "fail-fast: entity is nil"
```

## Error Handling

The package provides structured error handling:

```go
import "github.com/fluxorio/fluxor/pkg/persistence"

// Check for specific error types
if persistence.IsNotFound(err) {
    // Handle not found error
}

if persistence.IsTransactionError(err) {
    // Handle transaction error
}

// Access error details
if e, ok := err.(*persistence.Error); ok {
    fmt.Printf("Code: %s, Message: %s\n", e.Code, e.Message)
}
```

### Error Codes

- `NOT_FOUND`: Entity not found
- `INVALID_CONFIG`: Invalid configuration
- `INVALID_INPUT`: Invalid input parameters
- `TRANSACTION_ERROR`: Transaction operation failed
- `QUERY_ERROR`: Query execution failed
- `CONNECTION_ERROR`: Database connection error

## Best Practices

### 1. Use Context for Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

user, err := repo.FindByID(ctx, 1)
```

### 2. Always Handle Transactions Properly

```go
tx, err := repo.BeginTransaction(ctx)
if err != nil {
    return err
}

defer func() {
    if err != nil {
        tx.Rollback()
    }
}()

// Perform operations
err = tx.Repository().Create(ctx, entity)
if err != nil {
    return err
}

return tx.Commit()
```

### 3. Use Query Builder for Complex Queries

```go
// Good: Use query builder
query := persistence.NewQuery().
    WithFilter("status", "active").
    WithOrderBy("created_at", "DESC").
    WithLimit(10)

users, err := repo.FindAll(ctx, query)
```

### 4. Validate Configuration

```go
config := persistence.DefaultConfig("users", db)
if err := config.Validate(); err != nil {
    log.Fatal(err)
}

repo, err := persistence.NewSQLRepository(config)
```

### 5. Close Resources

```go
repo, err := persistence.NewSQLRepository(config)
if err != nil {
    log.Fatal(err)
}
defer repo.Close()
```

## Thread Safety

All repository implementations are thread-safe and can be used concurrently from multiple goroutines, as long as the underlying database connection pool is properly configured.

## Performance Considerations

- **Connection Pooling**: Use `dbruntime` package for efficient connection pooling
- **Query Timeout**: Set appropriate query timeouts to prevent hanging queries
- **Batch Operations**: For bulk operations, consider using transactions
- **Indexing**: Ensure proper database indexes for filtered fields

## Examples

### Error Handling

```go
ctx := context.Background()

// Find by ID - handle not found
user, err := repo.FindByID(ctx, 1)
if err != nil {
    if IsNotFound(err) {
        // Handle not found
        log.Printf("User not found: %v", err)
    } else {
        // Handle other errors
        log.Printf("Query error: %v", err)
    }
    return err
}

// Update - handle not found
err = repo.Update(ctx, 999, map[string]interface{}{"name": "Updated"})
if err != nil {
    if IsNotFound(err) {
        log.Printf("Entity not found for update")
    } else if IsTransactionError(err) {
        log.Printf("Transaction error: %v", err)
    }
    return err
}

// Access error details
if e, ok := err.(*Error); ok {
    fmt.Printf("Error Code: %s, Message: %s\n", e.Code, e.Message)
    if e.Cause != nil {
        fmt.Printf("Underlying error: %v\n", e.Cause)
    }
}
```

### Transaction with Rollback

```go
ctx := context.Background()

// Begin transaction
tx, err := repo.BeginTransaction(ctx)
if err != nil {
    log.Fatal(err)
}

// Always rollback on error
defer func() {
    if err != nil {
        if rollbackErr := tx.Rollback(); rollbackErr != nil {
            log.Printf("Rollback error: %v", rollbackErr)
        }
    }
}()

// Get transaction repository
txRepo := tx.Repository()

// Perform operations within transaction
err = txRepo.Create(ctx, entity1)
if err != nil {
    return err // Rollback will be called by defer
}

err = txRepo.Update(ctx, 2, entity2)
if err != nil {
    return err // Rollback will be called by defer
}

// Commit transaction
err = tx.Commit()
if err != nil {
    log.Fatal(err)
}
```

### Connection Pool Tuning

```go
import (
    "github.com/fluxorio/fluxor/pkg/dbruntime"
    "github.com/fluxorio/fluxor/pkg/persistence"
)

// Create optimized connection pool
poolConfig := dbruntime.PoolConfig{
    DSN:             "postgres://user:pass@localhost/dbname",
    DriverName:      "postgres",
    MaxOpenConns:    25,              // Maximum connections
    MaxIdleConns:    5,               // Idle connections to keep
    ConnMaxLifetime: 5 * time.Minute, // Connection lifetime
    ConnMaxIdleTime: 10 * time.Minute, // Idle timeout
}

pool, err := dbruntime.NewPool(poolConfig)
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

// Monitor pool statistics
stats := pool.Stats()
fmt.Printf("Open Connections: %d\n", stats.OpenConnections)
fmt.Printf("In Use: %d\n", stats.InUse)
fmt.Printf("Idle: %d\n", stats.Idle)
fmt.Printf("Wait Count: %d\n", stats.WaitCount)
fmt.Printf("Wait Duration: %v\n", stats.WaitDuration)

// Create repository with tuned config
config := persistence.DefaultConfig("users", pool.DB())
config.MaxRetries = 5                    // Retry transient errors
config.RetryDelay = 200 * time.Millisecond
config.QueryTimeout = 10 * time.Second  // Query timeout

repo, err := persistence.NewSQLRepository(config)
```

### Batch Operations

```go
ctx := context.Background()

// Batch create - more efficient than multiple Create() calls
entities := []interface{}{
    map[string]interface{}{"name": "User 1", "email": "user1@example.com"},
    map[string]interface{}{"name": "User 2", "email": "user2@example.com"},
    map[string]interface{}{"name": "User 3", "email": "user3@example.com"},
}

err := repo.BatchCreate(ctx, entities)
if err != nil {
    log.Fatal(err)
}

// Batch update
updateEntities := []interface{}{
    map[string]interface{}{"id": 1, "status": "inactive"},
    map[string]interface{}{"id": 2, "status": "inactive"},
}

err = repo.BatchUpdate(ctx, updateEntities)
if err != nil {
    log.Fatal(err)
}

// Batch delete
ids := []interface{}{1, 2, 3}
err = repo.BatchDelete(ctx, ids)
if err != nil {
    log.Fatal(err)
}
```

### Using with dbruntime Component

```go
import (
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/dbruntime"
    "github.com/fluxorio/fluxor/pkg/persistence"
)

type UserService struct {
    *core.BaseService
    db *dbruntime.DatabaseComponent
    repo *persistence.SQLRepository
}

func NewUserService() *UserService {
    return &UserService{
        BaseService: core.NewBaseService("user-service", "user.service"),
        db: dbruntime.NewDatabaseComponent(
            dbruntime.DefaultPoolConfig(
                "postgres://user:pass@localhost/dbname",
                "postgres",
            ),
        ),
    }
}

func (s *UserService) doStart(ctx core.FluxorContext) error {
    s.db.SetParent(s.BaseVerticle)
    if err := s.db.Start(ctx); err != nil {
        return err
    }

    // Create repository using database component
    config := persistence.DefaultConfig("users", s.db.DB())
    repo, err := persistence.NewSQLRepository(config)
    if err != nil {
        return err
    }
    s.repo = repo

    return nil
}

func (s *UserService) doStop(ctx core.FluxorContext) error {
    if s.repo != nil {
        s.repo.Close()
    }
    return s.db.Stop(ctx)
}
```

## Related Packages

- **pkg/dbruntime**: Database connection pooling
- **pkg/cache**: Caching layer for persistence operations
- **pkg/appendlog**: Append-only log for audit trails

---

**Package**: `github.com/fluxorio/fluxor/pkg/persistence`  
**Status**: ✅ Stable  
**Test Coverage**: 75%  
**Last Updated**: 2026-01-04
