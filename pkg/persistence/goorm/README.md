# GORM Package - GORM Integration

GORM integration for the Fluxor persistence package, providing type-safe ORM capabilities.

## 🎯 Overview

The `goorm` package provides:

- ✅ **GORM Integration**: Full GORM ORM support
- ✅ **Type-Safe Queries**: Use Go structs instead of maps
- ✅ **Repository Pattern**: Implements persistence.Repository interface
- ✅ **Transaction Support**: Full ACID transaction support
- ✅ **Soft Delete**: Optional soft delete support
- ✅ **Migrations**: GORM's migration capabilities

## Features

- **Type Safety**: Use Go structs for type-safe database operations
- **Associations**: GORM's powerful association features
- **Hooks**: GORM's before/after hooks
- **Validations**: GORM's validation system
- **Migrations**: Automatic schema migration
- **Query Builder**: GORM's fluent query builder

## Prerequisites

Install GORM:

```bash
go get -u gorm.io/gorm
go get -u gorm.io/driver/postgres  # For PostgreSQL
go get -u gorm.io/driver/mysql      # For MySQL
go get -u gorm.io/driver/sqlite     # For SQLite
```

## Quick Start

### Basic Setup

```go
import (
    "context"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "github.com/fluxorio/fluxor/pkg/persistence/goorm"
)

// Create GORM DB instance
dsn := "host=localhost user=user password=pass dbname=dbname port=5432 sslmode=disable"
db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
if err != nil {
    log.Fatal(err)
}

// Define model
type User struct {
    ID    uint   `gorm:"primaryKey"`
    Name  string
    Email string `gorm:"uniqueIndex"`
}

// Create GORM repository
config := goorm.Config{
    DB:      db,
    Model:   &User{},
    TableName: "users",
}

repo, err := goorm.NewGORMRepository(config)
if err != nil {
    log.Fatal(err)
}
defer repo.Close()

ctx := context.Background()

// Create user
user := &User{Name: "John Doe", Email: "john@example.com"}
err = repo.Create(ctx, user)

// Find by ID
foundUser, err := repo.FindByID(ctx, user.ID)
userResult := foundUser.(*User)

// Update user
user.Name = "Jane Doe"
err = repo.Update(ctx, user.ID, user)

// Delete user
err = repo.Delete(ctx, user.ID)
```

### With Query Builder

```go
// Build query
query := persistence.NewQuery().
    WithFilter("email", "john@example.com").
    WithOrderBy("created_at", "DESC").
    WithLimit(10)

// Find all matching users
results, err := repo.FindAll(ctx, query)
users := make([]*User, len(results))
for i, r := range results {
    users[i] = r.(*User)
}

// Find one
user, err := repo.FindOne(ctx, query)
```

### Transactions

```go
// Begin transaction
tx, err := repo.BeginTransaction(ctx)
if err != nil {
    log.Fatal(err)
}

// Get repository bound to transaction
txRepo := tx.Repository()

// Perform operations
user1 := &User{Name: "User 1", Email: "user1@example.com"}
err = txRepo.Create(ctx, user1)
if err != nil {
    tx.Rollback()
    return err
}

user2 := &User{Name: "User 2", Email: "user2@example.com"}
err = txRepo.Create(ctx, user2)
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

### Advanced GORM Features

```go
// Access underlying GORM DB for advanced operations
gormDB := repo.DB()

// Use GORM's query builder
var users []User
gormDB.WithContext(ctx).
    Where("age > ?", 18).
    Where("status = ?", "active").
    Preload("Profile").
    Find(&users)

// Use GORM associations
type User struct {
    ID      uint
    Name    string
    Profile Profile `gorm:"foreignKey:UserID"`
}

type Profile struct {
    ID     uint
    UserID uint
    Bio    string
}

// Preload associations
var user User
gormDB.WithContext(ctx).Preload("Profile").First(&user, 1)
```

### Migrations

```go
// Auto migrate
err := db.AutoMigrate(&User{}, &Profile{})
if err != nil {
    log.Fatal(err)
}

// Manual migrations
db.Migrator().CreateTable(&User{})
db.Migrator().AddColumn(&User{}, "Age")
db.Migrator().DropColumn(&User{}, "OldField")
```

### Soft Delete

```go
// Enable soft delete in config
config := goorm.Config{
    DB:              db,
    Model:           &User{},
    EnableSoftDelete: true,
}

// Model with soft delete
type User struct {
    ID        uint
    Name      string
    DeletedAt gorm.DeletedAt `gorm:"index"`
}

// Delete will soft delete
err := repo.Delete(ctx, userID)

// To hard delete, use GORM directly
gormDB := repo.DB()
gormDB.Unscoped().Delete(&User{}, userID)
```

## API Reference

### Config

```go
type Config struct {
    DB              *gorm.DB
    TableName       string
    Model           interface{}
    EnableSoftDelete bool
}

func (c *Config) Validate() error
```

### GORMRepository

```go
type GORMRepository struct {
    db     *gorm.DB
    config Config
}

func NewGORMRepository(config Config) (*GORMRepository, error)
func (r *GORMRepository) FindByID(ctx context.Context, id interface{}) (interface{}, error)
func (r *GORMRepository) FindAll(ctx context.Context, query *persistence.Query) ([]interface{}, error)
func (r *GORMRepository) FindOne(ctx context.Context, query *persistence.Query) (interface{}, error)
func (r *GORMRepository) Create(ctx context.Context, entity interface{}) error
func (r *GORMRepository) Update(ctx context.Context, id interface{}, entity interface{}) error
func (r *GORMRepository) Delete(ctx context.Context, id interface{}) error
func (r *GORMRepository) Count(ctx context.Context, query *persistence.Query) (int64, error)
func (r *GORMRepository) Exists(ctx context.Context, id interface{}) (bool, error)
func (r *GORMRepository) BeginTransaction(ctx context.Context) (persistence.Transaction, error)
func (r *GORMRepository) Close() error
func (r *GORMRepository) DB() *gorm.DB
```

## Best Practices

### 1. Use Type-Safe Models

```go
// ✅ Good: Define struct models
type User struct {
    ID    uint   `gorm:"primaryKey"`
    Name  string `gorm:"not null"`
    Email string `gorm:"uniqueIndex;not null"`
}

// ❌ Avoid: Using maps
entity := map[string]interface{}{
    "name": "John",
    "email": "john@example.com",
}
```

### 2. Use GORM Tags

```go
type User struct {
    ID        uint           `gorm:"primaryKey"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
    Name      string         `gorm:"size:255;not null"`
    Email     string         `gorm:"uniqueIndex;size:255"`
    Age       int            `gorm:"check:age > 0"`
}
```

### 3. Use Preload for Associations

```go
// ✅ Good: Preload associations
var user User
gormDB.WithContext(ctx).Preload("Profile").Preload("Orders").First(&user, 1)

// ❌ Avoid: N+1 queries
var user User
gormDB.First(&user, 1)
// Then querying Profile and Orders separately
```

### 4. Use Transactions for Multiple Operations

```go
// ✅ Good: Use transaction for multiple operations
tx, _ := repo.BeginTransaction(ctx)
txRepo := tx.Repository()
txRepo.Create(ctx, user1)
txRepo.Create(ctx, user2)
tx.Commit()

// ❌ Avoid: Multiple separate operations
repo.Create(ctx, user1)
repo.Create(ctx, user2) // If this fails, user1 is already created
```

### 5. Handle Errors Properly

```go
user, err := repo.FindByID(ctx, id)
if err != nil {
    if persistence.IsNotFound(err) {
        // Handle not found
    } else {
        // Handle other errors
    }
}
```

## Integration with Fluxor

### With dbruntime Pool

```go
import (
    "github.com/fluxorio/fluxor/pkg/dbruntime"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "github.com/fluxorio/fluxor/pkg/persistence/goorm"
)

// Create connection pool
pool, err := dbruntime.NewPool(dbruntime.PoolConfig{
    DSN:        dsn,
    DriverName: "postgres",
})
if err != nil {
    log.Fatal(err)
}

// Get underlying sql.DB
sqlDB := pool.DB()

// Create GORM DB from sql.DB
gormDB, err := gorm.Open(postgres.New(postgres.Config{
    Conn: sqlDB,
}), &gorm.Config{})

// Create repository
config := goorm.Config{
    DB:    gormDB,
    Model: &User{},
}
repo, err := goorm.NewGORMRepository(config)
```

## Examples

### Complete CRUD Example

```go
package main

import (
    "context"
    "log"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "github.com/fluxorio/fluxor/pkg/persistence/goorm"
)

type User struct {
    ID    uint   `gorm:"primaryKey"`
    Name  string
    Email string `gorm:"uniqueIndex"`
}

func main() {
    dsn := "host=localhost user=user password=pass dbname=dbname port=5432"
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal(err)
    }

    // Auto migrate
    db.AutoMigrate(&User{})

    // Create repository
    config := goorm.Config{
        DB:    db,
        Model: &User{},
    }
    repo, err := goorm.NewGORMRepository(config)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Create
    user := &User{Name: "John", Email: "john@example.com"}
    repo.Create(ctx, user)

    // Read
    found, _ := repo.FindByID(ctx, user.ID)
    log.Printf("Found: %+v", found)

    // Update
    user.Name = "Jane"
    repo.Update(ctx, user.ID, user)

    // Delete
    repo.Delete(ctx, user.ID)
}
```

## Related Packages

- **pkg/persistence**: Base persistence package
- **pkg/persistence/csql**: Custom SQL query builder
- **pkg/dbruntime**: Database connection pooling

---

**Package**: `github.com/fluxorio/fluxor/pkg/persistence/goorm`  
**Status**: ✅ Stable  
**Last Updated**: 2026-01-04
