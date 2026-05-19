package devops

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// loadEnvFile loads environment variables from a specified .env file.
// If envFilePath is empty, defaults to .env.local in project root.
func loadEnvFile(envFilePath string) (map[string]string, error) {
	envVars := make(map[string]string)

	var envPath string
	if envFilePath != "" {
		// Use specified path (can be absolute or relative)
		envPath = envFilePath
		if !filepath.IsAbs(envPath) {
			// If relative, resolve from current directory
			if absPath, err := filepath.Abs(envPath); err == nil {
				envPath = absPath
			}
		}
	} else {
		// Default: Find project root (look for go.mod)
		rootDir, err := findProjectRoot()
		if err != nil {
			// If can't find project root, try current directory
			rootDir, _ = os.Getwd()
		}
		// Look for .env.local in project root
		envPath = filepath.Join(rootDir, ".env.local")
	}
	
	// Debug: check if file exists
	if _, statErr := os.Stat(envPath); statErr != nil {
		// .env.local is optional, return empty map if not found
		// Try to find it in current directory as fallback
		// If using default path and file not found, try current directory as fallback
		if envFilePath == "" {
			if cwd, wdErr := os.Getwd(); wdErr == nil {
				fallbackPath := filepath.Join(cwd, ".env.local")
				if _, fallbackErr := os.Stat(fallbackPath); fallbackErr == nil {
					envPath = fallbackPath
				} else {
					// File not found, return empty map (silent - .env file is optional)
					return envVars, nil
				}
			} else {
				return envVars, nil
			}
		} else {
			// Specified file not found, return empty map
			return envVars, nil
		}
	}
	
	envFile, err := os.Open(envPath)
	if err != nil {
		// .env.local is optional, return empty map if not found
		return envVars, nil
	}
	defer envFile.Close()
	
	// Debug: log that we found and are reading .env.local (only if file exists)
	// Note: We don't log the actual values for security

	scanner := bufio.NewScanner(envFile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Remove quotes if present
			if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'')) {
				value = value[1 : len(value)-1]
			}

			envVars[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading .env.local: %w", err)
	}

	return envVars, nil
}

// findProjectRoot finds the project root by looking for go.mod file.
// Tries multiple strategies:
// 1. Start from current working directory
// 2. If not found, try from executable's directory
func findProjectRoot() (string, error) {
	// Strategy 1: Start from current working directory
	dir, err := os.Getwd()
	if err == nil {
		if root, found := searchForGoMod(dir); found {
			return root, nil
		}
	}

	// Strategy 2: Try from executable's directory
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		// Resolve symlinks
		if resolved, err := filepath.EvalSymlinks(execDir); err == nil {
			execDir = resolved
		}
		if root, found := searchForGoMod(execDir); found {
			return root, nil
		}
	}

	return "", fmt.Errorf("project root not found (go.mod not found)")
}

// searchForGoMod searches upward from dir for go.mod file.
func searchForGoMod(startDir string) (string, bool) {
	dir := startDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}
	return "", false
}

// ApplyEnvOverrides applies environment variable overrides to DeployTarget.
// Reads from .env file (defaults to .env.local) and environment variables.
// Environment variables take precedence over .env file.
// Priority: Environment variables > .env file > YAML config
//
// Reads from .env file with format:
//   SSH_HOST=hostname
//   SSH_USER=username
//   SSH_PASSWORD=password (optional, for password-based auth)
func ApplyEnvOverrides(target *DeployTarget, targetName string, envFilePath string) error {
	// Load .env file
	envVars, err := loadEnvFile(envFilePath)
	if err != nil {
		envFileName := ".env.local"
		if envFilePath != "" {
			envFileName = envFilePath
		}
		return fmt.Errorf("failed to load .env file (%s): %w", envFileName, err)
	}
	
	// Debug: log if .env file was loaded (but not the values for security)
	if len(envVars) > 0 {
		envFileName := ".env.local"
		if envFilePath != "" {
			envFileName = filepath.Base(envFilePath)
		}
		fmt.Printf("[Deploy] Loaded %d variables from %s\n", len(envVars), envFileName)
	}

	// Check for host override from .env.local (SSH_HOST)
	// Priority: Environment variable SSH_HOST > .env.local SSH_HOST > YAML config
	if host := os.Getenv("SSH_HOST"); host != "" {
		target.Host = host
	} else if host, ok := envVars["SSH_HOST"]; ok && host != "" {
		target.Host = host
	}

	// Check for user override from .env.local (SSH_USER)
	// Priority: Environment variable SSH_USER > .env.local SSH_USER > YAML config
	if user := os.Getenv("SSH_USER"); user != "" {
		target.User = user
	} else if user, ok := envVars["SSH_USER"]; ok && user != "" {
		target.User = user
	}

	// Check for ssh_key override
	// Note: SSH_PASSWORD is available in envVars but not used for SSH key auth
	// SSH key path can be overridden via SSH_KEY environment variable or .env.local
	if sshKey := os.Getenv("SSH_KEY"); sshKey != "" {
		target.SSHKey = sshKey
	} else if sshKey, ok := envVars["SSH_KEY"]; ok && sshKey != "" {
		target.SSHKey = sshKey
	}

	// Certbot email (for deploy -certbot)
	if email := os.Getenv("CERTBOT_EMAIL"); email != "" {
		target.Certbot.Email = email
	} else if email, ok := envVars["CERTBOT_EMAIL"]; ok && email != "" {
		target.Certbot.Email = email
	}

	return nil
}
