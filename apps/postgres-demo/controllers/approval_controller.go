package controllers

import (
	"fmt"

	"github.com/fluxorio/fluxor/apps/postgres-demo/models"
	"github.com/fluxorio/fluxor/apps/postgres-demo/services"
	"github.com/fluxorio/fluxor/pkg/web"
)

// ApprovalController handles approval workflow requests
type ApprovalController struct {
	approvalService *services.ApprovalService
}

// NewApprovalController creates a new approval controller
func NewApprovalController(approvalService *services.ApprovalService) *ApprovalController {
	return &ApprovalController{
		approvalService: approvalService,
	}
}

// CreateApproval creates a new approval request
func (c *ApprovalController) CreateApproval(ctx *web.FastRequestContext) error {
	var req models.CreateApprovalRequest
	if err := ctx.BindJSON(&req); err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_request",
			"message": "Invalid JSON",
		})
	}

	// Get user from JWT token
	userID := c.getUserID(ctx)
	if userID == nil {
		return ctx.JSON(401, map[string]interface{}{
			"error": "unauthorized",
		})
	}

	approvalID, err := c.approvalService.CreateApproval(ctx.Context(), req, *userID)
	if err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "create_approval_failed",
			"message": err.Error(),
		})
	}

	return ctx.JSON(201, map[string]interface{}{
		"id":      approvalID,
		"message": "Approval request created. Requires two approvals to be executed.",
	})
}

// GetApproval retrieves an approval by ID
func (c *ApprovalController) GetApproval(ctx *web.FastRequestContext) error {
	approvalIDStr := ctx.Param("id")
	if approvalIDStr == "" {
		return ctx.JSON(400, map[string]interface{}{
			"error": "approval_id is required",
		})
	}

	var approvalID int
	if _, err := fmt.Sscanf(approvalIDStr, "%d", &approvalID); err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error": "invalid approval_id",
		})
	}

	approval, err := c.approvalService.GetApproval(ctx.Context(), approvalID)
	if err != nil {
		return ctx.JSON(404, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return ctx.JSON(200, approval)
}

// GetPendingApprovals retrieves all pending approvals
func (c *ApprovalController) GetPendingApprovals(ctx *web.FastRequestContext) error {
	approvalTypeStr := ctx.Query("type")
	
	var approvalType *models.ApprovalType
	if approvalTypeStr != "" {
		at := models.ApprovalType(approvalTypeStr)
		approvalType = &at
	}

	approvals, err := c.approvalService.GetPendingApprovals(ctx.Context(), approvalType)
	if err != nil {
		return ctx.JSON(500, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return ctx.JSON(200, approvals)
}

// Approve approves an approval request
func (c *ApprovalController) Approve(ctx *web.FastRequestContext) error {
	approvalIDStr := ctx.Param("id")
	if approvalIDStr == "" {
		return ctx.JSON(400, map[string]interface{}{
			"error": "approval_id is required",
		})
	}

	var approvalID int
	if _, err := fmt.Sscanf(approvalIDStr, "%d", &approvalID); err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error": "invalid approval_id",
		})
	}

	// Get user from JWT token
	userID := c.getUserID(ctx)
	if userID == nil {
		return ctx.JSON(401, map[string]interface{}{
			"error": "unauthorized",
		})
	}

	err := c.approvalService.Approve(ctx.Context(), approvalID, *userID)
	if err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "approve_failed",
			"message": err.Error(),
		})
	}

	return ctx.JSON(200, map[string]interface{}{
		"message": "Approval recorded. Two approvals required for execution.",
	})
}

// Reject rejects an approval request
func (c *ApprovalController) Reject(ctx *web.FastRequestContext) error {
	approvalIDStr := ctx.Param("id")
	if approvalIDStr == "" {
		return ctx.JSON(400, map[string]interface{}{
			"error": "approval_id is required",
		})
	}

	var approvalID int
	if _, err := fmt.Sscanf(approvalIDStr, "%d", &approvalID); err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error": "invalid approval_id",
		})
	}

	var req models.ApproveRequest
	if err := ctx.BindJSON(&req); err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_request",
			"message": "Invalid JSON",
		})
	}

	if req.Action != "reject" {
		return ctx.JSON(400, map[string]interface{}{
			"error": "action must be 'reject'",
		})
	}

	// Get user from JWT token
	userID := c.getUserID(ctx)
	if userID == nil {
		return ctx.JSON(401, map[string]interface{}{
			"error": "unauthorized",
		})
	}

	reason := ""
	if req.RejectionReason != nil {
		reason = *req.RejectionReason
	}

	err := c.approvalService.Reject(ctx.Context(), approvalID, *userID, reason)
	if err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "reject_failed",
			"message": err.Error(),
		})
	}

	return ctx.JSON(200, map[string]interface{}{
		"message": "Approval rejected",
	})
}

// getUserID extracts user ID from JWT token
func (c *ApprovalController) getUserID(ctx *web.FastRequestContext) *string {
	// Try to get user from JWT token
	// This is a simplified version - in production, you'd extract from JWT claims
	userID := ctx.Get("user_id")
	if userID != nil {
		if str, ok := userID.(string); ok {
			return &str
		}
	}
	return nil
}
