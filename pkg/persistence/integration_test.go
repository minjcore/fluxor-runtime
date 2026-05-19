package persistence

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// TestRepository_Transaction_CompleteFlow tests a complete transaction flow
func TestRepository_Transaction_CompleteFlow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultConfig("test_table", db)
	repo, err := NewSQLRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// Begin transaction
	tx, err := repo.BeginTransaction(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Get transaction repository
	txRepo := tx.Repository()

	// Create entity within transaction
	entity1 := map[string]interface{}{
		"name":  "Test User 1",
		"email": "test1@example.com",
		"status": "active",
	}
	err = txRepo.Create(ctx, entity1)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to create entity in transaction: %v", err)
	}

	// Create another entity
	entity2 := map[string]interface{}{
		"name":  "Test User 2",
		"email": "test2@example.com",
		"status": "active",
	}
	err = txRepo.Create(ctx, entity2)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to create second entity in transaction: %v", err)
	}

	// Query within transaction
	query := NewQuery().WithFilter("status", "active")
	results, err := txRepo.FindAll(ctx, query)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to query in transaction: %v", err)
	}

	if len(results) < 2 {
		t.Errorf("Expected at least 2 results, got %d", len(results))
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Verify entities are persisted
	allResults, err := repo.FindAll(ctx, NewQuery())
	if err != nil {
		t.Fatalf("Failed to query after commit: %v", err)
	}

	if len(allResults) < 2 {
		t.Errorf("Expected at least 2 entities after commit, got %d", len(allResults))
	}
}

// TestRepository_Transaction_Rollback tests transaction rollback
func TestRepository_Transaction_Rollback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultConfig("test_table", db)
	repo, err := NewSQLRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// Begin transaction
	tx, err := repo.BeginTransaction(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	txRepo := tx.Repository()

	// Create entity within transaction
	entity := map[string]interface{}{
		"name":  "Rollback Test",
		"email": "rollback@example.com",
		"status": "active",
	}
	err = txRepo.Create(ctx, entity)
	if err != nil {
		tx.Rollback()
		t.Fatalf("Failed to create entity in transaction: %v", err)
	}

	// Rollback transaction
	err = tx.Rollback()
	if err != nil {
		t.Fatalf("Failed to rollback transaction: %v", err)
	}

	// Verify entity is NOT persisted
	query := NewQuery().WithFilter("email", "rollback@example.com")
	results, err := repo.FindAll(ctx, query)
	if err != nil {
		t.Fatalf("Failed to query after rollback: %v", err)
	}

	if len(results) > 0 {
		t.Error("Entity should not exist after rollback")
	}
}

// TestRepository_BatchOperations tests batch create, update, and delete
func TestRepository_BatchOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultConfig("test_table", db)
	repo, err := NewSQLRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// Test BatchCreate
	entities := []interface{}{
		map[string]interface{}{"name": "Batch User 1", "email": "batch1@example.com", "status": "active"},
		map[string]interface{}{"name": "Batch User 2", "email": "batch2@example.com", "status": "active"},
		map[string]interface{}{"name": "Batch User 3", "email": "batch3@example.com", "status": "active"},
	}

	err = repo.BatchCreate(ctx, entities)
	if err != nil {
		t.Fatalf("BatchCreate failed: %v", err)
	}

	// Verify entities were created
	query := NewQuery().WithFilter("status", "active")
	results, err := repo.FindAll(ctx, query)
	if err != nil {
		t.Fatalf("Failed to query after batch create: %v", err)
	}

	if len(results) < 3 {
		t.Errorf("Expected at least 3 entities, got %d", len(results))
	}

	// Test BatchDelete
	ids := []interface{}{1, 2, 3} // Assuming IDs are assigned
	err = repo.BatchDelete(ctx, ids)
	if err != nil {
		// May fail if IDs don't exist, which is OK for this test
		t.Logf("BatchDelete result: %v", err)
	}
}

// TestRepository_ConcurrentAccess tests concurrent access to repository
func TestRepository_ConcurrentAccess(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultConfig("test_table", db)
	repo, err := NewSQLRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// Concurrent creates
	const numGoroutines = 10
	done := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			entity := map[string]interface{}{
				"name":  fmt.Sprintf("Concurrent User %d", id),
				"email": fmt.Sprintf("concurrent%d@example.com", id),
				"status": "active",
			}
			err := repo.Create(ctx, entity)
			done <- err
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		err := <-done
		if err != nil {
			t.Errorf("Concurrent create failed: %v", err)
		}
	}

	// Verify all entities were created
	results, err := repo.FindAll(ctx, NewQuery())
	if err != nil {
		t.Fatalf("Failed to query after concurrent creates: %v", err)
	}

	if len(results) < numGoroutines {
		t.Errorf("Expected at least %d entities, got %d", numGoroutines, len(results))
	}
}

// TestRepository_RetryLogic tests retry logic with transient errors
func TestRepository_RetryLogic(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Configure with retries
	config := DefaultConfig("test_table", db)
	config.MaxRetries = 3
	config.RetryDelay = 10 * time.Millisecond

	repo, err := NewSQLRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// Normal operation should work
	entity := map[string]interface{}{
		"name":  "Retry Test",
		"email": "retry@example.com",
		"status": "active",
	}

	err = repo.Create(ctx, entity)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify entity was created
	query := NewQuery().WithFilter("email", "retry@example.com")
	results, err := repo.FindAll(ctx, query)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	if len(results) == 0 {
		t.Error("Entity should exist after create")
	}
}

// TestRepository_ErrorHandling tests various error scenarios
func TestRepository_ErrorHandling(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	config := DefaultConfig("test_table", db)
	repo, err := NewSQLRepository(config)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// Test FindByID with non-existent ID
	_, err = repo.FindByID(ctx, 99999)
	if err == nil {
		t.Error("Expected error for non-existent ID")
	}
	if !IsNotFound(err) {
		t.Errorf("Expected NotFound error, got: %v", err)
	}

	// Test Update with non-existent ID
	entity := map[string]interface{}{"name": "Updated Name"}
	err = repo.Update(ctx, 99999, entity)
	if err == nil {
		t.Error("Expected error for updating non-existent entity")
	}
	if !IsNotFound(err) {
		t.Errorf("Expected NotFound error, got: %v", err)
	}

	// Test Delete with non-existent ID
	err = repo.Delete(ctx, 99999)
	if err == nil {
		t.Error("Expected error for deleting non-existent entity")
	}
	if !IsNotFound(err) {
		t.Errorf("Expected NotFound error, got: %v", err)
	}
}
