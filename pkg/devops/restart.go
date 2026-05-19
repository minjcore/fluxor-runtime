package devops

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// RestartGoAppOptions contains options for restarting Go application service.
type RestartGoAppOptions struct {
	Force bool // Skip confirmation prompts
}

// RestartGoApp restarts the Go application systemd service.
func RestartGoApp(target DeployTarget, opts RestartGoAppOptions) error {
	// Validate Go app config
	if target.GoApp.ServiceName == "" {
		return fmt.Errorf("go_app service_name is required")
	}

	// Create SSH client
	client, err := NewSSHClient(target)
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}
	defer client.Close()

	// Confirm action unless force
	if !opts.Force {
		fmt.Printf("This will restart Go app service: %s\n", target.GoApp.ServiceName)
		fmt.Printf("Continue? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			return fmt.Errorf("restart cancelled by user")
		}
	}

	// Restart systemd service
	fmt.Printf("Restarting systemd service: %s\n", target.GoApp.ServiceName)
	if err := client.ExecuteCommandWithSudo(fmt.Sprintf("systemctl restart %s", target.GoApp.ServiceName)); err != nil {
		return fmt.Errorf("failed to restart service: %w", err)
	}

	// Check service status
	fmt.Printf("Checking service status...\n")
	if err := client.ExecuteCommandWithSudo(fmt.Sprintf("systemctl is-active %s", target.GoApp.ServiceName)); err != nil {
		return fmt.Errorf("service is not active after restart: %w", err)
	}

	fmt.Printf("✅ Successfully restarted Go app service: %s\n", target.GoApp.ServiceName)
	return nil
}

// RestartNginxOptions contains options for restarting nginx.
type RestartNginxOptions struct {
	Force bool // Skip confirmation prompts
}

// RestartNginx restarts nginx service.
func RestartNginx(target DeployTarget, opts RestartNginxOptions) error {
	// Create SSH client
	client, err := NewSSHClient(target)
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}
	defer client.Close()

	// Confirm action unless force
	if !opts.Force {
		fmt.Printf("This will restart nginx service.\n")
		fmt.Printf("Continue? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			return fmt.Errorf("restart cancelled by user")
		}
	}

	// Restart nginx
	fmt.Printf("Restarting nginx service...\n")
	if err := client.ExecuteCommandWithSudo("systemctl restart nginx"); err != nil {
		return fmt.Errorf("failed to restart nginx: %w", err)
	}

	// Check nginx status
	fmt.Printf("Checking nginx status...\n")
	if err := client.ExecuteCommandWithSudo("systemctl is-active nginx"); err != nil {
		return fmt.Errorf("nginx is not active after restart: %w", err)
	}

	fmt.Printf("✅ Successfully restarted nginx service\n")
	return nil
}
