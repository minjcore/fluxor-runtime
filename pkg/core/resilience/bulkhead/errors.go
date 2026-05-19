package bulkhead

import "fmt"

// Error represents a bulkhead operation error.
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
	ErrCodeBulkheadFull     = "BULKHEAD_FULL"
	ErrCodeNilContext       = "NIL_CONTEXT"
	ErrCodeNilFunction      = "NIL_FUNCTION"
	ErrCodeInvalidConfig    = "INVALID_CONFIG"
	ErrCodeContextCanceled  = "CONTEXT_CANCELED"
	ErrCodeContextTimeout   = "CONTEXT_TIMEOUT"
)

// NewError creates a new error with the given code and message.
func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}
