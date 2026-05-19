package timeout

import "fmt"

// Error represents a timeout operation error.
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
	ErrCodeTimeoutExceeded = "TIMEOUT_EXCEEDED"
	ErrCodeNilContext      = "NIL_CONTEXT"
	ErrCodeNilFunction     = "NIL_FUNCTION"
	ErrCodeInvalidTimeout  = "INVALID_TIMEOUT"
	ErrCodeContextCanceled = "CONTEXT_CANCELED"
)

// NewError creates a new error with the given code and message.
func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}
