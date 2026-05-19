package devops

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// DeployDockerComposeOptions contains options for deploying Docker Compose application.
type DeployDockerComposeOptions struct {
	Force bool // Skip confirmation prompts
}

// DefaultDeployDockerComposeOptions returns default options (no force, prompt for confirmation).
func DefaultDeployDockerComposeOptions() DeployDockerComposeOptions {
	return DeployDockerComposeOptions{Force: false}
}

// DeployDockerCompose deploys Docker Compose application to VPS using the default Docker runtime.
func DeployDockerCompose(target DeployTarget, opts DeployDockerComposeOptions) error {
	runtime := NewDefaultDocker()

	if err := runtime.Validate(target); err != nil {
		return err
	}

	if !opts.Force {
		fmt.Printf("This will deploy Docker Compose application to %s\n", target.Host)
		fmt.Printf("  - Compose File: %s\n", target.DockerCompose.ComposeFile)
		fmt.Printf("  - Remote Directory: %s\n", target.DockerCompose.RemoteDir)
		fmt.Printf("  - Project Name: %s\n", target.DockerCompose.ProjectName)
		fmt.Printf("Continue? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			return fmt.Errorf("deploy cancelled by user")
		}
	}

	client, err := NewSSHClient(target)
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}
	defer client.Close()

	fmt.Printf("Checking Docker installation...\n")
	if err := runtime.Prepare(client, target); err != nil {
		return err
	}

	fmt.Printf("Transferring docker-compose file to VPS...\n")
	fmt.Printf("Stopping existing containers (if any)...\n")
	fmt.Printf("Pulling images and starting containers...\n")
	if err := runtime.Deploy(client, target); err != nil {
		return err
	}

	fmt.Printf("Checking container status...\n")
	serviceState, _ := runtime.Status(client, target)
	if serviceState.Details != nil {
		if containers, ok := serviceState.Details["containers"].([]map[string]interface{}); ok && len(containers) > 0 {
			// optional: print container list
		}
	}

	fmt.Printf("✅ Successfully deployed Docker Compose application: %s\n", target.DockerCompose.ProjectName)

	stateManager, err := NewStateManager()
	if err == nil {
		serviceState.Name = target.DockerCompose.ProjectName
		serviceState.Type = "docker-compose"
		serviceState.Status = "running"
		serviceState.DeployedAt = time.Now()
		serviceState.LastChecked = time.Now()
		_ = stateManager.UpdateServiceState(target.Host, target.DockerCompose.ProjectName, serviceState)
	}

	return nil
}
