package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/fluxorio/fluxor/apps/postgres-demo/models"
	"github.com/fluxorio/fluxor/pkg/dbruntime"
)

// LedgerService handles ledger and accounting operations with double-entry bookkeeping
type LedgerService struct {
	db *dbruntime.DB
}

// NewLedgerService creates a new ledger service
func NewLedgerService(db *dbruntime.DB) *LedgerService {
	return &LedgerService{
		db: db,
	}
}

// ============================================================================
// ACCOUNT MANAGEMENT
// ============================================================================

// InitializeSystemAccounts creates all system accounts if they don't exist
func (s *LedgerService) InitializeSystemAccounts(ctx context.Context) error {
	accounts := []struct {
		code string
		name string
		typ  models.AccountType
	}{
		{models.AccCodeUserWallets, "User Wallets", models.AccountTypeAsset},
		{models.AccCodeCompanyCash, "Company Cash", models.AccountTypeAsset},
		{models.AccCodeFrozenFunds, "Frozen Funds", models.AccountTypeLiability},
		{models.AccCodeUserPayables, "User Payables", models.AccountTypeLiability},
		{models.AccCodeSalesRevenue, "Sales Revenue", models.AccountTypeRevenue},
		{models.AccCodeTransactionFees, "Transaction Fees", models.AccountTypeRevenue},
		{models.AccCodeRefunds, "Refunds", models.AccountTypeExpense},
		{models.AccCodeOperational, "Operational Expenses", models.AccountTypeExpense},
	}

	for _, acc := range accounts {
		query := `INSERT INTO accounts (code, name, type, is_system) 
			VALUES ($1, $2, $3, true)
			ON CONFLICT (code) DO NOTHING`
		_, err := s.db.ExecContext(ctx, query, acc.code, acc.name, acc.typ)
		if err != nil {
			return fmt.Errorf("failed to initialize account %s: %w", acc.code, err)
		}
	}

	return nil
}

// GetAccountByCode retrieves an account by its code
func (s *LedgerService) GetAccountByCode(ctx context.Context, code string) (*models.Account, error) {
	query := `SELECT id, code, name, type, parent_id, is_system, created_at 
		FROM accounts WHERE code = $1`
	var account models.Account
	err := s.db.QueryRowContext(ctx, query, code).Scan(
		&account.ID, &account.Code, &account.Name, &account.Type,
		&account.ParentID, &account.IsSystem, &account.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("account with code %s not found", code)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	return &account, nil
}

// ============================================================================
// JOURNAL ENTRY CREATION (DOUBLE-ENTRY BOOKKEEPING)
// ============================================================================

// CreateJournal creates a journal entry with multiple ledger entries
// All entries must balance: total debit = total credit
func (s *LedgerService) CreateJournal(
	ctx context.Context,
	dbTx *sql.Tx,
	refType models.ReferenceType,
	refID *int,
	description string,
	createdBy string,
	entries []models.JournalEntryInput,
) (int, error) {
	// Validate entries
	if len(entries) < 2 {
		return 0, fmt.Errorf("journal must have at least 2 entries")
	}

	// Validate balance (debit must equal credit)
	var totalDebit, totalCredit float64
	for _, entry := range entries {
		if entry.Debit < 0 || entry.Credit < 0 {
			return 0, fmt.Errorf("debit and credit amounts must be non-negative")
		}
		if entry.Debit > 0 && entry.Credit > 0 {
			return 0, fmt.Errorf("entry cannot have both debit and credit: account %s", entry.AccountCode)
		}
		if entry.Debit == 0 && entry.Credit == 0 {
			return 0, fmt.Errorf("entry must have either debit or credit: account %s", entry.AccountCode)
		}
		totalDebit += entry.Debit
		totalCredit += entry.Credit
	}

	const epsilon = 0.01 // Allow small floating point differences
	diff := totalDebit - totalCredit
	if diff < -epsilon || diff > epsilon {
		return 0, fmt.Errorf("journal entries must balance: debit=%.2f, credit=%.2f, diff=%.2f",
			totalDebit, totalCredit, diff)
	}

	// Verify all accounts exist and get their IDs
	accountIDs := make(map[string]int)
	for _, entry := range entries {
		account, err := s.GetAccountByCode(ctx, entry.AccountCode)
		if err != nil {
			return 0, fmt.Errorf("account %s not found: %w", entry.AccountCode, err)
		}
		accountIDs[entry.AccountCode] = account.ID
	}

	// Create journal
	journalQuery := `INSERT INTO journals (reference_type, reference_id, description, status, created_by)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`
	var journalID int
	err := dbTx.QueryRowContext(ctx, journalQuery,
		string(refType), refID, description, string(models.JournalStatusPending), createdBy).Scan(&journalID)
	if err != nil {
		return 0, fmt.Errorf("failed to create journal: %w", err)
	}

	// Create ledger entries
	for _, entry := range entries {
		accountID := accountIDs[entry.AccountCode]
		entryQuery := `INSERT INTO ledger_entries (journal_id, account_id, debit, credit)
			VALUES ($1, $2, $3, $4)`
		_, err = dbTx.ExecContext(ctx, entryQuery, journalID, accountID, entry.Debit, entry.Credit)
		if err != nil {
			return 0, fmt.Errorf("failed to create ledger entry for account %s: %w", entry.AccountCode, err)
		}
	}

	return journalID, nil
}

// PostJournal posts a pending journal (changes status from pending to posted)
func (s *LedgerService) PostJournal(ctx context.Context, dbTx *sql.Tx, journalID int) error {
	// Verify journal exists and is pending
	var currentStatus string
	checkQuery := `SELECT status FROM journals WHERE id = $1`
	err := dbTx.QueryRowContext(ctx, checkQuery, journalID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		return fmt.Errorf("journal %d not found", journalID)
	}
	if err != nil {
		return fmt.Errorf("failed to check journal status: %w", err)
	}
	if currentStatus != string(models.JournalStatusPending) {
		return fmt.Errorf("journal %d is not pending (current status: %s)", journalID, currentStatus)
	}

	// Update journal status to posted
	now := time.Now()
	updateQuery := `UPDATE journals SET status = $2, posted_at = $3 WHERE id = $1`
	_, err = dbTx.ExecContext(ctx, updateQuery, journalID, string(models.JournalStatusPosted), now)
	if err != nil {
		return fmt.Errorf("failed to post journal: %w", err)
	}

	return nil
}

// ReverseJournal reverses a posted journal by creating an opposite journal
func (s *LedgerService) ReverseJournal(
	ctx context.Context,
	dbTx *sql.Tx,
	journalID int,
	description string,
	createdBy string,
) (int, error) {
	// Get original journal with entries
	journal, entries, err := s.GetJournalWithEntries(ctx, journalID)
	if err != nil {
		return 0, fmt.Errorf("failed to get journal: %w", err)
	}

	if journal.Status.Status != models.JournalStatusPosted {
		return 0, fmt.Errorf("can only reverse posted journals (current: %s)", journal.Status.Status)
	}

	// Create reverse entries (swap debit and credit)
	reverseEntries := make([]models.JournalEntryInput, len(entries))
	for i, entry := range entries {
		account, err := s.GetAccountByCode(ctx, entry.Account.Code)
		if err != nil {
			return 0, fmt.Errorf("failed to get account: %w", err)
		}
		reverseEntries[i] = models.JournalEntryInput{
			AccountCode: account.Code,
			Debit:       entry.Credit, // Swap
			Credit:      entry.Debit,  // Swap
		}
	}

	// Create reverse journal
	reverseJournalID, err := s.CreateJournal(ctx, dbTx,
		models.RefTypeAdjustment,
		&journalID, // Reference to original journal
		description,
		createdBy,
		reverseEntries)
	if err != nil {
		return 0, fmt.Errorf("failed to create reverse journal: %w", err)
	}

	// Mark original journal as reversed
	updateQuery := `UPDATE journals SET status = $2 WHERE id = $1`
	_, err = dbTx.ExecContext(ctx, updateQuery, journalID, string(models.JournalStatusReversed))
	if err != nil {
		return 0, fmt.Errorf("failed to mark journal as reversed: %w", err)
	}

	// Post the reverse journal immediately
	err = s.PostJournal(ctx, dbTx, reverseJournalID)
	if err != nil {
		return 0, fmt.Errorf("failed to post reverse journal: %w", err)
	}

	return reverseJournalID, nil
}

// ============================================================================
// JOURNAL QUERIES
// ============================================================================

// GetJournalWithEntries retrieves a journal with all its ledger entries
func (s *LedgerService) GetJournalWithEntries(ctx context.Context, journalID int) (*models.Journal, []models.LedgerEntry, error) {
	// Get journal
	journalQuery := `SELECT id, reference_type, reference_id, description, status, posted_at, created_at, created_by
		FROM journals WHERE id = $1`
	var journal models.Journal
	var statusStr sql.NullString
	err := s.db.QueryRowContext(ctx, journalQuery, journalID).Scan(
		&journal.ID, &journal.ReferenceType, &journal.ReferenceID, &journal.Description,
		&statusStr, &journal.PostedAt, &journal.CreatedAt, &journal.CreatedBy)
	if err == sql.ErrNoRows {
		return nil, nil, fmt.Errorf("journal %d not found", journalID)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get journal: %w", err)
	}

	if statusStr.Valid {
		journal.Status = models.NullableJournalStatus{
			Status: models.JournalStatus(statusStr.String),
			Valid:  true,
		}
	}

	// Get entries with account details
	entriesQuery := `SELECT le.id, le.journal_id, le.account_id, le.debit, le.credit, le.created_at,
		a.code, a.name, a.type
		FROM ledger_entries le
		JOIN accounts a ON le.account_id = a.id
		WHERE le.journal_id = $1
		ORDER BY le.id`
	rows, err := s.db.QueryContext(ctx, entriesQuery, journalID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query entries: %w", err)
	}
	defer rows.Close()

	var entries []models.LedgerEntry
	for rows.Next() {
		var entry models.LedgerEntry
		var account models.Account
		err := rows.Scan(
			&entry.ID, &entry.JournalID, &entry.AccountID, &entry.Debit, &entry.Credit, &entry.CreatedAt,
			&account.Code, &account.Name, &account.Type)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan entry: %w", err)
		}
		entry.Account = &account
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating entries: %w", err)
	}

	journal.Entries = entries
	return &journal, entries, nil
}

// ============================================================================
// ACCOUNT BALANCE CALCULATION
// ============================================================================

// GetAccountBalance calculates the balance of an account
// For debit-normal accounts (Assets, Expenses): balance = debit - credit
// For credit-normal accounts (Liabilities, Equity, Revenue): balance = credit - debit
func (s *LedgerService) GetAccountBalance(ctx context.Context, accountCode string, asOf *time.Time) (*models.AccountBalance, error) {
	account, err := s.GetAccountByCode(ctx, accountCode)
	if err != nil {
		return nil, err
	}

	var asOfClause string
	var args []interface{}
	if asOf != nil {
		asOfClause = "AND le.created_at <= $2"
		args = []interface{}{account.ID, *asOf}
	} else {
		args = []interface{}{account.ID}
	}

	query := fmt.Sprintf(`SELECT 
		COALESCE(SUM(le.debit), 0) as total_debit,
		COALESCE(SUM(le.credit), 0) as total_credit
		FROM ledger_entries le
		JOIN journals j ON le.journal_id = j.id
		WHERE le.account_id = $1 AND j.status = 'posted' %s`, asOfClause)

	var totalDebit, totalCredit float64
	err = s.db.QueryRowContext(ctx, query, args...).Scan(&totalDebit, &totalCredit)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate balance: %w", err)
	}

	// Calculate balance based on account type
	var balance float64
	if account.Type.IsDebitNormal() {
		balance = totalDebit - totalCredit
	} else {
		balance = totalCredit - totalDebit
	}

	return &models.AccountBalance{
		AccountID:   account.ID,
		AccountCode: account.Code,
		AccountName: account.Name,
		AccountType: account.Type,
		TotalDebit:  totalDebit,
		TotalCredit: totalCredit,
		Balance:     balance,
	}, nil
}

// GetTrialBalance generates a trial balance report
func (s *LedgerService) GetTrialBalance(ctx context.Context, asOf *time.Time) (*models.TrialBalance, error) {
	var whereClause string
	var args []interface{}
	reportTime := time.Now()
	if asOf != nil {
		whereClause = "WHERE (le.id IS NULL OR (le.created_at <= $1 AND j.posted_at <= $1))"
		args = []interface{}{*asOf}
		reportTime = *asOf
	}

	query := fmt.Sprintf(`SELECT 
		a.id, a.code, a.name, a.type,
		COALESCE(SUM(le.debit), 0) as total_debit,
		COALESCE(SUM(le.credit), 0) as total_credit
		FROM accounts a
		LEFT JOIN ledger_entries le ON a.id = le.account_id
		LEFT JOIN journals j ON le.journal_id = j.id AND j.status = 'posted'
		%s
		GROUP BY a.id, a.code, a.name, a.type
		HAVING COALESCE(SUM(le.debit), 0) > 0 OR COALESCE(SUM(le.credit), 0) > 0
		ORDER BY a.code`, whereClause)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query trial balance: %w", err)
	}
	defer rows.Close()

	var accounts []models.AccountBalance
	var totalDebit, totalCredit float64

	for rows.Next() {
		var acc models.AccountBalance
		var accountType string
		err := rows.Scan(&acc.AccountID, &acc.AccountCode, &acc.AccountName, &accountType,
			&acc.TotalDebit, &acc.TotalCredit)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		acc.AccountType = models.AccountType(accountType)

		// Calculate balance based on account type
		if acc.AccountType.IsDebitNormal() {
			acc.Balance = acc.TotalDebit - acc.TotalCredit
		} else {
			acc.Balance = acc.TotalCredit - acc.TotalDebit
		}

		accounts = append(accounts, acc)
		totalDebit += acc.TotalDebit
		totalCredit += acc.TotalCredit
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating accounts: %w", err)
	}

	// Check if balanced
	const epsilon = 0.01
	isBalanced := (totalDebit-totalCredit) < epsilon && (totalCredit-totalDebit) < epsilon

	return &models.TrialBalance{
		AsOf:       reportTime,
		Accounts:   accounts,
		TotalDebit: totalDebit,
		TotalCredit: totalCredit,
		IsBalanced: isBalanced,
	}, nil
}

// GetAllAccountBalances returns balances for all accounts
func (s *LedgerService) GetAllAccountBalances(ctx context.Context, asOf *time.Time) ([]models.AccountBalance, error) {
	var whereClause string
	var args []interface{}
	if asOf != nil {
		whereClause = "AND le.created_at <= $1 AND (j.posted_at IS NULL OR j.posted_at <= $1)"
		args = []interface{}{*asOf}
	}

	query := fmt.Sprintf(`SELECT 
		a.id, a.code, a.name, a.type,
		COALESCE(SUM(le.debit), 0) as total_debit,
		COALESCE(SUM(le.credit), 0) as total_credit
		FROM accounts a
		LEFT JOIN ledger_entries le ON a.id = le.account_id
		LEFT JOIN journals j ON le.journal_id = j.id AND j.status = 'posted'
		WHERE 1=1 %s
		GROUP BY a.id, a.code, a.name, a.type
		ORDER BY a.code`, whereClause)

	var rows *sql.Rows
	var err error
	if len(args) > 0 {
		rows, err = s.db.QueryContext(ctx, query, args...)
	} else {
		rows, err = s.db.QueryContext(ctx, query)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query account balances: %w", err)
	}
	defer rows.Close()

	var balances []models.AccountBalance
	for rows.Next() {
		var acc models.AccountBalance
		var accountType string
		err := rows.Scan(&acc.AccountID, &acc.AccountCode, &acc.AccountName, &accountType,
			&acc.TotalDebit, &acc.TotalCredit)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		acc.AccountType = models.AccountType(accountType)

		// Calculate balance based on account type
		if acc.AccountType.IsDebitNormal() {
			acc.Balance = acc.TotalDebit - acc.TotalCredit
		} else {
			acc.Balance = acc.TotalCredit - acc.TotalDebit
		}

		balances = append(balances, acc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating accounts: %w", err)
	}

	return balances, nil
}

// GetJournals retrieves journals with optional filters
func (s *LedgerService) GetJournals(ctx context.Context, startDate, endDate *time.Time, status, referenceType string, limit int) ([]models.Journal, error) {
	var whereClauses []string
	var args []interface{}
	argIndex := 1

	if startDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("j.created_at >= $%d", argIndex))
		args = append(args, *startDate)
		argIndex++
	}
	if endDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("j.created_at <= $%d", argIndex))
		args = append(args, *endDate)
		argIndex++
	}
	if status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("j.status = $%d", argIndex))
		args = append(args, status)
		argIndex++
	}
	if referenceType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("j.reference_type = $%d", argIndex))
		args = append(args, referenceType)
		argIndex++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	query := fmt.Sprintf(`SELECT id, reference_type, reference_id, description, status, posted_at, created_at, created_by
		FROM journals j
		%s
		ORDER BY j.created_at DESC
		LIMIT $%d`, whereClause, argIndex)
	
	if limit <= 0 {
		limit = 100
	}
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query journals: %w", err)
	}
	defer rows.Close()

	var journals []models.Journal
	for rows.Next() {
		var journal models.Journal
		var statusStr sql.NullString
		err := rows.Scan(
			&journal.ID, &journal.ReferenceType, &journal.ReferenceID, &journal.Description,
			&statusStr, &journal.PostedAt, &journal.CreatedAt, &journal.CreatedBy)
		if err != nil {
			return nil, fmt.Errorf("failed to scan journal: %w", err)
		}

		if statusStr.Valid {
			journal.Status = models.NullableJournalStatus{
				Status: models.JournalStatus(statusStr.String),
				Valid:  true,
			}
		}

		journals = append(journals, journal)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating journals: %w", err)
	}

	return journals, nil
}

// GetAllAccounts returns all accounts
func (s *LedgerService) GetAllAccounts(ctx context.Context) ([]models.Account, error) {
	query := `SELECT id, code, name, type, parent_id, is_system, created_at 
		FROM accounts ORDER BY code`
	
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query accounts: %w", err)
	}
	defer rows.Close()

	var accounts []models.Account
	for rows.Next() {
		var account models.Account
		err := rows.Scan(
			&account.ID, &account.Code, &account.Name, &account.Type,
			&account.ParentID, &account.IsSystem, &account.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		accounts = append(accounts, account)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating accounts: %w", err)
	}

	return accounts, nil
}
