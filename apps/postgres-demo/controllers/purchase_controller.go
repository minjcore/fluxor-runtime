package controllers

import (
	"github.com/fluxorio/fluxor/apps/postgres-demo/models"
	"github.com/fluxorio/fluxor/apps/postgres-demo/services"
	"github.com/fluxorio/fluxor/pkg/web"
)

// PurchaseController handles purchase requests
type PurchaseController struct {
	purchaseService *services.PurchaseService
}

// NewPurchaseController creates a new purchase controller
func NewPurchaseController(purchaseService *services.PurchaseService) *PurchaseController {
	return &PurchaseController{
		purchaseService: purchaseService,
	}
}

// GetProducts returns all available products
func (c *PurchaseController) GetProducts(ctx *web.FastRequestContext) error {
	products, err := c.purchaseService.GetProducts(ctx.Context())
	if err != nil {
		return ctx.JSON(500, map[string]interface{}{
			"error": err.Error(),
		})
	}
	return ctx.JSON(200, products)
}

// MakePurchase processes a purchase request
func (c *PurchaseController) MakePurchase(ctx *web.FastRequestContext) error {
	var req models.PurchaseRequest
	if err := ctx.BindJSON(&req); err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_request",
			"message": "Invalid JSON",
		})
	}

	// ALWAYS use JWT token user_id, ignore request body user_id
	userID := GetUserIDFromContext(ctx)
	if userID == "" {
		userID = "demo_user" // Only fallback if JWT extraction fails
	}
	req.UserID = userID // Overwrite any user_id from request body

	result, err := c.purchaseService.MakePurchase(ctx.Context(), req)
	if err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "purchase_failed",
			"message": err.Error(),
		})
	}

	return ctx.JSON(200, result)
}

// GetOrders returns orders for a user
func (c *PurchaseController) GetOrders(ctx *web.FastRequestContext) error {
	// Always use JWT token user_id for consistency
	userID := GetUserIDFromContext(ctx)
	if userID == "" {
		userID = "demo_user" // Only fallback if JWT extraction fails
	}

	orders, err := c.purchaseService.GetOrders(ctx.Context(), userID)
	if err != nil {
		return ctx.JSON(500, map[string]interface{}{
			"error": err.Error(),
		})
	}
	return ctx.JSON(200, orders)
}
