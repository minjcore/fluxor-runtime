package fallback

import (
	"context"
	"errors"
	"strings"
)

// AlwaysFallback always returns true, executing fallback on all errors.
func AlwaysFallback(err error) bool {
	return err != nil
}

// NeverFallback always returns false, never executing fallback.
func NeverFallback(err error) bool {
	return false
}

// FallbackOnTimeout executes fallback only on timeout or deadline exceeded errors.
func FallbackOnTimeout(err error) bool {
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

// FallbackOnTemporary executes fallback on temporary errors.
// An error is considered temporary if it implements the Temporary() method
// and that method returns true, or if the error message contains "temporary".
func FallbackOnTemporary(err error) bool {
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

// FallbackOnNetworkError executes fallback on network-related errors.
func FallbackOnNetworkError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "dial") ||
		strings.Contains(errStr, "refused") ||
		strings.Contains(errStr, "reset") ||
		strings.Contains(errStr, "unreachable")
}

// FallbackOnErrorType executes fallback if the error is of a specific type.
func FallbackOnErrorType(target error) FallbackPredicate {
	return func(err error) bool {
		return errors.Is(err, target)
	}
}

// FallbackOnErrorMessage executes fallback if the error message contains the given substring.
func FallbackOnErrorMessage(substring string) FallbackPredicate {
	return func(err error) bool {
		if err == nil {
			return false
		}
		return strings.Contains(strings.ToLower(err.Error()), strings.ToLower(substring))
	}
}

// FallbackOnAny returns a predicate that executes fallback if any of the given predicates return true.
func FallbackOnAny(predicates ...FallbackPredicate) FallbackPredicate {
	return func(err error) bool {
		for _, p := range predicates {
			if p != nil && p(err) {
				return true
			}
		}
		return false
	}
}

// FallbackOnAll returns a predicate that executes fallback only if all of the given predicates return true.
func FallbackOnAll(predicates ...FallbackPredicate) FallbackPredicate {
	return func(err error) bool {
		if len(predicates) == 0 {
			return false
		}
		for _, p := range predicates {
			if p == nil || !p(err) {
				return false
			}
		}
		return true
	}
}

// FallbackOnNot returns a predicate that executes fallback if the given predicate returns false.
func FallbackOnNot(predicate FallbackPredicate) FallbackPredicate {
	return func(err error) bool {
		if predicate == nil {
			return err != nil
		}
		return !predicate(err)
	}
}
