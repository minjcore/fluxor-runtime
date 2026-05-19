package admin

import (
	"embed"
	"strings"

	"github.com/fluxorio/fluxor/pkg/web"
)

//go:embed admin.html admin.js admin.css
var adminFS embed.FS

// Handler serves the admin UI shell (sidebar, header, placeholder pages).
type Handler struct {
	// BasePath is the path prefix for admin (e.g. "/admin"). Used for redirects and links.
	BasePath string
}

// NewHandler creates a handler with the given base path (e.g. "/admin").
func NewHandler(basePath string) *Handler {
	if basePath == "" {
		basePath = "/admin"
	}
	basePath = strings.TrimSuffix(basePath, "/")
	return &Handler{BasePath: basePath}
}

// AdminHTMLHandler serves the admin shell for any admin page (dashboard, storage, ci).
func (h *Handler) AdminHTMLHandler(ctx *web.FastRequestContext) error {
	data, err := adminFS.ReadFile("admin.html")
	if err != nil {
		ctx.RequestCtx.SetStatusCode(500)
		ctx.RequestCtx.SetBodyString("Internal Server Error: Failed to read admin.html")
		return err
	}
	ctx.RequestCtx.SetContentType("text/html; charset=utf-8")
	ctx.RequestCtx.SetBody(data)
	return nil
}

// AdminJSHandler serves admin.js.
func (h *Handler) AdminJSHandler(ctx *web.FastRequestContext) error {
	data, err := adminFS.ReadFile("admin.js")
	if err != nil {
		ctx.RequestCtx.SetStatusCode(500)
		ctx.RequestCtx.SetBodyString("Internal Server Error")
		return err
	}
	ctx.RequestCtx.SetContentType("application/javascript; charset=utf-8")
	ctx.RequestCtx.SetBody(data)
	return nil
}

// AdminCSSHandler serves admin.css.
func (h *Handler) AdminCSSHandler(ctx *web.FastRequestContext) error {
	data, err := adminFS.ReadFile("admin.css")
	if err != nil {
		ctx.RequestCtx.SetStatusCode(500)
		ctx.RequestCtx.SetBodyString("Internal Server Error")
		return err
	}
	ctx.RequestCtx.SetContentType("text/css; charset=utf-8")
	ctx.RequestCtx.SetBody(data)
	return nil
}

// RedirectToDashboard redirects GET /admin to /admin/dashboard.
func (h *Handler) RedirectToDashboard(ctx *web.FastRequestContext) error {
	ctx.RequestCtx.Redirect(h.BasePath+"/dashboard", 302)
	return nil
}
