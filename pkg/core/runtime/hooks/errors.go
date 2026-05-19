package hooks

import "fmt"

// Error represents a hooks registry error.
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
	ErrCodeNilHook        = "NIL_HOOK"
	ErrCodeHookNotFound   = "HOOK_NOT_FOUND"
	ErrCodeHookExists     = "HOOK_EXISTS"
	ErrCodeInvalidPriority = "INVALID_PRIORITY"
	ErrCodeHookFailed     = "HOOK_FAILED"
	ErrCodeNilContext     = "NIL_CONTEXT"
	ErrCodeExecutionTimeout = "EXECUTION_TIMEOUT"
)

// NewError creates a new error with the given code and message.
func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}
