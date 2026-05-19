package retry

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
)

func TestAlwaysRetry(t *testing.T) {
	tests := []error{
		nil,
		errors.New("some error"),
		context.DeadlineExceeded,
		context.Canceled,
	}

	for _, err := range tests {
		if !AlwaysRetry(err) {
			t.Errorf("AlwaysRetry should always return true, got false for %v", err)
		}
	}
}

func TestNeverRetry(t *testing.T) {
	tests := []error{
		nil,
		errors.New("some error"),
		context.DeadlineExceeded,
	}

	for _, err := range tests {
		if NeverRetry(err) {
			t.Errorf("NeverRetry should always return false, got true for %v", err)
		}
	}
}

func TestRetryOnTimeout(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"context deadline exceeded", context.DeadlineExceeded, true},
		{"timeout error message", errors.New("timeout occurred"), true},
		{"deadline exceeded message", errors.New("deadline exceeded"), true},
		{"timed out message", errors.New("operation timed out"), true},
		{"regular error", errors.New("something went wrong"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RetryOnTimeout(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for error: %v", tt.expected, result, tt.err)
			}
		})
	}
}

func TestRetryOnTemporary(t *testing.T) {
	// Create a temporary error
	tempErr := &net.DNSError{
		Err:    "temporary failure",
		IsTimeout: false,
		IsTemporary: true,
	}

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"temporary DNS error", tempErr, true},
		{"temporary in message", errors.New("temporary failure"), true},
		{"retry in message", errors.New("please retry"), true},
		{"busy in message", errors.New("server is busy"), true},
		{"regular error", errors.New("something went wrong"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RetryOnTemporary(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for error: %v", tt.expected, result, tt.err)
			}
		})
	}
}

func TestRetryOnNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"network error", errors.New("network error"), true},
		{"connection error", errors.New("connection failed"), true},
		{"dial error", errors.New("dial tcp failed"), true},
		{"connection refused", errors.New("connection refused"), true},
		{"connection reset", errors.New("connection reset"), true},
		{"regular error", errors.New("something went wrong"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RetryOnNetworkError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for error: %v", tt.expected, result, tt.err)
			}
		})
	}
}

func TestRetryOnErrorType(t *testing.T) {
	targetErr := errors.New("target error")
	predicate := RetryOnErrorType(targetErr)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"exact match", targetErr, true},
		{"wrapped target", fmt.Errorf("wrapped: %w", targetErr), true},
		{"different error", errors.New("different error"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := predicate(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for error: %v", tt.expected, result, tt.err)
			}
		})
	}
}

func TestRetryOnErrorMessage(t *testing.T) {
	predicate := RetryOnErrorMessage("timeout", "network", "retry")

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"timeout in message", errors.New("request timeout"), true},
		{"network in message", errors.New("network failure"), true},
		{"retry in message", errors.New("please retry"), true},
		{"case insensitive", errors.New("NETWORK ERROR"), true},
		{"no match", errors.New("something else"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := predicate(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for error: %v", tt.expected, result, tt.err)
			}
		})
	}
}

func TestRetryOnAny(t *testing.T) {
	predicate := RetryOnAny(RetryOnTimeout, RetryOnNetworkError)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"timeout error", errors.New("timeout occurred"), true},
		{"network error", errors.New("network failure"), true},
		{"both match", errors.New("network timeout"), true},
		{"neither match", errors.New("something else"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := predicate(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for error: %v", tt.expected, result, tt.err)
			}
		})
	}
}

func TestRetryOnAll(t *testing.T) {
	// This is tricky - we need predicates that can both be true
	// Let's use message-based predicates
	predicate := RetryOnAll(
		RetryOnErrorMessage("network"),
		RetryOnErrorMessage("error"),
	)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"both substrings", errors.New("network error"), true},
		{"only network", errors.New("network issue"), false},
		{"only error", errors.New("some error"), false},
		{"neither", errors.New("something else"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := predicate(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for error: %v", tt.expected, result, tt.err)
			}
		})
	}
}

func TestRetryOnNot(t *testing.T) {
	predicate := RetryOnNot(RetryOnTimeout)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"timeout error", errors.New("timeout occurred"), false},
		{"regular error", errors.New("something else"), true},
		{"nil error", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := predicate(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for error: %v", tt.expected, result, tt.err)
			}
		})
	}
}
