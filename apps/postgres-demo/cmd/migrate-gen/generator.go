package main

import (
	"fmt"
	"strings"
)

// ============================================================================
// TYPE MAPPING FUNCTIONS
// ============================================================================

// mapGoTypeToSQL maps a Go type to PostgreSQL SQL type
func mapGoTypeToSQL(goType string, isNullable bool, comment string, columnName string) string {
	// Handle pointer types
	if strings.HasPrefix(goType, "*") {
		isNullable = true
		goType = strings.TrimPrefix(goType, "*")
	}

	// Check for specific type hints in comments (case-insensitive)
	commentLower := strings.ToLower(comment)
	if strings.Contains(commentLower, "json") || strings.Contains(commentLower, "jsonb") {
		return "JSONB"
	}
	if strings.Contains(strings.ToLower(comment), "text") || strings.Contains(strings.ToLower(columnName), "description") {
		return "TEXT"
	}

	// Handle qualified types (e.g., time.Time)
	if strings.Contains(goType, ".") {
		parts := strings.Split(goType, ".")
		if len(parts) == 2 {
			pkg := parts[0]
			typ := parts[1]
			if pkg == "time" && typ == "Time" {
				return "TIMESTAMP"
			}
		}
	}

	// Map basic types
	switch goType {
	case "int":
		// Check if it's an ID field by column name
		if columnName == "id" {
			return "SERIAL"
		}
		return "INTEGER"
	case "*int":
		return "INTEGER"
	case "int64":
		return "BIGINT"
	case "*int64":
		return "BIGINT"
	case "string":
		// Determine VARCHAR length from comment or use default
		length := inferVarcharLength(comment)
		if length > 0 {
			return fmt.Sprintf("VARCHAR(%d)", length)
		}
		return "VARCHAR(255)"
	case "*string":
		length := inferVarcharLength(comment)
		if length > 0 {
			return fmt.Sprintf("VARCHAR(%d)", length)
		}
		return "VARCHAR(255)"
	case "float64":
		// Check comment for precision hints
		if strings.Contains(strings.ToLower(comment), "balance") || 
		   strings.Contains(strings.ToLower(comment), "amount") ||
		   strings.Contains(strings.ToLower(comment), "money") {
			return "DECIMAL(15,2)"
		}
		return "DECIMAL(10,2)"
	case "*float64":
		if strings.Contains(strings.ToLower(comment), "balance") || 
		   strings.Contains(strings.ToLower(comment), "amount") ||
		   strings.Contains(strings.ToLower(comment), "money") {
			return "DECIMAL(15,2)"
		}
		return "DECIMAL(10,2)"
	case "bool":
		return "BOOLEAN"
	case "*bool":
		return "BOOLEAN"
	case "time.Time":
		return "TIMESTAMP"
	case "*time.Time":
		return "TIMESTAMP"
	default:
		// For custom types (like WalletType, ApprovalStatus), use VARCHAR
		// Check if it's an enum-like type
		return "VARCHAR(50)"
	}
}

// inferVarcharLength infers VARCHAR length from comment or field name
func inferVarcharLength(comment string) int {
	// Look for explicit length hints in comment
	// e.g., "VARCHAR(20)", "length: 100"
	if strings.Contains(comment, "VARCHAR(") {
		// Try to extract number
		start := strings.Index(comment, "VARCHAR(")
		if start != -1 {
			start += len("VARCHAR(")
			end := strings.Index(comment[start:], ")")
			if end != -1 {
				var length int
				fmt.Sscanf(comment[start:start+end], "%d", &length)
				if length > 0 {
					return length
				}
			}
		}
	}

	// Default lengths based on common patterns
	if strings.Contains(strings.ToLower(comment), "code") || 
	   strings.Contains(strings.ToLower(comment), "type") {
		return 20
	}
	if strings.Contains(strings.ToLower(comment), "key") {
		return 100
	}
	if strings.Contains(strings.ToLower(comment), "description") {
		return 0 // Use TEXT
	}

	return 0 // Use default
}

// getDefaultValue returns the default value for a field based on type and comment
func getDefaultValue(field *FieldInfo) string {
	// Check if explicit default in comment
	if field.DefaultValue != "" {
		return field.DefaultValue
	}

	// Type-based defaults
	if strings.Contains(field.SQLType, "TIMESTAMP") {
		if field.ColumnName == "created_at" || field.ColumnName == "updated_at" {
			return "CURRENT_TIMESTAMP"
		}
	}

	if strings.Contains(field.SQLType, "BOOLEAN") {
		if strings.Contains(strings.ToLower(field.Comment), "default true") {
			return "true"
		}
		if strings.Contains(strings.ToLower(field.Comment), "default false") {
			return "false"
		}
	}

	// Check comment for status defaults
	if field.ColumnName == "status" {
		if strings.Contains(strings.ToLower(field.Comment), "pending") {
			return "'pending'"
		}
	}

	return ""
}

// isNotNull determines if a column should be NOT NULL
func isNotNull(field *FieldInfo) bool {
	// Pointers are nullable
	if field.IsNullable {
		return false
	}

	// Check comment for explicit nullable hint
	if strings.Contains(strings.ToLower(field.Comment), "nullable") ||
	   strings.Contains(strings.ToLower(field.Comment), "optional") {
		return false
	}

	// Primary keys are always NOT NULL
	if field.IsPrimaryKey {
		return true
	}

	// Most fields are NOT NULL by default
	return true
}

// shouldCreateIndex determines if an index should be created for a field
func shouldCreateIndex(field *FieldInfo, model *ModelInfo) bool {
	// Foreign keys
	if strings.HasSuffix(field.ColumnName, "_id") && field.ColumnName != "id" {
		return true
	}

	// Timestamp fields
	if field.ColumnName == "created_at" || field.ColumnName == "updated_at" {
		return true
	}

	// Fields mentioned in comments as indexed
	if strings.Contains(strings.ToLower(field.Comment), "index") {
		return true
	}

	return false
}

// extractConstraints extracts CHECK constraints from comments
func extractConstraints(field *FieldInfo) []string {
	var constraints []string

	comment := strings.ToLower(field.Comment)

	// Check for enum-like constraints
	if strings.Contains(comment, "in (") || strings.Contains(comment, "one of") {
		// Extract enum values - simplified version
		// In practice, you'd want more sophisticated parsing
	}

	// Check for range constraints
	if strings.Contains(comment, "min:") || strings.Contains(comment, "max:") {
		// Extract min/max values
	}

	return constraints
}

// inferCompositePrimaryKey extracts composite primary key from model
func inferCompositePrimaryKey(model *ModelInfo) []string {
	// Already extracted in parser, but can add additional logic here
	return model.PrimaryKey
}

// validateModel validates that a model has required information
func validateModel(model *ModelInfo) error {
	if model.TableName == "" {
		return fmt.Errorf("model %s: missing table name", model.Name)
	}

	if len(model.Fields) == 0 {
		return fmt.Errorf("model %s: no fields found", model.Name)
	}

	// Check for primary key
	if len(model.PrimaryKey) == 0 {
		// Try to find ID field
		for _, field := range model.Fields {
			if field.ColumnName == "id" {
				model.PrimaryKey = []string{"id"}
				break
			}
		}
		// If still no primary key, warn but don't fail
	}

	return nil
}
