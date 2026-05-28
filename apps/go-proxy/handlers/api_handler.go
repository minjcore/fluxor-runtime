// Copyright (c) 2026 Fluxor Framework
// SPDX-License-Identifier: MIT

package handlers

import (
	"encoding/json"
	"io"
	"log"
	"strconv"

	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/goproxy/domain"
	"github.com/fluxorio/goproxy/storage"
)

// APIHandler handles REST API requests for the web UI
type APIHandler struct {
	storage storage.Storage
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(storage storage.Storage) *APIHandler {
	return &APIHandler{
		storage: storage,
	}
}

// RegisterRoutes registers API routes
func (h *APIHandler) RegisterRoutes(router *web.FastRouter) {
	// Module management APIs
	router.GETFast("/api/v1/modules", h.handleListModules)
	router.GETFast("/api/v1/modules/search", h.handleSearchModules)
	router.GETFast("/api/v1/modules/:module", h.handleGetModule)
	router.GETFast("/api/v1/modules/:module/versions", h.handleListVersions)
	router.GETFast("/api/v1/modules/:module/versions/:version", h.handleGetVersion)
	router.GETFast("/api/v1/modules/:module/stats", h.handleGetStats)

	// Upload API (requires auth)
	router.POSTFast("/api/v1/modules/:module/versions/:version", h.handleUploadVersion)
	router.DELETEFast("/api/v1/modules/:module/versions/:version", h.handleDeleteVersion)

	// Health check
	router.GETFast("/api/v1/health", h.handleHealth)
	router.GETFast("/api/v1/stats", h.handleOverallStats)
}

// handleListModules lists all modules
func (h *APIHandler) handleListModules(ctx *web.FastRequestContext) error {
	prefix := ctx.Query("prefix")
	limitStr := ctx.Query("limit")
	limit := 100

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	modules, err := h.storage.ListModules(ctx.Context(), prefix, limit)
	if err != nil {
		log.Printf("Error listing modules: %v", err)
		return ctx.JSON(500, map[string]string{"error": "internal server error"})
	}

	return ctx.JSON(200, map[string]interface{}{
		"modules": modules,
		"count":   len(modules),
	})
}

// handleSearchModules searches for modules
func (h *APIHandler) handleSearchModules(ctx *web.FastRequestContext) error {
	query := ctx.Query("q")
	if query == "" {
		return ctx.JSON(400, map[string]string{"error": "missing query parameter 'q'"})
	}

	limitStr := ctx.Query("limit")
	limit := 50

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	results, err := h.storage.SearchModules(ctx.Context(), query, limit)
	if err != nil {
		log.Printf("Error searching modules: %v", err)
		return ctx.JSON(500, map[string]string{"error": "internal server error"})
	}

	return ctx.JSON(200, map[string]interface{}{
		"results": results,
		"count":   len(results),
		"query":   query,
	})
}

// handleGetModule returns details for a specific module
func (h *APIHandler) handleGetModule(ctx *web.FastRequestContext) error {
	module := ctx.Param("module")
	if module == "" {
		return ctx.JSON(400, map[string]string{"error": "missing module path"})
	}

	// Unescape module path (may be URL encoded)
	module = domain.UnescapeModulePath(module)

	versions, err := h.storage.ListVersions(ctx.Context(), module)
	if err != nil {
		if domain.IsNotFound(err) {
			return ctx.JSON(404, map[string]string{"error": "module not found"})
		}
		log.Printf("Error getting module %s: %v", module, err)
		return ctx.JSON(500, map[string]string{"error": "internal server error"})
	}

	// Get latest version info
	var latestInfo *domain.VersionInfo
	if len(versions) > 0 {
		latestInfo, _ = h.storage.GetLatest(ctx.Context(), module)
	}

	// Get stats
	stats, _ := h.storage.GetModuleStats(ctx.Context(), module)

	return ctx.JSON(200, map[string]interface{}{
		"path":          module,
		"versions":      versions,
		"versionCount":  len(versions),
		"latestVersion": latestInfo,
		"stats":         stats,
	})
}

// handleListVersions lists all versions for a module
func (h *APIHandler) handleListVersions(ctx *web.FastRequestContext) error {
	module := ctx.Param("module")
	if module == "" {
		return ctx.JSON(400, map[string]string{"error": "missing module path"})
	}

	module = domain.UnescapeModulePath(module)

	versions, err := h.storage.ListVersions(ctx.Context(), module)
	if err != nil {
		if domain.IsNotFound(err) {
			return ctx.JSON(404, map[string]string{"error": "module not found"})
		}
		log.Printf("Error listing versions for %s: %v", module, err)
		return ctx.JSON(500, map[string]string{"error": "internal server error"})
	}

	return ctx.JSON(200, map[string]interface{}{
		"module":   module,
		"versions": versions,
		"count":    len(versions),
	})
}

// handleGetVersion returns details for a specific version
func (h *APIHandler) handleGetVersion(ctx *web.FastRequestContext) error {
	module := ctx.Param("module")
	version := ctx.Param("version")

	if module == "" || version == "" {
		return ctx.JSON(400, map[string]string{"error": "missing module or version"})
	}

	module = domain.UnescapeModulePath(module)

	info, err := h.storage.GetVersionInfo(ctx.Context(), module, version)
	if err != nil {
		if domain.IsNotFound(err) {
			return ctx.JSON(404, map[string]string{"error": "version not found"})
		}
		log.Printf("Error getting version info for %s@%s: %v", module, version, err)
		return ctx.JSON(500, map[string]string{"error": "internal server error"})
	}

	// Also get the go.mod content
	modContent, _ := h.storage.GetModFile(ctx.Context(), module, version)

	return ctx.JSON(200, map[string]interface{}{
		"module":  module,
		"version": info.Version,
		"time":    info.Time,
		"goMod":   string(modContent),
	})
}

// handleGetStats returns statistics for a module
func (h *APIHandler) handleGetStats(ctx *web.FastRequestContext) error {
	module := ctx.Param("module")
	if module == "" {
		return ctx.JSON(400, map[string]string{"error": "missing module path"})
	}

	module = domain.UnescapeModulePath(module)

	stats, err := h.storage.GetModuleStats(ctx.Context(), module)
	if err != nil {
		if domain.IsNotFound(err) {
			return ctx.JSON(404, map[string]string{"error": "module not found"})
		}
		log.Printf("Error getting stats for %s: %v", module, err)
		return ctx.JSON(500, map[string]string{"error": "internal server error"})
	}

	return ctx.JSON(200, stats)
}

// handleUploadVersion handles uploading a new module version
func (h *APIHandler) handleUploadVersion(ctx *web.FastRequestContext) error {
	module := ctx.Param("module")
	version := ctx.Param("version")

	if module == "" || version == "" {
		return ctx.JSON(400, map[string]string{"error": "missing module or version"})
	}

	module = domain.UnescapeModulePath(module)

	// Validate version format
	if !domain.IsValidVersion(version) {
		return ctx.JSON(400, map[string]string{"error": "invalid version format"})
	}

	// Parse multipart form
	form, err := ctx.RequestCtx.MultipartForm()
	if err != nil {
		return ctx.JSON(400, map[string]string{"error": "invalid multipart form"})
	}

	// Get zip file
	zipFiles := form.File["zip"]
	if len(zipFiles) == 0 {
		return ctx.JSON(400, map[string]string{"error": "missing zip file"})
	}

	zipFile, err := zipFiles[0].Open()
	if err != nil {
		return ctx.JSON(400, map[string]string{"error": "failed to open zip file"})
	}
	defer zipFile.Close()

	// Get go.mod content
	var modContent []byte
	modFiles := form.File["mod"]
	if len(modFiles) > 0 {
		modFile, err := modFiles[0].Open()
		if err != nil {
			return ctx.JSON(400, map[string]string{"error": "failed to open mod file"})
		}
		modContent, err = io.ReadAll(modFile)
		modFile.Close()
		if err != nil {
			return ctx.JSON(400, map[string]string{"error": "failed to read mod file"})
		}
	} else {
		// Try to get from form value
		if modValues := form.Value["mod"]; len(modValues) > 0 {
			modContent = []byte(modValues[0])
		}
	}

	if len(modContent) == 0 {
		return ctx.JSON(400, map[string]string{"error": "missing go.mod content"})
	}

	// Store the module
	if err := h.storage.PutModule(ctx.Context(), module, version, zipFile, modContent); err != nil {
		if domain.IsConflict(err) {
			return ctx.JSON(409, map[string]string{"error": "version already exists"})
		}
		log.Printf("Error uploading %s@%s: %v", module, version, err)
		return ctx.JSON(500, map[string]string{"error": "internal server error"})
	}

	log.Printf("Uploaded module %s@%s", module, version)
	return ctx.JSON(201, map[string]string{
		"message": "version uploaded successfully",
		"module":  module,
		"version": version,
	})
}

// handleDeleteVersion handles deleting a module version
func (h *APIHandler) handleDeleteVersion(ctx *web.FastRequestContext) error {
	module := ctx.Param("module")
	version := ctx.Param("version")

	if module == "" || version == "" {
		return ctx.JSON(400, map[string]string{"error": "missing module or version"})
	}

	module = domain.UnescapeModulePath(module)

	if err := h.storage.DeleteVersion(ctx.Context(), module, version); err != nil {
		if domain.IsNotFound(err) {
			return ctx.JSON(404, map[string]string{"error": "version not found"})
		}
		log.Printf("Error deleting %s@%s: %v", module, version, err)
		return ctx.JSON(500, map[string]string{"error": "internal server error"})
	}

	log.Printf("Deleted module %s@%s", module, version)
	return ctx.JSON(200, map[string]string{
		"message": "version deleted successfully",
		"module":  module,
		"version": version,
	})
}

// handleHealth returns health status
func (h *APIHandler) handleHealth(ctx *web.FastRequestContext) error {
	return ctx.JSON(200, map[string]interface{}{
		"status":  "healthy",
		"service": "goproxy",
	})
}

// handleOverallStats returns overall statistics
func (h *APIHandler) handleOverallStats(ctx *web.FastRequestContext) error {
	modules, err := h.storage.ListModules(ctx.Context(), "", 10000)
	if err != nil {
		log.Printf("Error getting overall stats: %v", err)
		return ctx.JSON(500, map[string]string{"error": "internal server error"})
	}

	var totalVersions int
	var totalDownloads int64

	for _, module := range modules {
		stats, err := h.storage.GetModuleStats(ctx.Context(), module)
		if err == nil {
			totalVersions += stats.VersionCount
			totalDownloads += stats.TotalDownloads
		}
	}

	return ctx.JSON(200, map[string]interface{}{
		"totalModules":   len(modules),
		"totalVersions":  totalVersions,
		"totalDownloads": totalDownloads,
	})
}

// UploadRequest represents an upload request body
type UploadRequest struct {
	Module  string `json:"module"`
	Version string `json:"version"`
	GoMod   string `json:"goMod"`
	ZipURL  string `json:"zipUrl,omitempty"` // Optional URL to download zip from
}

// ParseUploadRequest parses an upload request from JSON body
func ParseUploadRequest(body []byte) (*UploadRequest, error) {
	var req UploadRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return &req, nil
}
