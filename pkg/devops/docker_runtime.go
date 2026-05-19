package devops

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DockerRuntime is the interface for the full Docker Compose lifecycle (validate, prepare, deploy, destroy, status).
// Use NewDefaultDocker() for the default implementation.
//
// Part of the multi-runtime vision: fluxor deploy app.yaml → DockerRuntime; fluxor run app.py → pkg/shell.ProcessRuntime.
// Enables adding NewKubernetesRuntime(), NewDockerSwarmRuntime(), NewLocalDockerRuntime() later.
type DockerRuntime interface {
	Validate(target DeployTarget) error
	Prepare(client *SSHClient, target DeployTarget) error
	Deploy(client *SSHClient, target DeployTarget) error
	Destroy(client *SSHClient, target DeployTarget) error
	Status(client *SSHClient, target DeployTarget) (ServiceState, error)
}

// DefaultDockerRuntime is the default implementation of DockerRuntime (compose v2 on VPS via SSH).
type DefaultDockerRuntime struct{}

// NewDefaultDocker returns a DockerRuntime that implements the full lifecycle for Docker Compose deploy.
func NewDefaultDocker() DockerRuntime {
	return &DefaultDockerRuntime{}
}

// Validate checks target Docker Compose config and that the compose file exists.
func (d *DefaultDockerRuntime) Validate(target DeployTarget) error {
	if target.DockerCompose.ComposeFile == "" {
		return fmt.Errorf("docker_compose compose_file is required")
	}
	if target.DockerCompose.RemoteDir == "" {
		return fmt.Errorf("docker_compose remote_dir is required")
	}
	if target.DockerCompose.ProjectName == "" {
		return fmt.Errorf("docker_compose project_name is required")
	}
	composePath := ResolveConfigFile(target.DockerCompose.ComposeFile)
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		return fmt.Errorf("docker compose file does not exist: %s (checked: %s)", target.DockerCompose.ComposeFile, composePath)
	}
	return nil
}

// Prepare checks Docker and Docker Compose are installed and creates the remote directory.
func (d *DefaultDockerRuntime) Prepare(client *SSHClient, target DeployTarget) error {
	if err := client.ExecuteCommandWithSudo("docker --version"); err != nil {
		return fmt.Errorf("Docker is not installed on the server: %w", err)
	}
	if err := client.ExecuteCommandWithSudo("docker compose version"); err != nil {
		return fmt.Errorf("Docker Compose v2 is not installed on the server (docker compose): %w", err)
	}
	if err := client.ExecuteCommandWithSudo(fmt.Sprintf("mkdir -p %s", target.DockerCompose.RemoteDir)); err != nil {
		return fmt.Errorf("failed to create remote directory: %w", err)
	}
	return nil
}

// Deploy transfers the compose file and runs docker compose up -d (after down if any).
func (d *DefaultDockerRuntime) Deploy(client *SSHClient, target DeployTarget) error {
	composePath := ResolveConfigFile(target.DockerCompose.ComposeFile)
	remoteComposeFile := filepath.Join(target.DockerCompose.RemoteDir, "docker-compose.yml")
	if err := transferFile(client, composePath, remoteComposeFile, target); err != nil {
		return fmt.Errorf("failed to transfer docker-compose file: %w", err)
	}
	composeCmd := "docker compose"
	stopCmd := fmt.Sprintf("cd %s && %s -p %s down", target.DockerCompose.RemoteDir, composeCmd, target.DockerCompose.ProjectName)
	_ = client.ExecuteCommandWithSudo(stopCmd)
	upCmd := fmt.Sprintf("cd %s && %s -p %s up -d", target.DockerCompose.RemoteDir, composeCmd, target.DockerCompose.ProjectName)
	if err := client.ExecuteCommandWithSudo(upCmd); err != nil {
		return fmt.Errorf("failed to start Docker Compose services: %w", err)
	}
	return nil
}

// Destroy runs docker compose down for the project.
func (d *DefaultDockerRuntime) Destroy(client *SSHClient, target DeployTarget) error {
	cmd := fmt.Sprintf("cd %s && docker compose -p %s down", target.DockerCompose.RemoteDir, target.DockerCompose.ProjectName)
	return client.ExecuteCommandWithSudo(cmd)
}

// Status returns the current service state by running docker compose ps.
func (d *DefaultDockerRuntime) Status(client *SSHClient, target DeployTarget) (ServiceState, error) {
	cmd := fmt.Sprintf("cd %s && docker compose -p %s ps", target.DockerCompose.RemoteDir, target.DockerCompose.ProjectName)
	out, err := client.ExecuteCommandWithOutput(cmd)
	if err != nil {
		return ServiceState{}, fmt.Errorf("docker compose ps: %w", err)
	}
	containers := parseDockerComposePS(out)
	status := "running"
	if len(containers) == 0 {
		status = "stopped"
	}
	return ServiceState{
		Name:        target.DockerCompose.ProjectName,
		Type:        "docker-compose",
		Status:      status,
		LastChecked: time.Now(),
		Details: map[string]interface{}{
			"containers": containers,
		},
	}, nil
}
