# dbruntime Package - Complete Examples

## Error Handling

### Pool Error Handling

```go
import (
    "context"
    "github.com/fluxorio/fluxor/pkg/dbruntime"
)

// Create pool
pool, err := dbruntime.NewPool(dbruntime.DefaultPoolConfig(dsn, "postgres"))
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

ctx := context.Background()

// Query with error handling
rows, err := pool.Query(ctx, "SELECT * FROM users WHERE id = $1", userID)
if err != nil {
    if e, ok := err.(*dbruntime.Error); ok {
        switch e.Code {
        case "INVALID_STATE":
            log.Printf("Pool not initialized: %v", err)
        case "INVALID_INPUT":
            log.Printf("Invalid input: %v", err)
        default:
            log.Printf("Database error: %v", err)
        }
    } else {
        log.Printf("Unexpected error: %v", err)
    }
    return err
}
defer rows.Close()

// QueryRow - panics on invalid state/input (consistent with database/sql)
// Always check for nil pool before calling
row := pool.QueryRow(ctx, "SELECT name FROM users WHERE id = $1", userID)
var name string
if err := row.Scan(&name); err != nil {
    if err == sql.ErrNoRows {
        log.Printf("User not found")
    } else {
        log.Printf("Scan error: %v", err)
    }
    return err
}
```

### Component Error Handling

```go
import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/dbruntime"
)

type UserService struct {
    *core.BaseService
    db *dbruntime.DatabaseComponent
}

func (s *UserService) doStart(ctx core.FluxorContext) error {
    s.db.SetParent(s.BaseVerticle)
    return s.db.Start(ctx)
}

func (s *UserService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    userID := msg.Body().(string)
    
    // Query with error handling
    var name string
    err := s.db.QueryRow(
        ctx.Context(),
        "SELECT name FROM users WHERE id = $1",
        userID,
    ).Scan(&name)
    
    if err != nil {
        // Check error type
        if e, ok := err.(*core.EventBusError); ok {
            switch e.Code {
            case "NOT_STARTED":
                return s.Fail(msg, 503, "Database not ready")
            case "INVALID_INPUT":
                return s.Fail(msg, 400, "Invalid input")
            default:
                return s.Fail(msg, 500, "Database error")
            }
        }
        
        if err == sql.ErrNoRows {
            return s.Fail(msg, 404, "User not found")
        }
        
        return s.Fail(msg, 500, err.Error())
    }
    
    return s.Reply(msg, map[string]interface{}{"name": name})
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

pool, err := dbruntime.NewPool(config)
```

### High-Throughput Configuration

```go
// For high-throughput applications
config := dbruntime.PoolConfig{
    DSN:             dsn,
    DriverName:      "postgres",
    MaxOpenConns:    100,             // Higher for high throughput
    MaxIdleConns:    20,              // Keep more idle connections
    ConnMaxLifetime: 10 * time.Minute, // Longer lifetime
    ConnMaxIdleTime: 5 * time.Minute,  // Shorter idle time
}

pool, err := dbruntime.NewPool(config)
```

### Monitoring Pool Health

```go
// Monitor pool statistics
stats := pool.Stats()

fmt.Printf("Pool Statistics:\n")
fmt.Printf("  Open Connections: %d\n", stats.OpenConnections)
fmt.Printf("  In Use: %d\n", stats.InUse)
fmt.Printf("  Idle: %d\n", stats.Idle)
fmt.Printf("  Wait Count: %d\n", stats.WaitCount)
fmt.Printf("  Wait Duration: %v\n", stats.WaitDuration)

// Health check
if stats.OpenConnections >= config.MaxOpenConns {
    log.Warn("Pool at maximum capacity")
}

if stats.WaitCount > 0 {
    log.Warnf("Connections waiting: %d (duration: %v)", stats.WaitCount, stats.WaitDuration)
}
```

### Connection Pool Sizing Guidelines

```go
// Rule of thumb: MaxOpenConns = (DB max_connections) / (number of app instances)
// Example: DB allows 100 connections, 4 app instances
// Each instance: MaxOpenConns = 100 / 4 = 25

dbMaxConns := 100
numInstances := 4
maxOpenConns := dbMaxConns / numInstances

config := dbruntime.PoolConfig{
    DSN:          dsn,
    DriverName:   "postgres",
    MaxOpenConns: maxOpenConns,
    MaxIdleConns: maxOpenConns / 5, // 20% of max
}
```

## Transaction Examples

### Basic Transaction

```go
ctx := context.Background()

// Begin transaction
tx, err := pool.Begin(ctx)
if err != nil {
    log.Fatal(err)
}

// Always rollback on error
defer func() {
    if err != nil {
        tx.Rollback()
    }
}()

// Execute statements
_, err = tx.ExecContext(ctx, "INSERT INTO users (name) VALUES ($1)", "John")
if err != nil {
    return err
}

_, err = tx.ExecContext(ctx, "INSERT INTO profiles (user_id, bio) VALUES ($1, $2)", 1, "Bio")
if err != nil {
    return err
}

// Commit transaction
err = tx.Commit()
if err != nil {
    log.Fatal(err)
}
```

### Transaction with Options

```go
ctx := context.Background()

// Begin transaction with isolation level
opts := &sql.TxOptions{
    Isolation: sql.LevelReadCommitted,
    ReadOnly:  false,
}

tx, err := pool.BeginTx(ctx, opts)
if err != nil {
    log.Fatal(err)
}
defer tx.Rollback()

// Perform operations
// ...

err = tx.Commit()
```

## Component Lifecycle

### Complete Service Example

```go
type UserService struct {
    *core.BaseService
    db *dbruntime.DatabaseComponent
}

func NewUserService() *UserService {
    config := dbruntime.DefaultPoolConfig(
        "postgres://user:pass@localhost/dbname",
        "postgres",
    )
    return &UserService{
        BaseService: core.NewBaseService("user-service", "user.service"),
        db:          dbruntime.NewDatabaseComponent(config),
    }
}

func (s *UserService) doStart(ctx core.FluxorContext) error {
    // Set parent for component hierarchy
    s.db.SetParent(s.BaseVerticle)
    
    // Start database component
    if err := s.db.Start(ctx); err != nil {
        return err
    }
    
    // Verify connection
    if err := s.db.Ping(ctx.Context()); err != nil {
        return fmt.Errorf("database ping failed: %w", err)
    }
    
    return nil
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
    
    return s.Reply(msg, map[string]interface{}{"name": name})
}
```

## Best Practices

### 1. Always Use Context

```go
// ✅ Good: Use context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

rows, err := pool.Query(ctx, "SELECT * FROM users")

// ❌ Bad: No context
rows, err := pool.Query(context.Background(), "SELECT * FROM users")
```

### 2. Close Resources

```go
// ✅ Good: Always close rows
rows, err := pool.Query(ctx, "SELECT * FROM users")
if err != nil {
    return err
}
defer rows.Close()

// ✅ Good: Always close pool
pool, err := dbruntime.NewPool(config)
if err != nil {
    log.Fatal(err)
}
defer pool.Close()
```

### 3. Handle Errors Properly

```go
// ✅ Good: Check for specific error types
err := pool.Ping(ctx)
if err != nil {
    if e, ok := err.(*dbruntime.Error); ok {
        switch e.Code {
        case "INVALID_STATE":
            // Handle invalid state
        case "INVALID_INPUT":
            // Handle invalid input
        }
    }
    return err
}
```

### 4. Monitor Pool Statistics

```go
// ✅ Good: Monitor pool health
stats := pool.Stats()
if stats.OpenConnections >= config.MaxOpenConns {
    log.Warn("Pool at capacity")
}

if stats.WaitCount > 100 {
    log.Warn("High wait count - consider increasing pool size")
}
```

### 5. Use Transactions for Multiple Operations

```go
// ✅ Good: Use transaction for multiple operations
tx, err := pool.Begin(ctx)
if err != nil {
    return err
}
defer tx.Rollback()

tx.ExecContext(ctx, "INSERT INTO users ...")
tx.ExecContext(ctx, "INSERT INTO profiles ...")

err = tx.Commit()

// ❌ Bad: Multiple separate operations
pool.Exec(ctx, "INSERT INTO users ...")
pool.Exec(ctx, "INSERT INTO profiles ...") // If this fails, first insert is already committed
```
