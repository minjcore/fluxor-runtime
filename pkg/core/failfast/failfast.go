// Package failfast provides fail-fast assertions for programmer errors and invariants.
// Use it for invalid arguments, nil checks, and violated preconditions—not for expected
// runtime failures (network errors, validation failures, etc.), which should return errors.
//
// All panics include a stack trace so the call site is visible in logs or recover().
package failfast

import (
	"fmt"
	"reflect"
	"runtime/debug"
)

// Err panics if err != nil (fail-fast principle).
// Includes stack trace for debugging.
func Err(err error) {
	if err != nil {
		panic(fmt.Errorf("fail-fast: %w\n%s", err, debug.Stack()))
	}
}

// ErrWithContext panics if err != nil, wrapping the error with a formatted context message.
// Use when the same error type can originate from multiple call sites.
func ErrWithContext(err error, format string, args ...interface{}) {
	if err != nil {
		panic(fmt.Errorf("fail-fast: %s: %w\n%s", fmt.Sprintf(format, args...), err, debug.Stack()))
	}
}

// If panics if condition is false (i.e. requires condition to be true).
// Message supports fmt-style formatting via args.
// Includes stack trace for debugging.
func If(condition bool, message string, args ...interface{}) {
	if !condition {
		msg := fmt.Sprintf("fail-fast: "+message, args...)
		panic(fmt.Errorf("%s\n%s", msg, debug.Stack()))
	}
}

// NotNil panics if ptr is nil.
// Handles untyped nil, typed nil pointers, and nil functions.
// Includes stack trace for debugging.
func NotNil(ptr interface{}, name string) {
	var panicErr error
	if ptr == nil {
		panicErr = fmt.Errorf("fail-fast: %s is nil\n%s", name, debug.Stack())
	} else {
		v := reflect.ValueOf(ptr)
		if v.Kind() == reflect.Ptr && v.IsNil() {
			panicErr = fmt.Errorf("fail-fast: %s is nil\n%s", name, debug.Stack())
		}
		if v.Kind() == reflect.Func && v.IsNil() {
			panicErr = fmt.Errorf("fail-fast: %s is nil\n%s", name, debug.Stack())
		}
	}
	if panicErr != nil {
		panic(panicErr)
	}
}

// NotEmpty panics if s is empty. Use for required string arguments (e.g. name, address, key).
func NotEmpty(s string, name string) {
	if s == "" {
		panic(fmt.Errorf("fail-fast: %s cannot be empty\n%s", name, debug.Stack()))
	}
}

// Positive panics if n <= 0. Use for sizes, timeouts, counts that must be > 0.
func Positive(n int, name string) {
	if n <= 0 {
		panic(fmt.Errorf("fail-fast: %s must be positive, got %d\n%s", name, n, debug.Stack()))
	}
}

// NonNegative panics if n < 0. Use for capacities, indices, or counts that may be zero.
func NonNegative(n int, name string) {
	if n < 0 {
		panic(fmt.Errorf("fail-fast: %s must be non-negative, got %d\n%s", name, n, debug.Stack()))
	}
}
