package fallback

import (
	"context"
	"errors"
	"testing"
)

func TestAlwaysFallback(t *testing.T) {
	if !AlwaysFallback(errors.New("test error")) {
		t.Error("AlwaysFallback should return true for error")
	}

	if AlwaysFallback(nil) {
		t.Error("AlwaysFallback should return false for nil")
	}
}

func TestNeverFallback(t *testing.T) {
	if NeverFallback(errors.New("test error")) {
		t.Error("NeverFallback should return false for error")
	}

	if NeverFallback(nil) {
		t.Error("NeverFallback should return false for nil")
	}
}

func TestFallbackOnTimeout(t *testing.T) {
	// Deadline exceeded
	if !FallbackOnTimeout(context.DeadlineExceeded) {
		t.Error("Expected true for context.DeadlineExceeded")
	}

	// Timeout in message
	if !FallbackOnTimeout(errors.New("request timeout")) {
		t.Error("Expected true for timeout error message")
	}

	if !FallbackOnTimeout(errors.New("deadline exceeded")) {
		t.Error("Expected true for deadline exceeded error message")
	}

	if !FallbackOnTimeout(errors.New("operation timed out")) {
		t.Error("Expected true for timed out error message")
	}

	// Non-timeout error
	if FallbackOnTimeout(errors.New("network error")) {
		t.Error("Expected false for non-timeout error")
	}

	if FallbackOnTimeout(nil) {
		t.Error("Expected false for nil")
	}
}

func TestFallbackOnTemporary(t *testing.T) {
	// Temporary interface
	tempErr := &temporaryError{temp: true}
	if !FallbackOnTemporary(tempErr) {
		t.Error("Expected true for temporary error")
	}

	// Non-temporary interface with message that doesn't contain temporary keywords
	permErr := &permanentError{}
	if FallbackOnTemporary(permErr) {
		t.Error("Expected false for non-temporary error")
	}

	// Temporary in message
	if !FallbackOnTemporary(errors.New("temporary error")) {
		t.Error("Expected true for temporary error message")
	}

	if !FallbackOnTemporary(errors.New("please retry")) {
		t.Error("Expected true for retry error message")
	}

	if !FallbackOnTemporary(errors.New("service busy")) {
		t.Error("Expected true for busy error message")
	}

	// Non-temporary error
	if FallbackOnTemporary(errors.New("permanent error")) {
		t.Error("Expected false for non-temporary error")
	}

	if FallbackOnTemporary(nil) {
		t.Error("Expected false for nil")
	}
}

type temporaryError struct {
	temp bool
}

func (e *temporaryError) Error() string {
	return "temporary error"
}

func (e *temporaryError) Temporary() bool {
	return e.temp
}

type permanentError struct{}

func (e *permanentError) Error() string {
	return "permanent error"
}

func (e *permanentError) Temporary() bool {
	return false
}

func TestFallbackOnNetworkError(t *testing.T) {
	testCases := []struct {
		err      error
		expected bool
	}{
		{errors.New("network error"), true},
		{errors.New("connection failed"), true},
		{errors.New("dial tcp: connection refused"), true},
		{errors.New("connection reset"), true},
		{errors.New("host unreachable"), true},
		{errors.New("database error"), false},
		{errors.New("permission denied"), false},
		{nil, false},
	}

	for _, tc := range testCases {
		result := FallbackOnNetworkError(tc.err)
		if result != tc.expected {
			t.Errorf("FallbackOnNetworkError(%v) = %v, expected %v", tc.err, result, tc.expected)
		}
	}
}

func TestFallbackOnErrorType(t *testing.T) {
	targetErr := errors.New("target error")
	unwrappedErr := errors.New("unwrapped")

	predicate := FallbackOnErrorType(targetErr)

	// Same error
	if !predicate(targetErr) {
		t.Error("Expected true for same error")
	}

	// Different error
	if predicate(unwrappedErr) {
		t.Error("Expected false for different error")
	}

	if predicate(nil) {
		t.Error("Expected false for nil")
	}
}

func TestFallbackOnErrorMessage(t *testing.T) {
	predicate := FallbackOnErrorMessage("database")

	// Contains substring (case insensitive)
	if !predicate(errors.New("database error")) {
		t.Error("Expected true for error containing 'database'")
	}

	if !predicate(errors.New("DATABASE ERROR")) {
		t.Error("Expected true for error containing 'DATABASE' (case insensitive)")
	}

	// Doesn't contain substring
	if predicate(errors.New("network error")) {
		t.Error("Expected false for error not containing 'database'")
	}

	if predicate(nil) {
		t.Error("Expected false for nil")
	}
}

func TestFallbackOnAny(t *testing.T) {
	pred1 := FallbackOnTimeout
	pred2 := FallbackOnNetworkError
	predicate := FallbackOnAny(pred1, pred2)

	// Matches first predicate
	if !predicate(context.DeadlineExceeded) {
		t.Error("Expected true for timeout error")
	}

	// Matches second predicate
	if !predicate(errors.New("network error")) {
		t.Error("Expected true for network error")
	}

	// Matches neither
	if predicate(errors.New("database error")) {
		t.Error("Expected false for error matching neither predicate")
	}

	if predicate(nil) {
		t.Error("Expected false for nil")
	}

	// Empty predicates
	emptyPredicate := FallbackOnAny()
	if emptyPredicate(errors.New("any error")) {
		t.Error("Expected false for empty predicates")
	}

	// Nil predicates are skipped
	nilPredicate := FallbackOnAny(nil, pred1)
	if !nilPredicate(context.DeadlineExceeded) {
		t.Error("Expected true - nil predicates should be skipped")
	}
}

func TestFallbackOnAll(t *testing.T) {
	pred1 := func(err error) bool {
		return err != nil
	}
	pred2 := func(err error) bool {
		if err == nil {
			return false
		}
		return len(err.Error()) > 5
	}
	predicate := FallbackOnAll(pred1, pred2)

	// Matches both
	if !predicate(errors.New("long error message")) {
		t.Error("Expected true for error matching both predicates")
	}

	// Matches only first
	if predicate(errors.New("short")) {
		t.Error("Expected false for error matching only first predicate")
	}

	// Matches neither
	if predicate(nil) {
		t.Error("Expected false for nil")
	}

	// Empty predicates
	emptyPredicate := FallbackOnAll()
	if emptyPredicate(errors.New("any error")) {
		t.Error("Expected false for empty predicates")
	}

	// Nil predicates cause false
	nilPredicate := FallbackOnAll(nil, pred1)
	if nilPredicate(errors.New("error")) {
		t.Error("Expected false - nil predicates should cause false")
	}
}

func TestFallbackOnNot(t *testing.T) {
	predicate := FallbackOnNot(FallbackOnTimeout)

	// Original predicate returns true, so NOT should return false
	if predicate(context.DeadlineExceeded) {
		t.Error("Expected false - timeout should not trigger fallback with NOT")
	}

	// Original predicate returns false, so NOT should return true
	if !predicate(errors.New("database error")) {
		t.Error("Expected true - non-timeout should trigger fallback with NOT")
	}

	// Nil predicate
	nilPredicate := FallbackOnNot(nil)
	if !nilPredicate(errors.New("error")) {
		t.Error("Expected true - NOT of nil predicate should return true for errors")
	}

	if nilPredicate(nil) {
		t.Error("Expected false - NOT of nil predicate should return false for nil")
	}
}

func TestPredicateIntegration(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()

	// Test FallbackOnAny
	config1 := Config{
		Fallbacks: []Executable{
			func(ctx context.Context) error {
				return nil
			},
		},
		Predicate: FallbackOnAny(
			FallbackOnTimeout,
			FallbackOnNetworkError,
		),
	}

	// Should fallback on timeout
	err := manager.ExecuteWithConfig(ctx,
		func(ctx context.Context) error {
			return context.DeadlineExceeded
		},
		config1,
	)
	if err != nil {
		t.Errorf("Expected nil (fallback succeeded), got %v", err)
	}

	// Should not fallback on database error
	err = manager.ExecuteWithConfig(ctx,
		func(ctx context.Context) error {
			return errors.New("database error")
		},
		config1,
	)
	if err == nil {
		t.Error("Expected error (no fallback), got nil")
	}

	// Test FallbackOnNot
	config2 := Config{
		Fallbacks: []Executable{
			func(ctx context.Context) error {
				return nil
			},
		},
		Predicate: FallbackOnNot(FallbackOnTimeout),
	}

	// Should not fallback on timeout
	err = manager.ExecuteWithConfig(ctx,
		func(ctx context.Context) error {
			return context.DeadlineExceeded
		},
		config2,
	)
	if err == nil {
		t.Error("Expected error (no fallback on timeout with NOT), got nil")
	}

	// Should fallback on non-timeout error
	err = manager.ExecuteWithConfig(ctx,
		func(ctx context.Context) error {
			return errors.New("database error")
		},
		config2,
	)
	if err != nil {
		t.Errorf("Expected nil (fallback succeeded), got %v", err)
	}
}
