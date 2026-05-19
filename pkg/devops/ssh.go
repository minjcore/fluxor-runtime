package devops

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SSHClient wraps SSH command execution using sshpass.
type SSHClient struct {
	host     string
	user     string
	password string
	sshKey   string
}

// NewSSHClient creates a new SSH client. Prefers SSH key over password when key file exists.
// When using password, requires sshpass (brew install sshpass).
func NewSSHClient(target DeployTarget) (*SSHClient, error) {
	envVars, _ := loadEnvFile("")
	password := ""
	if p, ok := envVars["SSH_PASSWORD"]; ok && p != "" {
		password = p
	} else if p := os.Getenv("SSH_PASSWORD"); p != "" {
		password = p
	}

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

	if password == "" && sshKey == "" {
		return nil, fmt.Errorf("no authentication method: set SSH_PASSWORD in .env.local (use sshpass) or deploy.yaml ssh_key")
	}
	if password != "" {
		if _, err := exec.LookPath("sshpass"); err != nil {
			return nil, fmt.Errorf("sshpass required for password auth: brew install sshpass (macOS)")
		}
	}

	return &SSHClient{
		host:     target.Host,
		user:     target.User,
		password: password,
		sshKey:   sshKey,
	}, nil
}

// Close closes the SSH connection (no-op for sshpass, but kept for compatibility).
func (c *SSHClient) Close() error {
	// sshpass doesn't maintain persistent connections, so nothing to close
	return nil
}

// ExecuteCommandWithSudo executes a command with sudo privileges via SSH using sshpass.
func (c *SSHClient) ExecuteCommandWithSudo(cmd string) error {
	sudoCmd := fmt.Sprintf("sudo %s", cmd)
	return c.executeSSHCommand(sudoCmd)
}

// ExecuteCommand executes a command without sudo via SSH using sshpass.
func (c *SSHClient) ExecuteCommand(cmd string) error {
	return c.executeSSHCommand(cmd)
}

// executeSSHCommand executes an SSH command using sshpass with retry logic.
func (c *SSHClient) executeSSHCommand(cmd string) error {
	maxRetries := 3
	var lastErr error
	var output []byte
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			// Wait before retry (exponential backoff: 2s, 4s)
			waitTime := time.Duration(1<<uint(attempt-1)) * time.Second
			time.Sleep(waitTime)
		}
		
		var sshCmd *exec.Cmd

		if c.password != "" {
			// Use password authentication with sshpass
			// Add timeout and connection options for better reliability
			sshArgs := []string{
				"-o", "StrictHostKeyChecking=no",
				"-o", "ConnectTimeout=30",
				"-o", "ServerAliveInterval=60",
				"-o", "ServerAliveCountMax=3",
				fmt.Sprintf("%s@%s", c.user, c.host),
				cmd,
			}
			sshCmd = exec.Command("sshpass", "-e", "ssh")
			sshCmd.Args = append(sshCmd.Args, sshArgs...)
			sshCmd.Env = append(os.Environ(), fmt.Sprintf("SSHPASS=%s", c.password))
		} else if c.sshKey != "" {
			// Use SSH key only — disable password prompt so we fail fast if key not accepted
			sshArgs := []string{
				"-o", "StrictHostKeyChecking=no",
				"-o", "ConnectTimeout=30",
				"-o", "ServerAliveInterval=60",
				"-o", "ServerAliveCountMax=3",
				"-o", "PreferredAuthentications=publickey",
				"-o", "PasswordAuthentication=no",
				"-i", c.sshKey,
				fmt.Sprintf("%s@%s", c.user, c.host),
				cmd,
			}
			sshCmd = exec.Command("ssh", sshArgs...)
		} else {
			return fmt.Errorf("no authentication method available")
		}

		// Execute command
		output, lastErr = sshCmd.CombinedOutput()
		if lastErr == nil {
			// Success!
			return nil
		}
		
		// Check if error is connection-related (retryable)
		errStr := string(output)
		if !strings.Contains(errStr, "Connection reset") && 
		   !strings.Contains(errStr, "Connection closed") &&
		   !strings.Contains(errStr, "Connection refused") &&
		   !strings.Contains(errStr, "kex_exchange_identification") {
			// Non-retryable error, return immediately
			return fmt.Errorf("SSH command failed: %w\nOutput: %s", lastErr, errStr)
		}
	}
	
	// All retries failed
	return fmt.Errorf("SSH command failed after %d attempts: %w\nOutput: %s", maxRetries, lastErr, string(output))
}

// ExecuteCommandWithOutput executes a command and returns output
func (c *SSHClient) ExecuteCommandWithOutput(cmd string) (string, error) {
	maxRetries := 3
	var lastErr error
	var output []byte
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			waitTime := time.Duration(1<<uint(attempt-1)) * time.Second
			time.Sleep(waitTime)
		}
		
		var sshCmd *exec.Cmd

		if c.password != "" {
			sshArgs := []string{
				"-o", "StrictHostKeyChecking=no",
				"-o", "ConnectTimeout=30",
				"-o", "ServerAliveInterval=60",
				"-o", "ServerAliveCountMax=3",
				fmt.Sprintf("%s@%s", c.user, c.host),
				cmd,
			}
			sshCmd = exec.Command("sshpass", "-e", "ssh")
			sshCmd.Args = append(sshCmd.Args, sshArgs...)
			sshCmd.Env = append(os.Environ(), fmt.Sprintf("SSHPASS=%s", c.password))
		} else if c.sshKey != "" {
			sshArgs := []string{
				"-o", "StrictHostKeyChecking=no",
				"-o", "ConnectTimeout=30",
				"-o", "ServerAliveInterval=60",
				"-o", "ServerAliveCountMax=3",
				"-o", "PreferredAuthentications=publickey",
				"-o", "PasswordAuthentication=no",
				"-i", c.sshKey,
				fmt.Sprintf("%s@%s", c.user, c.host),
				cmd,
			}
			sshCmd = exec.Command("ssh", sshArgs...)
		} else {
			return "", fmt.Errorf("no authentication method available")
		}

		output, lastErr = sshCmd.CombinedOutput()
		if lastErr == nil {
			return string(output), nil
		}
		
		errStr := string(output)
		if !strings.Contains(errStr, "Connection reset") && 
		   !strings.Contains(errStr, "Connection closed") &&
		   !strings.Contains(errStr, "Connection refused") &&
		   !strings.Contains(errStr, "kex_exchange_identification") {
			return "", fmt.Errorf("SSH command failed: %w\nOutput: %s", lastErr, errStr)
		}
	}
	
	return "", fmt.Errorf("SSH command failed after %d attempts: %w\nOutput: %s", maxRetries, lastErr, string(output))
}

// expandPath expands a path that may start with ~ to the user's home directory.
func expandPath(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
