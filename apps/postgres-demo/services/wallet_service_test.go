package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/fluxorio/fluxor/apps/postgres-demo/models"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/dbruntime"
)

// setupTestDB creates an in-memory database for testing
func setupWalletTestDB(t *testing.T) *dbruntime.DB {
	t.Helper()

	// Use the same DSN as the main application
	poolConfig := dbruntime.PoolConfig{
		DSN:             "postgres://postgres:postgres@localhost:5432/fluxor_db?sslmode=disable",
		DriverName:      "postgres",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	db := dbruntime.NewDatabaseComponent(poolConfig)

	// Create GoCMD and FluxorContext for testing
	goCtx := context.Background()
	gocmd := core.NewGoCMD(goCtx)
	ctx := core.NewFluxorContext(goCtx, gocmd)
	
	if err := db.Start(ctx); err != nil {
		gocmd.Close()
		t.Skipf("Skipping test: database not available: %v", err)
		return nil
	}

	// Create tables
	schema := `
	CREATE TABLE IF NOT EXISTS wallet_types (
		code VARCHAR(20) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		description TEXT,
		icon VARCHAR(255),
		is_active BOOLEAN DEFAULT true,
		is_default BOOLEAN DEFAULT false,
		min_balance DECIMAL(15,2),
		max_balance DECIMAL(15,2),
		metadata JSONB,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS wallets (
		user_id VARCHAR(255) NOT NULL,
		wallet_type VARCHAR(20) NOT NULL DEFAULT 'primary',
		balance DECIMAL(10, 2) NOT NULL DEFAULT 0.00 CHECK (balance >= 0),
		frozen DECIMAL(10, 2) NOT NULL DEFAULT 0.00 CHECK (frozen >= 0),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, wallet_type),
		CHECK (balance >= frozen)
	);

	CREATE TABLE IF NOT EXISTS wallet_transactions (
		id SERIAL PRIMARY KEY,
		user_id VARCHAR(255) NOT NULL,
		wallet_type VARCHAR(20) NOT NULL DEFAULT 'primary',
		type VARCHAR(20) NOT NULL,
		amount DECIMAL(10, 2) NOT NULL,
		description TEXT,
		order_id INTEGER,
		status VARCHAR(20) DEFAULT 'completed' CHECK (status IN ('pending', 'completed', 'cancelled')),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id, wallet_type) REFERENCES wallets(user_id, wallet_type) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS wallet_types (
		code VARCHAR(20) PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		description TEXT,
		icon VARCHAR(255),
		is_active BOOLEAN DEFAULT true,
		is_default BOOLEAN DEFAULT false,
		min_balance DECIMAL(15,2),
		max_balance DECIMAL(15,2),
		metadata JSONB,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS accounts (
		id SERIAL PRIMARY KEY,
		code VARCHAR(20) UNIQUE NOT NULL,
		name VARCHAR(100) NOT NULL,
		type VARCHAR(20) NOT NULL CHECK (type IN ('asset', 'liability', 'equity', 'revenue', 'expense')),
		parent_id INTEGER REFERENCES accounts(id),
		is_system BOOLEAN DEFAULT false,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS journals (
		id SERIAL PRIMARY KEY,
		reference_type VARCHAR(50) NOT NULL,
		reference_id INTEGER,
		description TEXT,
		status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'posted', 'reversed')),
		posted_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		created_by VARCHAR(255)
	);

	CREATE TABLE IF NOT EXISTS ledger_entries (
		id SERIAL PRIMARY KEY,
		journal_id INTEGER NOT NULL REFERENCES journals(id) ON DELETE CASCADE,
		account_id INTEGER NOT NULL REFERENCES accounts(id),
		debit DECIMAL(15,2) DEFAULT 0 CHECK (debit >= 0),
		credit DECIMAL(15,2) DEFAULT 0 CHECK (credit >= 0),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT check_debit_or_credit CHECK ((debit > 0 AND credit = 0) OR (debit = 0 AND credit > 0))
	);
	`

	// Create tables
	if _, err := db.DB.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Insert default wallet types for testing
	walletTypesSQL := `
	INSERT INTO wallet_types (code, name, description, icon, is_active, is_default, min_balance, max_balance, metadata) VALUES
		('primary', 'Primary Wallet', 'Default/main wallet', 'wallet', true, true, 0.00, NULL, '{}'::jsonb),
		('savings', 'Savings Wallet', 'Savings wallet', 'savings', true, false, 0.00, NULL, '{}'::jsonb),
		('investment', 'Investment Wallet', 'Investment wallet', 'chart-line', true, false, 0.00, NULL, '{}'::jsonb),
		('business', 'Business Wallet', 'Business wallet', 'briefcase', true, false, 0.00, NULL, '{}'::jsonb),
		('escrow', 'Escrow Wallet', 'Escrow wallet', 'lock', true, false, 0.00, NULL, '{}'::jsonb)
	ON CONFLICT (code) DO NOTHING;
	`
	_, _ = db.DB.Exec(walletTypesSQL)
	
	// Add foreign key constraints after wallet_types exists
	_, _ = db.DB.Exec(`ALTER TABLE wallets DROP CONSTRAINT IF EXISTS wallets_wallet_type_fkey;
		ALTER TABLE wallets ADD CONSTRAINT wallets_wallet_type_fkey 
		FOREIGN KEY (wallet_type) REFERENCES wallet_types(code) ON DELETE RESTRICT;`)
	_, _ = db.DB.Exec(`ALTER TABLE wallet_transactions DROP CONSTRAINT IF EXISTS wallet_transactions_wallet_type_fkey;
		ALTER TABLE wallet_transactions ADD CONSTRAINT wallet_transactions_wallet_type_fkey 
		FOREIGN KEY (wallet_type) REFERENCES wallet_types(code) ON DELETE RESTRICT;`)

	// Cleanup existing test data (optional - only if needed)
	// This ensures tests start with a clean state
	_, _ = db.DB.Exec("DELETE FROM wallet_transactions")
	_, _ = db.DB.Exec("DELETE FROM wallets")
	_, _ = db.DB.Exec("DELETE FROM ledger_entries")
	_, _ = db.DB.Exec("DELETE FROM journals")
	// Note: accounts and wallet_types are not deleted as they're system/master data

	return db
}

// cleanupTestDB stops the database component
func cleanupTestDB(t *testing.T, db *dbruntime.DB) {
	t.Helper()
	if db != nil {
		goCtx := context.Background()
		gocmd := core.NewGoCMD(goCtx)
		defer gocmd.Close()
		
		ctx := core.NewFluxorContext(goCtx, gocmd)
		if err := db.Stop(ctx); err != nil {
			t.Logf("Error stopping database component: %v", err)
		}
	}
}

func TestWalletService_GetWalletBalance(t *testing.T) {
	db := setupWalletTestDB(t)
	if db == nil {
		return
	}
	defer cleanupTestDB(t, db)

	ctx := context.Background()
	service := NewWalletService(db)

	t.Run("Get balance for new user creates wallet", func(t *testing.T) {
		// Use unique user ID to avoid conflicts
		userID := "user1_test_new"
		balance, err := service.GetWalletBalance(ctx, userID)
		if err != nil {
			t.Fatalf("GetWalletBalance() error = %v", err)
		}
		if balance != 0 {
			t.Errorf("GetWalletBalance() = %v, want 0", balance)
		}
	})

	t.Run("Get balance returns correct available balance", func(t *testing.T) {
		// Use unique user ID to avoid conflicts
		userID := "user2_test_balance"
		
		// Add money first
		err := service.AddWalletBalance(ctx, userID, 100.0, "Test deposit")
		if err != nil {
			t.Fatalf("AddWalletBalance() error = %v", err)
		}

		balance, err := service.GetWalletBalance(ctx, userID)
		if err != nil {
			t.Fatalf("GetWalletBalance() error = %v", err)
		}
		if balance != 100.0 {
			t.Errorf("GetWalletBalance() = %v, want 100.0", balance)
		}
	})
}

func TestWalletService_GetWalletBalanceByType(t *testing.T) {
	db := setupWalletTestDB(t)
	if db == nil {
		return
	}
	defer cleanupTestDB(t, db)

	ctx := context.Background()
	service := NewWalletService(db)

	tests := []struct {
		name       string
		userID     string
		walletType models.WalletType
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "Primary wallet",
			userID:     "user3_test_bytype",
			walletType: models.WalletTypePrimary,
			wantErr:    false,
		},
		{
			name:       "Savings wallet",
			userID:     "user3_test_bytype",
			walletType: models.WalletTypeSavings,
			wantErr:    false,
		},
		{
			name:       "Investment wallet",
			userID:     "user3_test_bytype",
			walletType: models.WalletTypeInvestment,
			wantErr:    false,
		},
		{
			name:       "Invalid wallet type",
			userID:     "user3_test_bytype",
			walletType: models.WalletType("invalid"),
			wantErr:    true,
			errMsg:     "invalid wallet type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balance, err := service.GetWalletBalanceByType(ctx, tt.userID, tt.walletType)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetWalletBalanceByType() expected error but got none")
				}
				if err != nil && tt.errMsg != "" && err.Error()[:len(tt.errMsg)] != tt.errMsg {
					t.Errorf("GetWalletBalanceByType() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GetWalletBalanceByType() error = %v", err)
				}
				if balance != 0 {
					t.Errorf("GetWalletBalanceByType() = %v, want 0 for new wallet", balance)
				}
			}
		})
	}
}

func TestWalletService_AddWalletBalanceByType(t *testing.T) {
	db := setupWalletTestDB(t)
	if db == nil {
		return
	}
	defer cleanupTestDB(t, db)

	ctx := context.Background()
	service := NewWalletService(db)

	t.Run("Add balance to primary wallet", func(t *testing.T) {
		userID := "user4_test_add_primary"
		err := service.AddWalletBalanceByType(ctx, userID, models.WalletTypePrimary, 50.0, "Deposit")
		if err != nil {
			t.Fatalf("AddWalletBalanceByType() error = %v", err)
		}

		balance, err := service.GetWalletBalanceByType(ctx, userID, models.WalletTypePrimary)
		if err != nil {
			t.Fatalf("GetWalletBalanceByType() error = %v", err)
		}
		if balance != 50.0 {
			t.Errorf("Balance = %v, want 50.0", balance)
		}
	})

	t.Run("Add balance to savings wallet", func(t *testing.T) {
		userID := "user4_test_add_savings"
		// First add to primary
		err := service.AddWalletBalanceByType(ctx, userID, models.WalletTypePrimary, 50.0, "Primary deposit")
		if err != nil {
			t.Fatalf("AddWalletBalanceByType() error = %v", err)
		}
		
		// Then add to savings
		err = service.AddWalletBalanceByType(ctx, userID, models.WalletTypeSavings, 100.0, "Savings deposit")
		if err != nil {
			t.Fatalf("AddWalletBalanceByType() error = %v", err)
		}

		balance, err := service.GetWalletBalanceByType(ctx, userID, models.WalletTypeSavings)
		if err != nil {
			t.Fatalf("GetWalletBalanceByType() error = %v", err)
		}
		if balance != 100.0 {
			t.Errorf("Balance = %v, want 100.0", balance)
		}

		// Verify primary wallet balance unchanged
		primaryBalance, err := service.GetWalletBalanceByType(ctx, userID, models.WalletTypePrimary)
		if err != nil {
			t.Fatalf("GetWalletBalanceByType() error = %v", err)
		}
		if primaryBalance != 50.0 {
			t.Errorf("Primary balance = %v, want 50.0 (should be unchanged)", primaryBalance)
		}
	})

	t.Run("Multiple wallets for same user are independent", func(t *testing.T) {
		userID := "user5_test_multiple"
		// Add to investment wallet
		err := service.AddWalletBalanceByType(ctx, userID, models.WalletTypeInvestment, 200.0, "Investment")
		if err != nil {
			t.Fatalf("AddWalletBalanceByType() error = %v", err)
		}

		// Add to business wallet
		err = service.AddWalletBalanceByType(ctx, userID, models.WalletTypeBusiness, 300.0, "Business")
		if err != nil {
			t.Fatalf("AddWalletBalanceByType() error = %v", err)
		}

		investmentBalance, _ := service.GetWalletBalanceByType(ctx, userID, models.WalletTypeInvestment)
		businessBalance, _ := service.GetWalletBalanceByType(ctx, userID, models.WalletTypeBusiness)

		if investmentBalance != 200.0 {
			t.Errorf("Investment balance = %v, want 200.0", investmentBalance)
		}
		if businessBalance != 300.0 {
			t.Errorf("Business balance = %v, want 300.0", businessBalance)
		}
	})
}

func TestWalletService_FreezeAmountByType(t *testing.T) {
	db := setupWalletTestDB(t)
	if db == nil {
		return
	}
	defer cleanupTestDB(t, db)

	ctx := context.Background()
	service := NewWalletService(db)

	t.Run("Freeze amount from savings wallet", func(t *testing.T) {
		userID := "user6_test_freeze"
		// Setup: Add money to savings wallet
		err := service.AddWalletBalanceByType(ctx, userID, models.WalletTypeSavings, 100.0, "Initial deposit")
		if err != nil {
			t.Fatalf("AddWalletBalanceByType() error = %v", err)
		}

		// Start transaction
		tx, err := db.DB.BeginTx(ctx, &sql.TxOptions{
			Isolation: sql.LevelSerializable,
			ReadOnly:  false,
		})
		if err != nil {
			t.Fatalf("BeginTx() error = %v", err)
		}
		defer tx.Rollback()

		// Freeze amount
		orderID := 1
		transactionID, err := service.FreezeAmountByType(ctx, tx, userID, models.WalletTypeSavings, 50.0, "Purchase", &orderID)
		if err != nil {
			t.Fatalf("FreezeAmountByType() error = %v", err)
		}
		if transactionID == 0 {
			t.Error("FreezeAmountByType() returned 0 transaction ID")
		}

		// Check wallet state within the transaction
		var balance, frozen float64
		checkQuery := `SELECT balance, frozen FROM wallets WHERE user_id = $1 AND wallet_type = $2`
		err = tx.QueryRowContext(ctx, checkQuery, userID, string(models.WalletTypeSavings)).Scan(&balance, &frozen)
		if err != nil {
			t.Fatalf("Failed to query wallet state: %v", err)
		}

		// After freeze: balance = 50, frozen = 50, available = 0
		if balance != 50.0 {
			t.Errorf("Balance after freeze = %v, want 50.0", balance)
		}
		if frozen != 50.0 {
			t.Errorf("Frozen after freeze = %v, want 50.0", frozen)
		}

		availableBalance := balance - frozen
		if availableBalance != 0.0 {
			t.Errorf("Available balance = %v, want 0.0", availableBalance)
		}

		tx.Commit()
		
		// After commit, verify the state is persisted
		wallet, err := service.GetWalletDetailsByType(ctx, userID, models.WalletTypeSavings)
		if err != nil {
			t.Fatalf("GetWalletDetailsByType() error = %v", err)
		}
		if wallet.Balance != 50.0 {
			t.Errorf("Balance after commit = %v, want 50.0", wallet.Balance)
		}
		if wallet.Frozen != 50.0 {
			t.Errorf("Frozen after commit = %v, want 50.0", wallet.Frozen)
		}
	})

	t.Run("Freeze fails with insufficient balance", func(t *testing.T) {
		userID := "user7_test_insufficient"
		// Setup: Add money to wallet
		err := service.AddWalletBalanceByType(ctx, userID, models.WalletTypePrimary, 10.0, "Initial deposit")
		if err != nil {
			t.Fatalf("AddWalletBalanceByType() error = %v", err)
		}

		// Start transaction
		tx, err := db.DB.BeginTx(ctx, &sql.TxOptions{
			Isolation: sql.LevelSerializable,
			ReadOnly:  false,
		})
		if err != nil {
			t.Fatalf("BeginTx() error = %v", err)
		}
		defer tx.Rollback()

		// Try to freeze more than available
		_, err = service.FreezeAmountByType(ctx, tx, userID, models.WalletTypePrimary, 100.0, "Purchase", nil)
		if err == nil {
			t.Error("FreezeAmountByType() expected error for insufficient balance, got nil")
		}
		if err != nil && err.Error()[:21] != "insufficient wallet b" {
			t.Errorf("FreezeAmountByType() error = %v, want insufficient balance error", err)
		}
	})
}

func TestWalletService_CommitTransactionByType(t *testing.T) {
	db := setupWalletTestDB(t)
	if db == nil {
		return
	}
	defer cleanupTestDB(t, db)

	ctx := context.Background()
	service := NewWalletService(db)

	t.Run("Commit transaction completes purchase", func(t *testing.T) {
		userID := "user8_test_commit"
		// Setup: Add money
		err := service.AddWalletBalanceByType(ctx, userID, models.WalletTypePrimary, 100.0, "Initial deposit")
		if err != nil {
			t.Fatalf("AddWalletBalanceByType() error = %v", err)
		}

		// Start transaction
		tx, err := db.DB.BeginTx(ctx, &sql.TxOptions{
			Isolation: sql.LevelSerializable,
			ReadOnly:  false,
		})
		if err != nil {
			t.Fatalf("BeginTx() error = %v", err)
		}

		// Freeze amount
		orderID := 2
		transactionID, err := service.FreezeAmountByType(ctx, tx, userID, models.WalletTypePrimary, 30.0, "Purchase", &orderID)
		if err != nil {
			t.Fatalf("FreezeAmountByType() error = %v", err)
		}

		// Commit transaction
		err = service.CommitTransactionByType(ctx, tx, userID, models.WalletTypePrimary, 30.0, transactionID)
		if err != nil {
			tx.Rollback()
			t.Fatalf("CommitTransactionByType() error = %v", err)
		}

		err = tx.Commit()
		if err != nil {
			t.Fatalf("Commit() error = %v", err)
		}

		// Check wallet state
		wallet, err := service.GetWalletDetailsByType(ctx, userID, models.WalletTypePrimary)
		if err != nil {
			t.Fatalf("GetWalletDetailsByType() error = %v", err)
		}

		// After commit: balance = 70, frozen = 0
		if wallet.Balance != 70.0 {
			t.Errorf("Balance after commit = %v, want 70.0", wallet.Balance)
		}
		if wallet.Frozen != 0.0 {
			t.Errorf("Frozen after commit = %v, want 0.0", wallet.Frozen)
		}
	})
}

func TestWalletService_RollbackTransactionByType(t *testing.T) {
	db := setupWalletTestDB(t)
	if db == nil {
		return
	}
	defer cleanupTestDB(t, db)

	ctx := context.Background()
	service := NewWalletService(db)

	t.Run("Rollback transaction restores balance", func(t *testing.T) {
		userID := "user9_test_rollback"
		// Setup: Add money
		err := service.AddWalletBalanceByType(ctx, userID, models.WalletTypeBusiness, 200.0, "Initial deposit")
		if err != nil {
			t.Fatalf("AddWalletBalanceByType() error = %v", err)
		}

		// Start transaction
		tx, err := db.DB.BeginTx(ctx, &sql.TxOptions{
			Isolation: sql.LevelSerializable,
			ReadOnly:  false,
		})
		if err != nil {
			t.Fatalf("BeginTx() error = %v", err)
		}

		// Freeze amount
		orderID := 3
		transactionID, err := service.FreezeAmountByType(ctx, tx, userID, models.WalletTypeBusiness, 50.0, "Purchase", &orderID)
		if err != nil {
			t.Fatalf("FreezeAmountByType() error = %v", err)
		}

		// Rollback transaction
		err = service.RollbackTransactionByType(ctx, tx, userID, models.WalletTypeBusiness, 50.0, transactionID)
		if err != nil {
			tx.Rollback()
			t.Fatalf("RollbackTransactionByType() error = %v", err)
		}

		err = tx.Commit()
		if err != nil {
			t.Fatalf("Commit() error = %v", err)
		}

		// Check wallet state
		wallet, err := service.GetWalletDetailsByType(ctx, userID, models.WalletTypeBusiness)
		if err != nil {
			t.Fatalf("GetWalletDetailsByType() error = %v", err)
		}

		// After rollback: balance = 200, frozen = 0 (restored)
		if wallet.Balance != 200.0 {
			t.Errorf("Balance after rollback = %v, want 200.0", wallet.Balance)
		}
		if wallet.Frozen != 0.0 {
			t.Errorf("Frozen after rollback = %v, want 0.0", wallet.Frozen)
		}
	})
}

func TestWalletService_GetWalletTransactionsByType(t *testing.T) {
	db := setupWalletTestDB(t)
	if db == nil {
		return
	}
	defer cleanupTestDB(t, db)

	ctx := context.Background()
	service := NewWalletService(db)

	t.Run("Get transactions for specific wallet type", func(t *testing.T) {
		userID := "user10_test_transactions"
		// Add money to primary wallet
		err := service.AddWalletBalanceByType(ctx, userID, models.WalletTypePrimary, 100.0, "Primary deposit")
		if err != nil {
			t.Fatalf("AddWalletBalanceByType() error = %v", err)
		}

		// Add money to savings wallet
		err = service.AddWalletBalanceByType(ctx, userID, models.WalletTypeSavings, 200.0, "Savings deposit")
		if err != nil {
			t.Fatalf("AddWalletBalanceByType() error = %v", err)
		}

		// Get transactions for primary wallet
		primaryTxs, err := service.GetWalletTransactionsByType(ctx, userID, models.WalletTypePrimary)
		if err != nil {
			t.Fatalf("GetWalletTransactionsByType() error = %v", err)
		}
		if len(primaryTxs) != 1 {
			t.Errorf("Primary transactions count = %v, want 1", len(primaryTxs))
		}
		if len(primaryTxs) > 0 && primaryTxs[0].WalletType != models.WalletTypePrimary {
			t.Errorf("Transaction wallet type = %v, want %v", primaryTxs[0].WalletType, models.WalletTypePrimary)
		}

		// Get transactions for savings wallet
		savingsTxs, err := service.GetWalletTransactionsByType(ctx, userID, models.WalletTypeSavings)
		if err != nil {
			t.Fatalf("GetWalletTransactionsByType() error = %v", err)
		}
		if len(savingsTxs) != 1 {
			t.Errorf("Savings transactions count = %v, want 1", len(savingsTxs))
		}
		if len(savingsTxs) > 0 && savingsTxs[0].WalletType != models.WalletTypeSavings {
			t.Errorf("Transaction wallet type = %v, want %v", savingsTxs[0].WalletType, models.WalletTypeSavings)
		}
	})
}

func TestWalletService_WalletTypeValidation(t *testing.T) {
	db := setupWalletTestDB(t)
	if db == nil {
		return
	}
	defer cleanupTestDB(t, db)

	ctx := context.Background()
	service := NewWalletService(db)

	invalidTypes := []models.WalletType{
		models.WalletType(""),
		models.WalletType("invalid"),
		models.WalletType("checking"),
	}

	for _, invalidType := range invalidTypes {
		t.Run("Invalid wallet type: "+string(invalidType), func(t *testing.T) {
			userID := "user11_test_invalid_" + string(invalidType)
			_, err := service.GetWalletBalanceByType(ctx, userID, invalidType)
			if err == nil {
				t.Errorf("GetWalletBalanceByType() with invalid type %v expected error, got nil", invalidType)
			}

			err = service.AddWalletBalanceByType(ctx, userID, invalidType, 10.0, "Test")
			if err == nil {
				t.Errorf("AddWalletBalanceByType() with invalid type %v expected error, got nil", invalidType)
			}
		})
	}
}
