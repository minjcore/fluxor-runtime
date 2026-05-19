package retry

import "fmt"

// Error represents a retry operation error.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Message
}

// Error codes
const (
	ErrCodeMaxRetriesExceeded = "MAX_RETRIES_EXCEEDED"
	ErrCodeNilContext         = "NIL_CONTEXT"
	ErrCodeNilRetriable       = "NIL_RETRIABLE"
	ErrCodeContextTimeout     = "CONTEXT_TIMEOUT"
	ErrCodeContextCanceled    = "CONTEXT_CANCELED"
	ErrCodeInvalidConfig      = "INVALID_CONFIG"
)

// NewError creates a new error with the given code and message.
func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}
