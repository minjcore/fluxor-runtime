# CSQL Package - Custom SQL Query Builder

Advanced SQL query building and execution utilities for the Fluxor persistence package.

## 🎯 Overview

The `csql` package provides:

- ✅ **Fluent Query Builder**: Build complex SQL queries with a fluent API
- ✅ **Advanced SQL Features**: JOINs, GROUP BY, HAVING, subqueries
- ✅ **Raw SQL Execution**: Execute custom SQL queries when needed
- ✅ **Type-Safe Building**: Compile-time query construction
- ✅ **Fail-Fast Validation**: All operations validate inputs before execution

## Features

- **Fluent API**: Chain methods to build queries
- **Multiple JOIN Types**: INNER, LEFT, RIGHT joins
- **Complex WHERE Conditions**: Support for IN, BETWEEN, LIKE, etc.
- **Aggregation Support**: GROUP BY and HAVING clauses
- **Pagination**: Built-in LIMIT and OFFSET support
- **Raw SQL**: Execute custom SQL when needed

## Quick Start

### Basic Query Building

```go
import (
    "context"
    "github.com/fluxorio/fluxor/pkg/persistence"
    "github.com/fluxorio/fluxor/pkg/persistence/csql"
)

// Create CSQL repository
config := persistence.DefaultConfig("users", db)
repo, err := csql.NewCSQLRepository(config)
if err != nil {
    log.Fatal(err)
}

ctx := context.Background()

// Build a query
qb := csql.NewQueryBuilder("users").
    Select("id", "name", "email").
    WhereEq("status", "active").
    Where("created_at > ?", time.Now().AddDate(0, -1, 0)).
    OrderBy("created_at", "DESC").
    Limit(10).
    Offset(0)

// Execute query
rows, err := repo.Execute(ctx, qb)
defer rows.Close()

for rows.Next() {
    var id int
    var name, email string
    rows.Scan(&id, &name, &email)
    // Process row
}
```

### JOINs

```go
// Build query with JOIN
qb := csql.NewQueryBuilder("users").
    Select("users.id", "users.name", "profiles.bio").
    LeftJoin("profiles", "profiles.user_id = users.id").
    WhereEq("users.status", "active")

rows, err := repo.Execute(ctx, qb)
```

### Aggregations

```go
// GROUP BY with HAVING
qb := csql.NewQueryBuilder("orders").
    Select("user_id", "COUNT(*) as order_count", "SUM(total) as total_amount").
    GroupBy("user_id").
    Having("COUNT(*) > ?", 5).
    OrderBy("total_amount", "DESC")

rows, err := repo.Execute(ctx, qb)
```

### WHERE IN

```go
// WHERE IN clause
userIDs := []interface{}{1, 2, 3, 4, 5}
qb := csql.NewQueryBuilder("users").
    Select("*").
    WhereIn("id", userIDs)

rows, err := repo.Execute(ctx, qb)
```

### Raw SQL

```go
// Execute raw SQL
query := `
    SELECT u.*, COUNT(o.id) as order_count
    FROM users u
    LEFT JOIN orders o ON o.user_id = u.id
    WHERE u.status = $1
    GROUP BY u.id
    HAVING COUNT(o.id) > $2
    ORDER BY order_count DESC
    LIMIT $3
`

rows, err := repo.ExecuteRaw(ctx, query, "active", 5, 10)
```

### Single Row Query

```go
// Query for a single row
qb := csql.NewQueryBuilder("users").
    Select("*").
    WhereEq("id", 1).
    Limit(1)

row, err := repo.ExecuteOne(ctx, qb)
if err != nil {
    log.Fatal(err)
}

var user User
err = row.Scan(&user.ID, &user.Name, &user.Email)
```

### Executing Statements

```go
// Execute INSERT, UPDATE, DELETE
result, err := repo.Exec(ctx, 
    "UPDATE users SET status = $1 WHERE id = $2",
    "inactive", 123,
)

rowsAffected, _ := result.RowsAffected()
fmt.Printf("Updated %d rows\n", rowsAffected)
```

## API Reference

### QueryBuilder

```go
type QueryBuilder struct {
    // ... internal fields
}

// Builder methods
func NewQueryBuilder(table string) *QueryBuilder
func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder
func (qb *QueryBuilder) Join(table, on string) *QueryBuilder
func (qb *QueryBuilder) LeftJoin(table, on string) *QueryBuilder
func (qb *QueryBuilder) RightJoin(table, on string) *QueryBuilder
func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder
func (qb *QueryBuilder) WhereEq(column string, value interface{}) *QueryBuilder
func (qb *QueryBuilder) WhereIn(column string, values []interface{}) *QueryBuilder
func (qb *QueryBuilder) GroupBy(columns ...string) *QueryBuilder
func (qb *QueryBuilder) Having(condition string, args ...interface{}) *QueryBuilder
func (qb *QueryBuilder) OrderBy(column string, direction string) *QueryBuilder
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder
func (qb *QueryBuilder) Build() (string, []interface{})
```

### CSQLRepository

```go
type CSQLRepository struct {
    *persistence.SQLRepository
}

func NewCSQLRepository(config persistence.Config) (*CSQLRepository, error)
func (r *CSQLRepository) Execute(ctx context.Context, qb *QueryBuilder) (*sql.Rows, error)
func (r *CSQLRepository) ExecuteOne(ctx context.Context, qb *QueryBuilder) (*sql.Row, error)
func (r *CSQLRepository) ExecuteRaw(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
func (r *CSQLRepository) ExecuteRawOne(ctx context.Context, query string, args ...interface{}) *sql.Row
func (r *CSQLRepository) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
```

## Best Practices

### 1. Use Query Builder for Complex Queries

```go
// ✅ Good: Use query builder for complex queries
qb := csql.NewQueryBuilder("users").
    Select("id", "name").
    LeftJoin("profiles", "profiles.user_id = users.id").
    WhereEq("status", "active").
    OrderBy("name", "ASC")

// ❌ Avoid: Complex raw SQL strings
query := "SELECT id, name FROM users LEFT JOIN profiles ON ..."
```

### 2. Use Raw SQL for Simple Queries

```go
// ✅ Good: Simple queries can use raw SQL
rows, err := repo.ExecuteRaw(ctx, "SELECT * FROM users WHERE id = $1", userID)

// ❌ Overkill: Using query builder for simple queries
qb := csql.NewQueryBuilder("users").WhereEq("id", userID)
```

### 3. Always Use Context

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

rows, err := repo.Execute(ctx, qb)
```

### 4. Close Rows Properly

```go
rows, err := repo.Execute(ctx, qb)
if err != nil {
    return err
}
defer rows.Close() // Always close rows

for rows.Next() {
    // Process row
}
```

### 5. Use Parameterized Queries

```go
// ✅ Good: Parameterized query (safe from SQL injection)
qb.Where("name = ?", userName)

// ❌ Bad: String concatenation (SQL injection risk)
qb.Where(fmt.Sprintf("name = '%s'", userName))
```

## Examples

### Complex Reporting Query

```go
qb := csql.NewQueryBuilder("orders").
    Select(
        "DATE(created_at) as order_date",
        "COUNT(*) as order_count",
        "SUM(total) as total_revenue",
        "AVG(total) as avg_order_value",
    ).
    Where("created_at >= ?", startDate).
    Where("created_at <= ?", endDate).
    GroupBy("DATE(created_at)").
    Having("COUNT(*) > ?", 10).
    OrderBy("order_date", "DESC")

rows, err := repo.Execute(ctx, qb)
```

### Pagination

```go
func GetUsersPaginated(ctx context.Context, repo *csql.CSQLRepository, page, pageSize int) ([]User, error) {
    offset := (page - 1) * pageSize
    
    qb := csql.NewQueryBuilder("users").
        Select("*").
        WhereEq("status", "active").
        OrderBy("created_at", "DESC").
        Limit(pageSize).
        Offset(offset)
    
    rows, err := repo.Execute(ctx, qb)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var users []User
    for rows.Next() {
        var user User
        rows.Scan(&user.ID, &user.Name, &user.Email)
        users = append(users, user)
    }
    
    return users, nil
}
```

## Related Packages

- **pkg/persistence**: Base persistence package with Repository pattern
- **pkg/dbruntime**: Database connection pooling

---

**Package**: `github.com/fluxorio/fluxor/pkg/persistence/csql`  
**Status**: ✅ Stable  
**Last Updated**: 2026-01-04
