package health

import "fmt"

// Error represents a health check error.
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
	ErrCodeCheckTimeout     = "CHECK_TIMEOUT"
	ErrCodeInvalidChecker   = "INVALID_CHECKER"
	ErrCodeCheckerExists    = "CHECKER_EXISTS"
	ErrCodeInvalidThreshold = "INVALID_THRESHOLD"
	ErrCodeThresholdExists  = "THRESHOLD_EXISTS"
)

// NewError creates a new error with the given code and message.
func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}
