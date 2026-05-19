// Package persistence provides a generic, high-performance data persistence layer
// for the Fluxor framework with repository pattern, transaction support, and query builders.
//
// The persistence package abstracts database operations and provides:
//
//   - Repository Pattern: Clean separation of data access logic
//   - Transaction Support: ACID transactions with rollback capability
//   - Query Builder: Flexible query construction with filters, sorting, and pagination
//   - Fail-Fast Validation: All operations validate inputs before processing
//   - Context Support: All operations accept context.Context for cancellation
//   - Database Agnostic: Works with any SQL database via database/sql
//
// Example usage:
//
//	import (
//	    "context"
//	    "database/sql"
//	    "github.com/fluxorio/fluxor/pkg/persistence"
//	    _ "github.com/lib/pq"
//	)
//
//	db, _ := sql.Open("postgres", "postgres://...")
//	config := persistence.DefaultConfig("users", db)
//	repo, _ := persistence.NewSQLRepository(config)
//
//	// Create entity
//	entity := map[string]interface{}{"name": "John", "email": "john@example.com"}
//	repo.Create(ctx, entity)
//
//	// Find by ID
//	user, _ := repo.FindByID(ctx, 1)
//
//	// Query with filters
//	query := persistence.NewQuery().
//	    WithFilter("status", "active").
//	    WithOrderBy("created_at", "DESC").
//	    WithLimit(10)
//	users, _ := repo.FindAll(ctx, query)
//
//	// Transaction
//	tx, _ := repo.BeginTransaction(ctx)
//	defer tx.Rollback()
//	txRepo := tx.Repository()
//	txRepo.Create(ctx, entity)
//	tx.Commit()
package persistence
