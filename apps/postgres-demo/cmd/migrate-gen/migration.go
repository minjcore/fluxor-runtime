package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ============================================================================
// MIGRATION FILE MANAGEMENT
// ============================================================================

// generateMigrationFileName generates a migration file name with timestamp
func generateMigrationFileName(config GeneratorConfig, modelNames []string) string {
	timestamp := config.Timestamp.Format("20060102150405")
	
	var nameParts []string
	if len(modelNames) > 0 {
		// Use model names if specified
		nameParts = append(nameParts, strings.ToLower(strings.Join(modelNames, "_")))
	} else {
		nameParts = append(nameParts, "all_models")
	}
	
	nameParts = append(nameParts, "generated")
	
	filename := fmt.Sprintf("%s_%s.sql", timestamp, strings.Join(nameParts, "_"))
	return filename
}

// writeMigrationFile writes migration SQL to a file
func writeMigrationFile(config GeneratorConfig, content string, filename string) error {
	outputPath := filepath.Join(config.OutputDir, filename)

	if config.DryRun {
		fmt.Printf("[DRY RUN] Would write migration to: %s\n", outputPath)
		fmt.Printf("[DRY RUN] File size: %d bytes\n", len(content))
		return nil
	}

	// Ensure output directory exists
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write migration file: %w", err)
	}

	fmt.Printf("✓ Generated migration: %s\n", outputPath)
	return nil
}

// listExistingMigrations lists existing migration files in the output directory
func listExistingMigrations(outputDir string) ([]string, error) {
	files, err := os.ReadDir(outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read output directory: %w", err)
	}

	var migrations []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".sql") {
			migrations = append(migrations, file.Name())
		}
	}

	sort.Strings(migrations)
	return migrations, nil
}

// compareWithExisting compares generated SQL with existing migrations
func compareWithExisting(outputDir string, newContent string) (bool, string, error) {
	existing, err := listExistingMigrations(outputDir)
	if err != nil {
		return false, "", err
	}

	if len(existing) == 0 {
		return false, "", nil // No existing migrations to compare
	}

	// Read the most recent migration
	latestMigration := existing[len(existing)-1]
	latestPath := filepath.Join(outputDir, latestMigration)

	existingContent, err := os.ReadFile(latestPath)
	if err != nil {
		return false, "", fmt.Errorf("failed to read existing migration: %w", err)
	}

	// Simple comparison (in production, you might want more sophisticated diff)
	if strings.TrimSpace(string(existingContent)) == strings.TrimSpace(newContent) {
		return true, latestMigration, nil
	}

	return false, latestMigration, nil
}

// filterModels filters models based on config
func filterModels(models []ModelInfo, config GeneratorConfig) []ModelInfo {
	if len(config.ModelNames) == 0 {
		return models
	}

	var filtered []ModelInfo
	nameMap := make(map[string]bool)
	for _, name := range config.ModelNames {
		nameMap[name] = true
	}

	for _, model := range models {
		if nameMap[model.Name] {
			filtered = append(filtered, model)
		}
	}

	return filtered
}

// validateOutputDirectory validates that output directory is accessible
func validateOutputDirectory(outputDir string) error {
	// Check if directory exists
	info, err := os.Stat(outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist, will be created
			return nil
		}
		return fmt.Errorf("failed to access output directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("output path is not a directory: %s", outputDir)
	}

	return nil
}

// getModelsDirectory finds the models directory relative to current working directory
func getModelsDirectory() (string, error) {
	// Try common locations
	possiblePaths := []string{
		"models",
		"../models",
		"../../models",
		"apps/postgres-demo/models",
	}

	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			absPath, err := filepath.Abs(path)
			if err == nil {
				return absPath, nil
			}
		}
	}

	return "", fmt.Errorf("models directory not found. Tried: %v", possiblePaths)
}

// printMigrationSummary prints a summary of generated migrations
func printMigrationSummary(models []ModelInfo, filename string, config GeneratorConfig) {
	fmt.Printf("\n")
	fmt.Printf("Migration Generation Summary\n")
	fmt.Printf("============================\n")
	fmt.Printf("Models processed: %d\n", len(models))
	fmt.Printf("Output file: %s\n", filename)
	fmt.Printf("Output directory: %s\n", config.OutputDir)
	
	if config.DryRun {
		fmt.Printf("Mode: DRY RUN (no files written)\n")
	} else {
		fmt.Printf("Mode: WRITE\n")
	}

	fmt.Printf("\nGenerated tables:\n")
	for _, model := range models {
		fmt.Printf("  - %s (%s)\n", model.TableName, model.Name)
	}
	fmt.Printf("\n")
}
