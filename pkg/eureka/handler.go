package eureka

import (
	"fmt"
	"strings"

	"github.com/fluxorio/fluxor/pkg/web"
)

// Handler provides Eureka service registry HTTP handlers
type Handler struct {
	registry *Registry
}

// NewHandler creates a new Eureka handler
func NewHandler(registry *Registry) *Handler {
	return &Handler{
		registry: registry,
	}
}

// RegisterHandler handles service registration
// POST /eureka/apps/{appName}
func (h *Handler) RegisterHandler(ctx *web.FastRequestContext) error {
	appName := ctx.Param("appName")
	if appName == "" {
		return ctx.JSON(400, map[string]string{"error": "appName is required"})
	}

	var req RegistrationRequest
	if err := ctx.BindJSON(&req); err != nil {
		return ctx.JSON(400, map[string]string{"error": "invalid request body: " + err.Error()})
	}

	if req.Instance == nil {
		return ctx.JSON(400, map[string]string{"error": "instance is required"})
	}

	// Set service name from URL if not set
	if req.Instance.ServiceName == "" {
		req.Instance.ServiceName = appName
	}

	// Generate instance ID if not provided
	if req.Instance.InstanceID == "" {
		req.Instance.InstanceID = fmt.Sprintf("%s:%s:%d", req.Instance.ServiceName, req.Instance.Host, req.Instance.Port)
	}

	if err := h.registry.Register(req.Instance); err != nil {
		return ctx.JSON(500, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(204, nil) // No Content
}

// RenewHandler handles lease renewal (heartbeat)
// PUT /eureka/apps/{appName}/{instanceId}
func (h *Handler) RenewHandler(ctx *web.FastRequestContext) error {
	appName := ctx.Param("appName")
	instanceID := ctx.Param("instanceId")

	if appName == "" || instanceID == "" {
		return ctx.JSON(400, map[string]string{"error": "appName and instanceId are required"})
	}

	if err := h.registry.Renew(appName, instanceID); err != nil {
		if regErr, ok := err.(*RegistryError); ok && regErr.Code == "INSTANCE_NOT_FOUND" {
			return ctx.JSON(404, map[string]string{"error": err.Error()})
		}
		return ctx.JSON(500, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(200, map[string]string{"status": "OK"})
}

// UnregisterHandler handles service unregistration
// DELETE /eureka/apps/{appName}/{instanceId}
func (h *Handler) UnregisterHandler(ctx *web.FastRequestContext) error {
	appName := ctx.Param("appName")
	instanceID := ctx.Param("instanceId")

	if appName == "" || instanceID == "" {
		return ctx.JSON(400, map[string]string{"error": "appName and instanceId are required"})
	}

	if err := h.registry.Unregister(appName, instanceID); err != nil {
		if regErr, ok := err.(*RegistryError); ok && regErr.Code == "INSTANCE_NOT_FOUND" {
			return ctx.JSON(404, map[string]string{"error": err.Error()})
		}
		return ctx.JSON(500, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(200, map[string]string{"status": "OK"})
}

// GetApplicationHandler returns all instances of a service
// GET /eureka/apps/{appName}
func (h *Handler) GetApplicationHandler(ctx *web.FastRequestContext) error {
	appName := ctx.Param("appName")
	if appName == "" {
		return ctx.JSON(400, map[string]string{"error": "appName is required"})
	}

	// Check if we want all instances or just healthy ones
	status := strings.ToLower(ctx.Query("status"))
	includeExpired := status == "all"

	var instances []*ServiceInstance
	if includeExpired {
		instances = h.registry.GetAllInstances(appName)
	} else {
		instances = h.registry.GetInstances(appName)
	}

	response := DiscoveryResponse{
		ServiceName: appName,
		Instances:   instances,
	}

	return ctx.JSON(200, response)
}

// GetApplicationsHandler returns all registered services
// GET /eureka/apps
func (h *Handler) GetApplicationsHandler(ctx *web.FastRequestContext) error {
	services := h.registry.ListServices()
	response := make(map[string]interface{})
	response["applications"] = services
	return ctx.JSON(200, response)
}

// GetInstanceHandler returns a specific instance
// GET /eureka/apps/{appName}/{instanceId}
func (h *Handler) GetInstanceHandler(ctx *web.FastRequestContext) error {
	instanceID := ctx.Param("instanceId")
	if instanceID == "" {
		return ctx.JSON(400, map[string]string{"error": "instanceId is required"})
	}

	instance, exists := h.registry.GetInstance(instanceID)
	if !exists {
		return ctx.JSON(404, map[string]string{"error": "instance not found"})
	}

	return ctx.JSON(200, instance)
}

// UpdateStatusHandler updates instance status
// PUT /eureka/apps/{appName}/{instanceId}/status?value={status}
func (h *Handler) UpdateStatusHandler(ctx *web.FastRequestContext) error {
	appName := ctx.Param("appName")
	instanceID := ctx.Param("instanceId")
	statusValue := ctx.Query("value")

	if appName == "" || instanceID == "" {
		return ctx.JSON(400, map[string]string{"error": "appName and instanceId are required"})
	}

	if statusValue == "" {
		return ctx.JSON(400, map[string]string{"error": "status value is required"})
	}

	status := InstanceStatus(strings.ToUpper(statusValue))
	if err := h.registry.UpdateStatus(appName, instanceID, status); err != nil {
		if regErr, ok := err.(*RegistryError); ok && regErr.Code == "INSTANCE_NOT_FOUND" {
			return ctx.JSON(404, map[string]string{"error": err.Error()})
		}
		return ctx.JSON(500, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(200, map[string]string{"status": "OK"})
}

// DeleteStatusOverrideHandler removes status override
// DELETE /eureka/apps/{appName}/{instanceId}/status
func (h *Handler) DeleteStatusOverrideHandler(ctx *web.FastRequestContext) error {
	appName := ctx.Param("appName")
	instanceID := ctx.Param("instanceId")

	if appName == "" || instanceID == "" {
		return ctx.JSON(400, map[string]string{"error": "appName and instanceId are required"})
	}

	// Reset status to UP (default)
	if err := h.registry.UpdateStatus(appName, instanceID, InstanceStatusUp); err != nil {
		if regErr, ok := err.(*RegistryError); ok && regErr.Code == "INSTANCE_NOT_FOUND" {
			return ctx.JSON(404, map[string]string{"error": err.Error()})
		}
		return ctx.JSON(500, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(200, map[string]string{"status": "OK"})
}

// HealthHandler returns server health status
func (h *Handler) HealthHandler(ctx *web.FastRequestContext) error {
	return ctx.JSON(200, map[string]string{"status": "UP"})
}

// StatsHandler returns registry statistics
func (h *Handler) StatsHandler(ctx *web.FastRequestContext) error {
	stats := h.registry.GetStats()
	return ctx.JSON(200, stats)
}

// Register registers Eureka REST API routes with the given router
// prefix is the route prefix (e.g., "" for root, "/eureka" for /eureka prefix)
// registry is the service registry instance to use
func Register(router *web.FastRouter, prefix string, registry *Registry) {
	handler := NewHandler(registry)

	// Eureka-compatible REST API endpoints
	// Register instance: POST /eureka/apps/{appName}
	router.POSTFast(prefix+"/eureka/apps/:appName", handler.RegisterHandler)

	// Renew lease: PUT /eureka/apps/{appName}/{instanceId}
	router.PUTFast(prefix+"/eureka/apps/:appName/:instanceId", handler.RenewHandler)

	// Cancel registration: DELETE /eureka/apps/{appName}/{instanceId}
	router.DELETEFast(prefix+"/eureka/apps/:appName/:instanceId", handler.UnregisterHandler)

	// Get application: GET /eureka/apps/{appName}
	router.GETFast(prefix+"/eureka/apps/:appName", handler.GetApplicationHandler)

	// Get all applications: GET /eureka/apps
	router.GETFast(prefix+"/eureka/apps", handler.GetApplicationsHandler)

	// Get instance: GET /eureka/apps/{appName}/{instanceId}
	router.GETFast(prefix+"/eureka/apps/:appName/:instanceId", handler.GetInstanceHandler)

	// Update status: PUT /eureka/apps/{appName}/{instanceId}/status?value={status}
	router.PUTFast(prefix+"/eureka/apps/:appName/:instanceId/status", handler.UpdateStatusHandler)

	// Delete status override: DELETE /eureka/apps/{appName}/{instanceId}/status
	router.DELETEFast(prefix+"/eureka/apps/:appName/:instanceId/status", handler.DeleteStatusOverrideHandler)

	// Health check endpoint
	router.GETFast(prefix+"/health", handler.HealthHandler)

	// Stats endpoint
	router.GETFast(prefix+"/stats", handler.StatsHandler)
}
