package fallback

import "fmt"

// Error represents a fallback operation error.
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
	ErrCodeAllFallbacksExhausted = "ALL_FALLBACKS_EXHAUSTED"
	ErrCodeNilContext            = "NIL_CONTEXT"
	ErrCodeNilPrimaryFunction    = "NIL_PRIMARY_FUNCTION"
	ErrCodeNilFallbackFunction   = "NIL_FALLBACK_FUNCTION"
	ErrCodeInvalidConfig         = "INVALID_CONFIG"
)

// NewError creates a new error with the given code and message.
func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}
