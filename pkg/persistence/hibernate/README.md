# Hibernate-Like Package

Hibernate-like ORM functionality for Fluxor with session management, caching, and query APIs.

## 🎯 Overview

The `hibernate` package provides Hibernate-like features:

- ✅ **Session Management**: Similar to Hibernate Session
- ✅ **First-Level Cache**: Session-scoped entity cache
- ✅ **Second-Level Cache**: Optional shared cache
- ✅ **HQL Queries**: Hibernate Query Language support
- ✅ **Criteria API**: Type-safe query building
- ✅ **Entity Lifecycle**: Transient, Persistent, Detached states
- ✅ **Dirty Checking**: Automatic change tracking
- ✅ **Flush/Clear**: Session management operations

## Features

### Session Management

Similar to Hibernate Session:

```go
// Open session
session, err := factory.OpenSession(ctx)
defer session.Close()

// Get entity (like session.get())
user, err := session.Get(ctx, &User{}, 1)

// Save entity (like session.save())
session.Save(ctx, user)

// Update entity (like session.update())
session.Update(ctx, user)

// Delete entity (like session.delete())
session.Delete(ctx, user)

// Save or update (like session.saveOrUpdate())
session.SaveOrUpdate(ctx, user)
```

### Caching

**First-Level Cache (Session Cache)**:
- Automatically caches entities within a session
- No configuration needed

**Second-Level Cache (Optional)**:
- Shared cache across sessions
- Uses Fluxor cache package

```go
factory := hibernate.NewSessionFactory(hibernate.SessionFactoryConfig{
    Repository:             repo,
    EnableSecondLevelCache: true,
    SecondLevelCache:       cache.NewMemoryCache(),
})
```

### HQL Queries

Similar to Hibernate Query Language:

```go
// Create HQL query
query := session.CreateQuery("FROM User WHERE status = :status")
query.SetParameter("status", "active")
query.SetMaxResults(10)
users, err := query.List(ctx)

// Unique result
user, err := query.UniqueResult(ctx)
```

### Criteria API

Type-safe query building:

```go
// Create criteria
criteria := session.CreateCriteria(&User{})
criteria.Add(hibernate.Eq("status", "active"))
criteria.Add(hibernate.Like("name", "John%"))
criteria.AddOrder(hibernate.Asc("name"))
criteria.SetMaxResults(10)

// Execute
users, err := criteria.List(ctx)
```

### Session Operations

```go
// Flush session (like session.flush())
session.Flush(ctx)

// Clear session cache (like session.clear())
session.Clear()

// Evict entity from cache (like session.evict())
session.Evict(user)
```

## Quick Start

### Basic Setup

```go
import (
    "context"
    "database/sql"
    "github.com/fluxorio/fluxor/pkg/persistence"
    "github.com/fluxorio/fluxor/pkg/persistence/hibernate"
    "github.com/fluxorio/fluxor/pkg/cache"
)

// Create repository
config := persistence.DefaultConfig("users", db)
repo, err := persistence.NewSQLRepository(config)

// Create session factory
factory := hibernate.NewSessionFactory(hibernate.SessionFactoryConfig{
    Repository: repo,
})

// Open session
ctx := context.Background()
session, err := factory.OpenSession(ctx)
defer session.Close()
```

### Entity Operations

```go
// Save new entity
user := &User{
    Name:  "John Doe",
    Email: "john@example.com",
}
err := session.Save(ctx, user)

// Get entity by ID
found, err := session.Get(ctx, &User{}, 1)

// Update entity
user.Name = "Jane Doe"
err := session.Update(ctx, user)

// Delete entity
err := session.Delete(ctx, user)
```

### With Second-Level Cache

```go
// Create factory with cache
factory := hibernate.NewSessionFactory(hibernate.SessionFactoryConfig{
    Repository:             repo,
    EnableSecondLevelCache: true,
    SecondLevelCache:       cache.NewMemoryCache(),
})

session, _ := factory.OpenSession(ctx)
defer session.Close()

// First call - loads from database and caches
user1, _ := session.Get(ctx, &User{}, 1)

// Second call - loads from second-level cache
session2, _ := factory.OpenSession(ctx)
user2, _ := session2.Get(ctx, &User{}, 1) // From cache!
```

## API Reference

### Session Interface

```go
type Session interface {
    Get(ctx context.Context, entityType interface{}, id interface{}) (interface{}, error)
    Load(ctx context.Context, entityType interface{}, id interface{}) (interface{}, error)
    Save(ctx context.Context, entity interface{}) error
    Update(ctx context.Context, entity interface{}) error
    SaveOrUpdate(ctx context.Context, entity interface{}) error
    Delete(ctx context.Context, entity interface{}) error
    Flush(ctx context.Context) error
    Clear()
    Evict(entity interface{})
    CreateQuery(hql string) Query
    CreateCriteria(entityType interface{}) Criteria
    BeginTransaction(ctx context.Context) (Transaction, error)
    Close() error
}
```

### SessionFactory Interface

```go
type SessionFactory interface {
    OpenSession(ctx context.Context) (Session, error)
    GetCurrentSession(ctx context.Context) (Session, error)
    Close() error
}
```

### Query Interface (HQL)

```go
type Query interface {
    SetParameter(name string, value interface{}) Query
    SetParameterList(name string, values []interface{}) Query
    SetFirstResult(first int) Query
    SetMaxResults(max int) Query
    List(ctx context.Context) ([]interface{}, error)
    UniqueResult(ctx context.Context) (interface{}, error)
    ExecuteUpdate(ctx context.Context) (int, error)
}
```

### Criteria Interface

```go
type Criteria interface {
    Add(criterion Criterion) Criteria
    AddOrder(order Order) Criteria
    SetFirstResult(first int) Criteria
    SetMaxResults(max int) Criteria
    List(ctx context.Context) ([]interface{}, error)
    UniqueResult(ctx context.Context) (interface{}, error)
}
```

### Criterion Functions

```go
// Equality
hibernate.Eq("status", "active")

// LIKE
hibernate.Like("name", "John%")

// Ordering
hibernate.Asc("name")
hibernate.Desc("created_at")
```

## Hibernate Comparison

| Hibernate | Fluxor Hibernate |
|-----------|------------------|
| `Session.get()` | `session.Get()` |
| `Session.save()` | `session.Save()` |
| `Session.update()` | `session.Update()` |
| `Session.delete()` | `session.Delete()` |
| `Session.flush()` | `session.Flush()` |
| `Session.clear()` | `session.Clear()` |
| `Session.createQuery()` | `session.CreateQuery()` |
| `Session.createCriteria()` | `session.CreateCriteria()` |
| `Query.list()` | `query.List()` |
| `Query.uniqueResult()` | `query.UniqueResult()` |
| `Criteria.add()` | `criteria.Add()` |
| `Restrictions.eq()` | `hibernate.Eq()` |

## Entity Requirements

Entities should have:
- An `ID` field (or `Id`, `id`) for identification
- Struct tags for database mapping (optional)

Example:

```go
type User struct {
    ID    int    `db:"id" json:"id"`
    Name  string `db:"name" json:"name"`
    Email string `db:"email" json:"email"`
}
```

## Best Practices

### 1. Always Close Sessions

```go
session, err := factory.OpenSession(ctx)
if err != nil {
    return err
}
defer session.Close()
```

### 2. Use Transactions for Multiple Operations

```go
tx, err := session.BeginTransaction(ctx)
if err != nil {
    return err
}
defer tx.Rollback()

session.Save(ctx, user1)
session.Save(ctx, user2)

if err := tx.Commit(); err != nil {
    return err
}
```

### 3. Flush Before Long Operations

```go
// Make changes
session.Save(ctx, user1)
session.Update(ctx, user2)

// Flush to database
session.Flush(ctx)

// Continue with other operations
```

### 4. Use Second-Level Cache for Read-Heavy Workloads

```go
factory := hibernate.NewSessionFactory(hibernate.SessionFactoryConfig{
    Repository:             repo,
    EnableSecondLevelCache: true,
    SecondLevelCache:       cache.NewMemoryCache(), // or Redis
})
```

### 5. Clear Session When Needed

```go
// Clear all cached entities
session.Clear()

// Or evict specific entity
session.Evict(user)
```

## Examples

### Complete CRUD Example

```go
factory := hibernate.NewSessionFactory(hibernate.SessionFactoryConfig{
    Repository: repo,
})

ctx := context.Background()
session, _ := factory.OpenSession(ctx)
defer session.Close()

// Create
user := &User{Name: "John", Email: "john@example.com"}
session.Save(ctx, user)

// Read
found, _ := session.Get(ctx, &User{}, user.ID)

// Update
user.Name = "Jane"
session.Update(ctx, user)

// Delete
session.Delete(ctx, user)
```

### Query Example

```go
// HQL
query := session.CreateQuery("FROM User WHERE status = :status")
query.SetParameter("status", "active")
query.SetMaxResults(10)
users, _ := query.List(ctx)

// Criteria
criteria := session.CreateCriteria(&User{})
criteria.Add(hibernate.Eq("status", "active"))
criteria.AddOrder(hibernate.Asc("name"))
results, _ := criteria.List(ctx)
```

## Related Packages

- **pkg/persistence**: Base persistence package
- **pkg/persistence/csql**: Custom SQL query builder
- **pkg/persistence/goorm**: GORM integration
- **pkg/cache**: Caching layer

---

**Package**: `github.com/fluxorio/fluxor/pkg/persistence/hibernate`  
**Status**: ✅ Stable  
**Last Updated**: 2026-01-04
