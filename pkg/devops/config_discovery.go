package devops

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindDeployConfig automatically discovers deploy.yaml in project root.
// Search order:
// 1. Project root (where go.mod exists)
// 2. Current working directory
// 3. Return default "deploy.yaml" if not found (will error when loading)
func FindDeployConfig(customPath string) string {
	// If custom path is provided and is absolute, use it
	if customPath != "" && filepath.IsAbs(customPath) {
		return customPath
	}

	// If custom path is provided and file exists, use it
	if customPath != "" {
		if _, err := os.Stat(customPath); err == nil {
			// File exists, return as-is (will be resolved relative to cwd)
			return customPath
		}
	}

	// Try to find project root
	projectRoot, err := findProjectRootForConfig()
	if err == nil {
		// Check deploy.yaml in project root
		rootConfig := filepath.Join(projectRoot, "deploy.yaml")
		if _, err := os.Stat(rootConfig); err == nil {
			return rootConfig
		}
	}

	// Check current directory
	cwd, err := os.Getwd()
	if err == nil {
		cwdConfig := filepath.Join(cwd, "deploy.yaml")
		if _, err := os.Stat(cwdConfig); err == nil {
			return cwdConfig
		}
	}

	// If custom path was provided, return it (even if doesn't exist - will error later)
	if customPath != "" {
		return customPath
	}

	// Default: return "deploy.yaml" (relative to current directory)
	return "deploy.yaml"
}

// findProjectRootForConfig finds project root for config discovery.
// Uses the same logic as findProjectRoot in env.go but exported for config discovery.
func findProjectRootForConfig() (string, error) {
	// Strategy 1: Start from current working directory
	dir, err := os.Getwd()
	if err == nil {
		if root, found := searchForGoModForConfig(dir); found {
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
		if root, found := searchForGoModForConfig(execDir); found {
			return root, nil
		}
	}

	return "", fmt.Errorf("project root not found (go.mod not found)")
}

// searchForGoModForConfig searches upward from dir for go.mod file.
func searchForGoModForConfig(startDir string) (string, bool) {
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

// ResolveConfigFile resolves a config file path, checking both the original location
// and the repo/ folder in the project root. The repo/ folder is prioritized.
// Search order:
// 1. If path is absolute and exists, use it
// 2. Check in repo/ folder in project root (prioritized)
// 3. If path is relative, check in current directory or project root
// 4. Check in repo/ folder in current directory
// Returns the resolved absolute path if found, or the original path if not found (will error later)
func ResolveConfigFile(configPath string) string {
	// If path is absolute and exists, use it
	if filepath.IsAbs(configPath) {
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
		return configPath // Return as-is even if doesn't exist (will error later)
	}

	// Try to find project root
	projectRoot, err := findProjectRootForConfig()
	if err == nil {
		// Prioritize repo folder in project root
		repoPath := filepath.Join(projectRoot, "repo", configPath)
		if _, err := os.Stat(repoPath); err == nil {
			return repoPath
		}
	}

	// Check in current directory
	cwd, err := os.Getwd()
	if err == nil {
		// Check in repo folder in current directory (prioritized)
		repoPath := filepath.Join(cwd, "repo", configPath)
		if _, err := os.Stat(repoPath); err == nil {
			return repoPath
		}
		
		// Check in current directory
		cwdPath := filepath.Join(cwd, configPath)
		if _, err := os.Stat(cwdPath); err == nil {
			return cwdPath
		}
	}

	// Check in project root (if we found it)
	if err == nil {
		rootPath := filepath.Join(projectRoot, configPath)
		if _, err := os.Stat(rootPath); err == nil {
			return rootPath
		}
	}

	// Return original path (will error when trying to use it)
	return configPath
}
