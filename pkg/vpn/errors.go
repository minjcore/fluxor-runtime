package vpn

import "fmt"

// VPNError represents a VPN-specific error
type VPNError struct {
	Code    string
	Message string
	Details interface{}
	Err     error
}

func (e *VPNError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	if e.Details != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *VPNError) Unwrap() error {
	return e.Err
}

// NewVPNError creates a new VPN error
func NewVPNError(code, message string) *VPNError {
	return &VPNError{
		Code:    code,
		Message: message,
	}
}

// NewVPNErrorWithDetails creates a new VPN error with details
func NewVPNErrorWithDetails(code, message string, details interface{}) *VPNError {
	return &VPNError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// NewVPNErrorWithError creates a new VPN error wrapping another error
func NewVPNErrorWithError(code, message string, err error) *VPNError {
	return &VPNError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// ConfigError represents a configuration error
type ConfigError struct {
	Code    string
	Message string
	Err     error
}

func (e *ConfigError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}

// ProtocolError represents a protocol error
type ProtocolError struct {
	Code    string
	Message string
	Packet  *VPNPacket
	Err     error
}

func (e *ProtocolError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *ProtocolError) Unwrap() error {
	return e.Err
}

// Common error codes
const (
	ErrCodeInvalidPacket      = "INVALID_PACKET"
	ErrCodeInvalidState       = "INVALID_STATE"
	ErrCodeAuthenticationFailed = "AUTH_FAILED"
	ErrCodeRateLimited        = "RATE_LIMITED"
	ErrCodeIPPoolExhausted    = "IP_POOL_EXHAUSTED"
	ErrCodeClientNotFound     = "CLIENT_NOT_FOUND"
	ErrCodeConnectionClosed   = "CONNECTION_CLOSED"
	ErrCodeTimeout            = "TIMEOUT"
)
