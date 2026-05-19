// Package devops provides DevOps utilities and tools for Fluxor applications.
// This package is designed to support common DevOps operations such as health checks,
// metrics collection, deployment automation, and monitoring integration.
//
// This is a foundational package that can be extended with specific DevOps features
// as needed. Currently, it provides a basic structure for future enhancements.
package devops

import (
	"context"
	"time"
)

// HealthStatus represents the health status of a component or service.
type HealthStatus string

const (
	// HealthStatusHealthy indicates the component is healthy and operational.
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusDegraded indicates the component is operational but with reduced functionality.
	HealthStatusDegraded HealthStatus = "degraded"
	// HealthStatusUnhealthy indicates the component is not operational.
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents a health check result.
type HealthCheck struct {
	Status    HealthStatus           `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// Checker is an interface for components that can perform health checks.
type Checker interface {
	CheckHealth(ctx context.Context) (*HealthCheck, error)
}

// NewHealthCheck creates a new health check result.
func NewHealthCheck(status HealthStatus, message string) *HealthCheck {
	return &HealthCheck{
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}
}

// WithDetails adds details to the health check.
func (h *HealthCheck) WithDetails(key string, value interface{}) *HealthCheck {
	if h.Details == nil {
		h.Details = make(map[string]interface{})
	}
	h.Details[key] = value
	return h
}
