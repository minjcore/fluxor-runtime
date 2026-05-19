package core

import (
	"time"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// ValidateAddress validates an event bus address
func ValidateAddress(address string) error {
	if address == "" {
		return &EventBusError{Code: "INVALID_ADDRESS", Message: "address cannot be empty"}
	}
	if len(address) > 255 {
		return &EventBusError{Code: "INVALID_ADDRESS", Message: "address too long (max 255 characters)"}
	}
	return nil
}

// ValidateTimeout validates a timeout duration
func ValidateTimeout(timeout time.Duration) error {
	if timeout <= 0 {
		return &EventBusError{Code: "INVALID_TIMEOUT", Message: "timeout must be positive"}
	}
	if timeout > 5*time.Minute {
		return &EventBusError{Code: "INVALID_TIMEOUT", Message: "timeout too large (max 5 minutes)"}
	}
	return nil
}

// ValidateVerticle validates a verticle before deployment
func ValidateVerticle(verticle Verticle) error {
	if verticle == nil {
		return &EventBusError{Code: "INVALID_VERTICLE", Message: "verticle cannot be nil"}
	}
	return nil
}

// ValidateBody validates a message body
func ValidateBody(body interface{}) error {
	if body == nil {
		return &EventBusError{Code: "INVALID_BODY", Message: "body cannot be nil"}
	}
	return nil
}

// FailFast panics with an error (fail-fast principle)
// Deprecated: Use failfast.Err instead
func FailFast(err error) {
	failfast.Err(err)
}

// FailFastIf panics if condition is true
// Deprecated: Use failfast.If instead
func FailFastIf(condition bool, message string) {
	failfast.If(!condition, message)
}
