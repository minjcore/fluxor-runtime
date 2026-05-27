package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/fluxorio/fluxor/apps/postgres-demo/models"
	"github.com/fluxorio/fluxor/pkg/dbruntime"
	"github.com/shopspring/decimal"
)

// Use enum constants for transaction status
const (
	statusPending   = string(models.StatusPending)
	statusCompleted = string(models.StatusCompleted)
	statusCancelled = string(models.StatusCancelled)
)

// WalletService handles wallet and balance operations
type WalletService struct {
	db            *dbruntime.DB
	ledgerService *LedgerService      // Optional: if nil, ledger features are disabled
	ruleService   *WalletRuleService  // Optional: if nil, rule validation is disabled
}

// NewWalletService creates a new wallet service
func NewWalletService(db *dbruntime.DB) *WalletService {
	return &WalletService{
		db: db,
	}
}

// SetLedgerService sets the ledger service for double-entry bookkeeping
func (s *WalletService) SetLedgerService(ledgerService *LedgerService) {
	s.ledgerService = ledgerService
}

// SetRuleService sets the wallet rule service for rule validation
func (s *WalletService) SetRuleService(ruleService *WalletRuleService) {
	s.ruleService = ruleService
}

// ============================================================================
// WALLET TYPE MANAGEMENT
// ============================================================================

// GetWalletTypeInfo retrieves wallet type information from database
func (s *WalletService) GetWalletTypeInfo(ctx context.Context, walletType models.WalletType) (*models.WalletTypeInfo, error) {
	query := `SELECT code, name, description, icon, is_active, is_default, min_balance, max_balance, metadata, created_at, updated_at
		FROM wallet_types WHERE code = $1`
	
	var wt models.WalletTypeInfo
	var icon, metadata sql.NullString
	var minBalance, maxBalance sql.NullFloat64
	
	err := s.db.QueryRowContext(ctx, query, string(walletType)).Scan(
		&wt.Code, &wt.Name, &wt.Description, &icon, &wt.IsActive, &wt.IsDefault,
		&minBalance, &maxBalance, &metadata, &wt.CreatedAt, &wt.UpdatedAt)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("wallet type '%s' not found", walletType)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet type info: %w", err)
	}
	
	if icon.Valid {
		wt.Icon = &icon.String
	}
	if minBalance.Valid {
		wt.MinBalance = &minBalance.Float64
	}
	if maxBalance.Valid {
		wt.MaxBalance = &maxBalance.Float64
	}
	if metadata.Valid {
		wt.Metadata = &metadata.String
	}
	
	return &wt, nil
}

// GetAllWalletTypes retrieves all active wallet types
func (s *WalletService) GetAllWalletTypes(ctx context.Context, includeInactive bool) ([]models.WalletTypeInfo, error) {
	var query string
	if includeInactive {
		query = `SELECT code, name, description, icon, is_active, is_default, min_balance, max_balance, metadata, created_at, updated_at
			FROM wallet_types ORDER BY is_default DESC, code ASC`
	} else {
		query = `SELECT code, name, description, icon, is_active, is_default, min_balance, max_balance, metadata, created_at, updated_at
			FROM wallet_types WHERE is_active = true ORDER BY is_default DESC, code ASC`
	}
	
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query wallet types: %w", err)
	}
	defer rows.Close()
	
	var walletTypes []models.WalletTypeInfo
	for rows.Next() {
		var wt models.WalletTypeInfo
		var icon, metadata sql.NullString
		var minBalance, maxBalance sql.NullFloat64
		
		err := rows.Scan(
			&wt.Code, &wt.Name, &wt.Description, &icon, &wt.IsActive, &wt.IsDefault,
			&minBalance, &maxBalance, &metadata, &wt.CreatedAt, &wt.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan wallet type: %w", err)
		}
		
		if icon.Valid {
			wt.Icon = &icon.String
		}
		if minBalance.Valid {
			wt.MinBalance = &minBalance.Float64
		}
		if maxBalance.Valid {
			wt.MaxBalance = &maxBalance.Float64
		}
		if metadata.Valid {
			wt.Metadata = &metadata.String
		}
		
		walletTypes = append(walletTypes, wt)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating wallet types: %w", err)
	}
	
	return walletTypes, nil
}

// ValidateWalletType checks if a wallet type exists and is active in the database
func (s *WalletService) ValidateWalletType(ctx context.Context, walletType models.WalletType) error {
	query := `SELECT is_active FROM wallet_types WHERE code = $1`
	var isActive bool
	
	err := s.db.QueryRowContext(ctx, query, string(walletType)).Scan(&isActive)
	if err == sql.ErrNoRows {
		return fmt.Errorf("wallet type '%s' not found", walletType)
	}
	if err != nil {
		return fmt.Errorf("failed to validate wallet type: %w", err)
	}
	if !isActive {
		return fmt.Errorf("wallet type '%s' is not active", walletType)
	}
	
	return nil
}

// GetWalletBalance retrieves the wallet balance for a user's primary wallet
// Returns the available balance (balance - frozen)
// frozen represents pending amounts, so available = balance - frozen
func (s *WalletService) GetWalletBalance(ctx context.Context, userID string) (float64, error) {
	return s.GetWalletBalanceByType(ctx, userID, models.WalletTypePrimary)
}

// GetWalletBalanceByType retrieves the wallet balance for a specific wallet type
func (s *WalletService) GetWalletBalanceByType(ctx context.Context, userID string, walletType models.WalletType) (float64, error) {
	// Validate wallet type from database
	if err := s.ValidateWalletType(ctx, walletType); err != nil {
		return 0, err
	}
	
	query := `SELECT balance, frozen FROM wallets WHERE user_id = $1 AND wallet_type = $2`
	var balance, frozen float64

	err := s.db.QueryRowContext(ctx, query, userID, string(walletType)).Scan(&balance, &frozen)
	if err == sql.ErrNoRows {
		// Wallet doesn't exist, create it with 0 balance
		insertQuery := `INSERT INTO wallets (user_id, wallet_type, balance, frozen) VALUES ($1, $2, 0.00, 0.00) RETURNING balance, frozen`
		err = s.db.QueryRowContext(ctx, insertQuery, userID, string(walletType)).Scan(&balance, &frozen)
		if err != nil {
			return 0, fmt.Errorf("failed to create wallet: %w", err)
		}
		return balance - frozen, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to query wallet: %w", err)
	}

	// Return available balance (balance - frozen)
	// frozen is the sum of pending amounts, so available = balance - frozen
	return balance - frozen, nil
}

// GetWalletDetails retrieves full wallet information for primary wallet
func (s *WalletService) GetWalletDetails(ctx context.Context, userID string) (*models.Wallet, error) {
	return s.GetWalletDetailsByType(ctx, userID, models.WalletTypePrimary)
}

// GetWalletDetailsByType retrieves full wallet information for a specific wallet type
func (s *WalletService) GetWalletDetailsByType(ctx context.Context, userID string, walletType models.WalletType) (*models.Wallet, error) {
	// Validate wallet type from database
	if err := s.ValidateWalletType(ctx, walletType); err != nil {
		return nil, err
	}
	
	query := `SELECT user_id, wallet_type, balance, frozen FROM wallets WHERE user_id = $1 AND wallet_type = $2`
	var wallet models.Wallet

	err := s.db.QueryRowContext(ctx, query, userID, string(walletType)).Scan(&wallet.UserID, &wallet.WalletType, &wallet.Balance, &wallet.Frozen)
	if err == sql.ErrNoRows {
		// Wallet doesn't exist, create it with 0 balance
		insertQuery := `INSERT INTO wallets (user_id, wallet_type, balance, frozen) VALUES ($1, $2, 0.00, 0.00) RETURNING user_id, wallet_type, balance, frozen`
		err = s.db.QueryRowContext(ctx, insertQuery, userID, string(walletType)).Scan(&wallet.UserID, &wallet.WalletType, &wallet.Balance, &wallet.Frozen)
		if err != nil {
			return nil, fmt.Errorf("failed to create wallet: %w", err)
		}
		return &wallet, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query wallet: %w", err)
	}

	return &wallet, nil
}

// GetWalletTransactions retrieves wallet transactions for a user's primary wallet
func (s *WalletService) GetWalletTransactions(ctx context.Context, userID string) ([]models.WalletTransaction, error) {
	return s.GetWalletTransactionsByType(ctx, userID, models.WalletTypePrimary)
}

// GetWalletTransactionsByType retrieves wallet transactions for a specific wallet type
func (s *WalletService) GetWalletTransactionsByType(ctx context.Context, userID string, walletType models.WalletType) ([]models.WalletTransaction, error) {
	if !walletType.IsValid() {
		return nil, fmt.Errorf("invalid wallet type: %s", walletType)
	}
	
	query := `SELECT id, user_id, wallet_type, type, amount, description, order_id, status, created_at 
	          FROM wallet_transactions 
	          WHERE user_id = $1 AND wallet_type = $2
	          ORDER BY created_at DESC 
	          LIMIT 50`

	rows, err := s.db.QueryContext(ctx, query, userID, string(walletType))
	if err != nil {
		return nil, fmt.Errorf("failed to query wallet transactions: %w", err)
	}
	defer rows.Close()

	var transactions []models.WalletTransaction
	for rows.Next() {
		var t models.WalletTransaction
		var orderID sql.NullInt64
		var walletTypeStr string
		var createdAtStr string

		err := rows.Scan(&t.ID, &t.UserID, &walletTypeStr, &t.Type, &t.Amount, &t.Description, &orderID, &t.Status, &createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		t.WalletType = models.WalletType(walletTypeStr)
		if orderID.Valid {
			orderIDInt := int(orderID.Int64)
			t.OrderID = &orderIDInt
		}

		// Parse created_at timestamp
		if createdAt, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", createdAtStr); err == nil {
			t.CreatedAt = createdAt
		} else if createdAt, err := time.Parse("2006-01-02T15:04:05Z07:00", createdAtStr); err == nil {
			t.CreatedAt = createdAt
		} else if createdAt, err := time.Parse("2006-01-02 15:04:05", createdAtStr); err == nil {
			t.CreatedAt = createdAt
		}

		transactions = append(transactions, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}

// PendingInfo contains information about pending transactions
type PendingInfo struct {
	TotalPendingAmount      decimal.Decimal // Tổng số tiền đang pending (từ frozen) - sử dụng BigDecimal
	PendingTransactionsCount int            // Số lượng giao dịch đang pending
	PendingTransactionsSum   decimal.Decimal // Tổng amount từ các pending transactions - sử dụng BigDecimal
	IsConsistent            bool            // frozen có khớp với tổng pending transactions không
}

// GetPendingInfo retrieves pending transaction information for a user's primary wallet
func (s *WalletService) GetPendingInfo(ctx context.Context, userID string) (*PendingInfo, error) {
	return s.GetPendingInfoByType(ctx, userID, models.WalletTypePrimary)
}

// GetPendingInfoByType retrieves pending transaction information for a specific wallet type
// Kiểm tra tổng số tiền đang pending và số giao dịch pending
func (s *WalletService) GetPendingInfoByType(ctx context.Context, userID string, walletType models.WalletType) (*PendingInfo, error) {
	// Validate wallet type
	if err := s.ValidateWalletType(ctx, walletType); err != nil {
		return nil, err
	}

	// Get frozen amount from wallet (tổng số tiền đang pending)
	query := `SELECT frozen FROM wallets WHERE user_id = $1 AND wallet_type = $2`
	var frozenAmount decimal.Decimal
	err := s.db.QueryRowContext(ctx, query, userID, string(walletType)).Scan(&frozenAmount)
	if err == sql.ErrNoRows {
		// Wallet doesn't exist, no pending transactions
		return &PendingInfo{
			TotalPendingAmount:      decimal.Zero,
			PendingTransactionsCount: 0,
			PendingTransactionsSum:   decimal.Zero,
			IsConsistent:            true,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get frozen amount: %w", err)
	}

	// Get count and sum of pending transactions
	transactionQuery := `SELECT COUNT(*), COALESCE(SUM(amount), 0) 
		FROM wallet_transactions 
		WHERE user_id = $1 AND wallet_type = $2 AND status = $3`
	var pendingCount int
	var pendingSum decimal.Decimal
	err = s.db.QueryRowContext(ctx, transactionQuery, userID, string(walletType), statusPending).Scan(&pendingCount, &pendingSum)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending transactions: %w", err)
	}

	// Kiểm tra tính nhất quán: frozen có khớp với tổng pending transactions không
	// Sử dụng decimal comparison để tránh floating point errors
	isConsistent := frozenAmount.Equal(pendingSum)

	return &PendingInfo{
		TotalPendingAmount:      frozenAmount,
		PendingTransactionsCount: pendingCount,
		PendingTransactionsSum:   pendingSum,
		IsConsistent:            isConsistent,
	}, nil
}

// AddWalletBalance adds money to a user's primary wallet (for testing/admin purposes)
func (s *WalletService) AddWalletBalance(ctx context.Context, userID string, amount float64, description string) error {
	return s.AddWalletBalanceByType(ctx, userID, models.WalletTypePrimary, amount, description)
}

// AddWalletBalanceByType adds money to a specific wallet type
func (s *WalletService) AddWalletBalanceByType(ctx context.Context, userID string, walletType models.WalletType, amount float64, description string) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	// Validate wallet type from database
	if err := s.ValidateWalletType(ctx, walletType); err != nil {
		return err
	}
	
	// Get wallet type info to check constraints
	wtInfo, err := s.GetWalletTypeInfo(ctx, walletType)
	if err != nil {
		return err
	}
	
	// Validate against wallet rules (if rule service is configured)
	if s.ruleService != nil {
		result, err := s.ruleService.ValidateAddBalance(ctx, userID, walletType, amount)
		if err != nil {
			return fmt.Errorf("failed to validate add balance: %w", err)
		}
		if !result.IsValid {
			return fmt.Errorf("rule violation: %s", result.ErrorMessage)
		}
	}

	// Check min/max balance constraints from wallet_types (legacy support)
	if wtInfo.MaxBalance != nil {
		// Get current balance (wallet might not exist yet - that's OK, balance = 0)
		currentBalance := 0.0
		balance, err := s.GetWalletBalanceByType(ctx, userID, walletType)
		if err == nil {
			currentBalance = balance
		} else if err.Error()[:15] == "wallet type '" {
			// Validation error - return it
			return err
		}
		// If wallet doesn't exist, balance stays 0 (which is OK)
		
		if currentBalance+amount > *wtInfo.MaxBalance {
			return fmt.Errorf("amount exceeds maximum balance limit: current=%.2f, adding=%.2f, max=%.2f",
				currentBalance, amount, *wtInfo.MaxBalance)
		}
	}

	// Use transaction to ensure atomicity
	txOptions := &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	}

	dbTx, err := s.db.DB.BeginTx(ctx, txOptions)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer dbTx.Rollback()

	// Get or create wallet
	var currentBalance, currentFrozen float64
	walletQuery := `SELECT balance, frozen FROM wallets WHERE user_id = $1 AND wallet_type = $2 FOR UPDATE`
	err = dbTx.QueryRowContext(ctx, walletQuery, userID, string(walletType)).Scan(&currentBalance, &currentFrozen)
	if err == sql.ErrNoRows {
		// Create wallet with amount as balance, frozen = 0 (no pending amounts)
		insertWalletQuery := `INSERT INTO wallets (user_id, wallet_type, balance, frozen) VALUES ($1, $2, $3, 0.00) RETURNING balance, frozen`
		err = dbTx.QueryRowContext(ctx, insertWalletQuery, userID, string(walletType), amount).Scan(&currentBalance, &currentFrozen)
		if err != nil {
			return fmt.Errorf("failed to create wallet: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to query wallet: %w", err)
	} else {
		// Update balance on database level (frozen remains unchanged when adding money)
		// Database CHECK constraints ensure balance >= 0 and balance >= frozen
		updateWalletQuery := `UPDATE wallets 
			SET balance = balance + $1, updated_at = CURRENT_TIMESTAMP 
			WHERE user_id = $2 AND wallet_type = $3
			RETURNING balance, frozen`
		err = dbTx.QueryRowContext(ctx, updateWalletQuery, amount, userID, string(walletType)).Scan(&currentBalance, &currentFrozen)
		if err != nil {
			return fmt.Errorf("failed to update wallet: %w", err)
		}
	}

	// Create transaction record (credit transactions are always completed)
	var transactionID int
	transactionQuery := `INSERT INTO wallet_transactions (user_id, wallet_type, type, amount, description, status) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	err = dbTx.QueryRowContext(ctx, transactionQuery, userID, string(walletType), "credit", amount, description, statusCompleted).Scan(&transactionID)
	if err != nil {
		return fmt.Errorf("failed to create wallet transaction: %w", err)
	}

	// Create ledger journal entry (Double-entry: Debit User Wallets, Credit Company Cash)
	if s.ledgerService != nil {
		journalEntries := []models.JournalEntryInput{
			{AccountCode: models.AccCodeUserWallets, Debit: amount, Credit: 0},   // Asset increase
			{AccountCode: models.AccCodeCompanyCash, Debit: 0, Credit: amount}, // Cash decrease
		}
		journalID, err := s.ledgerService.CreateJournal(ctx, dbTx,
			models.RefTypeWalletTopup,
			&transactionID,
			fmt.Sprintf("Wallet top-up: %s - %s", userID, description),
			userID,
			journalEntries)
		if err != nil {
			return fmt.Errorf("failed to create ledger journal: %w", err)
		}

		// Post journal immediately (top-up is always complete)
		err = s.ledgerService.PostJournal(ctx, dbTx, journalID)
		if err != nil {
			return fmt.Errorf("failed to post journal: %w", err)
		}
	}

	// Commit transaction
	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// FreezeAmount freezes an amount in the primary wallet (for pending purchases)
func (s *WalletService) FreezeAmount(ctx context.Context, dbTx *sql.Tx, userID string, amount float64, description string, orderID *int) (int, error) {
	return s.FreezeAmountByType(ctx, dbTx, userID, models.WalletTypePrimary, amount, description, orderID)
}

// FreezeAmountByType freezes an amount in a specific wallet type
// Trừ trực tiếp từ balance và cộng vào frozen (đợi transaction pending)
// Balance giảm ngay khi freeze, frozen tăng lên
// Creates a pending transaction record
// Returns the transaction ID and error if insufficient balance
// Note: Wallet type validation should be done before calling this method
func (s *WalletService) FreezeAmountByType(ctx context.Context, dbTx *sql.Tx, userID string, walletType models.WalletType, amount float64, description string, orderID *int) (int, error) {
	// Validate freeze against wallet rules (if rule service is configured)
	// Note: This validation happens before transaction, so we use a separate connection
	if s.ruleService != nil {
		// Use a separate query context (not the transaction) for rule validation
		// Rules are read-only, so this is safe
		result, err := s.ruleService.ValidateFreeze(ctx, userID, walletType, amount)
		if err != nil {
			return 0, fmt.Errorf("failed to validate freeze: %w", err)
		}
		if !result.IsValid {
			return 0, fmt.Errorf("rule violation: %s", result.ErrorMessage)
		}
	}

	// Note: Wallet type validation should be done at service level before transaction starts
	// We validate here as well for safety (using the transaction connection)
	var isActive bool
	validateQuery := `SELECT is_active FROM wallet_types WHERE code = $1`
	err := dbTx.QueryRowContext(ctx, validateQuery, string(walletType)).Scan(&isActive)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("wallet type '%s' not found", walletType)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to validate wallet type: %w", err)
	}
	if !isActive {
		return 0, fmt.Errorf("wallet type '%s' is not active", walletType)
	}
	
	// Freeze the amount (trừ trực tiếp từ balance, cộng vào frozen)
	// Balance giảm ngay khi freeze, frozen tăng lên để đợi transaction pending
	// After freeze: balance = balance - amount, frozen = frozen + amount
	// Kiểm tra balance > 0 và balance >= amount là đủ
	// The UPDATE statement is atomic and locks the row automatically
	freezeQuery := `UPDATE wallets 
		SET balance = balance - $1, frozen = frozen + $1, updated_at = CURRENT_TIMESTAMP 
		WHERE user_id = $2 AND wallet_type = $3
		AND balance > 0
		AND balance >= $1
		RETURNING balance, frozen`
	var updatedBalance, updatedFrozen float64
	err = dbTx.QueryRowContext(ctx, freezeQuery, amount, userID, string(walletType)).Scan(&updatedBalance, &updatedFrozen)
	if err == sql.ErrNoRows {
		// Get current values for error message
		var currentBalance, currentFrozen float64
		checkQuery := `SELECT balance, frozen FROM wallets WHERE user_id = $1 AND wallet_type = $2`
		dbTx.QueryRowContext(ctx, checkQuery, userID, string(walletType)).Scan(&currentBalance, &currentFrozen)
		availableBalance := currentBalance - currentFrozen
		return 0, fmt.Errorf("insufficient wallet balance: required $%.2f, available: $%.2f (balance: $%.2f, frozen: $%.2f)",
			amount, availableBalance, currentBalance, currentFrozen)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to freeze wallet amount: %w", err)
	}

	// Create pending transaction record
	transactionQuery := `INSERT INTO wallet_transactions (user_id, wallet_type, type, amount, description, order_id, status) 
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	var transactionID int
	err = dbTx.QueryRowContext(ctx, transactionQuery, userID, string(walletType), "debit", amount, description, orderID, statusPending).Scan(&transactionID)
	if err != nil {
		return 0, fmt.Errorf("failed to create pending transaction: %w", err)
	}

	// Create ledger journal entry for freeze (Double-entry: Debit Frozen Funds, Credit User Wallets)
	if s.ledgerService != nil {
		journalEntries := []models.JournalEntryInput{
			{AccountCode: models.AccCodeFrozenFunds, Debit: amount, Credit: 0}, // Liability increase (we owe user)
			{AccountCode: models.AccCodeUserWallets, Debit: 0, Credit: amount}, // Asset decrease
		}
		journalID, err := s.ledgerService.CreateJournal(ctx, dbTx,
			models.RefTypeFreeze,
			orderID, // Reference to order
			fmt.Sprintf("Freeze amount for purchase: %s - %s", userID, description),
			userID,
			journalEntries)
		if err != nil {
			return 0, fmt.Errorf("failed to create freeze journal: %w", err)
		}
		// Keep journal as pending (will be posted when committed or reversed when rolled back)
		// Journal ID is stored in transaction metadata if needed (not implemented in wallet_transactions table)
		_ = journalID
	}

	return transactionID, nil
}

// CommitTransaction commits a pending transaction (completes a purchase)
func (s *WalletService) CommitTransaction(ctx context.Context, dbTx *sql.Tx, userID string, amount float64, transactionID int) error {
	// Get transaction to find wallet_type
	var walletTypeStr string
	txQuery := `SELECT wallet_type FROM wallet_transactions WHERE id = $1`
	err := dbTx.QueryRowContext(ctx, txQuery, transactionID).Scan(&walletTypeStr)
	if err != nil {
		return fmt.Errorf("failed to find transaction: %w", err)
	}
	walletType := models.WalletType(walletTypeStr)
	// Note: Wallet type validation is done from database in FreezeAmountByType
	// Here we just use the type from the transaction
	
	return s.CommitTransactionByType(ctx, dbTx, userID, walletType, amount, transactionID)
}

// CommitTransactionByType commits a pending transaction for a specific wallet type
// Clean frozen (balance đã bị trừ khi freeze rồi, chỉ cần clean frozen)
// Mark transaction as completed
// Call this after all validations (goods check, 3rd party services) are done
func (s *WalletService) CommitTransactionByType(ctx context.Context, dbTx *sql.Tx, userID string, walletType models.WalletType, amount float64, transactionID int) error {
	// Note: Wallet type is already validated when transaction was created
	
	// Clean frozen (balance đã bị trừ khi freeze, chỉ cần giảm frozen)
	// Mark transaction as completed
	commitQuery := `UPDATE wallets 
		SET frozen = frozen - $1, updated_at = CURRENT_TIMESTAMP 
		WHERE user_id = $2 AND wallet_type = $3
		AND frozen >= $1
		RETURNING balance, frozen`
	var finalBalanceCheck, finalFrozenCheck float64
	err := dbTx.QueryRowContext(ctx, commitQuery, amount, userID, string(walletType)).Scan(&finalBalanceCheck, &finalFrozenCheck)
	if err == sql.ErrNoRows {
		// Get current frozen for error message
		var currentFrozen float64
		dbTx.QueryRowContext(ctx, `SELECT frozen FROM wallets WHERE user_id = $1 AND wallet_type = $2`, userID, string(walletType)).Scan(&currentFrozen)
		return fmt.Errorf("wallet state mismatch during commit: frozen amount $%.2f is less than required $%.2f",
			currentFrozen, amount)
	}
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Mark transaction as completed
	updateTransactionQuery := `UPDATE wallet_transactions 
		SET status = $2 
		WHERE id = $1 AND status = $3`
	_, err = dbTx.ExecContext(ctx, updateTransactionQuery, transactionID, statusCompleted, statusPending)
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	// Create ledger journal entry for commit (Double-entry: Debit Company Cash, Credit Frozen Funds)
	// This completes the purchase - money moves from frozen to company
	if s.ledgerService != nil {
		journalEntries := []models.JournalEntryInput{
			{AccountCode: models.AccCodeCompanyCash, Debit: amount, Credit: 0},     // Company receives money
			{AccountCode: models.AccCodeFrozenFunds, Debit: 0, Credit: amount},     // Release frozen funds
		}
		journalID, err := s.ledgerService.CreateJournal(ctx, dbTx,
			models.RefTypePurchase,
			&transactionID,
			fmt.Sprintf("Commit purchase transaction #%d", transactionID),
			"system",
			journalEntries)
		if err != nil {
			return fmt.Errorf("failed to create commit journal: %w", err)
		}

		// Post journal immediately (purchase is complete)
		err = s.ledgerService.PostJournal(ctx, dbTx, journalID)
		if err != nil {
			return fmt.Errorf("failed to post commit journal: %w", err)
		}
	}

	return nil
}

// RollbackTransaction rolls back a pending transaction (if validation fails)
func (s *WalletService) RollbackTransaction(ctx context.Context, dbTx *sql.Tx, userID string, amount float64, transactionID int) error {
	// Get transaction to find wallet_type
	var walletTypeStr string
	txQuery := `SELECT wallet_type FROM wallet_transactions WHERE id = $1`
	err := dbTx.QueryRowContext(ctx, txQuery, transactionID).Scan(&walletTypeStr)
	if err != nil {
		return fmt.Errorf("failed to find transaction: %w", err)
	}
	walletType := models.WalletType(walletTypeStr)
	// Note: Wallet type validation is done from database in FreezeAmountByType
	// Here we just use the type from the transaction
	
	return s.RollbackTransactionByType(ctx, dbTx, userID, walletType, amount, transactionID)
}

// RollbackTransactionByType rolls back a pending transaction for a specific wallet type
// Restore balance và clean frozen (trả lại balance đã bị trừ khi freeze)
func (s *WalletService) RollbackTransactionByType(ctx context.Context, dbTx *sql.Tx, userID string, walletType models.WalletType, amount float64, transactionID int) error {
	// Note: Wallet type is already validated when transaction was created
	
	// Restore balance và clean frozen (trả lại balance, giảm frozen)
	rollbackQuery := `UPDATE wallets 
		SET balance = balance + $1, frozen = frozen - $1, updated_at = CURRENT_TIMESTAMP 
		WHERE user_id = $2 AND wallet_type = $3
		AND frozen >= $1
		RETURNING balance, frozen`
	var finalBalanceCheck, finalFrozenCheck float64
	err := dbTx.QueryRowContext(ctx, rollbackQuery, amount, userID, string(walletType)).Scan(&finalBalanceCheck, &finalFrozenCheck)
	if err == sql.ErrNoRows {
		// Get current frozen for error message
		var currentFrozen float64
		dbTx.QueryRowContext(ctx, `SELECT frozen FROM wallets WHERE user_id = $1 AND wallet_type = $2`, userID, string(walletType)).Scan(&currentFrozen)
		return fmt.Errorf("wallet state mismatch during rollback: frozen amount $%.2f is less than required $%.2f",
			currentFrozen, amount)
	}
	if err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	// Mark transaction as cancelled
	updateTransactionQuery := `UPDATE wallet_transactions 
		SET status = $2 
		WHERE id = $1 AND status = $3`
	_, err = dbTx.ExecContext(ctx, updateTransactionQuery, transactionID, statusCancelled, statusPending)
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	// Create ledger journal entry for rollback (Reverse the freeze: Debit User Wallets, Credit Frozen Funds)
	if s.ledgerService != nil {
		journalEntries := []models.JournalEntryInput{
			{AccountCode: models.AccCodeUserWallets, Debit: amount, Credit: 0},  // Restore user wallet
			{AccountCode: models.AccCodeFrozenFunds, Debit: 0, Credit: amount},  // Release frozen funds
		}
		journalID, err := s.ledgerService.CreateJournal(ctx, dbTx,
			models.RefTypeUnfreeze,
			&transactionID,
			fmt.Sprintf("Rollback transaction #%d - Refund to user", transactionID),
			"system",
			journalEntries)
		if err != nil {
			return fmt.Errorf("failed to create rollback journal: %w", err)
		}

		// Post journal immediately (rollback is complete)
		err = s.ledgerService.PostJournal(ctx, dbTx, journalID)
		if err != nil {
			return fmt.Errorf("failed to post rollback journal: %w", err)
		}
	}

	return nil
}

// ============================================================================
// WALLET TRANSFER
// ============================================================================

// TransferBetweenWallets transfers money from one wallet to another for the same user
// This is an atomic operation that:
// 1. Deducts amount from source wallet (debit transaction)
// 2. Adds amount to destination wallet (credit transaction)
// 3. Creates ledger journal entry (Debit destination wallet, Credit source wallet)
// 4. Creates transaction records for both wallets
func (s *WalletService) TransferBetweenWallets(ctx context.Context, userID string, fromWalletType, toWalletType models.WalletType, amount float64, description string) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	// Cannot transfer to the same wallet
	if fromWalletType == toWalletType {
		return fmt.Errorf("cannot transfer to the same wallet type")
	}

	// Validate both wallet types
	if err := s.ValidateWalletType(ctx, fromWalletType); err != nil {
		return fmt.Errorf("invalid source wallet type: %w", err)
	}
	if err := s.ValidateWalletType(ctx, toWalletType); err != nil {
		return fmt.Errorf("invalid destination wallet type: %w", err)
	}

	// Validate transfer against wallet rules (if rule service is configured)
	if s.ruleService != nil {
		result, err := s.ruleService.ValidateTransfer(ctx, userID, fromWalletType, toWalletType, amount)
		if err != nil {
			return fmt.Errorf("failed to validate transfer: %w", err)
		}
		if !result.IsValid {
			return fmt.Errorf("rule violation: %s", result.ErrorMessage)
		}
	}

	// Check available balance in source wallet
	availableBalance, err := s.GetWalletBalanceByType(ctx, userID, fromWalletType)
	if err != nil {
		return fmt.Errorf("failed to check source wallet balance: %w", err)
	}
	if availableBalance < amount {
		return fmt.Errorf("insufficient balance: available $%.2f, required $%.2f", availableBalance, amount)
	}

	// Use transaction to ensure atomicity
	txOptions := &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	}

	dbTx, err := s.db.DB.BeginTx(ctx, txOptions)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer dbTx.Rollback()

	// Lock and deduct from source wallet
	// Only deduct from balance (not frozen) - transfer uses available balance
	deductQuery := `UPDATE wallets 
		SET balance = balance - $1, updated_at = CURRENT_TIMESTAMP 
		WHERE user_id = $2 AND wallet_type = $3
		AND (balance - frozen) >= $1
		RETURNING balance, frozen`
	var fromBalance, fromFrozen float64
	err = dbTx.QueryRowContext(ctx, deductQuery, amount, userID, string(fromWalletType)).Scan(&fromBalance, &fromFrozen)
	if err == sql.ErrNoRows {
		// Get current values for error message
		var currentBalance, currentFrozen float64
		dbTx.QueryRowContext(ctx, `SELECT balance, frozen FROM wallets WHERE user_id = $1 AND wallet_type = $2`, userID, string(fromWalletType)).Scan(&currentBalance, &currentFrozen)
		return fmt.Errorf("insufficient available balance: available $%.2f (balance: $%.2f, frozen: $%.2f), required $%.2f",
			currentBalance-currentFrozen, currentBalance, currentFrozen, amount)
	}
	if err != nil {
		return fmt.Errorf("failed to deduct from source wallet: %w", err)
	}

	// Add to destination wallet (create if doesn't exist)
	creditQuery := `INSERT INTO wallets (user_id, wallet_type, balance, frozen) 
		VALUES ($1, $2, $3, 0.00)
		ON CONFLICT (user_id, wallet_type) 
		DO UPDATE SET balance = wallets.balance + $3, updated_at = CURRENT_TIMESTAMP
		RETURNING balance, frozen`
	var toBalance, toFrozen float64
	err = dbTx.QueryRowContext(ctx, creditQuery, userID, string(toWalletType), amount).Scan(&toBalance, &toFrozen)
	if err != nil {
		return fmt.Errorf("failed to add to destination wallet: %w", err)
	}

	// Create debit transaction record for source wallet
	debitDescription := fmt.Sprintf("Transfer to %s: %s", toWalletType, description)
	debitTransactionQuery := `INSERT INTO wallet_transactions (user_id, wallet_type, type, amount, description, status) 
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	var debitTransactionID int
	err = dbTx.QueryRowContext(ctx, debitTransactionQuery, userID, string(fromWalletType), "debit", amount, debitDescription, statusCompleted).Scan(&debitTransactionID)
	if err != nil {
		return fmt.Errorf("failed to create debit transaction: %w", err)
	}

	// Create credit transaction record for destination wallet
	creditDescription := fmt.Sprintf("Transfer from %s: %s", fromWalletType, description)
	creditTransactionQuery := `INSERT INTO wallet_transactions (user_id, wallet_type, type, amount, description, status) 
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	var creditTransactionID int
	err = dbTx.QueryRowContext(ctx, creditTransactionQuery, userID, string(toWalletType), "credit", amount, creditDescription, statusCompleted).Scan(&creditTransactionID)
	if err != nil {
		return fmt.Errorf("failed to create credit transaction: %w", err)
	}

	// Create ledger journal entry (Double-entry: Debit destination wallet, Credit source wallet)
	// This records the transfer in the accounting system
	if s.ledgerService != nil {
		// Both wallets use AccCodeUserWallets account
		// Debit the destination (increases its asset), Credit the source (decreases its asset)
		journalEntries := []models.JournalEntryInput{
			{AccountCode: models.AccCodeUserWallets, Debit: amount, Credit: 0}, // Destination wallet receives
			{AccountCode: models.AccCodeUserWallets, Debit: 0, Credit: amount}, // Source wallet gives
		}
		journalID, err := s.ledgerService.CreateJournal(ctx, dbTx,
			models.RefTypeTransfer,
			&debitTransactionID, // Reference to debit transaction
			fmt.Sprintf("Transfer: %s -> %s (%s) - %s", fromWalletType, toWalletType, userID, description),
			userID,
			journalEntries)
		if err != nil {
			return fmt.Errorf("failed to create transfer journal: %w", err)
		}

		// Post journal immediately (transfer is complete)
		err = s.ledgerService.PostJournal(ctx, dbTx, journalID)
		if err != nil {
			return fmt.Errorf("failed to post transfer journal: %w", err)
		}
	}

	// Commit transaction
	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transfer transaction: %w", err)
	}

	return nil
}

// ============================================================================
// AUTO-CREATE WALLETS FOR USER
// ============================================================================

// AutoCreateWalletsForUser automatically creates wallets for a user based on active wallet types
// 
// NOTE: Wallet system is SEPARATED from users system. This method only requires a user_id (string).
// It does NOT require a users table or foreign key constraints. The wallet system operates
// independently and can work with any user_id identifier.
//
// This should be called when a new user account is created (or manually for existing users).
// It creates wallets for all active wallet types in the wallet_types table.
//
// Parameters:
//   - userID: User identifier (string). Can be any unique identifier, doesn't need to exist in users table.
//
// Returns:
//   - error: Returns error if wallet creation fails
func (s *WalletService) AutoCreateWalletsForUser(ctx context.Context, userID string) error {
	// Get all active wallet types
	walletTypes, err := s.GetAllWalletTypes(ctx, false) // false = only active
	if err != nil {
		return fmt.Errorf("failed to get active wallet types: %w", err)
	}

	if len(walletTypes) == 0 {
		// No active wallet types, nothing to create
		return nil
	}

	// Use transaction to ensure atomicity
	txOptions := &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	}

	dbTx, err := s.db.DB.BeginTx(ctx, txOptions)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer dbTx.Rollback()

	// Create wallets for each active wallet type
	// Use INSERT ... ON CONFLICT DO NOTHING to avoid errors if wallet already exists
	for _, wt := range walletTypes {
		if !wt.IsActive {
			continue // Skip inactive types (shouldn't happen with GetAllWalletTypes(false), but be safe)
		}

		// Check if wallet already exists
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM wallets WHERE user_id = $1 AND wallet_type = $2)`
		err = dbTx.QueryRowContext(ctx, checkQuery, userID, wt.Code).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check wallet existence: %w", err)
		}

		if exists {
			// Wallet already exists, skip
			continue
		}

		// Create wallet with initial balance 0.00 and frozen 0.00
		// Check min_balance constraint if defined
		initialBalance := 0.00
		if wt.MinBalance != nil && *wt.MinBalance > 0 {
			initialBalance = *wt.MinBalance
		}

		insertQuery := `INSERT INTO wallets (user_id, wallet_type, balance, frozen) 
			VALUES ($1, $2, $3, 0.00)`
		_, err = dbTx.ExecContext(ctx, insertQuery, userID, wt.Code, initialBalance)
		if err != nil {
			return fmt.Errorf("failed to create wallet %s for user %s: %w", wt.Code, userID, err)
		}
	}

	// Commit transaction
	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("failed to commit wallet creation transaction: %w", err)
	}

	return nil
}

// CreateWalletForUser creates a specific wallet type for a user
// 
// NOTE: Wallet system is SEPARATED from users system. This method only requires a user_id (string).
// It does NOT require a users table or foreign key constraints.
//
// This is a helper method that can be used to create a single wallet for a user.
//
// Parameters:
//   - userID: User identifier (string). Can be any unique identifier.
//   - walletType: The type of wallet to create (must be active in wallet_types table)
//
// Returns:
//   - error: Returns error if wallet creation fails
func (s *WalletService) CreateWalletForUser(ctx context.Context, userID string, walletType models.WalletType) error {
	// Validate wallet type
	if err := s.ValidateWalletType(ctx, walletType); err != nil {
		return err
	}

	// Get wallet type info
	wtInfo, err := s.GetWalletTypeInfo(ctx, walletType)
	if err != nil {
		return err
	}

	if !wtInfo.IsActive {
		return fmt.Errorf("wallet type '%s' is not active", walletType)
	}

	// Check if wallet already exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM wallets WHERE user_id = $1 AND wallet_type = $2)`
	err = s.db.QueryRowContext(ctx, checkQuery, userID, string(walletType)).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check wallet existence: %w", err)
	}

	if exists {
		// Wallet already exists
		return nil
	}

	// Determine initial balance
	initialBalance := 0.00
	if wtInfo.MinBalance != nil && *wtInfo.MinBalance > 0 {
		initialBalance = *wtInfo.MinBalance
	}

	// Create wallet
	insertQuery := `INSERT INTO wallets (user_id, wallet_type, balance, frozen) 
		VALUES ($1, $2, $3, 0.00)`
	_, err = s.db.ExecContext(ctx, insertQuery, userID, string(walletType), initialBalance)
	if err != nil {
		return fmt.Errorf("failed to create wallet: %w", err)
	}

	return nil
}

