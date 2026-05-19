package debug

import "fmt"

// Error represents a debug operation error.
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
	ErrCodeNilContext       = "NIL_CONTEXT"
	ErrCodeCollectTimeout   = "COLLECT_TIMEOUT"
	ErrCodeInvalidCollector = "INVALID_COLLECTOR"
	ErrCodeCollectorExists   = "COLLECTOR_EXISTS"
)

// NewError creates a new error with the given code and message.
func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}
