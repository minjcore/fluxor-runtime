package errorhandler

import (
	"net/http"
)

// HTTPError represents an HTTP error response
type HTTPError struct {
	Code    ErrorCode              `json:"code"`
	Message string                 `json:"message"`
	Status  int                    `json:"status"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// ToHTTPError converts a FluxorError to an HTTPError
func ToHTTPError(err error) HTTPError {
	if err == nil {
		return HTTPError{
			Code:    ErrorCodeUnknown,
			Message: "Unknown error",
			Status:  http.StatusInternalServerError,
		}
	}
	
	fluxorErr, ok := As(err)
	if !ok {
		// If it's not a FluxorError, create one
		fluxorErr = Wrap(err, ErrorCodeInternal, err.Error())
	}
	
	status := StatusCodeFromErrorCode(fluxorErr.Code)
	
	return HTTPError{
		Code:    fluxorErr.Code,
		Message: fluxorErr.Message,
		Status:  status,
		Context: fluxorErr.Context,
	}
}

// StatusCodeFromErrorCode maps an ErrorCode to an HTTP status code
func StatusCodeFromErrorCode(code ErrorCode) int {
	switch code {
	case ErrorCodeValidation:
		return http.StatusBadRequest
	case ErrorCodeNotFound:
		return http.StatusNotFound
	case ErrorCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrorCodeForbidden:
		return http.StatusForbidden
	case ErrorCodeConflict:
		return http.StatusConflict
	case ErrorCodeTimeout:
		return http.StatusRequestTimeout
	case ErrorCodeRateLimit:
		return http.StatusTooManyRequests
	case ErrorCodeServiceUnavailable:
		return http.StatusServiceUnavailable
	case ErrorCodeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// ErrorCodeFromStatusCode maps an HTTP status code to an ErrorCode
func ErrorCodeFromStatusCode(status int) ErrorCode {
	switch status {
	case http.StatusBadRequest:
		return ErrorCodeValidation
	case http.StatusNotFound:
		return ErrorCodeNotFound
	case http.StatusUnauthorized:
		return ErrorCodeUnauthorized
	case http.StatusForbidden:
		return ErrorCodeForbidden
	case http.StatusConflict:
		return ErrorCodeConflict
	case http.StatusRequestTimeout:
		return ErrorCodeTimeout
	case http.StatusTooManyRequests:
		return ErrorCodeRateLimit
	case http.StatusServiceUnavailable:
		return ErrorCodeServiceUnavailable
	case http.StatusInternalServerError:
		return ErrorCodeInternal
	default:
		return ErrorCodeUnknown
	}
}

// NewHTTPError creates a new HTTPError from an ErrorCode and message
func NewHTTPError(code ErrorCode, message string, status int) HTTPError {
	return HTTPError{
		Code:    code,
		Message: message,
		Status:  status,
		Context: make(map[string]interface{}),
	}
}

// WithContext adds context to the HTTPError
func (e HTTPError) WithContext(key string, value interface{}) HTTPError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}
