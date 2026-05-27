package main

import (
	"fmt"
	"strings"
)

// ============================================================================
// SQL GENERATION FUNCTIONS
// ============================================================================

// generateCreateTable generates CREATE TABLE SQL statement
func generateCreateTable(model *ModelInfo) string {
	var sb strings.Builder

	// Header comment
	sb.WriteString("-- ============================================================================\n")
	sb.WriteString(fmt.Sprintf("-- CREATE %s TABLE\n", strings.ToUpper(model.TableName)))
	if model.Comment != "" {
		// Use first line of comment as description
		lines := strings.Split(model.Comment, "\n")
		if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
			sb.WriteString(fmt.Sprintf("-- %s\n", strings.TrimSpace(lines[0])))
		}
	}
	sb.WriteString("-- ============================================================================\n\n")

	// CREATE TABLE statement
	sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", model.TableName))

	// Generate column definitions
	var columnDefs []string
	for i, field := range model.Fields {
		colDef := generateColumnDefinition(field, model, i == len(model.Fields)-1)
		columnDefs = append(columnDefs, colDef)
	}

	sb.WriteString(strings.Join(columnDefs, ",\n"))
	sb.WriteString("\n);\n\n")

	// Generate indexes
	if len(model.Indexes) > 0 {
		for _, idx := range model.Indexes {
			sb.WriteString(generateIndex(model.TableName, idx))
			sb.WriteString("\n")
		}
	}

	// Generate foreign keys
	if len(model.ForeignKeys) > 0 {
		for _, fk := range model.ForeignKeys {
			sb.WriteString(generateForeignKey(model.TableName, fk))
			sb.WriteString("\n")
		}
	}

	// Generate comments
	sb.WriteString(generateTableComments(model))

	return sb.String()
}

// generateColumnDefinition generates a single column definition
func generateColumnDefinition(field FieldInfo, model *ModelInfo, isLast bool) string {
	var parts []string

	// Column name
	parts = append(parts, fmt.Sprintf("    %s", field.ColumnName))

	// SQL type
	sqlType := field.SQLType
	if field.IsNullable && !strings.Contains(strings.ToLower(field.Comment), "not null") {
		// Type is already correct for nullable
	} else if !field.IsNullable {
		// Ensure NOT NULL for non-nullable fields
	}

	parts = append(parts, sqlType)

	// Primary key - always add PRIMARY KEY constraint
	if field.IsPrimaryKey && len(model.PrimaryKey) == 1 {
		parts = append(parts, "PRIMARY KEY")
	}

	// NOT NULL constraint
	if isNotNull(&field) && !field.IsPrimaryKey {
		parts = append(parts, "NOT NULL")
	}

	// Default value
	if defaultValue := getDefaultValue(&field); defaultValue != "" {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", defaultValue))
	}

	// Inline comment
	if field.Comment != "" && !strings.Contains(field.Comment, "Table:") {
		// Add comment as SQL comment
		comment := strings.TrimSpace(field.Comment)
		// Truncate long comments
		if len(comment) > 60 {
			comment = comment[:57] + "..."
		}
		parts = append(parts, fmt.Sprintf("-- %s", comment))
	}

	return strings.Join(parts, " ")
}

// generateIndex generates CREATE INDEX statement
func generateIndex(tableName string, idx IndexInfo) string {
	var sb strings.Builder

	if idx.Unique {
		columns := strings.Join(idx.Columns, ", ")
		sb.WriteString(fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s(%s);", 
			idx.Name, tableName, columns))
	} else {
		columns := strings.Join(idx.Columns, ", ")
		sb.WriteString(fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s);", 
			idx.Name, tableName, columns))
	}

	return sb.String()
}

// generateForeignKey generates ALTER TABLE ADD FOREIGN KEY statement
func generateForeignKey(tableName string, fk ForeignKeyInfo) string {
	onDelete := fk.OnDelete
	if onDelete == "" {
		onDelete = "RESTRICT"
	}

	onUpdate := fk.OnUpdate
	if onUpdate == "" {
		onUpdate = "RESTRICT"
	}

	constraintName := fmt.Sprintf("%s_%s_fkey", tableName, fk.Column)

	return fmt.Sprintf(
		"ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s) ON DELETE %s ON UPDATE %s;",
		tableName, constraintName, fk.Column, fk.RefTable, fk.RefColumn, onDelete, onUpdate,
	)
}

// generateTableComments generates COMMENT ON statements
func generateTableComments(model *ModelInfo) string {
	var sb strings.Builder

	// Table comment
	if model.Comment != "" {
		// Use first meaningful line
		lines := strings.Split(model.Comment, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.Contains(line, "Table:") {
				// Escape single quotes
				comment := strings.ReplaceAll(line, "'", "''")
				sb.WriteString(fmt.Sprintf("COMMENT ON TABLE %s IS '%s';\n", 
					model.TableName, comment))
				break
			}
		}
	}

	// Column comments
	for _, field := range model.Fields {
		if field.Comment != "" && !strings.Contains(field.Comment, "Table:") {
			comment := strings.TrimSpace(field.Comment)
			// Escape single quotes
			comment = strings.ReplaceAll(comment, "'", "''")
			// Truncate if too long
			if len(comment) > 200 {
				comment = comment[:197] + "..."
			}
			sb.WriteString(fmt.Sprintf("COMMENT ON COLUMN %s.%s IS '%s';\n",
				model.TableName, field.ColumnName, comment))
		}
	}

	return sb.String()
}

// generateCompositePrimaryKey generates PRIMARY KEY constraint for composite keys
func generateCompositePrimaryKey(tableName string, columns []string) string {
	if len(columns) <= 1 {
		return ""
	}

	cols := strings.Join(columns, ", ")
	return fmt.Sprintf("ALTER TABLE %s ADD PRIMARY KEY (%s);", tableName, cols)
}

// formatSQL formats SQL with proper indentation (basic version)
func formatSQL(sql string) string {
	// Basic formatting - in production, you might want a more sophisticated formatter
	lines := strings.Split(sql, "\n")
	var formatted []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			formatted = append(formatted, trimmed)
		}
	}

	return strings.Join(formatted, "\n")
}

// generateMigrationFileContent generates complete migration file content
func generateMigrationFileContent(models []ModelInfo, config GeneratorConfig) string {
	var sb strings.Builder

	// File header
	sb.WriteString("-- ============================================================================\n")
	sb.WriteString("-- GENERATED MIGRATION FILE\n")
	sb.WriteString("-- Generated from Go models - DO NOT EDIT MANUALLY\n")
	sb.WriteString(fmt.Sprintf("-- Generated at: %s\n", config.Timestamp.Format("2006-01-02 15:04:05")))
	sb.WriteString("-- ============================================================================\n\n")

	// Generate table for each model
	for i, model := range models {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(generateCreateTable(&model))

		// Add composite primary key if needed
		if len(model.PrimaryKey) > 1 {
			sb.WriteString("\n")
			sb.WriteString(generateCompositePrimaryKey(model.TableName, model.PrimaryKey))
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
