package web

import "net/http"

// HTTPStatusError is implemented by errors that should map to a specific HTTP status (not only 500).
type HTTPStatusError interface {
	error
	StatusCode() int
}

// StatusError is a concrete HTTP status error for handlers (validation, auth, business rules).
type StatusError struct {
	Code   string // optional; defaults to "request_error" in Gin wrap when empty
	Status int    // HTTP status; defaults to 500 if zero
	Msg    string
}

func (e *StatusError) Error() string {
	if e == nil {
		return ""
	}
	return e.Msg
}

// StatusCode returns the HTTP status for this error.
func (e *StatusError) StatusCode() int {
	if e == nil {
		return http.StatusInternalServerError
	}
	if e.Status == 0 {
		return http.StatusInternalServerError
	}
	return e.Status
}
