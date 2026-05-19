package persistence

import (
	"fmt"
	"strings"
)

// Error represents a persistence error
type Error struct {
	Code    string
	Message string
	Cause   error
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("persistence error [%s]: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("persistence error [%s]: %s", e.Code, e.Message)
}

// SafeError returns a sanitized error message safe for production use
// It removes potentially sensitive information from the underlying error
func (e *Error) SafeError() string {
	// Return only the code and generic message, without the underlying cause
	// which may contain sensitive database details
	return fmt.Sprintf("persistence error [%s]: %s", e.Code, e.Message)
}

// SanitizeError sanitizes an error for production use
// If the error is a persistence.Error, returns SafeError()
// Otherwise, returns a generic error message
func SanitizeError(err error) string {
	if err == nil {
		return ""
	}
	
	if e, ok := err.(*Error); ok {
		return e.SafeError()
	}
	
	// For non-persistence errors, sanitize by removing potentially sensitive patterns
	errStr := err.Error()
	
	// Remove common sensitive patterns (DSN, passwords, etc.)
	sensitivePatterns := []string{
		"password",
		"pwd",
		"dsn",
		"connection string",
		"host=",
		"user=",
		"password=",
	}
	
	lowerErr := strings.ToLower(errStr)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerErr, pattern) {
			// Return generic error without sensitive details
			return "database operation failed"
		}
	}
	
	// If no sensitive patterns found, return as-is but limit length
	if len(errStr) > 200 {
		return errStr[:200] + "..."
	}
	
	return errStr
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Cause
}

// Error codes
const (
	ErrCodeNotFound      = "NOT_FOUND"
	ErrCodeInvalidConfig = "INVALID_CONFIG"
	ErrCodeInvalidInput  = "INVALID_INPUT"
	ErrCodeTransaction   = "TRANSACTION_ERROR"
	ErrCodeQuery         = "QUERY_ERROR"
	ErrCodeConnection    = "CONNECTION_ERROR"
)

// NewError creates a new persistence error
func NewError(code, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// IsNotFound checks if an error is a "not found" error
func IsNotFound(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrCodeNotFound
	}
	return false
}

// IsTransactionError checks if an error is a transaction error
func IsTransactionError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrCodeTransaction
	}
	return false
}
