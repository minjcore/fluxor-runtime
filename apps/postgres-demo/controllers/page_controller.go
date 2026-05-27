package controllers

import (
	"github.com/fluxorio/fluxor/apps/postgres-demo/views"
	"github.com/fluxorio/fluxor/pkg/web"
)

// PageController handles page requests
type PageController struct{}

// NewPageController creates a new page controller
func NewPageController() *PageController {
	return &PageController{}
}

// ShowLogin displays the login page
func (c *PageController) ShowLogin(ctx *web.FastRequestContext) error {
	ctx.RequestCtx.SetContentType("text/html; charset=utf-8")
	ctx.RequestCtx.SetStatusCode(200)
	ctx.RequestCtx.WriteString(views.GetLoginHTML())
	return nil
}

// ShowRegister displays the registration page
func (c *PageController) ShowRegister(ctx *web.FastRequestContext) error {
	ctx.RequestCtx.SetContentType("text/html; charset=utf-8")
	ctx.RequestCtx.SetStatusCode(200)
	ctx.RequestCtx.WriteString(views.GetRegisterHTML())
	return nil
}
