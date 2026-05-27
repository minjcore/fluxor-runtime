package controllers

import (
	"github.com/fluxorio/fluxor/apps/postgres-demo/views"
	"github.com/fluxorio/fluxor/pkg/web"
)

// DashboardController handles dashboard requests
type DashboardController struct{}

// NewDashboardController creates a new dashboard controller
func NewDashboardController() *DashboardController {
	return &DashboardController{}
}

// ShowDashboard displays the dashboard page
func (c *DashboardController) ShowDashboard(ctx *web.FastRequestContext) error {
	ctx.RequestCtx.SetContentType("text/html; charset=utf-8")
	ctx.RequestCtx.SetStatusCode(200)
	ctx.RequestCtx.WriteString(views.GetDashboardHTML())
	return nil
}
