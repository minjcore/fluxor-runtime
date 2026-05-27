package controllers

import (
	"github.com/fluxorio/fluxor/apps/postgres-demo/models"
	"github.com/fluxorio/fluxor/apps/postgres-demo/services"
	"github.com/fluxorio/fluxor/pkg/web"
)

// SystemSettingController handles system setting management requests
type SystemSettingController struct {
	settingService *services.SystemSettingService
}

// NewSystemSettingController creates a new system setting controller
func NewSystemSettingController(settingService *services.SystemSettingService) *SystemSettingController {
	return &SystemSettingController{
		settingService: settingService,
	}
}

// GetAllSettings returns all system settings
func (c *SystemSettingController) GetAllSettings(ctx *web.FastRequestContext) error {
	category := ctx.Query("category")
	
	var settings []models.SystemSetting
	var err error
	
	if category != "" {
		settings, err = c.settingService.GetSettingsByCategory(ctx.Context(), category)
	} else {
		settings, err = c.settingService.GetAllSettings(ctx.Context(), nil)
	}
	
	if err != nil {
		return ctx.JSON(500, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return ctx.JSON(200, settings)
}

// GetSetting returns a specific setting by key
func (c *SystemSettingController) GetSetting(ctx *web.FastRequestContext) error {
	key := ctx.Param("key")
	if key == "" {
		return ctx.JSON(400, map[string]interface{}{
			"error": "setting key is required",
		})
	}

	setting, err := c.settingService.GetSetting(ctx.Context(), key)
	if err != nil {
		return ctx.JSON(404, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return ctx.JSON(200, setting)
}

// GetSettingValue returns a setting value (parsed according to data type)
func (c *SystemSettingController) GetSettingValue(ctx *web.FastRequestContext) error {
	key := ctx.Param("key")
	if key == "" {
		return ctx.JSON(400, map[string]interface{}{
			"error": "setting key is required",
		})
	}

	value, err := c.settingService.GetSettingValue(ctx.Context(), key)
	if err != nil {
		return ctx.JSON(404, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return ctx.JSON(200, map[string]interface{}{
		"key":   key,
		"value": value,
	})
}

// CreateSetting creates a new system setting
func (c *SystemSettingController) CreateSetting(ctx *web.FastRequestContext) error {
	var req models.CreateSettingRequest
	if err := ctx.BindJSON(&req); err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_request",
			"message": "Invalid JSON",
		})
	}

	// Get user from JWT token (if available)
	userID := c.getUserID(ctx)

	settingID, err := c.settingService.CreateSetting(ctx.Context(), req, userID)
	if err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "create_setting_failed",
			"message": err.Error(),
		})
	}

	return ctx.JSON(201, map[string]interface{}{
		"id":      settingID,
		"message": "Setting created successfully",
	})
}

// UpdateSetting updates an existing system setting
func (c *SystemSettingController) UpdateSetting(ctx *web.FastRequestContext) error {
	key := ctx.Param("key")
	if key == "" {
		return ctx.JSON(400, map[string]interface{}{
			"error": "setting key is required",
		})
	}

	var req models.UpdateSettingRequest
	if err := ctx.BindJSON(&req); err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_request",
			"message": "Invalid JSON",
		})
	}

	// Get user from JWT token (if available)
	userID := c.getUserID(ctx)

	err := c.settingService.UpdateSetting(ctx.Context(), key, req, userID)
	if err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "update_setting_failed",
			"message": err.Error(),
		})
	}

	return ctx.JSON(200, map[string]interface{}{
		"message": "Setting updated successfully",
	})
}

// DeleteSetting deletes a system setting
func (c *SystemSettingController) DeleteSetting(ctx *web.FastRequestContext) error {
	key := ctx.Param("key")
	if key == "" {
		return ctx.JSON(400, map[string]interface{}{
			"error": "setting key is required",
		})
	}

	err := c.settingService.DeleteSetting(ctx.Context(), key)
	if err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "delete_setting_failed",
			"message": err.Error(),
		})
	}

	return ctx.JSON(200, map[string]interface{}{
		"message": "Setting deleted successfully",
	})
}

// getUserID extracts user ID from JWT token (if available)
func (c *SystemSettingController) getUserID(ctx *web.FastRequestContext) *string {
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
