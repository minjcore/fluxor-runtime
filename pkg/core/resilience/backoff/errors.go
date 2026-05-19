package backoff

import "fmt"

// Error represents a backoff operation error.
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
	ErrCodeInvalidConfig = "INVALID_CONFIG"
	ErrCodeNilContext    = "NIL_CONTEXT"
	ErrCodeContextCanceled = "CONTEXT_CANCELED"
	ErrCodeContextTimeout  = "CONTEXT_TIMEOUT"
)

// NewError creates a new error with the given code and message.
func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}
