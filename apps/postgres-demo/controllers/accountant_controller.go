package controllers

import (
	"fmt"
	"time"

	"github.com/fluxorio/fluxor/apps/postgres-demo/services"
	"github.com/fluxorio/fluxor/apps/postgres-demo/views"
	"github.com/fluxorio/fluxor/pkg/web"
)

// AccountantController handles accountant dashboard requests
type AccountantController struct {
	ledgerService *services.LedgerService
}

// NewAccountantController creates a new accountant controller
func NewAccountantController(ledgerService *services.LedgerService) *AccountantController {
	return &AccountantController{
		ledgerService: ledgerService,
	}
}

// ShowAccountantDashboard displays the accountant dashboard page
func (c *AccountantController) ShowAccountantDashboard(ctx *web.FastRequestContext) error {
	ctx.RequestCtx.SetContentType("text/html; charset=utf-8")
	ctx.RequestCtx.SetStatusCode(200)
	ctx.RequestCtx.WriteString(views.GetAccountantDashboardHTML())
	return nil
}

// GetTrialBalance returns the trial balance
func (c *AccountantController) GetTrialBalance(ctx *web.FastRequestContext) error {
	// Parse date range from query params (optional)
	endDate := ctx.Query("end_date")
	
	var endTime *time.Time
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			// Set to end of day
			t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			endTime = &t
		}
	}

	trialBalance, err := c.ledgerService.GetTrialBalance(ctx.Context(), endTime)
	if err != nil {
		return ctx.JSON(500, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return ctx.JSON(200, trialBalance)
}

// GetAccountBalances returns balances for all accounts
func (c *AccountantController) GetAccountBalances(ctx *web.FastRequestContext) error {
	// Parse date range from query params (optional)
	endDate := ctx.Query("end_date")
	
	var endTime *time.Time
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			endTime = &t
		}
	}

	balances, err := c.ledgerService.GetAllAccountBalances(ctx.Context(), endTime)
	if err != nil {
		return ctx.JSON(500, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return ctx.JSON(200, balances)
}

// GetJournals returns journals with optional filters
func (c *AccountantController) GetJournals(ctx *web.FastRequestContext) error {
	// Parse query params
	startDate := ctx.Query("start_date")
	endDate := ctx.Query("end_date")
	status := ctx.Query("status")
	referenceType := ctx.Query("reference_type")
	limitStr := ctx.Query("limit")
	
	var startTime, endTime *time.Time
	if startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			startTime = &t
		}
	}
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			endTime = &t
		}
	}

	limit := 100 // default
	if limitStr != "" {
		if l, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || l != 1 {
			limit = 100
		}
	}

	journals, err := c.ledgerService.GetJournals(ctx.Context(), startTime, endTime, status, referenceType, limit)
	if err != nil {
		return ctx.JSON(500, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return ctx.JSON(200, journals)
}

// GetJournalWithEntries returns a specific journal with its entries
func (c *AccountantController) GetJournalWithEntries(ctx *web.FastRequestContext) error {
	journalIDStr := ctx.Param("id")
	if journalIDStr == "" {
		return ctx.JSON(400, map[string]interface{}{
			"error": "journal_id is required",
		})
	}

	var journalID int
	if _, err := fmt.Sscanf(journalIDStr, "%d", &journalID); err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error": "invalid journal_id",
		})
	}

	journal, entries, err := c.ledgerService.GetJournalWithEntries(ctx.Context(), journalID)
	if err != nil {
		return ctx.JSON(500, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return ctx.JSON(200, map[string]interface{}{
		"journal": journal,
		"entries": entries,
	})
}

// GetAccounts returns all accounts
func (c *AccountantController) GetAccounts(ctx *web.FastRequestContext) error {
	accounts, err := c.ledgerService.GetAllAccounts(ctx.Context())
	if err != nil {
		return ctx.JSON(500, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return ctx.JSON(200, accounts)
}
