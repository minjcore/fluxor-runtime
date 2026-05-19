package persistence

import (
	"fmt"
	"regexp"
	"strings"
)

// sqlIdentifierPattern matches valid SQL identifiers (alphanumeric + underscore)
// SQL identifiers must start with a letter or underscore, followed by letters, digits, or underscores
var sqlIdentifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// ValidateSQLIdentifier validates that a string is a safe SQL identifier
// Fail-fast: Returns error if identifier is invalid
// Valid identifiers: alphanumeric + underscore, must start with letter or underscore
func ValidateSQLIdentifier(identifier string) error {
	if identifier == "" {
		return &Error{Code: "INVALID_INPUT", Message: "SQL identifier cannot be empty"}
	}

	// Check for SQL injection attempts
	if strings.Contains(identifier, ";") ||
		strings.Contains(identifier, "--") ||
		strings.Contains(identifier, "/*") ||
		strings.Contains(identifier, "*/") ||
		strings.Contains(identifier, "'") ||
		strings.Contains(identifier, "\"") ||
		strings.Contains(identifier, "`") ||
		strings.Contains(identifier, "(") ||
		strings.Contains(identifier, ")") ||
		strings.Contains(identifier, " ") ||
		strings.Contains(identifier, "\n") ||
		strings.Contains(identifier, "\r") ||
		strings.Contains(identifier, "\t") {
		return &Error{
			Code:    "INVALID_INPUT",
			Message: fmt.Sprintf("SQL identifier contains invalid characters: %s", identifier),
		}
	}

	// Validate against pattern
	if !sqlIdentifierPattern.MatchString(identifier) {
		return &Error{
			Code:    "INVALID_INPUT",
			Message: fmt.Sprintf("SQL identifier is not valid: %s (must match [a-zA-Z_][a-zA-Z0-9_]*)", identifier),
		}
	}

	return nil
}

// QuoteSQLIdentifier quotes a SQL identifier for safe use in queries
// Uses double quotes (PostgreSQL/MySQL standard)
// Only quotes if necessary (contains special characters or is a reserved word)
func QuoteSQLIdentifier(identifier string) string {
	// If already quoted, return as-is
	if len(identifier) >= 2 && identifier[0] == '"' && identifier[len(identifier)-1] == '"' {
		return identifier
	}

	// Quote the identifier
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

// ValidateAndQuoteSQLIdentifier validates and quotes a SQL identifier
// Fail-fast: Returns error if identifier is invalid
func ValidateAndQuoteSQLIdentifier(identifier string) (string, error) {
	if err := ValidateSQLIdentifier(identifier); err != nil {
		return "", err
	}
	return QuoteSQLIdentifier(identifier), nil
}

// ValidateSQLIdentifiers validates multiple SQL identifiers
// Fail-fast: Returns error if any identifier is invalid
func ValidateSQLIdentifiers(identifiers []string) error {
	for _, id := range identifiers {
		if err := ValidateSQLIdentifier(id); err != nil {
			return err
		}
	}
	return nil
}

// ValidateTableName validates a table name
// Fail-fast: Returns error if table name is invalid
func ValidateTableName(tableName string) error {
	return ValidateSQLIdentifier(tableName)
}

// ValidateFieldName validates a field/column name
// Fail-fast: Returns error if field name is invalid
func ValidateFieldName(fieldName string) error {
	// Support qualified names (table.field)
	parts := strings.Split(fieldName, ".")
	for _, part := range parts {
		if err := ValidateSQLIdentifier(strings.TrimSpace(part)); err != nil {
			return err
		}
	}
	return nil
}

// ValidateFieldNames validates multiple field names
// Fail-fast: Returns error if any field name is invalid
func ValidateFieldNames(fieldNames []string) error {
	for _, fieldName := range fieldNames {
		if err := ValidateFieldName(fieldName); err != nil {
			return err
		}
	}
	return nil
}

// QuoteFieldName quotes a field name (supports qualified names like "table.field")
func QuoteFieldName(fieldName string) (string, error) {
	parts := strings.Split(fieldName, ".")
	quotedParts := make([]string, len(parts))
	for i, part := range parts {
		quoted, err := ValidateAndQuoteSQLIdentifier(strings.TrimSpace(part))
		if err != nil {
			return "", err
		}
		quotedParts[i] = quoted
	}
	return strings.Join(quotedParts, "."), nil
}
