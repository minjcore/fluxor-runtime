package models

import (
	"time"
)

// ============================================================================
// WALLET RULE TYPES
// ============================================================================

// WalletRuleType represents the type of wallet rule
type WalletRuleType string

const (
	// RuleTypeMinBalance enforces minimum balance requirement
	RuleTypeMinBalance WalletRuleType = "min_balance"
	// RuleTypeMaxBalance enforces maximum balance limit
	RuleTypeMaxBalance WalletRuleType = "max_balance"
	// RuleTypeDailyLimit enforces daily transaction/transfer limit
	RuleTypeDailyLimit WalletRuleType = "daily_limit"
	// RuleTypeTransferLimit enforces per-transaction transfer limit
	RuleTypeTransferLimit WalletRuleType = "transfer_limit"
	// RuleTypeTransactionLimit enforces per-transaction limit
	RuleTypeTransactionLimit WalletRuleType = "transaction_limit"
	// RuleTypeWithdrawalLimit enforces withdrawal limit
	RuleTypeWithdrawalLimit WalletRuleType = "withdrawal_limit"
	// RuleTypeDepositLimit enforces deposit limit
	RuleTypeDepositLimit WalletRuleType = "deposit_limit"
	// RuleTypeFreezeLimit enforces freeze amount limit
	RuleTypeFreezeLimit WalletRuleType = "freeze_limit"
	// RuleTypeCustom for custom business rules
	RuleTypeCustom WalletRuleType = "custom"
)

// String returns the string representation of WalletRuleType
func (rt WalletRuleType) String() string {
	return string(rt)
}

// IsValid checks if the rule type is valid
func (rt WalletRuleType) IsValid() bool {
	switch rt {
	case RuleTypeMinBalance, RuleTypeMaxBalance, RuleTypeDailyLimit,
		RuleTypeTransferLimit, RuleTypeTransactionLimit, RuleTypeWithdrawalLimit,
		RuleTypeDepositLimit, RuleTypeFreezeLimit, RuleTypeCustom:
		return true
	}
	return false
}

// ============================================================================
// OPERATION TYPES
// ============================================================================

// WalletOperationType represents the type of wallet operation
type WalletOperationType string

const (
	OperationTypeTransfer    WalletOperationType = "transfer"
	OperationTypeAddBalance  WalletOperationType = "add_balance"
	OperationTypeFreeze      WalletOperationType = "freeze"
	OperationTypePurchase    WalletOperationType = "purchase"
	OperationTypeWithdrawal  WalletOperationType = "withdrawal"
	OperationTypeDeposit     WalletOperationType = "deposit"
	OperationTypeAll         WalletOperationType = "all"
)

// String returns the string representation
func (ot WalletOperationType) String() string {
	return string(ot)
}

// ============================================================================
// PERIOD TYPES
// ============================================================================

// RulePeriodType represents the period for limit rules
type RulePeriodType string

const (
	PeriodPerTransaction RulePeriodType = "per_transaction"
	PeriodDaily         RulePeriodType = "daily"
	PeriodWeekly        RulePeriodType = "weekly"
	PeriodMonthly       RulePeriodType = "monthly"
	PeriodYearly        RulePeriodType = "yearly"
)

// String returns the string representation
func (pt RulePeriodType) String() string {
	return string(pt)
}

// ============================================================================
// WALLET RULE MODEL
// ============================================================================

// WalletRule represents a business rule for wallet operations
// Table: wallet_rules
type WalletRule struct {
	ID            int                `db:"id" json:"id"`
	RuleName      string             `db:"rule_name" json:"rule_name"`
	RuleType      WalletRuleType     `db:"rule_type" json:"rule_type"`
	WalletType    *string            `db:"wallet_type" json:"wallet_type,omitempty"`     // NULL = all wallet types
	UserID        *string            `db:"user_id" json:"user_id,omitempty"`             // NULL = all users
	OperationType *string            `db:"operation_type" json:"operation_type,omitempty"` // NULL = all operations
	MinValue      *float64           `db:"min_value" json:"min_value,omitempty"`
	MaxValue      *float64           `db:"max_value" json:"max_value,omitempty"`
	PeriodType    *string            `db:"period_type" json:"period_type,omitempty"`     // NULL, 'daily', 'weekly', 'monthly', 'per_transaction'
	IsActive      bool               `db:"is_active" json:"is_active"`
	Priority      int                `db:"priority" json:"priority"`                   // Lower = higher priority
	Description   string             `db:"description" json:"description"`
	ErrorMessage  *string            `db:"error_message" json:"error_message,omitempty"`
	Metadata      *string            `db:"metadata" json:"metadata,omitempty"`           // JSON metadata
	CreatedAt     time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `db:"updated_at" json:"updated_at"`
}

// ============================================================================
// RULE VALIDATION RESULT
// ============================================================================

// RuleValidationResult represents the result of rule validation
type RuleValidationResult struct {
	IsValid      bool     `json:"is_valid"`
	RuleName     string   `json:"rule_name,omitempty"`
	ErrorMessage string   `json:"error_message,omitempty"`
	RuleType     string   `json:"rule_type,omitempty"`
}

// ============================================================================
// RULE QUERY PARAMETERS
// ============================================================================

// RuleQueryParams represents parameters for querying applicable rules
type RuleQueryParams struct {
	WalletType    *WalletType
	UserID        *string
	OperationType *WalletOperationType
	RuleType      *WalletRuleType
	OnlyActive    bool
}
