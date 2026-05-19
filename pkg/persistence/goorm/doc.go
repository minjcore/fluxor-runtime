// Package goorm provides GORM integration for the Fluxor persistence package.
//
// The goorm package provides:
//
//   - GORM Integration: Full GORM ORM support
//   - Type-Safe Queries: Use Go structs instead of maps
//   - Repository Pattern: Implements persistence.Repository interface
//   - Transaction Support: Full ACID transaction support
//   - Soft Delete: Optional soft delete support
//   - Migrations: GORM's migration capabilities
//
// Example usage:
//
//	import (
//	    "context"
//	    "gorm.io/driver/postgres"
//	    "gorm.io/gorm"
//	    "github.com/fluxorio/fluxor/pkg/persistence/goorm"
//	)
//
//	db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
//	config := goorm.Config{
//	    DB:    db,
//	    Model: &User{},
//	}
//	repo, _ := goorm.NewGORMRepository(config)
//
//	// Create
//	user := &User{Name: "John", Email: "john@example.com"}
//	repo.Create(ctx, user)
//
//	// Find
//	found, _ := repo.FindByID(ctx, user.ID)
package goorm
