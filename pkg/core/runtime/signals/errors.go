package signals

import "fmt"

// Error represents a signal handler error.
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
	ErrCodeAlreadyStarted    = "ALREADY_STARTED"
	ErrCodeAlreadyStopped    = "ALREADY_STOPPED"
	ErrCodeNilContext        = "NIL_CONTEXT"
	ErrCodeNilCallback       = "NIL_CALLBACK"
	ErrCodeChannelClosed     = "CHANNEL_CLOSED"
	ErrCodeHandlerStopped    = "HANDLER_STOPPED"
	ErrCodeShutdownTimeout   = "SHUTDOWN_TIMEOUT"
)

// NewError creates a new error with the given code and message.
func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}
