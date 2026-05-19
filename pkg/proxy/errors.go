package proxy

import "fmt"

// ProxyError represents a proxy-specific error
type ProxyError struct {
	Code    string
	Message string
	Details interface{}
	Err     error
}

func (e *ProxyError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	if e.Details != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *ProxyError) Unwrap() error {
	return e.Err
}

// NewProxyError creates a new proxy error
func NewProxyError(code, message string) *ProxyError {
	return &ProxyError{
		Code:    code,
		Message: message,
	}
}

// NewProxyErrorWithDetails creates a new proxy error with details
func NewProxyErrorWithDetails(code, message string, details interface{}) *ProxyError {
	return &ProxyError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// NewProxyErrorWithError creates a new proxy error wrapping another error
func NewProxyErrorWithError(code, message string, err error) *ProxyError {
	return &ProxyError{
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

// BackendError represents a backend-specific error
type BackendError struct {
	Code    string
	Message string
	Backend *Backend
	Err     error
}

func (e *BackendError) Error() string {
	backendURL := "unknown"
	if e.Backend != nil {
		backendURL = e.Backend.URL
	}
	if e.Err != nil {
		return fmt.Sprintf("%s [%s]: %s (%v)", e.Code, backendURL, e.Message, e.Err)
	}
	return fmt.Sprintf("%s [%s]: %s", e.Code, backendURL, e.Message)
}

func (e *BackendError) Unwrap() error {
	return e.Err
}

// NewBackendError creates a new backend error
func NewBackendError(code, message string, backend *Backend, err error) *BackendError {
	return &BackendError{
		Code:    code,
		Message: message,
		Backend: backend,
		Err:     err,
	}
}

// Common error codes
const (
	ErrCodeNoHealthyBackends = "NO_HEALTHY_BACKENDS"
	ErrCodeBackendTimeout    = "BACKEND_TIMEOUT"
	ErrCodeBackendError      = "BACKEND_ERROR"
	ErrCodeRateLimited       = "RATE_LIMITED"
	ErrCodeMaxConnections    = "MAX_CONNECTIONS"
	ErrCodeCircuitBreakerOpen = "CIRCUIT_BREAKER_OPEN"
	ErrCodeInvalidBackend     = "INVALID_BACKEND"
)
