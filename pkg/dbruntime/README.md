### dbconnect,dbstate

# Database Runtime Package - Connection Pooling (HikariCP Equivalent)

Package `dbruntime` provides database connection pooling for Fluxor, similar to HikariCP in Java.

## Features

- ✅ **HikariCP-like configuration**: Similar API and behavior
- ✅ **Built-in pooling**: Uses Go's `database/sql` built-in connection pooling
- ✅ **Premium Pattern integration**: `DatabaseComponent` for Fluxor
- ✅ **Pool statistics**: Monitor pool health and performance
- ✅ **Context support**: Full context.Context integration

## Quick Start

### Basic Pool Usage

```go
import "github.com/fluxorio/fluxor/pkg/dbruntime"

// Create pool configuration (similar to HikariConfig)
config := dbruntime.DefaultPoolConfig(
    "postgres://user:pass@localhost/dbname",
    "postgres",
)

// Create pool (similar to HikariDataSource)
pool, err := dbruntime.NewPool(config)
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

// Use pool (connections automatically managed)
ctx := context.Background()
rows, err := pool.Query(ctx, "SELECT * FROM users")
```

### Premium Pattern Integration

```go
import (
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/dbruntime"
)

type UserService struct {
    *core.BaseService
    db *dbruntime.DatabaseComponent
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
    return s.db.Start(ctx)
}

func (s *UserService) doStop(ctx core.FluxorContext) error {
    return s.db.Stop(ctx)
}

func (s *UserService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    userID := msg.Body().(string)
    
    var name string
    err := s.db.QueryRow(
        ctx.Context(),
        "SELECT name FROM users WHERE id = $1",
        userID,
    ).Scan(&name)
    
    if err != nil {
        return s.Fail(msg, 500, err.Error())
    }
    
    return s.Reply(msg, map[string]interface{}{
        "id":   userID,
        "name": name,
    })
}
```

## Configuration

### PoolConfig (HikariCP-like)

```go
config := dbruntime.PoolConfig{
    DSN:             "postgres://user:pass@localhost/dbname",
    DriverName:      "postgres",
    MaxOpenConns:    25,              // maximumPoolSize
    MaxIdleConns:    5,               // minimumIdle
    ConnMaxLifetime: 5 * time.Minute, // connectionTimeout
    ConnMaxIdleTime: 10 * time.Minute, // idleTimeout
}
```

### Default Configuration

```go
// HikariCP-like defaults
config := dbruntime.DefaultPoolConfig(dsn, "postgres")
// MaxOpenConns: 25
// MaxIdleConns: 5
// ConnMaxLifetime: 5 minutes
// ConnMaxIdleTime: 10 minutes
```

## API Reference

### Pool

```go
// Create pool
pool, err := dbruntime.NewPool(config)

// Get underlying *sql.DB
db := pool.DB()

// Query operations
rows, err := pool.Query(ctx, query, args...)
row := pool.QueryRow(ctx, query, args...)
result, err := pool.Exec(ctx, query, args...)

// Transactions
tx, err := pool.Begin(ctx)
tx, err := pool.BeginTx(ctx, opts)

// Health check
err := pool.Ping(ctx)

// Statistics
stats := pool.Stats()

// Close pool
err := pool.Close()
```

### DatabaseComponent

```go
// Create component
component := dbruntime.NewDatabaseComponent(config)

// Lifecycle (Premium Pattern)
component.Start(ctx)
component.Stop(ctx)

// Database operations (same as Pool)
rows, err := component.Query(ctx, query, args...)
row := component.QueryRow(ctx, query, args...)
result, err := component.Exec(ctx, query, args...)
tx, err := component.Begin(ctx)

// Access pool
pool := component.Pool()
db := component.DB()

// Statistics
stats := component.Stats()
```

## Pool Statistics

Monitor pool health (similar to HikariPoolMXBean):

```go
stats := pool.Stats()

fmt.Printf("Open Connections: %d\n", stats.OpenConnections)
fmt.Printf("In Use: %d\n", stats.InUse)
fmt.Printf("Idle: %d\n", stats.Idle)
fmt.Printf("Wait Count: %d\n", stats.WaitCount)
fmt.Printf("Wait Duration: %v\n", stats.WaitDuration)
```

## Comparison with HikariCP

| HikariCP (Java) | Go db Package | Notes |
|----------------|--------------|-------|
| `HikariConfig` | `PoolConfig` | Similar configuration |
| `HikariDataSource` | `Pool` | Connection pool |
| `maximumPoolSize` | `MaxOpenConns` | Max connections |
| `minimumIdle` | `MaxIdleConns` | Min idle connections |
| `connectionTimeout` | `ConnMaxLifetime` | Connection lifetime |
| `idleTimeout` | `ConnMaxIdleTime` | Idle timeout |
| `HikariPoolMXBean` | `Pool.Stats()` | Pool statistics |
| Auto pool management | ✅ Built-in | Automatic |

## Best Practices

1. **Configure pool size** based on database `max_connections`
2. **Use context** for all operations (timeout, cancellation)
3. **Monitor statistics** regularly
4. **Close resources** properly (rows, transactions)
5. **Use Premium Pattern** for Fluxor integration

## Supported Databases

- PostgreSQL (`postgres` driver)
- MySQL (`mysql` driver)
- SQLite (`sqlite3` driver)
- Any database with `database/sql` driver

## Error Handling

### Pool Error Handling

```go
// Query returns error
rows, err := pool.Query(ctx, "SELECT * FROM users")
if err != nil {
    if e, ok := err.(*dbruntime.Error); ok {
        switch e.Code {
        case "INVALID_STATE":
            log.Printf("Pool not initialized")
        case "INVALID_INPUT":
            log.Printf("Invalid input: %v", err)
        }
    }
    return err
}
defer rows.Close()

// QueryRow panics on invalid state/input (consistent with database/sql)
// Always check pool is not nil before calling
row := pool.QueryRow(ctx, "SELECT name FROM users WHERE id = $1", userID)
var name string
if err := row.Scan(&name); err != nil {
    if err == sql.ErrNoRows {
        log.Printf("User not found")
    }
    return err
}
```

### Component Error Handling

```go
// Component methods return errors (not panics)
rows, err := component.Query(ctx, "SELECT * FROM users")
if err != nil {
    if e, ok := err.(*core.EventBusError); ok {
        switch e.Code {
        case "NOT_STARTED":
            log.Printf("Component not started")
        case "INVALID_INPUT":
            log.Printf("Invalid input")
        }
    }
    return err
}
```

## Connection Pool Tuning

### Basic Configuration

```go
config := dbruntime.PoolConfig{
    DSN:             "postgres://user:pass@localhost/dbname",
    DriverName:      "postgres",
    MaxOpenConns:    25,              // Maximum open connections
    MaxIdleConns:    5,               // Maximum idle connections
    ConnMaxLifetime: 5 * time.Minute, // Connection max lifetime
    ConnMaxIdleTime:  10 * time.Minute, // Connection max idle time
}
```

### High-Throughput Configuration

```go
// For high-throughput applications
config := dbruntime.PoolConfig{
    DSN:             dsn,
    DriverName:      "postgres",
    MaxOpenConns:    100,             // Higher for high throughput
    MaxIdleConns:    20,              // Keep more idle connections
    ConnMaxLifetime: 10 * time.Minute,
    ConnMaxIdleTime:  5 * time.Minute,
}
```

### Monitoring Pool Health

```go
stats := pool.Stats()

fmt.Printf("Open Connections: %d\n", stats.OpenConnections)
fmt.Printf("In Use: %d\n", stats.InUse)
fmt.Printf("Idle: %d\n", stats.Idle)
fmt.Printf("Wait Count: %d\n", stats.WaitCount)

// Health check
if stats.OpenConnections >= config.MaxOpenConns {
    log.Warn("Pool at maximum capacity")
}
```

### Sizing Guidelines

- **MaxOpenConns**: `(DB max_connections) / (number of app instances)`
- **MaxIdleConns**: `MaxOpenConns / 5` (20% of max)
- **ConnMaxLifetime**: 5-10 minutes (prevents stale connections)
- **ConnMaxIdleTime**: 10 minutes (releases idle connections)

## See Also

- [EXAMPLES.md](./EXAMPLES.md) - Complete examples with error handling and tuning
- [DATABASE_POOLING.md](../../DATABASE_POOLING.md) - Complete guide
- [BASE_CLASSES.md](../core/BASE_CLASSES.md) - Premium Pattern documentation
