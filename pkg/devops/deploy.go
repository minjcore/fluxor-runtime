package devops

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"
)

// DeployGoAppOptions contains options for deploying Go application.
type DeployGoAppOptions struct {
	Force bool // Skip confirmation prompts
}

// DeployGoApp builds, transfers, and sets up Go application on VPS.
func DeployGoApp(target DeployTarget, opts DeployGoAppOptions) error {
	// Validate Go app config
	if target.GoApp.Source == "" {
		return fmt.Errorf("go_app source is required")
	}
	if target.GoApp.ServiceName == "" {
		return fmt.Errorf("go_app service_name is required")
	}
	if target.GoApp.BinaryPath == "" {
		return fmt.Errorf("go_app binary_path is required")
	}
	if target.AppDir == "" {
		return fmt.Errorf("app_dir is required")
	}

	// Confirm action unless force
	if !opts.Force {
		fmt.Printf("This will deploy Go app to %s\n", target.Host)
		fmt.Printf("  - Source: %s\n", target.GoApp.Source)
		fmt.Printf("  - Binary: %s\n", target.GoApp.BinaryPath)
		fmt.Printf("  - Service: %s\n", target.GoApp.ServiceName)
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

	// Step 1: Build Go binary
	fmt.Printf("Building Go binary from %s...\n", target.GoApp.Source)
	binaryPath, err := buildGoApp(target.GoApp)
	if err != nil {
		return fmt.Errorf("failed to build Go app: %w", err)
	}
	defer func() {
		// Clean up local binary after deployment
		if err := os.Remove(binaryPath); err != nil {
			fmt.Printf("Warning: failed to remove temp binary: %v\n", err)
		}
	}()

	// Step 2: Create SSH client
	client, err := NewSSHClient(target)
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}
	defer client.Close()

	// Step 3: Create directories on VPS
	fmt.Printf("Creating directories on VPS...\n")
	binaryDir := filepath.Dir(target.GoApp.BinaryPath)
	if err := client.ExecuteCommandWithSudo(fmt.Sprintf("mkdir -p %s", binaryDir)); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Step 4: Transfer binary to VPS
	fmt.Printf("Transferring binary to VPS...\n")
	if err := transferFile(client, binaryPath, target.GoApp.BinaryPath, target); err != nil {
		return fmt.Errorf("failed to transfer binary: %w", err)
	}

	// Step 4b: Transfer config.json if present in source (app needs it in WorkingDirectory)
	sourcePath := target.GoApp.Source
	if !filepath.IsAbs(sourcePath) {
		if wd, err := os.Getwd(); err == nil {
			sourcePath = filepath.Join(wd, sourcePath)
		}
	}
	configDest := filepath.Join(binaryDir, "config.json")
	for _, name := range []string{"config.production.json", "config.json"} {
		localConfig := filepath.Join(sourcePath, name)
		if _, err := os.Stat(localConfig); err == nil {
			fmt.Printf("Transferring %s to VPS...\n", name)
			if err := transferFile(client, localConfig, configDest, target); err != nil {
				return fmt.Errorf("failed to transfer config: %w", err)
			}
			break
		}
	}

	// Step 5: Make binary executable
	fmt.Printf("Making binary executable...\n")
	if err := client.ExecuteCommandWithSudo(fmt.Sprintf("chmod +x %s", target.GoApp.BinaryPath)); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Step 6: Create systemd service file
	fmt.Printf("Creating systemd service file...\n")
	if err := createSystemdService(client, target); err != nil {
		return fmt.Errorf("failed to create systemd service: %w", err)
	}

	// Step 7: Reload systemd and enable service
	fmt.Printf("Reloading systemd daemon...\n")
	if err := client.ExecuteCommandWithSudo("systemctl daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	fmt.Printf("Enabling systemd service: %s\n", target.GoApp.ServiceName)
	if err := client.ExecuteCommandWithSudo(fmt.Sprintf("systemctl enable %s", target.GoApp.ServiceName)); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	// Step 8: Start service
	fmt.Printf("Starting systemd service: %s\n", target.GoApp.ServiceName)
	if err := client.ExecuteCommandWithSudo(fmt.Sprintf("systemctl start %s", target.GoApp.ServiceName)); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	fmt.Printf("✅ Successfully deployed Go app: %s\n", target.GoApp.ServiceName)
	
	// Update state after successful deployment
	stateManager, err := NewStateManager()
	if err == nil {
		serviceState := ServiceState{
			Name:        target.GoApp.ServiceName,
			Type:        "go-app",
			Status:      "running",
			DeployedAt:  time.Now(),
			LastChecked: time.Now(),
		}
		// Try to get PID
		if pidOutput, err := client.ExecuteCommandWithOutput(fmt.Sprintf("systemctl show --property=MainPID --value %s", target.GoApp.ServiceName)); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(pidOutput)); err == nil {
				serviceState.PID = pid
			}
		}
		_ = stateManager.UpdateServiceState(target.Host, target.GoApp.ServiceName, serviceState)
	}
	
	return nil
}

// buildGoApp builds Go application binary for Linux.
func buildGoApp(config GoAppConfig) (string, error) {
	// Determine binary name
	binaryName := config.BinaryName
	if binaryName == "" {
		binaryName = filepath.Base(config.BinaryPath)
	}

	// Create temp directory for build
	tmpDir, err := os.MkdirTemp("", "fluxor-deploy-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	binaryPath := filepath.Join(tmpDir, binaryName)

	// Check if source is a directory with main.go
	sourcePath := config.Source
	if !filepath.IsAbs(sourcePath) {
		// Try to find project root
		wd, err := os.Getwd()
		if err == nil {
			sourcePath = filepath.Join(wd, config.Source)
		}
	}

	// Check if source directory exists
	if info, err := os.Stat(sourcePath); err != nil || !info.IsDir() {
		return "", fmt.Errorf("source directory not found: %s", sourcePath)
	}

	// Build command: GOOS=linux GOARCH=amd64 go build -o <binary> <source>
	buildCmd := exec.Command("go", "build", "-o", binaryPath, sourcePath)
	buildCmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	buildCmd.Dir = sourcePath // Set working directory to source

	output, err := buildCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to build: %w\nOutput: %s", err, string(output))
	}

	// Verify binary was created
	if _, err := os.Stat(binaryPath); err != nil {
		return "", fmt.Errorf("binary was not created: %w", err)
	}

	fmt.Printf("  Built binary: %s\n", binaryPath)
	return binaryPath, nil
}

// transferFile transfers a file to VPS using scp with sshpass.
// Transfers to /tmp first (user can write), then moves to final location with sudo.
func transferFile(client *SSHClient, localPath, remotePath string, target DeployTarget) error {
	// Get password from .env file
	// Use empty string to default to .env.local
	envVars, _ := loadEnvFile("")
	password := ""
	if p, ok := envVars["SSH_PASSWORD"]; ok && p != "" {
		password = p
	} else if p := os.Getenv("SSH_PASSWORD"); p != "" {
		password = p
	}

	// Get SSH key
	sshKey := target.SSHKey
	if sshKey == "" {
		if k, ok := envVars["SSH_KEY"]; ok && k != "" {
			sshKey = k
		} else if k := os.Getenv("SSH_KEY"); k != "" {
			sshKey = k
		}
	}
	if sshKey != "" {
		sshKey = expandPath(sshKey)
	}

	// Use temp location that user can write to (/tmp)
	remoteTempPath := fmt.Sprintf("/tmp/%s", filepath.Base(localPath))

	// Transfer to temp location (user can write to /tmp)
	// Add retry logic for connection issues
	maxRetries := 3
	var lastErr error
	var output []byte
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			// Wait before retry (exponential backoff: 2s, 4s, 8s)
			waitTime := time.Duration(1<<uint(attempt-1)) * time.Second
			fmt.Printf("  Retrying file transfer (attempt %d/%d) after %v...\n", attempt, maxRetries, waitTime)
			time.Sleep(waitTime)
		}
		
		var cmd *exec.Cmd
		if password != "" {
			// Use sshpass with scp, add timeout and connection options
			scpCmd := fmt.Sprintf("scp -o StrictHostKeyChecking=no -o ConnectTimeout=30 -o ServerAliveInterval=60 -o ServerAliveCountMax=3 %s %s@%s:%s", localPath, target.User, target.Host, remoteTempPath)
			cmd = exec.Command("sshpass", "-e", "sh", "-c", scpCmd)
			cmd.Env = append(os.Environ(), fmt.Sprintf("SSHPASS=%s", password))
		} else if sshKey != "" {
			// Use SSH key only — disable password prompt
			scpCmd := fmt.Sprintf("scp -o StrictHostKeyChecking=no -o ConnectTimeout=30 -o ServerAliveInterval=60 -o ServerAliveCountMax=3 -o PreferredAuthentications=publickey -o PasswordAuthentication=no -i %s %s %s@%s:%s", sshKey, localPath, target.User, target.Host, remoteTempPath)
			cmd = exec.Command("sh", "-c", scpCmd)
		} else {
			return fmt.Errorf("no authentication method available for file transfer")
		}

		output, lastErr = cmd.CombinedOutput()
		if lastErr == nil {
			// Success!
			break
		}
		
		// Check if error is connection-related (retryable)
		errStr := string(output)
		if !strings.Contains(errStr, "Connection reset") && 
		   !strings.Contains(errStr, "Connection closed") &&
		   !strings.Contains(errStr, "Connection refused") &&
		   !strings.Contains(errStr, "kex_exchange_identification") {
			// Non-retryable error, return immediately
			return fmt.Errorf("failed to transfer file: %w\nOutput: %s", lastErr, errStr)
		}
	}
	
	if lastErr != nil {
		return fmt.Errorf("failed to transfer file after %d attempts: %w\nOutput: %s", maxRetries, lastErr, string(output))
	}

	// Move from temp to final location with sudo
	moveCmd := fmt.Sprintf("mv %s %s", remoteTempPath, remotePath)
	if err := client.ExecuteCommandWithSudo(moveCmd); err != nil {
		return fmt.Errorf("failed to move file to final location: %w", err)
	}

	// Set ownership
	chownCmd := fmt.Sprintf("chown %s:%s %s", target.User, target.User, remotePath)
	if err := client.ExecuteCommandWithSudo(chownCmd); err != nil {
		return fmt.Errorf("failed to set ownership: %w", err)
	}

	return nil
}

// createSystemdService creates systemd service file on VPS.
func createSystemdService(client *SSHClient, target DeployTarget) error {
	serviceTemplate := `[Unit]
Description={{.ServiceName}}
After=network.target
StartLimitIntervalSec=60
StartLimitBurst=3

[Service]
Type=simple
Restart=on-failure
RestartSec=10
User={{.User}}
Group={{.User}}
WorkingDirectory={{.WorkingDir}}

{{range $key, $value := .Env}}
Environment="{{$key}}={{$value}}"
{{end}}

ExecStart={{.BinaryPath}}
ExecReload=/bin/kill -HUP $MAINPID

StandardOutput=journal
StandardError=journal
SyslogIdentifier={{.ServiceName}}

LimitNOFILE=65536
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
`

	tmpl, err := template.New("service").Parse(serviceTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse service template: %w", err)
	}

	// Prepare template data
	workingDir := filepath.Dir(target.GoApp.BinaryPath)
	env := target.GoApp.Env
	if env == nil {
		env = make(map[string]string)
	}

	data := struct {
		ServiceName string
		User        string
		WorkingDir  string
		BinaryPath  string
		Env         map[string]string
	}{
		ServiceName: target.GoApp.ServiceName,
		User:        target.User,
		WorkingDir:  workingDir,
		BinaryPath:  target.GoApp.BinaryPath,
		Env:         env,
	}

	// Generate service file content
	var serviceContent strings.Builder
	if err := tmpl.Execute(&serviceContent, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Write service file to temp file locally
	tmpFile, err := os.CreateTemp("", "systemd-service-*.service")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(serviceContent.String()); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write service file: %w", err)
	}
	tmpFile.Close()

	// Transfer service file to VPS
	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", target.GoApp.ServiceName)
	if err := transferFile(client, tmpFile.Name(), servicePath, target); err != nil {
		return fmt.Errorf("failed to transfer service file: %w", err)
	}

	return nil
}

// DeployNginxOptions contains options for deploying nginx configuration.
type DeployNginxOptions struct {
	Force bool // Skip confirmation prompts
}

// DeployNginx deploys nginx configuration file to VPS.
func DeployNginx(target DeployTarget, opts DeployNginxOptions) error {
	// Validate nginx config
	if target.Nginx.ConfigSource == "" {
		return fmt.Errorf("nginx config_source is required")
	}
	if target.Nginx.ConfigDest == "" {
		return fmt.Errorf("nginx config_dest is required")
	}
	if target.Nginx.EnabledPath == "" {
		return fmt.Errorf("nginx enabled_path is required")
	}

	// Resolve config file path (check repo folder)
	resolvedConfigPath := ResolveConfigFile(target.Nginx.ConfigSource)
	if _, err := os.Stat(resolvedConfigPath); os.IsNotExist(err) {
		return fmt.Errorf("nginx config source file does not exist: %s (checked: %s)", target.Nginx.ConfigSource, resolvedConfigPath)
	}

	// Confirm action unless force
	if !opts.Force {
		fmt.Printf("This will deploy nginx config to %s\n", target.Host)
		fmt.Printf("  - Source: %s\n", target.Nginx.ConfigSource)
		fmt.Printf("  - Destination: %s\n", target.Nginx.ConfigDest)
		fmt.Printf("  - Enabled Path: %s\n", target.Nginx.EnabledPath)
		fmt.Printf("  - Site Name: %s\n", target.Nginx.SiteName)
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

	// Create SSH client
	client, err := NewSSHClient(target)
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}
	defer client.Close()

	// Transfer nginx config file (use resolved path)
	fmt.Printf("Transferring nginx config to VPS...\n")
	if err := transferFile(client, resolvedConfigPath, target.Nginx.ConfigDest, target); err != nil {
		return fmt.Errorf("failed to transfer nginx config: %w", err)
	}

	// Create symlink in sites-enabled
	fmt.Printf("Creating symlink in sites-enabled...\n")
	// Remove existing symlink if it exists
	if err := client.ExecuteCommandWithSudo(fmt.Sprintf("rm -f %s", target.Nginx.EnabledPath)); err != nil {
		// Continue even if removal fails (might not exist)
	}
	// Create new symlink
	if err := client.ExecuteCommandWithSudo(fmt.Sprintf("ln -s %s %s", target.Nginx.ConfigDest, target.Nginx.EnabledPath)); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	// Test nginx configuration
	fmt.Printf("Testing nginx configuration...\n")
	if err := client.ExecuteCommandWithSudo("nginx -t"); err != nil {
		return fmt.Errorf("nginx config test failed: %w", err)
	}

	// Restart nginx: starts if inactive, applies new config if already running
	fmt.Printf("Restarting nginx service...\n")
	if err := client.ExecuteCommandWithSudo("systemctl restart nginx"); err != nil {
		return fmt.Errorf("failed to restart nginx: %w (on VPS check: systemctl status nginx; if not installed: apt install nginx)", err)
	}

	fmt.Printf("✅ Successfully deployed nginx config for %s\n", target.Nginx.SiteName)
	
	// Update state after successful deployment
	stateManager, err := NewStateManager()
	if err == nil {
		// Update nginx service state
		nginxState := ServiceState{
			Name:        "nginx-" + target.Nginx.SiteName,
			Type:        "nginx",
			Status:      "running",
			DeployedAt:  time.Now(),
			LastChecked: time.Now(),
		}
		// Check if nginx is actually running
		if output, err := client.ExecuteCommandWithOutput("systemctl is-active nginx"); err == nil {
			if strings.TrimSpace(output) != "active" {
				nginxState.Status = "stopped"
			}
		}
		_ = stateManager.UpdateServiceState(target.Host, "nginx-"+target.Nginx.SiteName, nginxState)
		
		// Update config state
		configState := ConfigState{
			Name:       target.Nginx.SiteName,
			Type:       "nginx",
			Path:       target.Nginx.ConfigDest,
			DeployedAt: time.Now(),
			Active:     true,
		}
		_ = stateManager.UpdateConfigState(target.Host, "nginx-"+target.Nginx.SiteName, configState)
	}
	
	return nil
}
