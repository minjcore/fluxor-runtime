package errorhandler

import (
	"fmt"
	"time"
)

// ErrorCode represents a standardized error code
type ErrorCode string

const (
	// ErrorCodeUnknown represents an unknown error
	ErrorCodeUnknown ErrorCode = "UNKNOWN"
	// ErrorCodeValidation represents a validation error
	ErrorCodeValidation ErrorCode = "VALIDATION_ERROR"
	// ErrorCodeNotFound represents a not found error
	ErrorCodeNotFound ErrorCode = "NOT_FOUND"
	// ErrorCodeUnauthorized represents an unauthorized error
	ErrorCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	// ErrorCodeForbidden represents a forbidden error
	ErrorCodeForbidden ErrorCode = "FORBIDDEN"
	// ErrorCodeConflict represents a conflict error
	ErrorCodeConflict ErrorCode = "CONFLICT"
	// ErrorCodeInternal represents an internal server error
	ErrorCodeInternal ErrorCode = "INTERNAL_ERROR"
	// ErrorCodeTimeout represents a timeout error
	ErrorCodeTimeout ErrorCode = "TIMEOUT"
	// ErrorCodeRateLimit represents a rate limit error
	ErrorCodeRateLimit ErrorCode = "RATE_LIMIT"
	// ErrorCodeServiceUnavailable represents a service unavailable error
	ErrorCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)

// ErrorSeverity represents the severity of an error
type ErrorSeverity string

const (
	// ErrorSeverityLow represents a low severity error
	ErrorSeverityLow ErrorSeverity = "low"
	// ErrorSeverityMedium represents a medium severity error
	ErrorSeverityMedium ErrorSeverity = "medium"
	// ErrorSeverityHigh represents a high severity error
	ErrorSeverityHigh ErrorSeverity = "high"
	// ErrorSeverityCritical represents a critical severity error
	ErrorSeverityCritical ErrorSeverity = "critical"
)

// FluxorError is a structured error type with additional context
type FluxorError struct {
	// Code is the error code
	Code ErrorCode `json:"code"`
	// Message is the error message
	Message string `json:"message"`
	// Severity is the error severity
	Severity ErrorSeverity `json:"severity,omitempty"`
	// Cause is the underlying error (if any)
	Cause error `json:"-"`
	// Context contains additional context information
	Context map[string]interface{} `json:"context,omitempty"`
	// Timestamp is when the error occurred
	Timestamp time.Time `json:"timestamp"`
	// StackTrace is the stack trace (if available)
	StackTrace string `json:"stack_trace,omitempty"`
}

// Error implements the error interface
func (e *FluxorError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *FluxorError) Unwrap() error {
	return e.Cause
}

// WithContext adds context to the error
func (e *FluxorError) WithContext(key string, value interface{}) *FluxorError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithSeverity sets the error severity
func (e *FluxorError) WithSeverity(severity ErrorSeverity) *FluxorError {
	e.Severity = severity
	return e
}

// New creates a new FluxorError
func New(code ErrorCode, message string) *FluxorError {
	return &FluxorError{
		Code:      code,
		Message:  message,
		Severity:  ErrorSeverityMedium,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
	}
}

// Wrap wraps an existing error with a FluxorError
func Wrap(err error, code ErrorCode, message string) *FluxorError {
	if err == nil {
		return nil
	}
	
	fluxorErr := New(code, message)
	fluxorErr.Cause = err
	
	// If the error is already a FluxorError, preserve its context
	if existing, ok := err.(*FluxorError); ok {
		fluxorErr.Context = existing.Context
		fluxorErr.Severity = existing.Severity
	}
	
	return fluxorErr
}

// Is checks if the error matches the given code
func Is(err error, code ErrorCode) bool {
	if err == nil {
		return false
	}
	
	if fluxorErr, ok := err.(*FluxorError); ok {
		return fluxorErr.Code == code
	}
	
	return false
}

// As extracts a FluxorError from the error chain
func As(err error) (*FluxorError, bool) {
	if err == nil {
		return nil, false
	}
	
	if fluxorErr, ok := err.(*FluxorError); ok {
		return fluxorErr, true
	}
	
	return nil, false
}
