package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/fluxorio/fluxor/pkg/config"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/dbruntime"
	"github.com/fluxorio/fluxor/pkg/persistence"
)

// TestTransactionIsolationLevels tests the new BeginTransactionWithOptions feature
func TestTransactionIsolationLevels(t *testing.T) {
	// Setup: Create database component
	appConfig := loadTestConfig(t)
	poolConfig := dbruntime.PoolConfig{
		DSN:             appConfig.DSN,
		DriverName:      "postgres",
		MaxOpenConns:    appConfig.MaxOpenConns,
		MaxIdleConns:    appConfig.MaxIdleConns,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	db := dbruntime.NewDatabaseComponent(poolConfig)
	ctx := core.NewFluxorContext(context.Background())
	
	if err := db.Start(ctx); err != nil {
		t.Fatalf("Failed to start database: %v", err)
	}
	defer db.Stop(ctx)

	// Create repository
	repoConfig := persistence.DefaultConfig("test_users", db.DB())
	repo, err := persistence.NewSQLRepository(repoConfig)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Setup test table
	setupTestTable(t, db.DB())
	defer cleanupTestTable(t, db.DB())

	// Test 1: Default isolation level (Read Committed)
	t.Run("DefaultIsolationLevel", func(t *testing.T) {
		tx, err := repo.BeginTransaction(ctx.Context())
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}
		defer tx.Rollback()

		txRepo := tx.Repository()
		err = txRepo.Create(ctx.Context(), map[string]interface{}{
			"name":  "Test User 1",
			"email": "test1@example.com",
		})
		if err != nil {
			t.Fatalf("Failed to create entity: %v", err)
		}

		if err := tx.Commit(); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}
		t.Log("✅ Default isolation level transaction successful")
	})

	// Test 2: Serializable isolation level
	t.Run("SerializableIsolationLevel", func(t *testing.T) {
		opts := &sql.TxOptions{
			Isolation: sql.LevelSerializable,
			ReadOnly:  false,
		}
		tx, err := repo.BeginTransactionWithOptions(ctx.Context(), opts)
		if err != nil {
			t.Fatalf("Failed to begin transaction with options: %v", err)
		}
		defer tx.Rollback()

		txRepo := tx.Repository()
		err = txRepo.Create(ctx.Context(), map[string]interface{}{
			"name":  "Test User 2",
			"email": "test2@example.com",
		})
		if err != nil {
			t.Fatalf("Failed to create entity: %v", err)
		}

		if err := tx.Commit(); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}
		t.Log("✅ Serializable isolation level transaction successful")
	})

	// Test 3: Read-only transaction
	t.Run("ReadOnlyTransaction", func(t *testing.T) {
		opts := &sql.TxOptions{
			Isolation: sql.LevelReadCommitted,
			ReadOnly:  true,
		}
		tx, err := repo.BeginTransactionWithOptions(ctx.Context(), opts)
		if err != nil {
			t.Fatalf("Failed to begin read-only transaction: %v", err)
		}
		defer tx.Rollback()

		// Read-only transaction should allow reads
		txRepo := tx.Repository()
		_, err = txRepo.FindAll(ctx.Context(), persistence.NewQuery().WithLimit(10))
		if err != nil {
			t.Fatalf("Failed to read in read-only transaction: %v", err)
		}

		if err := tx.Commit(); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}
		t.Log("✅ Read-only transaction successful")
	})
}

// TestSavepointSupport tests the new savepoint feature
func TestSavepointSupport(t *testing.T) {
	// Setup: Create database component
	appConfig := loadTestConfig(t)
	poolConfig := dbruntime.PoolConfig{
		DSN:             appConfig.DSN,
		DriverName:      "postgres",
		MaxOpenConns:    appConfig.MaxOpenConns,
		MaxIdleConns:    appConfig.MaxIdleConns,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	db := dbruntime.NewDatabaseComponent(poolConfig)
	ctx := core.NewFluxorContext(context.Background())
	
	if err := db.Start(ctx); err != nil {
		t.Fatalf("Failed to start database: %v", err)
	}
	defer db.Stop(ctx)

	// Create repository
	repoConfig := persistence.DefaultConfig("test_users", db.DB())
	repo, err := persistence.NewSQLRepository(repoConfig)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Setup test table
	setupTestTable(t, db.DB())
	defer cleanupTestTable(t, db.DB())

	// Test 1: Create savepoint and rollback to it
	t.Run("SavepointRollback", func(t *testing.T) {
		tx, err := repo.BeginTransaction(ctx.Context())
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}
		defer tx.Rollback()

		txRepo := tx.Repository()

		// Create first entity
		err = txRepo.Create(ctx.Context(), map[string]interface{}{
			"name":  "User Before Savepoint",
			"email": "before@example.com",
		})
		if err != nil {
			t.Fatalf("Failed to create entity: %v", err)
		}

		// Create savepoint
		if err := tx.Savepoint(ctx.Context(), "sp1"); err != nil {
			t.Fatalf("Failed to create savepoint: %v", err)
		}
		t.Log("✅ Savepoint created successfully")

		// Create second entity after savepoint
		err = txRepo.Create(ctx.Context(), map[string]interface{}{
			"name":  "User After Savepoint",
			"email": "after@example.com",
		})
		if err != nil {
			t.Fatalf("Failed to create entity after savepoint: %v", err)
		}

		// Rollback to savepoint (should undo second entity)
		if err := tx.RollbackToSavepoint(ctx.Context(), "sp1"); err != nil {
			t.Fatalf("Failed to rollback to savepoint: %v", err)
		}
		t.Log("✅ Rollback to savepoint successful")

		// Commit transaction (should only have first entity)
		if err := tx.Commit(); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}

		// Verify: Only first entity should exist
		results, err := repo.FindAll(ctx.Context(), persistence.NewQuery())
		if err != nil {
			t.Fatalf("Failed to query results: %v", err)
		}

		// Count entities with "before" in email
		beforeCount := 0
		for _, result := range results {
			entity := result.(map[string]interface{})
			email := entity["email"].(string)
			if email == "before@example.com" {
				beforeCount++
			}
			if email == "after@example.com" {
				t.Errorf("Entity after savepoint should not exist after rollback")
			}
		}

		if beforeCount == 0 {
			t.Errorf("Entity before savepoint should exist")
		}
		t.Log("✅ Savepoint rollback verification successful")
	})

	// Test 2: Release savepoint
	t.Run("ReleaseSavepoint", func(t *testing.T) {
		tx, err := repo.BeginTransaction(ctx.Context())
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}
		defer tx.Rollback()

		// Create savepoint
		if err := tx.Savepoint(ctx.Context(), "sp2"); err != nil {
			t.Fatalf("Failed to create savepoint: %v", err)
		}

		// Release savepoint
		if err := tx.ReleaseSavepoint(ctx.Context(), "sp2"); err != nil {
			t.Fatalf("Failed to release savepoint: %v", err)
		}
		t.Log("✅ Savepoint released successfully")

		// Try to rollback to released savepoint (should fail)
		err = tx.RollbackToSavepoint(ctx.Context(), "sp2")
		if err == nil {
			t.Errorf("Rollback to released savepoint should fail")
		}
		t.Log("✅ Rollback to released savepoint correctly failed")

		if err := tx.Commit(); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}
	})

	// Test 3: Multiple savepoints
	t.Run("MultipleSavepoints", func(t *testing.T) {
		tx, err := repo.BeginTransaction(ctx.Context())
		if err != nil {
			t.Fatalf("Failed to begin transaction: %v", err)
		}
		defer tx.Rollback()

		txRepo := tx.Repository()

		// Create entity 1
		err = txRepo.Create(ctx.Context(), map[string]interface{}{
			"name":  "User 1",
			"email": "user1@example.com",
		})
		if err != nil {
			t.Fatalf("Failed to create entity 1: %v", err)
		}

		// Create savepoint 1
		if err := tx.Savepoint(ctx.Context(), "sp3"); err != nil {
			t.Fatalf("Failed to create savepoint 1: %v", err)
		}

		// Create entity 2
		err = txRepo.Create(ctx.Context(), map[string]interface{}{
			"name":  "User 2",
			"email": "user2@example.com",
		})
		if err != nil {
			t.Fatalf("Failed to create entity 2: %v", err)
		}

		// Create savepoint 2
		if err := tx.Savepoint(ctx.Context(), "sp4"); err != nil {
			t.Fatalf("Failed to create savepoint 2: %v", err)
		}

		// Create entity 3
		err = txRepo.Create(ctx.Context(), map[string]interface{}{
			"name":  "User 3",
			"email": "user3@example.com",
		})
		if err != nil {
			t.Fatalf("Failed to create entity 3: %v", err)
		}

		// Rollback to savepoint 2 (should undo entity 3)
		if err := tx.RollbackToSavepoint(ctx.Context(), "sp4"); err != nil {
			t.Fatalf("Failed to rollback to savepoint 2: %v", err)
		}

		// Rollback to savepoint 1 (should undo entity 2)
		if err := tx.RollbackToSavepoint(ctx.Context(), "sp3"); err != nil {
			t.Fatalf("Failed to rollback to savepoint 1: %v", err)
		}

		// Commit (should only have entity 1)
		if err := tx.Commit(); err != nil {
			t.Fatalf("Failed to commit: %v", err)
		}

		t.Log("✅ Multiple savepoints test successful")
	})
}

// TestErrorSanitization tests the new error sanitization feature
func TestErrorSanitization(t *testing.T) {
	// Test 1: SafeError method
	t.Run("SafeError", func(t *testing.T) {
		err := persistence.NewError(
			persistence.ErrCodeQuery,
			"query execution failed",
			fmt.Errorf("connection failed: password=secret123"),
		)

		safeMsg := err.SafeError()
		if safeMsg == "" {
			t.Errorf("SafeError should return a message")
		}

		// Safe error should not contain the cause
		if contains(safeMsg, "password") || contains(safeMsg, "secret123") {
			t.Errorf("SafeError should not contain sensitive information: %s", safeMsg)
		}

		t.Logf("✅ SafeError: %s", safeMsg)
	})

	// Test 2: SanitizeError function
	t.Run("SanitizeError", func(t *testing.T) {
		// Test with persistence error
		persistenceErr := persistence.NewError(
			persistence.ErrCodeConnection,
			"connection failed",
			fmt.Errorf("dsn=postgres://user:password@host/db"),
		)

		sanitized := persistence.SanitizeError(persistenceErr)
		if contains(sanitized, "password") || contains(sanitized, "dsn=") {
			t.Errorf("SanitizeError should remove sensitive info: %s", sanitized)
		}
		t.Logf("✅ Sanitized persistence error: %s", sanitized)

		// Test with generic error containing sensitive info
		genericErr := fmt.Errorf("connection failed: password=secret123 host=localhost")
		sanitized = persistence.SanitizeError(genericErr)
		if contains(sanitized, "password") || contains(sanitized, "secret123") {
			t.Errorf("SanitizeError should remove sensitive info from generic errors: %s", sanitized)
		}
		t.Logf("✅ Sanitized generic error: %s", sanitized)
	})
}

// Helper functions

func loadTestConfig(t *testing.T) AppConfig {
	var appConfig AppConfig
	configPath := "application.properties"
	if err := config.LoadProperties(configPath, &appConfig); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Set defaults if not provided
	if appConfig.MaxOpenConns == 0 {
		appConfig.MaxOpenConns = 25
	}
	if appConfig.MaxIdleConns == 0 {
		appConfig.MaxIdleConns = 5
	}

	return appConfig
}

func setupTestTable(t *testing.T, db *sql.DB) {
	// Drop table if exists
	_, _ = db.Exec("DROP TABLE IF EXISTS test_users")

	// Create table
	_, err := db.Exec(`
		CREATE TABLE test_users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}
}

func cleanupTestTable(t *testing.T, db *sql.DB) {
	_, err := db.Exec("DROP TABLE IF EXISTS test_users")
	if err != nil {
		t.Logf("Warning: Failed to cleanup test table: %v", err)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// RunTests runs all persistence feature tests
func RunTests() {
	log.Println("=" + strings.Repeat("=", 60))
	log.Println("Running Persistence Feature Tests")
	log.Println("=" + strings.Repeat("=", 60))

	// Test error sanitization (doesn't require database)
	log.Println("\n[Test 1] Error Sanitization")
	testErrorSanitization()
	log.Println("✅ Error sanitization tests passed")

	// Test transaction isolation levels
	log.Println("\n[Test 2] Transaction Isolation Levels")
	if err := testTransactionIsolationLevels(); err != nil {
		log.Printf("❌ Transaction isolation level tests failed: %v", err)
		return
	}
	log.Println("✅ Transaction isolation level tests passed")

	// Test savepoint support
	log.Println("\n[Test 3] Savepoint Support")
	if err := testSavepointSupport(); err != nil {
		log.Printf("❌ Savepoint tests failed: %v", err)
		return
	}
	log.Println("✅ Savepoint tests passed")

	log.Println("\n" + strings.Repeat("=", 62))
	log.Println("✅ All persistence feature tests completed successfully!")
	log.Println(strings.Repeat("=", 62))
}

// testErrorSanitization tests error sanitization (no database required)
func testErrorSanitization() {
	// Test SafeError method
	err := persistence.NewError(
		persistence.ErrCodeQuery,
		"query execution failed",
		fmt.Errorf("connection failed: password=secret123"),
	)

	safeMsg := err.SafeError()
	if safeMsg == "" {
		log.Fatal("SafeError should return a message")
	}

	// Safe error should not contain the cause
	if contains(safeMsg, "password") || contains(safeMsg, "secret123") {
		log.Fatalf("SafeError should not contain sensitive information: %s", safeMsg)
	}
	log.Printf("  ✓ SafeError sanitizes sensitive data: %s", safeMsg)

	// Test SanitizeError function with persistence error
	persistenceErr := persistence.NewError(
		persistence.ErrCodeConnection,
		"connection failed",
		fmt.Errorf("dsn=postgres://user:password@host/db"),
	)

	sanitized := persistence.SanitizeError(persistenceErr)
	if contains(sanitized, "password") || contains(sanitized, "dsn=") {
		log.Fatalf("SanitizeError should remove sensitive info: %s", sanitized)
	}
	log.Printf("  ✓ SanitizeError removes sensitive info from persistence errors")

	// Test with generic error containing sensitive info
	genericErr := fmt.Errorf("connection failed: password=secret123 host=localhost")
	sanitized = persistence.SanitizeError(genericErr)
	if contains(sanitized, "password") || contains(sanitized, "secret123") {
		log.Fatalf("SanitizeError should remove sensitive info from generic errors: %s", sanitized)
	}
	log.Printf("  ✓ SanitizeError removes sensitive info from generic errors")
}

// testTransactionIsolationLevels tests transaction isolation levels
func testTransactionIsolationLevels() error {
	// Setup: Create database component
	appConfig := loadTestConfig()
	poolConfig := dbruntime.PoolConfig{
		DSN:             appConfig.DSN,
		DriverName:      "postgres",
		MaxOpenConns:    appConfig.MaxOpenConns,
		MaxIdleConns:    appConfig.MaxIdleConns,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	db := dbruntime.NewDatabaseComponent(poolConfig)
	ctx := core.NewFluxorContext(context.Background())
	
	if err := db.Start(ctx); err != nil {
		return fmt.Errorf("failed to start database: %w", err)
	}
	defer db.Stop(ctx)

	// Create repository
	repoConfig := persistence.DefaultConfig("test_users", db.DB())
	repo, err := persistence.NewSQLRepository(repoConfig)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}
	defer repo.Close()

	// Setup test table
	if err := setupTestTable(db.DB()); err != nil {
		return fmt.Errorf("failed to setup test table: %w", err)
	}
	defer cleanupTestTable(db.DB())

	// Test 1: Default isolation level
	log.Println("  Testing default isolation level...")
	tx, err := repo.BeginTransaction(ctx.Context())
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	txRepo := tx.Repository()
	err = txRepo.Create(ctx.Context(), map[string]interface{}{
		"name":  "Test User 1",
		"email": "test1@example.com",
	})
	if err != nil {
		return fmt.Errorf("failed to create entity: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	log.Println("  ✓ Default isolation level transaction successful")

	// Test 2: Serializable isolation level
	log.Println("  Testing Serializable isolation level...")
	opts := &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	}
	tx, err = repo.BeginTransactionWithOptions(ctx.Context(), opts)
	if err != nil {
		return fmt.Errorf("failed to begin transaction with options: %w", err)
	}
	defer tx.Rollback()

	txRepo = tx.Repository()
	err = txRepo.Create(ctx.Context(), map[string]interface{}{
		"name":  "Test User 2",
		"email": "test2@example.com",
	})
	if err != nil {
		return fmt.Errorf("failed to create entity: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	log.Println("  ✓ Serializable isolation level transaction successful")

	// Test 3: Read-only transaction
	log.Println("  Testing read-only transaction...")
	opts = &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
	}
	tx, err = repo.BeginTransactionWithOptions(ctx.Context(), opts)
	if err != nil {
		return fmt.Errorf("failed to begin read-only transaction: %w", err)
	}
	defer tx.Rollback()

	// Read-only transaction should allow reads
	txRepo = tx.Repository()
	_, err = txRepo.FindAll(ctx.Context(), persistence.NewQuery().WithLimit(10))
	if err != nil {
		return fmt.Errorf("failed to read in read-only transaction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	log.Println("  ✓ Read-only transaction successful")

	return nil
}

// testSavepointSupport tests savepoint functionality
func testSavepointSupport() error {
	// Setup: Create database component
	appConfig := loadTestConfig()
	poolConfig := dbruntime.PoolConfig{
		DSN:             appConfig.DSN,
		DriverName:      "postgres",
		MaxOpenConns:    appConfig.MaxOpenConns,
		MaxIdleConns:    appConfig.MaxIdleConns,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	db := dbruntime.NewDatabaseComponent(poolConfig)
	ctx := core.NewFluxorContext(context.Background())
	
	if err := db.Start(ctx); err != nil {
		return fmt.Errorf("failed to start database: %w", err)
	}
	defer db.Stop(ctx)

	// Create repository
	repoConfig := persistence.DefaultConfig("test_users", db.DB())
	repo, err := persistence.NewSQLRepository(repoConfig)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}
	defer repo.Close()

	// Setup test table
	if err := setupTestTable(db.DB()); err != nil {
		return fmt.Errorf("failed to setup test table: %w", err)
	}
	defer cleanupTestTable(db.DB())

	// Test 1: Create savepoint and rollback to it
	log.Println("  Testing savepoint rollback...")
	tx, err := repo.BeginTransaction(ctx.Context())
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	txRepo := tx.Repository()

	// Create first entity
	err = txRepo.Create(ctx.Context(), map[string]interface{}{
		"name":  "User Before Savepoint",
		"email": "before@example.com",
	})
	if err != nil {
		return fmt.Errorf("failed to create entity: %w", err)
	}

	// Create savepoint
	if err := tx.Savepoint(ctx.Context(), "sp1"); err != nil {
		return fmt.Errorf("failed to create savepoint: %w", err)
	}

	// Create second entity after savepoint
	err = txRepo.Create(ctx.Context(), map[string]interface{}{
		"name":  "User After Savepoint",
		"email": "after@example.com",
	})
	if err != nil {
		return fmt.Errorf("failed to create entity after savepoint: %w", err)
	}

	// Rollback to savepoint (should undo second entity)
	if err := tx.RollbackToSavepoint(ctx.Context(), "sp1"); err != nil {
		return fmt.Errorf("failed to rollback to savepoint: %w", err)
	}

	// Commit transaction (should only have first entity)
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Verify: Only first entity should exist
	results, err := repo.FindAll(ctx.Context(), persistence.NewQuery())
	if err != nil {
		return fmt.Errorf("failed to query results: %w", err)
	}

	// Check that after@example.com doesn't exist
	for _, result := range results {
		entity := result.(map[string]interface{})
		email := entity["email"].(string)
		if email == "after@example.com" {
			return fmt.Errorf("entity after savepoint should not exist after rollback")
		}
	}
	log.Println("  ✓ Savepoint rollback successful")

	// Test 2: Release savepoint
	log.Println("  Testing savepoint release...")
	tx, err = repo.BeginTransaction(ctx.Context())
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create savepoint
	if err := tx.Savepoint(ctx.Context(), "sp2"); err != nil {
		return fmt.Errorf("failed to create savepoint: %w", err)
	}

	// Release savepoint
	if err := tx.ReleaseSavepoint(ctx.Context(), "sp2"); err != nil {
		return fmt.Errorf("failed to release savepoint: %w", err)
	}

	// Try to rollback to released savepoint (should fail)
	err = tx.RollbackToSavepoint(ctx.Context(), "sp2")
	if err == nil {
		return fmt.Errorf("rollback to released savepoint should fail")
	}
	log.Println("  ✓ Savepoint release successful")

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Test 3: Multiple savepoints
	log.Println("  Testing multiple savepoints...")
	tx, err = repo.BeginTransaction(ctx.Context())
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	txRepo = tx.Repository()

	// Create entity 1
	err = txRepo.Create(ctx.Context(), map[string]interface{}{
		"name":  "User 1",
		"email": "user1@example.com",
	})
	if err != nil {
		return fmt.Errorf("failed to create entity 1: %w", err)
	}

	// Create savepoint 1
	if err := tx.Savepoint(ctx.Context(), "sp3"); err != nil {
		return fmt.Errorf("failed to create savepoint 1: %w", err)
	}

	// Create entity 2
	err = txRepo.Create(ctx.Context(), map[string]interface{}{
		"name":  "User 2",
		"email": "user2@example.com",
	})
	if err != nil {
		return fmt.Errorf("failed to create entity 2: %w", err)
	}

	// Create savepoint 2
	if err := tx.Savepoint(ctx.Context(), "sp4"); err != nil {
		return fmt.Errorf("failed to create savepoint 2: %w", err)
	}

	// Create entity 3
	err = txRepo.Create(ctx.Context(), map[string]interface{}{
		"name":  "User 3",
		"email": "user3@example.com",
	})
	if err != nil {
		return fmt.Errorf("failed to create entity 3: %w", err)
	}

	// Rollback to savepoint 2 (should undo entity 3)
	if err := tx.RollbackToSavepoint(ctx.Context(), "sp4"); err != nil {
		return fmt.Errorf("failed to rollback to savepoint 2: %w", err)
	}

	// Rollback to savepoint 1 (should undo entity 2)
	if err := tx.RollbackToSavepoint(ctx.Context(), "sp3"); err != nil {
		return fmt.Errorf("failed to rollback to savepoint 1: %w", err)
	}

	// Commit (should only have entity 1)
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	log.Println("  ✓ Multiple savepoints test successful")

	return nil
}

// Helper functions (updated to return errors instead of using testing.T)

func loadTestConfig() AppConfig {
	var appConfig AppConfig
	configPath := "application.properties"
	if err := config.LoadProperties(configPath, &appConfig); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set defaults if not provided
	if appConfig.MaxOpenConns == 0 {
		appConfig.MaxOpenConns = 25
	}
	if appConfig.MaxIdleConns == 0 {
		appConfig.MaxIdleConns = 5
	}

	return appConfig
}

func setupTestTable(db *sql.DB) error {
	// Drop table if exists
	_, _ = db.Exec("DROP TABLE IF EXISTS test_users")

	// Create table
	_, err := db.Exec(`
		CREATE TABLE test_users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func cleanupTestTable(db *sql.DB) {
	_, _ = db.Exec("DROP TABLE IF EXISTS test_users")
}
