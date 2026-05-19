package web

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

type errWithCode struct {
	msg string
	n   int
}

func (e *errWithCode) Error() string   { return e.msg }
func (e *errWithCode) StatusCode() int { return e.n }

func TestGinHandlerHTTPError_StatusError(t *testing.T) {
	st, code, msg := ginHandlerHTTPError(&StatusError{Status: http.StatusTeapot, Code: "tea", Msg: "x"})
	if st != http.StatusTeapot || code != "tea" || msg != "x" {
		t.Fatalf("got %d %q %q", st, code, msg)
	}
}

func TestGinHandlerHTTPError_wrappedStatusCoder(t *testing.T) {
	inner := &errWithCode{msg: "nope", n: http.StatusUnauthorized}
	wrapped := fmt.Errorf("outer: %w", inner)
	st, code, msg := ginHandlerHTTPError(wrapped)
	if st != http.StatusUnauthorized || code != "request_error" {
		t.Fatalf("got %d %q %q", st, code, msg)
	}
	if msg != wrapped.Error() {
		t.Fatalf("message: %q", msg)
	}
}

func TestGinHandlerHTTPError_plain(t *testing.T) {
	st, code, msg := ginHandlerHTTPError(errors.New("fail"))
	if st != http.StatusInternalServerError || code != "internal_error" || msg != "fail" {
		t.Fatalf("got %d %q %q", st, code, msg)
	}
}
