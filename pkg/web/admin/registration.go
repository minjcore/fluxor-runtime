package admin

import (
	"strings"

	"github.com/fluxorio/fluxor/pkg/web"
)

// Register registers admin UI routes on the given router.
// prefix is the route prefix (e.g. "" or "/app"). Admin routes will be at prefix+"/admin", prefix+"/admin/dashboard", etc.
func Register(router *web.FastRouter, prefix string) {
	prefix = strings.TrimSuffix(prefix, "/")
	base := prefix + "/admin"
	h := NewHandler(base)

	// GET /admin -> redirect to /admin/dashboard
	router.GETFast(base, h.RedirectToDashboard)
	// GET /admin/dashboard, /admin/storage, /admin/ci -> same shell (JS shows content by path)
	router.GETFast(base+"/dashboard", h.AdminHTMLHandler)
	router.GETFast(base+"/storage", h.AdminHTMLHandler)
	router.GETFast(base+"/ci", h.AdminHTMLHandler)
	// Static assets
	router.GETFast(base+"/admin.js", h.AdminJSHandler)
	router.GETFast(base+"/admin.css", h.AdminCSSHandler)
}
