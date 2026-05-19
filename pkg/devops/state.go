package devops

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ApplicationState represents the current state of a deployed application
type ApplicationState struct {
	Target        string                 `json:"target"`
	Host          string                 `json:"host"`
	LastUpdated   time.Time              `json:"last_updated"`
	Services      map[string]ServiceState `json:"services"`
	Configurations map[string]ConfigState `json:"configurations"`
	Resources     ResourceState          `json:"resources"`
	Health        HealthState            `json:"health"`
}

// ServiceState represents the state of a service (Go app, Docker, etc.)
type ServiceState struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"` // "go-app", "docker-compose", "nginx"
	Status      string    `json:"status"` // "running", "stopped", "error", "unknown"
	PID         int       `json:"pid,omitempty"`
	Version     string    `json:"version,omitempty"`
	DeployedAt  time.Time `json:"deployed_at,omitempty"`
	LastChecked time.Time `json:"last_checked"`
	Error       string    `json:"error,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// ConfigState represents the state of a configuration
type ConfigState struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"` // "nginx", "systemd", "docker-compose"
	Path        string    `json:"path"`
	DeployedAt  time.Time `json:"deployed_at"`
	Checksum    string    `json:"checksum,omitempty"`
	Active      bool      `json:"active"`
}

// ResourceState represents server resource information
type ResourceState struct {
	DiskUsage   DiskUsage   `json:"disk_usage"`
	MemoryUsage MemoryUsage `json:"memory_usage"`
	CPUUsage    CPUUsage    `json:"cpu_usage,omitempty"`
	LastChecked time.Time   `json:"last_checked"`
}

// DiskUsage represents disk usage information
type DiskUsage struct {
	Total     int64   `json:"total"`     // bytes
	Used      int64   `json:"used"`      // bytes
	Available int64   `json:"available"` // bytes
	Percent   float64 `json:"percent"`
}

// MemoryUsage represents memory usage information
type MemoryUsage struct {
	Total     int64   `json:"total"`     // bytes
	Used      int64   `json:"used"`     // bytes
	Available int64   `json:"available"` // bytes
	Percent   float64 `json:"percent"`
}

// CPUUsage represents CPU usage information
type CPUUsage struct {
	Percent float64 `json:"percent"`
	LoadAvg []float64 `json:"load_avg,omitempty"`
}

// HealthState represents overall health status
type HealthState struct {
	Status      string    `json:"status"` // "healthy", "degraded", "unhealthy"
	LastChecked time.Time `json:"last_checked"`
	Checks      []HealthCheckDetail `json:"checks,omitempty"`
}

// HealthCheckDetail represents an individual health check detail
type HealthCheckDetail struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

// StateManager manages application state
type StateManager struct {
	stateDir string
}

// NewStateManager creates a new state manager
func NewStateManager() (*StateManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	stateDir := filepath.Join(homeDir, ".fluxor-cli", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	return &StateManager{
		stateDir: stateDir,
	}, nil
}

// GetState retrieves the state for a target
func (sm *StateManager) GetState(target string) (*ApplicationState, error) {
	statePath := sm.getStatePath(target)
	
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // State doesn't exist yet
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state ApplicationState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// SaveState saves the state for a target
func (sm *StateManager) SaveState(target string, state *ApplicationState) error {
	statePath := sm.getStatePath(target)
	
	state.LastUpdated = time.Now()
	
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// UpdateServiceState updates the state of a service
func (sm *StateManager) UpdateServiceState(target string, serviceName string, serviceState ServiceState) error {
	state, err := sm.GetState(target)
	if err != nil {
		return err
	}

	if state == nil {
		state = &ApplicationState{
			Target:        target,
			Services:      make(map[string]ServiceState),
			Configurations: make(map[string]ConfigState),
		}
	}

	if state.Services == nil {
		state.Services = make(map[string]ServiceState)
	}

	serviceState.LastChecked = time.Now()
	state.Services[serviceName] = serviceState

	return sm.SaveState(target, state)
}

// UpdateConfigState updates the state of a configuration
func (sm *StateManager) UpdateConfigState(target string, configName string, configState ConfigState) error {
	state, err := sm.GetState(target)
	if err != nil {
		return err
	}

	if state == nil {
		state = &ApplicationState{
			Target:        target,
			Services:      make(map[string]ServiceState),
			Configurations: make(map[string]ConfigState),
		}
	}

	if state.Configurations == nil {
		state.Configurations = make(map[string]ConfigState)
	}

	state.Configurations[configName] = configState

	return sm.SaveState(target, state)
}

// UpdateResourceState updates resource state
func (sm *StateManager) UpdateResourceState(target string, resourceState ResourceState) error {
	state, err := sm.GetState(target)
	if err != nil {
		return err
	}

	if state == nil {
		state = &ApplicationState{
			Target:        target,
			Services:      make(map[string]ServiceState),
			Configurations: make(map[string]ConfigState),
		}
	}

	resourceState.LastChecked = time.Now()
	state.Resources = resourceState

	return sm.SaveState(target, state)
}

// UpdateHealthState updates health state
func (sm *StateManager) UpdateHealthState(target string, healthState HealthState) error {
	state, err := sm.GetState(target)
	if err != nil {
		return err
	}

	if state == nil {
		state = &ApplicationState{
			Target:        target,
			Services:      make(map[string]ServiceState),
			Configurations: make(map[string]ConfigState),
		}
	}

	healthState.LastChecked = time.Now()
	state.Health = healthState

	return sm.SaveState(target, state)
}

// GetAllStates returns all saved states
func (sm *StateManager) GetAllStates() (map[string]*ApplicationState, error) {
	states := make(map[string]*ApplicationState)

	files, err := os.ReadDir(sm.stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return states, nil
		}
		return nil, fmt.Errorf("failed to read state directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		target := file.Name()
		// Remove .json extension
		if len(target) > 5 && target[len(target)-5:] == ".json" {
			target = target[:len(target)-5]
		}

		state, err := sm.GetState(target)
		if err != nil {
			continue // Skip invalid state files
		}

		if state != nil {
			states[target] = state
		}
	}

	return states, nil
}

// getStatePath returns the path to the state file for a target
func (sm *StateManager) getStatePath(target string) string {
	return filepath.Join(sm.stateDir, target+".json")
}
