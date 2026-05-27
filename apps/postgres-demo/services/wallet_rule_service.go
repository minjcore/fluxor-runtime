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

// WalletRuleService handles wallet rule validation and management
type WalletRuleService struct {
	db *dbruntime.DB
}

// NewWalletRuleService creates a new wallet rule service
func NewWalletRuleService(db *dbruntime.DB) *WalletRuleService {
	return &WalletRuleService{
		db: db,
	}
}

// ============================================================================
// RULE QUERIES
// ============================================================================

// GetApplicableRules retrieves rules that apply to a specific operation
func (s *WalletRuleService) GetApplicableRules(ctx context.Context, params models.RuleQueryParams) ([]models.WalletRule, error) {
	var whereClauses []string
	var args []interface{}
	argIndex := 1

	// Build WHERE clause based on parameters
	whereClauses = append(whereClauses, fmt.Sprintf("is_active = $%d", argIndex))
	args = append(args, params.OnlyActive)
	argIndex++

	if params.WalletType != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("(wallet_type = $%d OR wallet_type IS NULL)", argIndex))
		args = append(args, string(*params.WalletType))
		argIndex++
	}

	if params.UserID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("(user_id = $%d OR user_id IS NULL)", argIndex))
		args = append(args, *params.UserID)
		argIndex++
	}

	if params.OperationType != nil {
		opTypeStr := string(*params.OperationType)
		whereClauses = append(whereClauses, fmt.Sprintf("(operation_type = $%d OR operation_type IS NULL OR operation_type = 'all')", argIndex))
		args = append(args, opTypeStr)
		argIndex++
	}

	if params.RuleType != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("rule_type = $%d", argIndex))
		args = append(args, string(*params.RuleType))
		argIndex++
	}

	whereClause := "WHERE " + strings.Join(whereClauses, " AND ")
	query := fmt.Sprintf(`SELECT id, rule_name, rule_type, wallet_type, user_id, operation_type, 
		min_value, max_value, period_type, is_active, priority, description, error_message, metadata, 
		created_at, updated_at
		FROM wallet_rules
		%s
		ORDER BY priority ASC, id ASC`, whereClause)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query rules: %w", err)
	}
	defer rows.Close()

	var rules []models.WalletRule
	for rows.Next() {
		var rule models.WalletRule
		var walletType, userID, operationType, periodType, errorMessage, metadata sql.NullString
		var minValue, maxValue sql.NullFloat64

		err := rows.Scan(
			&rule.ID, &rule.RuleName, &rule.RuleType,
			&walletType, &userID, &operationType,
			&minValue, &maxValue, &periodType,
			&rule.IsActive, &rule.Priority, &rule.Description,
			&errorMessage, &metadata,
			&rule.CreatedAt, &rule.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}

		if walletType.Valid {
			rule.WalletType = &walletType.String
		}
		if userID.Valid {
			rule.UserID = &userID.String
		}
		if operationType.Valid {
			rule.OperationType = &operationType.String
		}
		if periodType.Valid {
			rule.PeriodType = &periodType.String
		}
		if errorMessage.Valid {
			rule.ErrorMessage = &errorMessage.String
		}
		if metadata.Valid {
			rule.Metadata = &metadata.String
		}
		if minValue.Valid {
			rule.MinValue = &minValue.Float64
		}
		if maxValue.Valid {
			rule.MaxValue = &maxValue.Float64
		}

		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rules: %w", err)
	}

	return rules, nil
}

// ============================================================================
// RULE VALIDATION
// ============================================================================

// ValidateTransfer validates a transfer operation against applicable rules
func (s *WalletRuleService) ValidateTransfer(ctx context.Context, userID string, fromWalletType, toWalletType models.WalletType, amount float64) (*models.RuleValidationResult, error) {
	// Get applicable rules for transfer operation
	params := models.RuleQueryParams{
		WalletType:    &fromWalletType,
		UserID:        &userID,
		OperationType: func() *models.WalletOperationType { op := models.OperationTypeTransfer; return &op }(),
		OnlyActive:    true,
	}

	rules, err := s.GetApplicableRules(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	// Check transfer_limit rules
	for _, rule := range rules {
		if rule.RuleType == models.RuleTypeTransferLimit {
			if rule.MinValue != nil && amount < *rule.MinValue {
				errorMsg := fmt.Sprintf("Transfer amount $%.2f is below minimum $%.2f", amount, *rule.MinValue)
				if rule.ErrorMessage != nil {
					errorMsg = *rule.ErrorMessage
				}
				return &models.RuleValidationResult{
					IsValid:      false,
					RuleName:     rule.RuleName,
					ErrorMessage: errorMsg,
					RuleType:     string(rule.RuleType),
				}, nil
			}
			if rule.MaxValue != nil && amount > *rule.MaxValue {
				errorMsg := fmt.Sprintf("Transfer amount $%.2f exceeds maximum $%.2f", amount, *rule.MaxValue)
				if rule.ErrorMessage != nil {
					errorMsg = *rule.ErrorMessage
				}
				return &models.RuleValidationResult{
					IsValid:      false,
					RuleName:     rule.RuleName,
					ErrorMessage: errorMsg,
					RuleType:     string(rule.RuleType),
				}, nil
			}
		}
	}

	// Check daily_limit rules (if period_type is daily)
	for _, rule := range rules {
		if rule.RuleType == models.RuleTypeDailyLimit && rule.PeriodType != nil && *rule.PeriodType == "daily" {
			// Calculate daily transfer amount for this user and wallet type
			today := time.Now().Format("2006-01-02")
			query := `SELECT COALESCE(SUM(amount), 0) 
				FROM wallet_transactions 
				WHERE user_id = $1 AND wallet_type = $2 
				AND type = 'debit' 
				AND status = 'completed'
				AND DATE(created_at) = $3`
			
			var dailyTotal float64
			err := s.db.QueryRowContext(ctx, query, userID, string(fromWalletType), today).Scan(&dailyTotal)
			if err != nil && err != sql.ErrNoRows {
				return nil, fmt.Errorf("failed to check daily limit: %w", err)
			}

			if rule.MaxValue != nil && (dailyTotal+amount) > *rule.MaxValue {
				errorMsg := fmt.Sprintf("Daily transfer limit exceeded: current $%.2f, adding $%.2f, limit $%.2f", dailyTotal, amount, *rule.MaxValue)
				if rule.ErrorMessage != nil {
					errorMsg = *rule.ErrorMessage
				}
				return &models.RuleValidationResult{
					IsValid:      false,
					RuleName:     rule.RuleName,
					ErrorMessage: errorMsg,
					RuleType:     string(rule.RuleType),
				}, nil
			}
		}
	}

	return &models.RuleValidationResult{IsValid: true}, nil
}

// ValidateBalance validates balance against min/max balance rules
func (s *WalletRuleService) ValidateBalance(ctx context.Context, userID string, walletType models.WalletType, currentBalance, newBalance float64) (*models.RuleValidationResult, error) {
	params := models.RuleQueryParams{
		WalletType: &walletType,
		UserID:     &userID,
		OnlyActive: true,
	}

	rules, err := s.GetApplicableRules(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	// Check min_balance rules
	for _, rule := range rules {
		if rule.RuleType == models.RuleTypeMinBalance && rule.MinValue != nil {
			if newBalance < *rule.MinValue {
				errorMsg := fmt.Sprintf("Balance $%.2f is below minimum $%.2f", newBalance, *rule.MinValue)
				if rule.ErrorMessage != nil {
					errorMsg = *rule.ErrorMessage
				}
				return &models.RuleValidationResult{
					IsValid:      false,
					RuleName:     rule.RuleName,
					ErrorMessage: errorMsg,
					RuleType:     string(rule.RuleType),
				}, nil
			}
		}
	}

	// Check max_balance rules
	for _, rule := range rules {
		if rule.RuleType == models.RuleTypeMaxBalance && rule.MaxValue != nil {
			if newBalance > *rule.MaxValue {
				errorMsg := fmt.Sprintf("Balance $%.2f exceeds maximum $%.2f", newBalance, *rule.MaxValue)
				if rule.ErrorMessage != nil {
					errorMsg = *rule.ErrorMessage
				}
				return &models.RuleValidationResult{
					IsValid:      false,
					RuleName:     rule.RuleName,
					ErrorMessage: errorMsg,
					RuleType:     string(rule.RuleType),
				}, nil
			}
		}
	}

	return &models.RuleValidationResult{IsValid: true}, nil
}

// ValidateAddBalance validates adding balance against applicable rules
func (s *WalletRuleService) ValidateAddBalance(ctx context.Context, userID string, walletType models.WalletType, amount float64) (*models.RuleValidationResult, error) {
	params := models.RuleQueryParams{
		WalletType:    &walletType,
		UserID:        &userID,
		OperationType: func() *models.WalletOperationType { op := models.OperationTypeAddBalance; return &op }(),
		OnlyActive:    true,
	}

	rules, err := s.GetApplicableRules(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	// Get current balance
	query := `SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1 AND wallet_type = $2`
	var currentBalance float64
	err = s.db.QueryRowContext(ctx, query, userID, string(walletType)).Scan(&currentBalance)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get current balance: %w", err)
	}

	newBalance := currentBalance + amount

	// Check deposit_limit rules
	for _, rule := range rules {
		if rule.RuleType == models.RuleTypeDepositLimit {
			if rule.MaxValue != nil && amount > *rule.MaxValue {
				errorMsg := fmt.Sprintf("Deposit amount $%.2f exceeds maximum $%.2f", amount, *rule.MaxValue)
				if rule.ErrorMessage != nil {
					errorMsg = *rule.ErrorMessage
				}
				return &models.RuleValidationResult{
					IsValid:      false,
					RuleName:     rule.RuleName,
					ErrorMessage: errorMsg,
					RuleType:     string(rule.RuleType),
				}, nil
			}
		}
	}

	// Validate new balance against min/max balance rules
	return s.ValidateBalance(ctx, userID, walletType, currentBalance, newBalance)
}

// ValidateFreeze validates freeze operation against applicable rules
func (s *WalletRuleService) ValidateFreeze(ctx context.Context, userID string, walletType models.WalletType, amount float64) (*models.RuleValidationResult, error) {
	params := models.RuleQueryParams{
		WalletType:    &walletType,
		UserID:        &userID,
		OperationType: func() *models.WalletOperationType { op := models.OperationTypeFreeze; return &op }(),
		OnlyActive:    true,
	}

	rules, err := s.GetApplicableRules(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	// Check freeze_limit rules
	for _, rule := range rules {
		if rule.RuleType == models.RuleTypeFreezeLimit {
			if rule.MaxValue != nil && amount > *rule.MaxValue {
				errorMsg := fmt.Sprintf("Freeze amount $%.2f exceeds maximum $%.2f", amount, *rule.MaxValue)
				if rule.ErrorMessage != nil {
					errorMsg = *rule.ErrorMessage
				}
				return &models.RuleValidationResult{
					IsValid:      false,
					RuleName:     rule.RuleName,
					ErrorMessage: errorMsg,
					RuleType:     string(rule.RuleType),
				}, nil
			}
		}
	}

	return &models.RuleValidationResult{IsValid: true}, nil
}

// ============================================================================
// RULE MANAGEMENT
// ============================================================================

// CreateRule creates a new wallet rule
func (s *WalletRuleService) CreateRule(ctx context.Context, rule models.WalletRule) (int, error) {
	query := `INSERT INTO wallet_rules (rule_name, rule_type, wallet_type, user_id, operation_type, 
		min_value, max_value, period_type, is_active, priority, description, error_message, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id`
	
	var ruleID int
	err := s.db.QueryRowContext(ctx, query,
		rule.RuleName, rule.RuleType,
		rule.WalletType, rule.UserID, rule.OperationType,
		rule.MinValue, rule.MaxValue, rule.PeriodType,
		rule.IsActive, rule.Priority, rule.Description,
		rule.ErrorMessage, rule.Metadata).Scan(&ruleID)
	if err != nil {
		return 0, fmt.Errorf("failed to create rule: %w", err)
	}

	return ruleID, nil
}

// UpdateRule updates an existing wallet rule
func (s *WalletRuleService) UpdateRule(ctx context.Context, ruleID int, rule models.WalletRule) error {
	query := `UPDATE wallet_rules 
		SET rule_name = $2, rule_type = $3, wallet_type = $4, user_id = $5, operation_type = $6,
			min_value = $7, max_value = $8, period_type = $9, is_active = $10, priority = $11,
			description = $12, error_message = $13, metadata = $14, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`
	
	_, err := s.db.ExecContext(ctx, query,
		ruleID, rule.RuleName, rule.RuleType,
		rule.WalletType, rule.UserID, rule.OperationType,
		rule.MinValue, rule.MaxValue, rule.PeriodType,
		rule.IsActive, rule.Priority, rule.Description,
		rule.ErrorMessage, rule.Metadata)
	if err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	return nil
}

// DeleteRule deletes a wallet rule
func (s *WalletRuleService) DeleteRule(ctx context.Context, ruleID int) error {
	query := `DELETE FROM wallet_rules WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, ruleID)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}
	return nil
}

// GetAllRules retrieves all wallet rules
func (s *WalletRuleService) GetAllRules(ctx context.Context, includeInactive bool) ([]models.WalletRule, error) {
	var query string
	if includeInactive {
		query = `SELECT id, rule_name, rule_type, wallet_type, user_id, operation_type, 
			min_value, max_value, period_type, is_active, priority, description, error_message, metadata, 
			created_at, updated_at
			FROM wallet_rules
			ORDER BY priority ASC, id ASC`
	} else {
		query = `SELECT id, rule_name, rule_type, wallet_type, user_id, operation_type, 
			min_value, max_value, period_type, is_active, priority, description, error_message, metadata, 
			created_at, updated_at
			FROM wallet_rules
			WHERE is_active = true
			ORDER BY priority ASC, id ASC`
	}

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rules: %w", err)
	}
	defer rows.Close()

	var rules []models.WalletRule
	for rows.Next() {
		var rule models.WalletRule
		var walletType, userID, operationType, periodType, errorMessage, metadata sql.NullString
		var minValue, maxValue sql.NullFloat64

		err := rows.Scan(
			&rule.ID, &rule.RuleName, &rule.RuleType,
			&walletType, &userID, &operationType,
			&minValue, &maxValue, &periodType,
			&rule.IsActive, &rule.Priority, &rule.Description,
			&errorMessage, &metadata,
			&rule.CreatedAt, &rule.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}

		if walletType.Valid {
			rule.WalletType = &walletType.String
		}
		if userID.Valid {
			rule.UserID = &userID.String
		}
		if operationType.Valid {
			rule.OperationType = &operationType.String
		}
		if periodType.Valid {
			rule.PeriodType = &periodType.String
		}
		if errorMessage.Valid {
			rule.ErrorMessage = &errorMessage.String
		}
		if metadata.Valid {
			rule.Metadata = &metadata.String
		}
		if minValue.Valid {
			rule.MinValue = &minValue.Float64
		}
		if maxValue.Valid {
			rule.MaxValue = &maxValue.Float64
		}

		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rules: %w", err)
	}

	return rules, nil
}
