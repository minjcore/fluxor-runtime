package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// ============================================================================
// ACCOUNT TYPES & ENUMS
// ============================================================================

// AccountType represents the type of accounting account
type AccountType string

const (
	// AccountTypeAsset represents asset accounts (Tài sản)
	AccountTypeAsset AccountType = "asset"
	// AccountTypeLiability represents liability accounts (Nợ phải trả)
	AccountTypeLiability AccountType = "liability"
	// AccountTypeEquity represents equity accounts (Vốn chủ sở hữu)
	AccountTypeEquity AccountType = "equity"
	// AccountTypeRevenue represents revenue accounts (Doanh thu)
	AccountTypeRevenue AccountType = "revenue"
	// AccountTypeExpense represents expense accounts (Chi phí)
	AccountTypeExpense AccountType = "expense"
)

// String returns the string representation of AccountType
func (at AccountType) String() string {
	return string(at)
}

// IsValid checks if the account type is valid
func (at AccountType) IsValid() bool {
	switch at {
	case AccountTypeAsset, AccountTypeLiability, AccountTypeEquity, AccountTypeRevenue, AccountTypeExpense:
		return true
	}
	return false
}

// IsDebitNormal returns true if this account type normally has a debit balance
// Assets and Expenses are debit-normal, Liabilities, Equity, and Revenue are credit-normal
func (at AccountType) IsDebitNormal() bool {
	return at == AccountTypeAsset || at == AccountTypeExpense
}

// ============================================================================
// JOURNAL STATUS
// ============================================================================

// JournalStatus represents the status of a journal entry
type JournalStatus string

const (
	// JournalStatusPending indicates the journal is pending (not yet posted)
	JournalStatusPending JournalStatus = "pending"
	// JournalStatusPosted indicates the journal has been posted to the ledger
	JournalStatusPosted JournalStatus = "posted"
	// JournalStatusReversed indicates the journal has been reversed
	JournalStatusReversed JournalStatus = "reversed"
)

// String returns the string representation of JournalStatus
func (js JournalStatus) String() string {
	return string(js)
}

// IsValid checks if the journal status is valid
func (js JournalStatus) IsValid() bool {
	switch js {
	case JournalStatusPending, JournalStatusPosted, JournalStatusReversed:
		return true
	}
	return false
}

// NullableJournalStatus wraps JournalStatus to handle NULL values from database
type NullableJournalStatus struct {
	Status JournalStatus
	Valid  bool
}

// Scan implements the sql.Scanner interface
func (njs *NullableJournalStatus) Scan(value interface{}) error {
	if value == nil {
		njs.Status = ""
		njs.Valid = false
		return nil
	}

	var s sql.NullString
	if err := s.Scan(value); err != nil {
		return err
	}

	njs.Valid = s.Valid
	if s.Valid {
		njs.Status = JournalStatus(s.String)
	}
	return nil
}

// MarshalJSON implements json.Marshaler
func (njs NullableJournalStatus) MarshalJSON() ([]byte, error) {
	if !njs.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(njs.Status)
}

// ============================================================================
// REFERENCE TYPES (for journal entries)
// ============================================================================

// ReferenceType represents the type of transaction that created the journal
type ReferenceType string

const (
	RefTypeWalletTopup   ReferenceType = "wallet_topup"
	RefTypePurchase      ReferenceType = "purchase"
	RefTypeRefund        ReferenceType = "refund"
	RefTypeTransfer      ReferenceType = "transfer"
	RefTypeAdjustment    ReferenceType = "adjustment"
	RefTypeFee           ReferenceType = "fee"
	RefTypeFreeze        ReferenceType = "freeze"
	RefTypeUnfreeze      ReferenceType = "unfreeze"
)

// String returns the string representation of ReferenceType
func (rt ReferenceType) String() string {
	return string(rt)
}

// ============================================================================
// SYSTEM ACCOUNT CODES
// ============================================================================

const (
	// Asset accounts (1xxx)
	AccCodeUserWallets  = "1100" // User wallet balances
	AccCodeCompanyCash  = "1200" // Company cash/bank

	// Liability accounts (2xxx)
	AccCodeFrozenFunds  = "2100" // Frozen amounts (pending transactions)
	AccCodeUserPayables = "2200" // Amounts owed to users

	// Revenue accounts (3xxx)
	AccCodeSalesRevenue    = "3100" // Revenue from sales
	AccCodeTransactionFees = "3200" // Transaction fees

	// Expense accounts (4xxx)
	AccCodeRefunds     = "4100" // Refund expenses
	AccCodeOperational = "4200" // Operational expenses
)

// ============================================================================
// MODELS
// ============================================================================

// Account represents a ledger account (Tài khoản kế toán)
// Table: accounts
type Account struct {
	ID        int         `db:"id" json:"id"`
	Code      string      `db:"code" json:"code"`           // Unique account code (e.g., '1100')
	Name      string      `db:"name" json:"name"`           // Account name
	Type      AccountType `db:"type" json:"type"`           // Account type
	ParentID  *int        `db:"parent_id" json:"parent_id,omitempty"` // Parent account for hierarchy
	IsSystem  bool        `db:"is_system" json:"is_system"` // System accounts cannot be deleted
	CreatedAt time.Time   `db:"created_at" json:"created_at"`
}

// Journal represents a journal entry (Bút toán)
// A journal groups multiple ledger entries that must balance (debit = credit)
// Table: journals
type Journal struct {
	ID            int                   `db:"id" json:"id"`
	ReferenceType ReferenceType         `db:"reference_type" json:"reference_type"` // Type of transaction
	ReferenceID   *int                  `db:"reference_id" json:"reference_id,omitempty"` // ID of related entity
	Description   string                `db:"description" json:"description"`
	Status        NullableJournalStatus `db:"status" json:"status"`
	PostedAt      *time.Time            `db:"posted_at" json:"posted_at,omitempty"`
	CreatedAt     time.Time             `db:"created_at" json:"created_at"`
	CreatedBy     string                `db:"created_by" json:"created_by"`

	// Nested data (populated by service)
	Entries []LedgerEntry `db:"-" json:"entries,omitempty"`
}

// LedgerEntry represents a single debit or credit entry in the ledger
// Table: ledger_entries
type LedgerEntry struct {
	ID        int       `db:"id" json:"id"`
	JournalID int       `db:"journal_id" json:"journal_id"`
	AccountID int       `db:"account_id" json:"account_id"`
	Debit     float64   `db:"debit" json:"debit"`   // Debit amount (0 if credit)
	Credit    float64   `db:"credit" json:"credit"` // Credit amount (0 if debit)
	CreatedAt time.Time `db:"created_at" json:"created_at"`

	// Joined data (populated by service)
	Account *Account `db:"-" json:"account,omitempty"`
}

// ============================================================================
// HELPER TYPES
// ============================================================================

// JournalEntryInput is used to create a new ledger entry
type JournalEntryInput struct {
	AccountCode string  // Account code (e.g., '1100')
	Debit       float64 // Debit amount
	Credit      float64 // Credit amount
}

// AccountBalance represents the balance of an account
type AccountBalance struct {
	AccountID   int         `json:"account_id"`
	AccountCode string      `json:"account_code"`
	AccountName string      `json:"account_name"`
	AccountType AccountType `json:"account_type"`
	TotalDebit  float64     `json:"total_debit"`
	TotalCredit float64     `json:"total_credit"`
	Balance     float64     `json:"balance"` // Calculated based on account type
}

// TrialBalance represents a trial balance report
type TrialBalance struct {
	AsOf       time.Time        `json:"as_of"`
	Accounts   []AccountBalance `json:"accounts"`
	TotalDebit float64          `json:"total_debit"`
	TotalCredit float64         `json:"total_credit"`
	IsBalanced bool             `json:"is_balanced"`
}
