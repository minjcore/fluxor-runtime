package state

import "fmt"

// Error represents a state management error.
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
	ErrCodeInvalidState        = "INVALID_STATE"
	ErrCodeInvalidTransition   = "INVALID_TRANSITION"
	ErrCodeNilContext          = "NIL_CONTEXT"
	ErrCodeTransitionTimeout   = "TRANSITION_TIMEOUT"
	ErrCodeAlreadyStarted      = "ALREADY_STARTED"
	ErrCodeAlreadyStopped      = "ALREADY_STOPPED"
)

// NewError creates a new error with the given code and message.
func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}
