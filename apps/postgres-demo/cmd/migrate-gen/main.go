package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"
)

// ============================================================================
// MAIN ENTRY POINT
// ============================================================================

func main() {
	// Parse command line flags
	var (
		modelsDir  = flag.String("models", "", "Directory containing model files (default: auto-detect)")
		outputDir  = flag.String("output", "migrations", "Output directory for migration files")
		dryRun     = flag.Bool("dry-run", false, "Preview changes without writing files")
		modelNames = flag.String("models-list", "", "Comma-separated list of specific models to generate (empty = all)")
		verbose    = flag.Bool("verbose", false, "Enable verbose output")
	)
	flag.Parse()

	// Initialize configuration
	config := GeneratorConfig{
		OutputDir:   *outputDir,
		DryRun:      *dryRun,
		Timestamp:   time.Now(),
		ModelNames:  parseModelNames(*modelNames),
	}

	// Find models directory
	if *modelsDir == "" {
		var err error
		config.ModelsDir, err = getModelsDirectory()
		if err != nil {
			log.Fatalf("Error finding models directory: %v", err)
		}
	} else {
		absPath, err := filepath.Abs(*modelsDir)
		if err != nil {
			log.Fatalf("Error resolving models directory path: %v", err)
		}
		config.ModelsDir = absPath
	}

	// Validate output directory
	if err := validateOutputDirectory(config.OutputDir); err != nil {
		log.Fatalf("Error validating output directory: %v", err)
	}

	if *verbose {
		fmt.Printf("Models directory: %s\n", config.ModelsDir)
		fmt.Printf("Output directory: %s\n", config.OutputDir)
		fmt.Printf("Dry run: %v\n", config.DryRun)
		fmt.Printf("\n")
	}

	// Parse models
	fmt.Printf("Parsing models from: %s\n", config.ModelsDir)
	models, err := parseModelsDirectory(config.ModelsDir)
	if err != nil {
		log.Fatalf("Error parsing models: %v", err)
	}

	if len(models) == 0 {
		log.Fatalf("No models found in %s", config.ModelsDir)
	}

	fmt.Printf("Found %d model(s)\n", len(models))

	// Filter models if specified
	filteredModels := filterModels(models, config)
	if len(filteredModels) == 0 {
		log.Fatalf("No models match the filter criteria")
	}

	// Validate models
	for i := range filteredModels {
		if err := validateModel(&filteredModels[i]); err != nil {
			log.Printf("Warning: %v", err)
		}
	}

	// Generate SQL
	fmt.Printf("Generating SQL for %d model(s)...\n", len(filteredModels))
	sqlContent := generateMigrationFileContent(filteredModels, config)

	// Generate filename
	filename := generateMigrationFileName(config, getModelNames(filteredModels))

	// Compare with existing (if not dry run)
	if !config.DryRun {
		isSame, existingFile, err := compareWithExisting(config.OutputDir, sqlContent)
		if err == nil && isSame {
			fmt.Printf("✓ Generated SQL matches existing migration: %s\n", existingFile)
			fmt.Printf("  No new migration file created.\n")
			return
		}
	}

	// Write migration file
	if err := writeMigrationFile(config, sqlContent, filename); err != nil {
		log.Fatalf("Error writing migration file: %v", err)
	}

	// Print summary
	printMigrationSummary(filteredModels, filename, config)

	if config.DryRun {
		fmt.Printf("\n[DRY RUN] Migration content preview:\n")
		fmt.Printf("=====================================\n")
		// Print first 50 lines
		lines := strings.Split(sqlContent, "\n")
		maxLines := 50
		if len(lines) < maxLines {
			maxLines = len(lines)
		}
		for i := 0; i < maxLines; i++ {
			fmt.Println(lines[i])
		}
		if len(lines) > maxLines {
			fmt.Printf("... (%d more lines)\n", len(lines)-maxLines)
		}
	}
}

// parseModelNames parses comma-separated model names
func parseModelNames(input string) []string {
	if input == "" {
		return []string{}
	}

	names := strings.Split(input, ",")
	var result []string
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			result = append(result, name)
		}
	}
	return result
}

// getModelNames extracts model names from model info slice
func getModelNames(models []ModelInfo) []string {
	var names []string
	for _, model := range models {
		names = append(names, model.Name)
	}
	return names
}
