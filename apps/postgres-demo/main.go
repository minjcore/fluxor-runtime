package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"

	"github.com/fluxorio/fluxor/pkg/config"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/dbruntime"
	"github.com/fluxorio/fluxor/pkg/entrypoint"
	"github.com/fluxorio/fluxor/pkg/persistence"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/fluxor/apps/postgres-demo/services"
)

// AppConfig holds the application configuration
// Field names are converted to lowercase for property matching (case-insensitive)
type AppConfig struct {
	DSN             string // Matches "DSN" or "dsn" in properties file
	MaxOpenConns    int    // Matches "MaxOpenConns" or "maxopenconns" in properties file
	MaxIdleConns    int    // Matches "MaxIdleConns" or "maxidleconns" in properties file
	ConnMaxLifetime string // Matches "ConnMaxLifetime" or "connmaxlifetime" in properties file
	ConnMaxIdleTime string // Matches "ConnMaxIdleTime" or "connmaxidletime" in properties file
}

type PostgresDemoVerticle struct {
	*core.BaseVerticle
	db              *dbruntime.DB
	server          *web.FastHTTPServer
	purchaseService *services.PurchaseService

	healthCheckTicker  *time.Ticker
	healthCheckCtx     context.Context
	healthCheckCancel  context.CancelFunc
	healthCheckWg      sync.WaitGroup
}

func NewPostgresDemoVerticle() *PostgresDemoVerticle {
	return &PostgresDemoVerticle{
		BaseVerticle: core.NewBaseVerticle("postgres-demo"),
	}
}

// Start initializes the database component and executes a test query
func (v *PostgresDemoVerticle) Start(ctx core.FluxorContext) error {
	// Call base Start to setup event loop
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}

	// Load configuration from application.properties
	var appConfig AppConfig
	configPath := "application.properties"
	if err := config.LoadProperties(configPath, &appConfig); err != nil {
		return fmt.Errorf("failed to load config from %s: %w", configPath, err)
	}

	log.Printf("Loaded configuration from %s", configPath)
	log.Printf("DSN: %s", maskDSN(appConfig.DSN))
	log.Printf("MaxOpenConns: %d", appConfig.MaxOpenConns)
	log.Printf("MaxIdleConns: %d", appConfig.MaxIdleConns)

	// Parse duration strings
	connMaxLifetime, err := time.ParseDuration(appConfig.ConnMaxLifetime)
	if err != nil {
		connMaxLifetime = 5 * time.Minute // Default
		log.Printf("Warning: Invalid ConnMaxLifetime, using default: %v", connMaxLifetime)
	}

	connMaxIdleTime, err := time.ParseDuration(appConfig.ConnMaxIdleTime)
	if err != nil {
		connMaxIdleTime = 10 * time.Minute // Default
		log.Printf("Warning: Invalid ConnMaxIdleTime, using default: %v", connMaxIdleTime)
	}

	// Open database
	db, err := dbruntime.Open(dbruntime.Config{
		DSN:             appConfig.DSN,
		Driver:          "postgres",
		MaxOpen:         appConfig.MaxOpenConns,
		MaxIdle:         appConfig.MaxIdleConns,
		ConnMaxLifetime: connMaxLifetime,
		ConnMaxIdleTime: connMaxIdleTime,
	})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	v.db = db

	// Smoke test
	var one int
	if err := v.db.QueryRowContext(ctx.Context(), "SELECT 1").Scan(&one); err != nil {
		return fmt.Errorf("db ping query failed: %w", err)
	}
	stats := v.db.Stats()
	log.Printf("Database ready (open=%d idle=%d)", stats.OpenConnections, stats.Idle)

	// Initialize ledger service (for double-entry bookkeeping)
	log.Println("\n[Ledger Service] Initializing ledger service...")
	ledgerService := services.NewLedgerService(v.db)
	if err := ledgerService.InitializeSystemAccounts(ctx.Context()); err != nil {
		log.Printf("Warning: Failed to initialize system accounts: %v", err)
	} else {
		log.Println("✅ System accounts initialized")
	}

	// Initialize wallet rule service
	log.Println("\n[Wallet Rule Service] Initializing wallet rule service...")
	walletRuleService := services.NewWalletRuleService(v.db)
	log.Println("✅ Wallet rule service initialized")

	// Initialize wallet service
	log.Println("\n[Wallet Service] Initializing wallet service...")
	walletService := services.NewWalletService(v.db)
	walletService.SetLedgerService(ledgerService) // Enable ledger integration
	walletService.SetRuleService(walletRuleService) // Enable rule validation
	
	// Verify wallet types are available
	walletTypes, err := walletService.GetAllWalletTypes(ctx.Context(), false)
	if err != nil {
		log.Printf("Warning: Failed to load wallet types: %v", err)
	} else {
		log.Printf("✅ Wallet types loaded: %d active types", len(walletTypes))
	}
	
	// Initialize auth service (for user registration with auto-wallet creation)
	log.Println("\n[Auth Service] Initializing auth service...")
	authService := services.NewAuthService(jwtSecret, defaultUsername, defaultPassword)
	authService.SetDatabase(v.db) // Enable user registration
	authService.SetWalletService(walletService) // Enable auto-creating wallets on registration
	
	// Initialize purchase service
	log.Println("\n[Purchase Service] Initializing purchase service...")
	purchaseService, err := services.NewPurchaseService(v.db, walletService)
	if err != nil {
		log.Printf("Warning: Failed to create purchase service: %v", err)
	} else {
		v.purchaseService = purchaseService
		
		// Setup purchase tables
		if err := purchaseService.SetupPurchaseTables(ctx.Context()); err != nil {
			log.Printf("Warning: Failed to setup purchase tables: %v", err)
		} else {
			log.Println("✅ Purchase tables created")
			
			// Seed products
			if err := purchaseService.SeedProducts(ctx.Context()); err != nil {
				log.Printf("Warning: Failed to seed products: %v", err)
			} else {
				log.Println("✅ Products seeded")
			}
		}
	}

	// Start HTTP web UI
	log.Println("\n[Web UI] Starting HTTP server on :8081...")
	webServer, err := setupWebUI(ctx.GoCMD(), v.db, v.purchaseService, walletService, authService, v)
	if err != nil {
		log.Printf("Warning: Failed to setup web UI: %v", err)
	} else {
		v.server = webServer
		// Start server in background
		go func() {
			if err := webServer.Start(); err != nil {
				log.Printf("Web server error: %v", err)
			}
		}()
		log.Println("✅ Web UI available at http://localhost:8081")
	}

	// Start periodic health check query every 2 seconds
	v.healthCheckCtx, v.healthCheckCancel = context.WithCancel(ctx.Context())
	v.healthCheckTicker = time.NewTicker(2 * time.Second)
	v.healthCheckWg.Add(1)
	
	go func() {
		defer v.healthCheckWg.Done()
		log.Println("\n[Health Check] Starting periodic health check (every 2 seconds)...")
		
		for {
			select {
			case <-v.healthCheckCtx.Done():
				log.Println("[Health Check] Stopping periodic health check")
				return
			case <-v.healthCheckTicker.C:
				// Execute health check query synchronously
				var value int
				row := v.db.QueryRowContext(v.healthCheckCtx, "SELECT 1")
				if err := row.Scan(&value); err != nil {
					fmt.Printf("%s[Health Check]%s Query failed: %v\n", colorLightBlue, colorReset, err)
				} else {
					// Use colored log format matching the image
					fmt.Print(formatHealthCheckLog(value))
					fmt.Print("\n")
				}
			}
		}
	}()

	return nil
}


// Stop stops the database component
// This will stop both the connection executor and pool manager
func (v *PostgresDemoVerticle) Stop(ctx core.FluxorContext) error {
	// Stop web server
	if v.server != nil {
		log.Println("[Web UI] Stopping HTTP server...")
		if err := v.server.Stop(); err != nil {
			log.Printf("Error stopping web server: %v", err)
		}
	}

	// Stop periodic health check
	if v.healthCheckCancel != nil {
		v.healthCheckCancel()
	}
	if v.healthCheckTicker != nil {
		v.healthCheckTicker.Stop()
	}
	v.healthCheckWg.Wait()
	
	if v.db != nil {
		if err := v.db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}
	return v.BaseVerticle.Stop(ctx)
}

// ANSI color codes for terminal output
const (
	colorReset     = "\033[0m"
	colorLightGray = "\033[90m"
	colorLightBlue = "\033[94m"
	colorOrange    = "\033[38;5;214m" // Lighter orange/amber color (was 208, now 214 for lighter shade)
)

// formatHealthCheckLog formats the health check log with colors
// Format: [date in orange] [time in orange] [Health Check in light blue] Query successful: [result in orange]
func formatHealthCheckLog(result int) string {
	now := time.Now()
	date := now.Format("2006/01/02")
	timeStr := now.Format("15:04:05")
	
	// Build the colored log message by concatenating colored parts:
	// - Date in orange
	// - Time in orange
	// - Brackets [ ] in light gray
	// - "Health Check" text in light blue
	// - "Query successful: " in light gray
	// - Result number in orange
	msg := fmt.Sprintf("%s%s%s", colorOrange, date, colorReset) + // Date (orange)
		" " +
		fmt.Sprintf("%s%s%s", colorOrange, timeStr, colorReset) + // Time (orange)
		" " +
		fmt.Sprintf("%s[%sHealth Check%s]%s", colorLightGray, colorLightBlue, colorReset, colorLightGray) + // [Health Check]
		" Query successful: " +
		fmt.Sprintf("%s%d%s", colorOrange, result, colorReset) // Result (orange)
	
	return msg
}

// maskDSN masks password in DSN for logging
func maskDSN(dsn string) string {
	// Simple masking: replace password with ***
	// Format: postgres://user:password@host:port/db
	if len(dsn) < 20 {
		return "***"
	}
	// Find @ symbol and mask password part
	for i := 0; i < len(dsn); i++ {
		if dsn[i] == '@' {
			// Find : before @
			for j := i - 1; j >= 0; j-- {
				if dsn[j] == ':' {
					return dsn[:j+1] + "***" + dsn[i:]
				}
			}
		}
	}
	return "***"
}

func main() {
	// Check if running in test mode
	if len(os.Args) > 1 && os.Args[1] == "--test" {
		runPersistenceTests()
		return
	}

	log.Println("Starting PostgreSQL Demo Application...")

	// Create Fluxor application (empty config path - we load properties manually in verticle)
	app, err := entrypoint.NewMainVerticle("")
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Deploy PostgresDemoVerticle
	_, err = app.DeployVerticle(NewPostgresDemoVerticle())
	if err != nil {
		log.Fatalf("Failed to deploy verticle: %v", err)
	}

	// Start application (blocks until SIGINT/SIGTERM)
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	// Stop application (called automatically by app.Start() on signal)
	if err := app.Stop(); err != nil {
		log.Printf("Error stopping application: %v", err)
	}

	log.Println("Application stopped")
}

// runPersistenceTests runs all persistence feature tests
func runPersistenceTests() {
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
	if strings.Contains(strings.ToLower(safeMsg), "password") || strings.Contains(strings.ToLower(safeMsg), "secret123") {
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
	if strings.Contains(strings.ToLower(sanitized), "password") || strings.Contains(strings.ToLower(sanitized), "dsn=") {
		log.Fatalf("SanitizeError should remove sensitive info: %s", sanitized)
	}
	log.Printf("  ✓ SanitizeError removes sensitive info from persistence errors")

	// Test with generic error containing sensitive info
	genericErr := fmt.Errorf("connection failed: password=secret123 host=localhost")
	sanitized = persistence.SanitizeError(genericErr)
	if strings.Contains(strings.ToLower(sanitized), "password") || strings.Contains(strings.ToLower(sanitized), "secret123") {
		log.Fatalf("SanitizeError should remove sensitive info from generic errors: %s", sanitized)
	}
	log.Printf("  ✓ SanitizeError removes sensitive info from generic errors")
}

// testTransactionIsolationLevels tests transaction isolation levels
func testTransactionIsolationLevels() error {
	appConfig := loadTestConfig()
	db, err := dbruntime.Open(dbruntime.Config{DSN: appConfig.DSN, Driver: "postgres"})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	goCtx := context.Background()
	gocmd := core.NewGoCMD(goCtx)
	ctx := core.NewFluxorContext(goCtx, gocmd)
	defer gocmd.Close()

	// Create repository
	repoConfig := persistence.DefaultConfig("test_users", db.DB)
	repo, err := persistence.NewSQLRepository(repoConfig)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}
	defer repo.Close()

	// Setup test table
	if err := setupTestTable(db.DB); err != nil {
		return fmt.Errorf("failed to setup test table: %w", err)
	}
	defer cleanupTestTable(db.DB)

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
	appConfig := loadTestConfig()
	db, err := dbruntime.Open(dbruntime.Config{DSN: appConfig.DSN, Driver: "postgres"})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	goCtx := context.Background()
	gocmd := core.NewGoCMD(goCtx)
	ctx := core.NewFluxorContext(goCtx, gocmd)
	defer gocmd.Close()

	// Create repository
	repoConfig := persistence.DefaultConfig("test_users", db.DB)
	repo, err := persistence.NewSQLRepository(repoConfig)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}
	defer repo.Close()

	// Setup test table
	if err := setupTestTable(db.DB); err != nil {
		return fmt.Errorf("failed to setup test table: %w", err)
	}
	defer cleanupTestTable(db.DB)

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

// Helper functions for tests

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
