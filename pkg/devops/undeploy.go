package devops

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// UndeployNginxOptions contains options for undeploying nginx configuration.
type UndeployNginxOptions struct {
	Force bool // Skip confirmation prompts
}

// UndeployNginx removes nginx configuration and restarts nginx.
func UndeployNginx(target DeployTarget, opts UndeployNginxOptions) error {
	// Validate nginx config
	if target.Nginx.ConfigDest == "" {
		return fmt.Errorf("nginx config_dest is required")
	}
	if target.Nginx.EnabledPath == "" {
		return fmt.Errorf("nginx enabled_path is required")
	}

	// Create SSH client
	client, err := NewSSHClient(target)
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}
	defer client.Close()

	// Confirm action unless force
	if !opts.Force {
		fmt.Printf("This will remove nginx config for %s\n", target.Nginx.SiteName)
		fmt.Printf("  - %s\n", target.Nginx.EnabledPath)
		fmt.Printf("  - %s\n", target.Nginx.ConfigDest)
		fmt.Printf("And restart nginx service.\n")
		fmt.Printf("Continue? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			return fmt.Errorf("undeploy cancelled by user")
		}
	}

	// Remove symlink in sites-enabled
	fmt.Printf("Removing nginx symlink: %s\n", target.Nginx.EnabledPath)
	if err := client.ExecuteCommandWithSudo(fmt.Sprintf("rm -f %s", target.Nginx.EnabledPath)); err != nil {
		return fmt.Errorf("failed to remove symlink: %w", err)
	}

	// Remove config file in sites-available
	fmt.Printf("Removing nginx config: %s\n", target.Nginx.ConfigDest)
	if err := client.ExecuteCommandWithSudo(fmt.Sprintf("rm -f %s", target.Nginx.ConfigDest)); err != nil {
		return fmt.Errorf("failed to remove config file: %w", err)
	}

	// Test nginx configuration
	fmt.Printf("Testing nginx configuration...\n")
	if err := client.ExecuteCommandWithSudo("nginx -t"); err != nil {
		return fmt.Errorf("nginx config test failed: %w", err)
	}

	// Restart nginx to apply changes
	fmt.Printf("Restarting nginx service...\n")
	if err := client.ExecuteCommandWithSudo("systemctl restart nginx"); err != nil {
		return fmt.Errorf("failed to restart nginx: %w", err)
	}

	fmt.Printf("✅ Successfully undeployed nginx config for %s\n", target.Nginx.SiteName)
	return nil
}

// UndeployGoAppOptions contains options for undeploying Go application service.
type UndeployGoAppOptions struct {
	RemoveFiles bool // Remove application binary files
	Force       bool // Skip confirmation prompts
}

// UndeployGoApp stops and removes Go application systemd service.
func UndeployGoApp(target DeployTarget, opts UndeployGoAppOptions) error {
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
		fmt.Printf("This will stop and remove Go app service: %s\n", target.GoApp.ServiceName)
		if target.GoApp.BinaryPath != "" {
			fmt.Printf("  - Service: %s\n", target.GoApp.ServiceName)
			fmt.Printf("  - Binary: %s\n", target.GoApp.BinaryPath)
		}
		if opts.RemoveFiles && target.GoApp.BinaryPath != "" {
			fmt.Printf("  - Will remove binary file\n")
		}
		fmt.Printf("Continue? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			return fmt.Errorf("undeploy cancelled by user")
		}
	}

	// Stop systemd service
	fmt.Printf("Stopping systemd service: %s\n", target.GoApp.ServiceName)
	if err := client.ExecuteCommandWithSudo(fmt.Sprintf("systemctl stop %s", target.GoApp.ServiceName)); err != nil {
		// Service might not be running, continue anyway
		fmt.Printf("  Warning: failed to stop service (may not be running): %v\n", err)
	}

	// Disable systemd service
	fmt.Printf("Disabling systemd service: %s\n", target.GoApp.ServiceName)
	if err := client.ExecuteCommandWithSudo(fmt.Sprintf("systemctl disable %s", target.GoApp.ServiceName)); err != nil {
		// Service might not be enabled, continue anyway
		fmt.Printf("  Warning: failed to disable service (may not be enabled): %v\n", err)
	}

	// Remove systemd service file
	serviceFile := fmt.Sprintf("/etc/systemd/system/%s.service", target.GoApp.ServiceName)
	fmt.Printf("Removing systemd service file: %s\n", serviceFile)
	if err := client.ExecuteCommandWithSudo(fmt.Sprintf("rm -f %s", serviceFile)); err != nil {
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	// Reload systemd daemon
	fmt.Printf("Reloading systemd daemon...\n")
	if err := client.ExecuteCommandWithSudo("systemctl daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Remove binary file if requested
	if opts.RemoveFiles && target.GoApp.BinaryPath != "" {
		fmt.Printf("Removing binary file: %s\n", target.GoApp.BinaryPath)
		if err := client.ExecuteCommandWithSudo(fmt.Sprintf("rm -f %s", target.GoApp.BinaryPath)); err != nil {
			return fmt.Errorf("failed to remove binary file: %w", err)
		}
	}

	fmt.Printf("✅ Successfully undeployed Go app service: %s\n", target.GoApp.ServiceName)
	return nil
}
