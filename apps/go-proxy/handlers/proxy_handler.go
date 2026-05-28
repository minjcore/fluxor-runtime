// Copyright (c) 2026 Fluxor Framework
// SPDX-License-Identifier: MIT

package handlers

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/goproxy/domain"
	"github.com/fluxorio/goproxy/storage"
)

// ProxyHandler handles GOPROXY protocol requests
type ProxyHandler struct {
	storage storage.Storage
}

// NewProxyHandler creates a new GOPROXY protocol handler
func NewProxyHandler(storage storage.Storage) *ProxyHandler {
	return &ProxyHandler{
		storage: storage,
	}
}

// RegisterRoutes registers GOPROXY protocol routes.
// The module path can contain slashes, so we use SetDefaultHandler
// which catches all requests not matched by specific routes above.
func (h *ProxyHandler) RegisterRoutes(router *web.FastRouter) {
	router.SetDefaultHandler(h.handleProxyRequest)
}

// handleProxyRequest routes requests to the appropriate handler based on the path
func (h *ProxyHandler) handleProxyRequest(ctx *web.FastRequestContext) error {
	path := strings.TrimPrefix(string(ctx.RequestCtx.Path()), "/")
	if path == "" {
		return ctx.JSON(400, map[string]string{"error": "missing module path"})
	}

	// Parse the path to determine the request type
	// Format: {module}/@v/list, {module}/@v/{version}.info, etc.

	// Serve go-import meta tag for `go get` tool (vanity redirect)
	if string(ctx.RequestCtx.QueryArgs().Peek("go-get")) == "1" {
		module := unescapeModule(path)
		return h.handleGoGet(ctx, module)
	}

	// Check for @latest suffix
	if strings.HasSuffix(path, "/@latest") {
		module := strings.TrimSuffix(path, "/@latest")
		return h.handleLatest(ctx, module)
	}

	// Check for @v/ segment
	atVIndex := strings.Index(path, "/@v/")
	if atVIndex == -1 {
		return ctx.JSON(404, map[string]string{"error": "invalid path format"})
	}

	module := path[:atVIndex]
	suffix := path[atVIndex+4:] // Skip "/@v/"

	// Route to appropriate handler
	switch {
	case suffix == "list":
		return h.handleList(ctx, module)
	case strings.HasSuffix(suffix, ".info"):
		version := strings.TrimSuffix(suffix, ".info")
		return h.handleInfo(ctx, module, version)
	case strings.HasSuffix(suffix, ".mod"):
		version := strings.TrimSuffix(suffix, ".mod")
		return h.handleMod(ctx, module, version)
	case strings.HasSuffix(suffix, ".zip"):
		version := strings.TrimSuffix(suffix, ".zip")
		return h.handleZip(ctx, module, version)
	default:
		return ctx.JSON(404, map[string]string{"error": "unknown request type"})
	}
}

// handleList handles GET /{module}/@v/list
// Returns a list of available versions, one per line
func (h *ProxyHandler) handleList(ctx *web.FastRequestContext, module string) error {
	module = unescapeModule(module)

	versions, err := h.storage.ListVersions(ctx.Context(), module)
	if err != nil {
		if domain.IsNotFound(err) {
			return ctx.Text(404, "not found")
		}
		log.Printf("Error listing versions for %s: %v", module, err)
		return ctx.Text(500, "internal server error")
	}

	// Return versions as plain text, one per line
	ctx.RequestCtx.Response.Header.Set("Content-Type", "text/plain; charset=utf-8")
	ctx.RequestCtx.Response.Header.Set("Cache-Control", "public, max-age=300")
	return ctx.Text(200, strings.Join(versions, "\n"))
}

// handleInfo handles GET /{module}/@v/{version}.info
// Returns version metadata as JSON
func (h *ProxyHandler) handleInfo(ctx *web.FastRequestContext, module, version string) error {
	module = unescapeModule(module)

	info, err := h.storage.GetVersionInfo(ctx.Context(), module, version)
	if err != nil {
		if domain.IsNotFound(err) {
			return ctx.Text(404, "not found")
		}
		log.Printf("Error getting version info for %s@%s: %v", module, version, err)
		return ctx.Text(500, "internal server error")
	}

	ctx.RequestCtx.Response.Header.Set("Content-Type", "application/json")
	ctx.RequestCtx.Response.Header.Set("Cache-Control", "public, max-age=86400") // 24 hours
	return ctx.JSON(200, info)
}

// handleMod handles GET /{module}/@v/{version}.mod
// Returns the go.mod file contents
func (h *ProxyHandler) handleMod(ctx *web.FastRequestContext, module, version string) error {
	module = unescapeModule(module)

	modContent, err := h.storage.GetModFile(ctx.Context(), module, version)
	if err != nil {
		if domain.IsNotFound(err) {
			return ctx.Text(404, "not found")
		}
		log.Printf("Error getting mod file for %s@%s: %v", module, version, err)
		return ctx.Text(500, "internal server error")
	}

	ctx.RequestCtx.Response.Header.Set("Content-Type", "text/plain; charset=utf-8")
	ctx.RequestCtx.Response.Header.Set("Cache-Control", "public, max-age=86400") // 24 hours
	return ctx.Text(200, string(modContent))
}

// handleZip handles GET /{module}/@v/{version}.zip
// Returns the source archive
func (h *ProxyHandler) handleZip(ctx *web.FastRequestContext, module, version string) error {
	module = unescapeModule(module)

	// Increment download counter
	go func() {
		if err := h.storage.IncrementDownloads(ctx.Context(), module, version); err != nil {
			log.Printf("Error incrementing downloads for %s@%s: %v", module, version, err)
		}
	}()

	zipReader, err := h.storage.GetZip(ctx.Context(), module, version)
	if err != nil {
		if domain.IsNotFound(err) {
			return ctx.Text(404, "not found")
		}
		log.Printf("Error getting zip for %s@%s: %v", module, version, err)
		return ctx.Text(500, "internal server error")
	}
	defer zipReader.Close()

	// Stream the zip file
	ctx.RequestCtx.Response.Header.Set("Content-Type", "application/zip")
	ctx.RequestCtx.Response.Header.Set("Cache-Control", "public, max-age=86400") // 24 hours
	ctx.RequestCtx.Response.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s@%s.zip", 
		escapeFilename(module), version))

	// Copy zip content to response
	_, err = io.Copy(ctx.RequestCtx.Response.BodyWriter(), zipReader)
	if err != nil {
		log.Printf("Error streaming zip for %s@%s: %v", module, version, err)
		return err
	}

	return nil
}

// handleLatest handles GET /{module}/@latest
// Returns the latest version info
func (h *ProxyHandler) handleLatest(ctx *web.FastRequestContext, module string) error {
	module = unescapeModule(module)

	info, err := h.storage.GetLatest(ctx.Context(), module)
	if err != nil {
		if domain.IsNotFound(err) {
			return ctx.Text(404, "not found")
		}
		log.Printf("Error getting latest for %s: %v", module, err)
		return ctx.Text(500, "internal server error")
	}

	ctx.RequestCtx.Response.Header.Set("Content-Type", "application/json")
	ctx.RequestCtx.Response.Header.Set("Cache-Control", "public, max-age=300") // 5 minutes
	return ctx.JSON(200, info)
}

// handleGoGet serves a go-import meta tag so `go get` can resolve the module.
// The proxy itself is the authoritative source — no external VCS redirect needed.
func (h *ProxyHandler) handleGoGet(ctx *web.FastRequestContext, module string) error {
	// Derive the proxy root from the Host header (e.g. "go.nivic.dev")
	host := string(ctx.RequestCtx.Host())
	if host == "" {
		host = "go.nivic.dev"
	}
	importPath := module
	proxyURL := "https://" + host

	html := fmt.Sprintf(`<!DOCTYPE html><html><head>
<meta name="go-import" content="%s mod %s">
</head><body>
<p><a href="%s">%s</a></p>
</body></html>`, importPath, proxyURL, proxyURL, importPath)

	ctx.RequestCtx.Response.Header.Set("Content-Type", "text/html; charset=utf-8")
	return ctx.Text(200, html)
}

// unescapeModule unescapes a module path from URL format
// Go module paths in URLs have uppercase letters escaped with '!'
func unescapeModule(escaped string) string {
	return domain.UnescapeModulePath(escaped)
}

// escapeFilename escapes a module path for use in Content-Disposition
func escapeFilename(module string) string {
	// Replace slashes with underscores for filename
	return strings.ReplaceAll(module, "/", "_")
}
