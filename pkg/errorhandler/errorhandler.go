// Package errorhandler provides comprehensive error handling utilities for Fluxor applications.
//
// Features:
//   - Structured error types with codes and severity
//   - Error wrapping and context
//   - Panic recovery utilities
//   - HTTP error transformation
//   - Error handler chains and filters
//
// Example usage:
//
//	err := errorhandler.New(errorhandler.ErrorCodeNotFound, "Resource not found")
//	err = err.WithContext("resource_id", "123")
//
//	httpErr := errorhandler.ToHTTPError(err)
//	// Returns HTTP 404 with proper error structure
package errorhandler
