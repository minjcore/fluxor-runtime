package retry

import (
	"context"
	"errors"
	"strings"
)

// RetryPredicate determines whether an error should trigger a retry.
// Returns true if the error should be retried, false otherwise.
type RetryPredicate func(err error) bool

// AlwaysRetry always returns true, retrying on all errors.
func AlwaysRetry(err error) bool {
	return true
}

// NeverRetry always returns false, never retrying.
func NeverRetry(err error) bool {
	return false
}

// RetryOnTimeout retries only on timeout or deadline exceeded errors.
func RetryOnTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	// Check for common timeout error messages
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "timed out")
}

// RetryOnTemporary retries on temporary errors.
// An error is considered temporary if it implements the Temporary() method
// and that method returns true, or if the error message contains "temporary".
func RetryOnTemporary(err error) bool {
	if err == nil {
		return false
	}
	
	// Check if error implements Temporary() method
	type temporary interface {
		Temporary() bool
	}
	if t, ok := err.(temporary); ok && t.Temporary() {
		return true
	}
	
	// Check error message
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "temporary") ||
		strings.Contains(errStr, "retry") ||
		strings.Contains(errStr, "busy")
}

// RetryOnNetworkError retries on network-related errors.
func RetryOnNetworkError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "dial") ||
		strings.Contains(errStr, "refused") ||
		strings.Contains(errStr, "reset")
}

// RetryOnErrorType creates a predicate that retries on errors matching a specific type.
func RetryOnErrorType(target error) RetryPredicate {
	return func(err error) bool {
		return errors.Is(err, target)
	}
}

// RetryOnErrorMessage creates a predicate that retries if the error message
// contains any of the specified substrings (case-insensitive).
func RetryOnErrorMessage(substrings ...string) RetryPredicate {
	return func(err error) bool {
		if err == nil {
			return false
		}
		errStr := strings.ToLower(err.Error())
		for _, substr := range substrings {
			if strings.Contains(errStr, strings.ToLower(substr)) {
				return true
			}
		}
		return false
	}
}

// RetryOnAny combines multiple predicates with OR logic.
// Returns true if any predicate returns true.
func RetryOnAny(predicates ...RetryPredicate) RetryPredicate {
	return func(err error) bool {
		for _, p := range predicates {
			if p(err) {
				return true
			}
		}
		return false
	}
}

// RetryOnAll combines multiple predicates with AND logic.
// Returns true only if all predicates return true.
func RetryOnAll(predicates ...RetryPredicate) RetryPredicate {
	return func(err error) bool {
		for _, p := range predicates {
			if !p(err) {
				return false
			}
		}
		return true
	}
}

// RetryOnNot negates a predicate.
// Returns true if the predicate returns false.
func RetryOnNot(predicate RetryPredicate) RetryPredicate {
	return func(err error) bool {
		return !predicate(err)
	}
}
