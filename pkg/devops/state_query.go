package devops

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// QueryState queries the current state of deployed applications on a target
func QueryState(target DeployTarget) (*ApplicationState, error) {
	client, err := NewSSHClient(target)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH client: %w", err)
	}
	defer client.Close()

	// Try to load existing state to preserve deployed_at timestamps
	stateManager, _ := NewStateManager()
	existingState, _ := stateManager.GetState(target.Host)
	
	state := &ApplicationState{
		Target:        target.Host,
		Host:          target.Host,
		Services:      make(map[string]ServiceState),
		Configurations: make(map[string]ConfigState),
		LastUpdated:   time.Now(),
	}
	
	// Preserve existing deployed_at timestamps when updating services below

	// Query Go app service state
	if target.GoApp.ServiceName != "" {
		serviceState, err := queryGoAppState(client, target)
		if err == nil {
			// Preserve deployed_at from existing state if available
			if existingState != nil {
				if existing, ok := existingState.Services[target.GoApp.ServiceName]; ok {
					if existing.DeployedAt.After(time.Time{}) {
						serviceState.DeployedAt = existing.DeployedAt
					}
				}
			}
			state.Services[target.GoApp.ServiceName] = serviceState
		}
	}

	// Query Docker Compose state
	if target.DockerCompose.ProjectName != "" {
		dockerState, err := queryDockerComposeState(client, target)
		if err == nil {
			// Preserve deployed_at from existing state if available
			if existingState != nil {
				if existing, ok := existingState.Services[target.DockerCompose.ProjectName]; ok {
					if existing.DeployedAt.After(time.Time{}) {
						dockerState.DeployedAt = existing.DeployedAt
					}
				}
			}
			state.Services[target.DockerCompose.ProjectName] = dockerState
		}
	}

	// Query Nginx state
	if target.Nginx.SiteName != "" {
		nginxState, err := queryNginxState(client, target)
		if err == nil {
			// Preserve deployed_at from existing state if available
			nginxServiceName := "nginx-" + target.Nginx.SiteName
			if existingState != nil {
				if existing, ok := existingState.Services[nginxServiceName]; ok {
					if existing.DeployedAt.After(time.Time{}) {
						nginxState.DeployedAt = existing.DeployedAt
					}
				}
			}
			state.Services[nginxServiceName] = nginxState
			
			// Query nginx config state
			configState := queryNginxConfigState(client, target)
			if configState.Path != "" {
				// Preserve deployed_at from existing config if available
				if existingState != nil {
					if existing, ok := existingState.Configurations[nginxServiceName]; ok {
						if existing.DeployedAt.After(time.Time{}) {
							configState.DeployedAt = existing.DeployedAt
						}
					}
				}
				state.Configurations[nginxServiceName] = configState
			}
		}
	}

	// Query resource state
	resourceState, err := queryResourceState(client)
	if err == nil {
		state.Resources = resourceState
	}

	// Query health state
	healthState, err := queryHealthState(client, target)
	if err == nil {
		state.Health = healthState
	}

	return state, nil
}

// queryNginxConfigState queries the state of nginx configuration
func queryNginxConfigState(client *SSHClient, target DeployTarget) ConfigState {
	configState := ConfigState{
		Name:   target.Nginx.SiteName,
		Type:   "nginx",
		Path:   target.Nginx.ConfigDest,
		Active: false,
	}
	
	// Check if config file exists
	if err := client.ExecuteCommand(fmt.Sprintf("test -f %s", target.Nginx.ConfigDest)); err == nil {
		configState.Active = true
	}
	
	// Check if symlink exists (config is enabled)
	if err := client.ExecuteCommand(fmt.Sprintf("test -L %s", target.Nginx.EnabledPath)); err == nil {
		configState.Active = true
	}
	
	return configState
}

// queryGoAppState queries the state of a Go application service
func queryGoAppState(client *SSHClient, target DeployTarget) (ServiceState, error) {
	serviceName := target.GoApp.ServiceName
	state := ServiceState{
		Name:        serviceName,
		Type:        "go-app",
		Status:      "unknown",
		LastChecked: time.Now(),
	}

	// Check if service is active
	output, err := client.ExecuteCommandWithOutput(fmt.Sprintf("systemctl is-active %s", serviceName))
	if err == nil {
		if strings.TrimSpace(output) == "active" {
			state.Status = "running"
		} else {
			state.Status = "stopped"
		}
	} else {
		state.Status = "error"
		state.Error = err.Error()
	}

	// Get PID if running
	if state.Status == "running" {
		pidOutput, err := client.ExecuteCommandWithOutput(fmt.Sprintf("systemctl show --property=MainPID --value %s", serviceName))
		if err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(pidOutput)); err == nil {
				state.PID = pid
			}
		}
	}

	// Get service status details
	statusOutput, err := client.ExecuteCommandWithOutput(fmt.Sprintf("systemctl status %s --no-pager", serviceName))
	if err == nil {
		state.Details = map[string]interface{}{
			"status_output": statusOutput,
		}
	}

	return state, nil
}

// queryDockerComposeState queries the state of Docker Compose services
func queryDockerComposeState(client *SSHClient, target DeployTarget) (ServiceState, error) {
	projectName := target.DockerCompose.ProjectName
	remoteDir := target.DockerCompose.RemoteDir
	state := ServiceState{
		Name:        projectName,
		Type:        "docker-compose",
		Status:      "unknown",
		LastChecked: time.Now(),
	}

	// Use docker compose v2
	composeCmd := "docker compose"
	if err := client.ExecuteCommand("docker compose version"); err != nil {
		// If docker compose v2 is not available, return error
		state.Status = "error"
		state.Error = "Docker Compose v2 is not available"
		return state, nil
	}

	// Get container status
	psCmd := fmt.Sprintf("cd %s && %s -p %s ps --format json", remoteDir, composeCmd, projectName)
	output, err := client.ExecuteCommandWithOutput(psCmd)
	if err != nil {
		state.Status = "error"
		state.Error = err.Error()
		return state, nil
	}

	// Parse container status
	containers := parseDockerComposePS(output)
	if len(containers) == 0 {
		state.Status = "stopped"
	} else {
		allRunning := true
		for _, container := range containers {
			if state, ok := container["state"].(string); ok && state != "running" {
				allRunning = false
				break
			} else if !ok {
				allRunning = false
				break
			}
		}
		if allRunning {
			state.Status = "running"
		} else {
			state.Status = "degraded"
		}
		state.Details = map[string]interface{}{
			"containers": containers,
		}
	}

	return state, nil
}

// queryNginxState queries the state of Nginx
func queryNginxState(client *SSHClient, target DeployTarget) (ServiceState, error) {
	state := ServiceState{
		Name:        "nginx-" + target.Nginx.SiteName,
		Type:        "nginx",
		Status:      "unknown",
		LastChecked: time.Now(),
	}

	// Check if nginx is running
	output, err := client.ExecuteCommandWithOutput("systemctl is-active nginx")
	if err == nil {
		if strings.TrimSpace(output) == "active" {
			state.Status = "running"
		} else {
			state.Status = "stopped"
		}
	} else {
		state.Status = "error"
		state.Error = err.Error()
	}

	// Check if config is enabled
	if state.Status == "running" {
		enabledOutput, err := client.ExecuteCommandWithOutput(fmt.Sprintf("test -L %s && echo enabled || echo disabled", target.Nginx.EnabledPath))
		if err == nil {
			if strings.Contains(enabledOutput, "enabled") {
				state.Details = map[string]interface{}{
					"config_enabled": true,
				}
			}
		}
	}

	return state, nil
}

// queryResourceState queries server resource usage
func queryResourceState(client *SSHClient) (ResourceState, error) {
	state := ResourceState{
		LastChecked: time.Now(),
	}

	// Get disk usage
	dfOutput, err := client.ExecuteCommandWithOutput("df -B1 / | tail -1")
	if err == nil {
		parts := strings.Fields(dfOutput)
		if len(parts) >= 4 {
			if total, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				state.DiskUsage.Total = total
			}
			if used, err := strconv.ParseInt(parts[2], 10, 64); err == nil {
				state.DiskUsage.Used = used
			}
			if available, err := strconv.ParseInt(parts[3], 10, 64); err == nil {
				state.DiskUsage.Available = available
			}
			if state.DiskUsage.Total > 0 {
				state.DiskUsage.Percent = float64(state.DiskUsage.Used) / float64(state.DiskUsage.Total) * 100
			}
		}
	}

	// Get memory usage
	memInfo, err := client.ExecuteCommandWithOutput("cat /proc/meminfo | grep -E 'MemTotal|MemAvailable'")
	if err == nil {
		lines := strings.Split(memInfo, "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				value, err := strconv.ParseInt(parts[1], 10, 64)
				if err != nil {
					continue
				}
				value *= 1024 // Convert KB to bytes
				if strings.Contains(line, "MemTotal") {
					state.MemoryUsage.Total = value
				} else if strings.Contains(line, "MemAvailable") {
					state.MemoryUsage.Available = value
				}
			}
		}
		if state.MemoryUsage.Total > 0 {
			state.MemoryUsage.Used = state.MemoryUsage.Total - state.MemoryUsage.Available
			state.MemoryUsage.Percent = float64(state.MemoryUsage.Used) / float64(state.MemoryUsage.Total) * 100
		}
	}

	return state, nil
}

// queryHealthState queries overall health status
func queryHealthState(client *SSHClient, target DeployTarget) (HealthState, error) {
	health := HealthState{
		Status:      "unknown",
		LastChecked: time.Now(),
		Checks:      []HealthCheckDetail{},
	}

	// Check systemd services
	if target.GoApp.ServiceName != "" {
		output, err := client.ExecuteCommandWithOutput(fmt.Sprintf("systemctl is-active %s", target.GoApp.ServiceName))
		check := HealthCheckDetail{
			Name:      "go-app-" + target.GoApp.ServiceName,
			CheckedAt: time.Now(),
		}
		if err == nil && strings.TrimSpace(output) == "active" {
			check.Status = "healthy"
			check.Message = "Service is running"
		} else {
			check.Status = "unhealthy"
			check.Message = "Service is not running"
		}
		health.Checks = append(health.Checks, check)
	}

	// Determine overall health
	allHealthy := true
	for _, check := range health.Checks {
		if check.Status != "healthy" {
			allHealthy = false
			break
		}
	}

	if allHealthy && len(health.Checks) > 0 {
		health.Status = "healthy"
	} else if len(health.Checks) > 0 {
		health.Status = "degraded"
	}

	return health, nil
}

// parseDockerComposePS parses docker compose ps output
func parseDockerComposePS(output string) []map[string]interface{} {
	var containers []map[string]interface{}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "NAME") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			container := map[string]interface{}{
				"name":  parts[0],
				"state": parts[1],
			}
			containers = append(containers, container)
		}
	}
	return containers
}

