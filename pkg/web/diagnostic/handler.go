package diagnostic

import (
	"fmt"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
)

// Handler provides diagnostic HTTP handlers
type Handler struct {
	gocmd core.GoCMD
}

// NewHandler creates a new diagnostic handler
func NewHandler(gocmd core.GoCMD) *Handler {
	return &Handler{
		gocmd: gocmd,
	}
}

// DeploymentHandler handles GET /api/diagnostic/deployment/:id
func (h *Handler) DeploymentHandler(ctx *web.FastRequestContext) error {
	deploymentID := ctx.Param("id")
	if deploymentID == "" {
		return ctx.JSON(400, map[string]interface{}{
			"error": "deployment ID is required",
		})
	}

	diagnostic, err := h.gocmd.GetDeploymentDiagnostic(deploymentID)
	if err != nil {
		if err, ok := err.(*core.EventBusError); ok && err.Code == "DEPLOYMENT_NOT_FOUND" {
			return ctx.JSON(404, map[string]interface{}{
				"error": err.Message,
			})
		}
		return ctx.JSON(500, map[string]interface{}{
			"error": fmt.Sprintf("failed to get deployment diagnostic: %v", err),
		})
	}

	return ctx.JSON(200, diagnostic)
}

// SystemHandler handles GET /api/diagnostic/system
func (h *Handler) SystemHandler(ctx *web.FastRequestContext) error {
	diagnostic := h.gocmd.GetSystemDiagnostic()
	return ctx.JSON(200, diagnostic)
}

// AllDeploymentsHandler handles GET /api/diagnostic/deployments
func (h *Handler) AllDeploymentsHandler(ctx *web.FastRequestContext) error {
	diagnostics := h.gocmd.GetAllDeploymentDiagnostics()
	return ctx.JSON(200, map[string]interface{}{
		"deployments": diagnostics,
		"count":       len(diagnostics),
	})
}
