package core

import (
	"time"
)

// DeploymentDiagnostic contains detailed information about a single deployment
type DeploymentDiagnostic struct {
	ID            string            `json:"id"`
	State         DeploymentState   `json:"state"`
	VerticleType  string            `json:"verticle_type"`
	StartTime     *time.Time        `json:"start_time,omitempty"`
	StartDuration time.Duration     `json:"start_duration"`
	StartTimeout  time.Duration     `json:"start_timeout"`
	Error         string            `json:"error,omitempty"`
	StateHistory  []StateTransition `json:"state_history"`
	LastUpdated   time.Time         `json:"last_updated"`
}

// StateTransition records a state change
type StateTransition struct {
	From      DeploymentState `json:"from"`
	To        DeploymentState `json:"to"`
	Timestamp time.Time       `json:"timestamp"`
	Error     string          `json:"error,omitempty"`
}

// SystemDiagnostic contains overall system information
type SystemDiagnostic struct {
	Timestamp          time.Time               `json:"timestamp"`
	TotalDeployments   int                     `json:"total_deployments"`
	DeploymentsByState map[DeploymentState]int `json:"deployments_by_state"`
	StartTimeout       time.Duration           `json:"start_timeout"`
	HealthStatus       string                  `json:"health_status"` // "HEALTHY", "DEGRADED", "UNHEALTHY"
	DeploymentDetails  []DeploymentDiagnostic  `json:"deployment_details,omitempty"`
}

// HealthStatus constants
const (
	HealthStatusHealthy   = "HEALTHY"
	HealthStatusDegraded  = "DEGRADED"
	HealthStatusUnhealthy = "UNHEALTHY"
)
