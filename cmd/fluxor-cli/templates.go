package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates/service/* templates/app/*
var templateFS embed.FS

// TemplateType represents a scaffold template type
type TemplateType string

const (
	TemplateTypeService TemplateType = "service"
	TemplateTypeApp     TemplateType = "app"
	TemplateTypeWorkflow TemplateType = "workflow"
)

// TemplateData holds data for template execution
type TemplateData struct {
	ServiceName     string
	ServiceNameLower string
	AppName         string
	Name            string
}

// TemplateLoader loads and executes templates
type TemplateLoader struct {
	templateType TemplateType
	templateFS   fs.FS
}

// NewTemplateLoader creates a new template loader
func NewTemplateLoader(templateType TemplateType) *TemplateLoader {
	// Get the subdirectory for this template type
	subFS, err := fs.Sub(templateFS, "templates/"+string(templateType))
	if err != nil {
		// If subdirectory doesn't exist, use empty FS
		subFS = templateFS
	}

	return &TemplateLoader{
		templateType: templateType,
		templateFS:   subFS,
	}
}

// ExecuteTemplate executes a template file and writes to destination
func (tl *TemplateLoader) ExecuteTemplate(templateFile, destPath string, data TemplateData) error {
	// Read template file
	content, err := fs.ReadFile(tl.templateFS, templateFile)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", templateFile, err)
	}

	// Parse template
	tmpl, err := template.New(templateFile).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templateFile, err)
	}

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", destPath, err)
	}
	defer destFile.Close()

	// Execute template
	if err := tmpl.Execute(destFile, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templateFile, err)
	}

	return nil
}

// ExecuteAllTemplates executes all templates in the template directory
func (tl *TemplateLoader) ExecuteAllTemplates(destDir string, data TemplateData) error {
	return fs.WalkDir(tl.templateFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Skip non-template files
		if !hasTemplateExtension(path) {
			return nil
		}

		// Determine destination filename (remove .tmpl extension)
		destFileName := removeTemplateExtension(path)
		destPath := filepath.Join(destDir, destFileName)

		// Execute template
		if err := tl.ExecuteTemplate(path, destPath, data); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", path, err)
		}

		// Set restrictive permissions for config files
		if destFileName == "application.properties" || destFileName == "config.json" || destFileName == "go.mod" {
			if err := os.Chmod(destPath, 0600); err != nil {
				return fmt.Errorf("failed to set permissions for %s: %w", destPath, err)
			}
		}

		return nil
	})
}

// hasTemplateExtension checks if a file has a template extension
func hasTemplateExtension(filename string) bool {
	return filepath.Ext(filename) == ".tmpl"
}

// removeTemplateExtension removes the .tmpl extension from a filename
func removeTemplateExtension(filename string) string {
	if hasTemplateExtension(filename) {
		return filename[:len(filename)-5] // Remove .tmpl (5 characters)
	}
	return filename
}

// ListTemplates lists available template types
func ListTemplates() ([]string, error) {
	entries, err := fs.ReadDir(templateFS, "templates")
	if err != nil {
		return nil, fmt.Errorf("failed to read templates directory: %w", err)
	}

	var templates []string
	for _, entry := range entries {
		if entry.IsDir() {
			templates = append(templates, entry.Name())
		}
	}

	return templates, nil
}
