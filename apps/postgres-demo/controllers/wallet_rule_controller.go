package controllers

import (
	"fmt"

	"github.com/fluxorio/fluxor/apps/postgres-demo/models"
	"github.com/fluxorio/fluxor/apps/postgres-demo/services"
	"github.com/fluxorio/fluxor/pkg/web"
)

// WalletRuleController handles wallet rule management requests
type WalletRuleController struct {
	ruleService *services.WalletRuleService
}

// NewWalletRuleController creates a new wallet rule controller
func NewWalletRuleController(ruleService *services.WalletRuleService) *WalletRuleController {
	return &WalletRuleController{
		ruleService: ruleService,
	}
}

// GetAllRules returns all wallet rules
func (c *WalletRuleController) GetAllRules(ctx *web.FastRequestContext) error {
	includeInactive := ctx.Query("include_inactive") == "true"
	
	rules, err := c.ruleService.GetAllRules(ctx.Context(), includeInactive)
	if err != nil {
		return ctx.JSON(500, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return ctx.JSON(200, rules)
}

// CreateRule creates a new wallet rule
func (c *WalletRuleController) CreateRule(ctx *web.FastRequestContext) error {
	var rule struct {
		RuleName      string   `json:"rule_name"`
		RuleType      string   `json:"rule_type"`
		WalletType    *string  `json:"wallet_type,omitempty"`
		UserID        *string  `json:"user_id,omitempty"`
		OperationType *string  `json:"operation_type,omitempty"`
		MinValue      *float64 `json:"min_value,omitempty"`
		MaxValue      *float64 `json:"max_value,omitempty"`
		PeriodType    *string  `json:"period_type,omitempty"`
		IsActive      bool     `json:"is_active"`
		Priority      int      `json:"priority"`
		Description   string   `json:"description"`
		ErrorMessage  *string  `json:"error_message,omitempty"`
		Metadata      *string  `json:"metadata,omitempty"`
	}
	
	if err := ctx.BindJSON(&rule); err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_request",
			"message": "Invalid JSON",
		})
	}

	walletRule := models.WalletRule{
		RuleName:      rule.RuleName,
		RuleType:      models.WalletRuleType(rule.RuleType),
		WalletType:    rule.WalletType,
		UserID:        rule.UserID,
		OperationType: rule.OperationType,
		MinValue:      rule.MinValue,
		MaxValue:      rule.MaxValue,
		PeriodType:    rule.PeriodType,
		IsActive:      rule.IsActive,
		Priority:      rule.Priority,
		Description:   rule.Description,
		ErrorMessage:  rule.ErrorMessage,
		Metadata:      rule.Metadata,
	}

	ruleID, err := c.ruleService.CreateRule(ctx.Context(), walletRule)
	if err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "create_rule_failed",
			"message": err.Error(),
		})
	}

	return ctx.JSON(201, map[string]interface{}{
		"id":      ruleID,
		"message": "Rule created successfully",
	})
}

// UpdateRule updates an existing wallet rule
func (c *WalletRuleController) UpdateRule(ctx *web.FastRequestContext) error {
	ruleIDStr := ctx.Param("id")
	if ruleIDStr == "" {
		return ctx.JSON(400, map[string]interface{}{
			"error": "rule_id is required",
		})
	}

	var ruleID int
	if _, err := fmt.Sscanf(ruleIDStr, "%d", &ruleID); err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error": "invalid rule_id",
		})
	}

	var rule struct {
		RuleName      string   `json:"rule_name"`
		RuleType      string   `json:"rule_type"`
		WalletType    *string  `json:"wallet_type,omitempty"`
		UserID        *string  `json:"user_id,omitempty"`
		OperationType *string  `json:"operation_type,omitempty"`
		MinValue      *float64 `json:"min_value,omitempty"`
		MaxValue      *float64 `json:"max_value,omitempty"`
		PeriodType    *string  `json:"period_type,omitempty"`
		IsActive      bool     `json:"is_active"`
		Priority      int      `json:"priority"`
		Description   string   `json:"description"`
		ErrorMessage  *string  `json:"error_message,omitempty"`
		Metadata      *string  `json:"metadata,omitempty"`
	}
	
	if err := ctx.BindJSON(&rule); err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_request",
			"message": "Invalid JSON",
		})
	}

	walletRule := models.WalletRule{
		RuleName:      rule.RuleName,
		RuleType:      models.WalletRuleType(rule.RuleType),
		WalletType:    rule.WalletType,
		UserID:        rule.UserID,
		OperationType: rule.OperationType,
		MinValue:      rule.MinValue,
		MaxValue:      rule.MaxValue,
		PeriodType:    rule.PeriodType,
		IsActive:      rule.IsActive,
		Priority:      rule.Priority,
		Description:   rule.Description,
		ErrorMessage:  rule.ErrorMessage,
		Metadata:      rule.Metadata,
	}

	err := c.ruleService.UpdateRule(ctx.Context(), ruleID, walletRule)
	if err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "update_rule_failed",
			"message": err.Error(),
		})
	}

	return ctx.JSON(200, map[string]interface{}{
		"message": "Rule updated successfully",
	})
}

// DeleteRule deletes a wallet rule
func (c *WalletRuleController) DeleteRule(ctx *web.FastRequestContext) error {
	ruleIDStr := ctx.Param("id")
	if ruleIDStr == "" {
		return ctx.JSON(400, map[string]interface{}{
			"error": "rule_id is required",
		})
	}

	var ruleID int
	if _, err := fmt.Sscanf(ruleIDStr, "%d", &ruleID); err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error": "invalid rule_id",
		})
	}

	err := c.ruleService.DeleteRule(ctx.Context(), ruleID)
	if err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "delete_rule_failed",
			"message": err.Error(),
		})
	}

	return ctx.JSON(200, map[string]interface{}{
		"message": "Rule deleted successfully",
	})
}
