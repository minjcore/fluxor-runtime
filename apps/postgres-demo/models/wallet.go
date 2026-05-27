package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// ============================================================================
// WALLET TYPE
// ============================================================================

// WalletType represents the type of wallet (used as code/key)
type WalletType string

const (
	// WalletTypePrimary is the default/main wallet for a user
	WalletTypePrimary WalletType = "primary"
	// WalletTypeSavings is a savings wallet (typically with interest or restrictions)
	WalletTypeSavings WalletType = "savings"
	// WalletTypeInvestment is for investment purposes
	WalletTypeInvestment WalletType = "investment"
	// WalletTypeBusiness is for business transactions
	WalletTypeBusiness WalletType = "business"
	// WalletTypeEscrow is for holding funds in escrow
	WalletTypeEscrow WalletType = "escrow"
)

// WalletTypeInfo represents wallet type information from database
// Table: wallet_types
type WalletTypeInfo struct {
	Code        string  `db:"code" json:"code"`               // Primary key, e.g., 'primary', 'savings'
	Name        string  `db:"name" json:"name"`               // Display name
	Description string  `db:"description" json:"description"` // Description
	Icon        *string `db:"icon" json:"icon,omitempty"`     // Icon name/URL (optional)
	IsActive    bool    `db:"is_active" json:"is_active"`     // Whether this wallet type is enabled
	IsDefault   bool    `db:"is_default" json:"is_default"`   // Whether this is the default wallet type
	MinBalance  *float64 `db:"min_balance" json:"min_balance,omitempty"` // Minimum balance required (optional)
	MaxBalance  *float64 `db:"max_balance" json:"max_balance,omitempty"` // Maximum balance allowed (optional)
	Metadata    *string  `db:"metadata" json:"metadata,omitempty"`       // JSON metadata (optional)
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// String returns the string representation of WalletType
func (wt WalletType) String() string {
	return string(wt)
}

// IsValid checks if the wallet type is valid
func (wt WalletType) IsValid() bool {
	switch wt {
	case WalletTypePrimary, WalletTypeSavings, WalletTypeInvestment, WalletTypeBusiness, WalletTypeEscrow:
		return true
	}
	return false
}

// NullableWalletType wraps WalletType to handle NULL values from database
type NullableWalletType struct {
	Type  WalletType
	Valid bool
}

// Scan implements the sql.Scanner interface
func (nwt *NullableWalletType) Scan(value interface{}) error {
	if value == nil {
		nwt.Type = ""
		nwt.Valid = false
		return nil
	}

	var s sql.NullString
	if err := s.Scan(value); err != nil {
		return err
	}

	nwt.Valid = s.Valid
	if s.Valid {
		nwt.Type = WalletType(s.String)
	}
	return nil
}

// MarshalJSON implements json.Marshaler
func (nwt NullableWalletType) MarshalJSON() ([]byte, error) {
	if !nwt.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(nwt.Type)
}

// ============================================================================
// TRANSACTION STATUS
// ============================================================================

// TransactionStatus represents the status of a wallet transaction
type TransactionStatus string

const (
	// StatusPending indicates the transaction is pending (amount is frozen)
	StatusPending TransactionStatus = "pending"
	// StatusCompleted indicates the transaction has been completed
	StatusCompleted TransactionStatus = "completed"
	// StatusCancelled indicates the transaction was cancelled/rolled back
	StatusCancelled TransactionStatus = "cancelled"
)

// String returns the string representation of the status
func (s TransactionStatus) String() string {
	return string(s)
}

// IsValid checks if the status is a valid TransactionStatus
func (s TransactionStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusCompleted, StatusCancelled:
		return true
	}
	return false
}

// NullableStatus wraps TransactionStatus to handle NULL values from database
type NullableStatus struct {
	Status TransactionStatus
	Valid  bool // Valid is true if Status is not NULL
}

// Scan implements the sql.Scanner interface for NullableStatus
func (ns *NullableStatus) Scan(value interface{}) error {
	if value == nil {
		ns.Status = ""
		ns.Valid = false
		return nil
	}

	var s sql.NullString
	if err := s.Scan(value); err != nil {
		return err
	}

	ns.Valid = s.Valid
	if s.Valid {
		ns.Status = TransactionStatus(s.String)
	}
	return nil
}

// MarshalJSON implements json.Marshaler for NullableStatus
func (ns NullableStatus) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return json.Marshal(nil)
	}
	return json.Marshal(ns.Status)
}

// Wallet represents a user's wallet
// Table: wallets
// Columns: user_id (VARCHAR(255)), wallet_type (VARCHAR(20)), balance (DECIMAL(10,2)), frozen (DECIMAL(10,2)), created_at (TIMESTAMP), updated_at (TIMESTAMP)
// Primary key: (user_id, wallet_type) - allows multiple wallets per user
type Wallet struct {
	UserID     string     `db:"user_id" json:"user_id"`       // User identifier (part of composite PK)
	WalletType WalletType `db:"wallet_type" json:"wallet_type"` // Wallet type (part of composite PK)
	Balance    float64    `db:"balance" json:"balance"`       // Current wallet balance
	Frozen     float64    `db:"frozen" json:"frozen"`         // Frozen/pending amount (not available for use)
}

// WalletTransaction represents a wallet transaction
// Table: wallet_transactions
// Columns: id (SERIAL PRIMARY KEY), user_id (VARCHAR(255)), wallet_type (VARCHAR(20)), type (VARCHAR(20)), amount (DECIMAL(10,2)), description (TEXT), order_id (INTEGER FK, nullable), status (VARCHAR(20)), created_at (TIMESTAMP)
type WalletTransaction struct {
	ID          int            `db:"id" json:"id"`                       // Primary key, auto-increment
	UserID      string         `db:"user_id" json:"user_id"`             // User who owns the wallet
	WalletType  WalletType     `db:"wallet_type" json:"wallet_type"`     // Wallet type (for multi-wallet support)
	Type        string         `db:"type" json:"type"`                   // Transaction type: "debit" (trừ tiền) or "credit" (nạp tiền)
	Amount      float64        `db:"amount" json:"amount"`               // Transaction amount
	Description string         `db:"description" json:"description"`     // Transaction description
	OrderID     *int           `db:"order_id" json:"order_id,omitempty"` // Optional: link to order if transaction is from purchase
	Status      NullableStatus `db:"status" json:"status"`               // Transaction status: pending, completed, or cancelled
	CreatedAt   time.Time      `db:"created_at" json:"created_at"`       // Transaction timestamp
}
