package registry

import "fmt"

// Error represents a registry error.
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
	ErrCodeEmptyName        = "EMPTY_NAME"
	ErrCodeNilComponent     = "NIL_COMPONENT"
	ErrCodeAlreadyExists    = "ALREADY_EXISTS"
	ErrCodeNotFound         = "NOT_FOUND"
	ErrCodeMaxComponents    = "MAX_COMPONENTS"
	ErrCodeNilContext       = "NIL_CONTEXT"
	ErrCodeContextCanceled  = "CONTEXT_CANCELED"
	ErrCodeValidationFailed = "VALIDATION_FAILED"
)

// NewError creates a new error with the given code and message.
func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}
