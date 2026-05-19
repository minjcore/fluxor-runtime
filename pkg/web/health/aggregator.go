package health

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fluxorio/fluxor/pkg/web"
)

// Aggregator aggregates health check results
type Aggregator struct {
	registry *Registry
}

// NewAggregator creates a new health check aggregator
func NewAggregator(registry *Registry) *Aggregator {
	if registry == nil {
		registry = globalRegistry
	}
	return &Aggregator{registry: registry}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
}

// Handler returns a FastRequestHandler for the /health endpoint
func Handler() web.FastRequestHandler {
	aggregator := NewAggregator(nil)
	return aggregator.HandleHealth
}

// ReadyHandler returns a FastRequestHandler for the /ready endpoint
// Returns 503 if any check fails, 200 if all checks pass
func ReadyHandler() web.FastRequestHandler {
	aggregator := NewAggregator(nil)
	return aggregator.HandleReady
}

// HandleHealth handles the /health endpoint
func (a *Aggregator) HandleHealth(ctx *web.FastRequestContext) error {
	results := a.registry.Check(ctx.Context())

	// Determine overall status
	overallStatus := StatusUp
	for _, result := range results {
		if result.Status == StatusDown {
			overallStatus = StatusDown
			break
		}
	}

	response := HealthResponse{
		Status:    string(overallStatus),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    results,
		RequestID: ctx.RequestID(),
	}

	statusCode := 200
	if overallStatus == StatusDown {
		statusCode = 503
	}

	return ctx.JSON(statusCode, response)
}

// HandleReady handles the /ready endpoint
// Returns 503 if any critical check fails, 200 if all checks pass
func (a *Aggregator) HandleReady(ctx *web.FastRequestContext) error {
	results := a.registry.Check(ctx.Context())

	// Determine overall status
	overallStatus := StatusUp
	for _, result := range results {
		if result.Status == StatusDown {
			overallStatus = StatusDown
			break
		}
	}

	response := HealthResponse{
		Status:    string(overallStatus),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    results,
		RequestID: ctx.RequestID(),
	}

	statusCode := 200
	if overallStatus == StatusDown {
		statusCode = 503
	}

	return ctx.JSON(statusCode, response)
}

// GetHealthStatus returns the current health status without HTTP response
func (a *Aggregator) GetHealthStatus(ctx context.Context) (Status, map[string]CheckResult) {
	results := a.registry.Check(ctx)

	overallStatus := StatusUp
	for _, result := range results {
		if result.Status == StatusDown {
			overallStatus = StatusDown
			break
		}
	}

	return overallStatus, results
}

// FormatHealthResponse formats health check results as JSON
func FormatHealthResponse(status Status, checks map[string]CheckResult, requestID string) ([]byte, error) {
	response := HealthResponse{
		Status:    string(status),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    checks,
		RequestID: requestID,
	}

	return json.Marshal(response)
}

// FormatHealthResponseString formats health check results as JSON string
func FormatHealthResponseString(status Status, checks map[string]CheckResult, requestID string) (string, error) {
	data, err := FormatHealthResponse(status, checks, requestID)
	if err != nil {
		return "", fmt.Errorf("failed to format health response: %w", err)
	}
	return string(data), nil
}
