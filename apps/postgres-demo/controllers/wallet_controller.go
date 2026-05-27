package controllers

import (
	"fmt"

	"github.com/fluxorio/fluxor/apps/postgres-demo/models"
	"github.com/fluxorio/fluxor/apps/postgres-demo/services"
	"github.com/fluxorio/fluxor/pkg/web"
)

// WalletController handles wallet-related requests
type WalletController struct {
	walletService *services.WalletService
}

// NewWalletController creates a new wallet controller
func NewWalletController(walletService *services.WalletService) *WalletController {
	return &WalletController{
		walletService: walletService,
	}
}

// GetBalance returns the wallet balance for the authenticated user
func (c *WalletController) GetBalance(ctx *web.FastRequestContext) error {
	// Get username from JWT token claims using helper
	userID := GetUserIDFromContext(ctx)
	if userID == "" {
		userID = "demo_user" // Fallback if JWT extraction fails
	}

	wallet, err := c.walletService.GetWalletDetails(ctx.Context(), userID)
	if err != nil {
		return ctx.JSON(500, map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Calculate available balance from wallet details (balance - frozen)
	availableBalance := wallet.Balance - wallet.Frozen

	return ctx.JSON(200, map[string]interface{}{
		"user_id":          userID,
		"balance":          wallet.Balance,
		"frozen":           wallet.Frozen,
		"available_balance": availableBalance,
	})
}

// GetTransactions returns wallet transactions for the authenticated user
func (c *WalletController) GetTransactions(ctx *web.FastRequestContext) error {
	// Get username from JWT token claims using helper
	userID := GetUserIDFromContext(ctx)
	if userID == "" {
		userID = "demo_user" // Fallback if JWT extraction fails
	}

	transactions, err := c.walletService.GetWalletTransactions(ctx.Context(), userID)
	if err != nil {
		return ctx.JSON(500, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return ctx.JSON(200, transactions)
}

// AddBalance adds money to the wallet (for testing/admin)
func (c *WalletController) AddBalance(ctx *web.FastRequestContext) error {
	var req struct {
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_request",
			"message": "Invalid JSON",
		})
	}

	// Get username from JWT token claims using helper
	userID := GetUserIDFromContext(ctx)
	if userID == "" {
		userID = "demo_user" // Fallback if JWT extraction fails
	}

	description := req.Description
	if description == "" {
		description = "Wallet top-up"
	}

	err := c.walletService.AddWalletBalance(ctx.Context(), userID, req.Amount, description)
	if err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "add_balance_failed",
			"message": err.Error(),
		})
	}

	// Get updated balance
	balance, err := c.walletService.GetWalletBalance(ctx.Context(), userID)
	if err != nil {
		return ctx.JSON(500, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return ctx.JSON(200, map[string]interface{}{
		"user_id": userID,
		"balance": balance,
		"message": "Balance added successfully",
	})
}

// Transfer transfers money between wallets
func (c *WalletController) Transfer(ctx *web.FastRequestContext) error {
	var req struct {
		FromWalletType string  `json:"from_wallet_type"`
		ToWalletType   string  `json:"to_wallet_type"`
		Amount         float64 `json:"amount"`
		Description    string  `json:"description"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_request",
			"message": "Invalid JSON",
		})
	}

	// Get username from JWT token claims using helper
	userID := GetUserIDFromContext(ctx)
	if userID == "" {
		userID = "demo_user" // Fallback if JWT extraction fails
	}

	// Validate wallet types
	fromWalletType := models.WalletType(req.FromWalletType)
	toWalletType := models.WalletType(req.ToWalletType)

	description := req.Description
	if description == "" {
		description = fmt.Sprintf("Transfer from %s to %s", fromWalletType, toWalletType)
	}

	// Perform transfer
	err := c.walletService.TransferBetweenWallets(ctx.Context(), userID, fromWalletType, toWalletType, req.Amount, description)
	if err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "transfer_failed",
			"message": err.Error(),
		})
	}

	// Get updated balances
	fromBalance, err := c.walletService.GetWalletBalanceByType(ctx.Context(), userID, fromWalletType)
	if err != nil {
		return ctx.JSON(500, map[string]interface{}{
			"error": err.Error(),
		})
	}

	toBalance, err := c.walletService.GetWalletBalanceByType(ctx.Context(), userID, toWalletType)
	if err != nil {
		return ctx.JSON(500, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return ctx.JSON(200, map[string]interface{}{
		"user_id":          userID,
		"from_wallet_type": fromWalletType,
		"to_wallet_type":   toWalletType,
		"amount":           req.Amount,
		"from_balance":     fromBalance,
		"to_balance":       toBalance,
		"message":          "Transfer completed successfully",
	})
}
