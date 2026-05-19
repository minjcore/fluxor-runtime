package devops

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DeployNodeAppOptions contains options for deploying Node.js application.
type DeployNodeAppOptions struct {
	Force bool // Skip confirmation prompts
}

// DeployNodeApp builds Node.js app and deploys HTML files and assets to VPS.
func DeployNodeApp(target DeployTarget, opts DeployNodeAppOptions) error {
	// Validate Node app config
	if target.NodeApp.Source == "" {
		return fmt.Errorf("node_app source is required")
	}
	if target.NodeApp.BuildOutput == "" {
		return fmt.Errorf("node_app build_output is required")
	}
	if target.NodeApp.DestDir == "" {
		return fmt.Errorf("node_app dest_dir is required")
	}
	if target.AppDir == "" {
		return fmt.Errorf("app_dir is required")
	}

	// Confirm action unless force
	if !opts.Force {
		fmt.Printf("This will deploy Node.js app HTML files and assets to %s\n", target.Host)
		fmt.Printf("  - Source: %s\n", target.NodeApp.Source)
		fmt.Printf("  - Build Output: %s\n", target.NodeApp.BuildOutput)
		fmt.Printf("  - Destination: %s\n", target.NodeApp.DestDir)
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

	// Build Node.js app
	fmt.Printf("Building Node.js app from %s...\n", target.NodeApp.Source)
	if err := buildNodeApp(target.NodeApp.Source); err != nil {
		return fmt.Errorf("failed to build Node.js app: %w", err)
	}

	// Find HTML files in build output
	buildOutputPath := filepath.Join(target.NodeApp.Source, target.NodeApp.BuildOutput)
	htmlFiles, err := findHTMLFiles(buildOutputPath)
	if err != nil {
		return fmt.Errorf("failed to find HTML files: %w", err)
	}

	if len(htmlFiles) == 0 {
		return fmt.Errorf("no HTML files found in %s", buildOutputPath)
	}

	fmt.Printf("  Found %d HTML file(s)\n", len(htmlFiles))

	// Find assets folder
	assetsPath, err := findAssetsFolder(buildOutputPath)
	if err == nil {
		// Count files in assets folder
		assetCount, err := countFilesInDirectory(assetsPath)
		if err == nil {
			fmt.Printf("  Found assets folder with %d file(s)\n", assetCount)
		}
	}

	// Create SSH client
	client, err := NewSSHClient(target)
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}
	defer client.Close()

	// Create destination directory on VPS
	fmt.Printf("Creating directories on VPS...\n")
	if err := client.ExecuteCommandWithSudo(fmt.Sprintf("mkdir -p %s", target.NodeApp.DestDir)); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Transfer HTML files
	fmt.Printf("Transferring HTML files to VPS...\n")
	for _, htmlFile := range htmlFiles {
		// Get relative path from build output directory
		relPath, err := filepath.Rel(buildOutputPath, htmlFile)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Destination path on VPS
		destPath := filepath.Join(target.NodeApp.DestDir, relPath)

		// Create parent directory on VPS if needed
		destDir := filepath.Dir(destPath)
		if destDir != target.NodeApp.DestDir {
			if err := client.ExecuteCommandWithSudo(fmt.Sprintf("mkdir -p %s", destDir)); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}
		}

		// Transfer file
		fmt.Printf("  Transferring %s -> %s\n", relPath, destPath)
		if err := transferFile(client, htmlFile, destPath, target); err != nil {
			return fmt.Errorf("failed to transfer %s: %w", htmlFile, err)
		}
	}

	// Transfer assets folder if it exists
	if assetsPath != "" {
		fmt.Printf("Transferring assets to VPS...\n")
		if err := transferAssetsFolder(client, assetsPath, target.NodeApp.DestDir, target); err != nil {
			return fmt.Errorf("failed to transfer assets folder: %w", err)
		}
		fmt.Printf("✅ Successfully deployed Node.js app HTML files and assets\n")
		} else {
			fmt.Printf("✅ Successfully deployed Node.js app HTML files\n")
		}

		// Update state after successful deployment
		stateManager, err := NewStateManager()
		if err == nil {
			configState := ConfigState{
				Name:       "node-app",
				Type:       "node-app",
				Path:       target.NodeApp.DestDir,
				DeployedAt: time.Now(),
				Active:     true,
			}
			_ = stateManager.UpdateConfigState(target.Host, "node-app", configState)
		}

		return nil
	}

// buildNodeApp builds the Node.js application.
func buildNodeApp(sourceDir string) error {
	// Check if source directory exists
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", sourceDir)
	}

	// Check if package.json exists
	packageJSON := filepath.Join(sourceDir, "package.json")
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		return fmt.Errorf("package.json not found in %s", sourceDir)
	}

	// Run npm install if node_modules doesn't exist
	nodeModules := filepath.Join(sourceDir, "node_modules")
	if _, err := os.Stat(nodeModules); os.IsNotExist(err) {
		fmt.Printf("  Installing dependencies...\n")
		cmd := exec.Command("npm", "install")
		cmd.Dir = sourceDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to install dependencies: %w\nOutput: %s", err, string(output))
		}
	}

	// Run npm run build
	fmt.Printf("  Running npm run build...\n")
	cmd := exec.Command("npm", "run", "build")
	cmd.Dir = sourceDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to build: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// findHTMLFiles finds all HTML files in the given directory recursively.
func findHTMLFiles(dir string) ([]string, error) {
	var htmlFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if file has .html extension
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".html") {
			htmlFiles = append(htmlFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return htmlFiles, nil
}

// findAssetsFolder finds the assets folder in the build output directory.
func findAssetsFolder(buildOutputPath string) (string, error) {
	assetsPath := filepath.Join(buildOutputPath, "assets")
	
	info, err := os.Stat(assetsPath)
	if err != nil {
		return "", err
	}
	
	if !info.IsDir() {
		return "", fmt.Errorf("assets path exists but is not a directory")
	}
	
	return assetsPath, nil
}

// countFilesInDirectory counts the number of files (not directories) in a directory recursively.
func countFilesInDirectory(dir string) (int, error) {
	count := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	return count, err
}

// transferAssetsFolder recursively transfers all files in the assets folder to VPS.
func transferAssetsFolder(client *SSHClient, assetsPath, destDir string, target DeployTarget) error {
	// Ensure assets directory exists on VPS
	assetsDestDir := filepath.Join(destDir, "assets")
	if err := client.ExecuteCommandWithSudo(fmt.Sprintf("mkdir -p %s", assetsDestDir)); err != nil {
		return fmt.Errorf("failed to create assets directory: %w", err)
	}

	// Walk through all files in assets folder
	err := filepath.Walk(assetsPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path from assets folder
		relPath, err := filepath.Rel(assetsPath, filePath)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Destination path on VPS
		destPath := filepath.Join(assetsDestDir, relPath)

		// Create parent directory on VPS if needed
		destParentDir := filepath.Dir(destPath)
		if destParentDir != assetsDestDir {
			if err := client.ExecuteCommandWithSudo(fmt.Sprintf("mkdir -p %s", destParentDir)); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}
		}

		// Transfer file
		fmt.Printf("  Transferring assets/%s -> %s\n", relPath, destPath)
		if err := transferFile(client, filePath, destPath, target); err != nil {
			return fmt.Errorf("failed to transfer %s: %w", filePath, err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
