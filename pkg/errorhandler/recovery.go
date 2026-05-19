package errorhandler

import (
	"fmt"
	"runtime/debug"
)

// RecoveryHandler is a function that handles recovered panics
type RecoveryHandler func(panicValue interface{}, stackTrace []byte) error

// Recover recovers from a panic and calls the handler
func Recover(handler RecoveryHandler) {
	if r := recover(); r != nil {
		stackTrace := debug.Stack()
		if handler != nil {
			if err := handler(r, stackTrace); err != nil {
				// If handler returns an error, we can't do much
				// Log it or handle it as needed
				_ = err
			}
		}
	}
}

// RecoverWithError recovers from a panic and returns it as an error
func RecoverWithError() error {
	if r := recover(); r != nil {
		stackTrace := debug.Stack()
		return New(ErrorCodeInternal, fmt.Sprintf("panic recovered: %v", r)).
			WithContext("stack_trace", string(stackTrace)).
			WithSeverity(ErrorSeverityCritical)
	}
	return nil
}

// RecoverAndWrap recovers from a panic and wraps it in a FluxorError
func RecoverAndWrap(code ErrorCode, message string) error {
	if r := recover(); r != nil {
		stackTrace := debug.Stack()
		err := New(code, message)
		if panicErr, ok := r.(error); ok {
			err.Cause = panicErr
		} else {
			err.Cause = fmt.Errorf("%v", r)
		}
		err.StackTrace = string(stackTrace)
		err.Severity = ErrorSeverityCritical
		return err
	}
	return nil
}

// SafeCall executes a function and recovers from any panics
func SafeCall(fn func() error, handler RecoveryHandler) (err error) {
	defer func() {
		if r := recover(); r != nil {
			stackTrace := debug.Stack()
			if handler != nil {
				err = handler(r, stackTrace)
			} else {
				err = New(ErrorCodeInternal, fmt.Sprintf("panic in SafeCall: %v", r)).
					WithContext("stack_trace", string(stackTrace)).
					WithSeverity(ErrorSeverityCritical)
			}
		}
	}()
	
	return fn()
}

// SafeCallWithDefault executes a function and returns a default error on panic
func SafeCallWithDefault(fn func() error, defaultErr error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if defaultErr != nil {
				err = defaultErr
			} else {
				err = New(ErrorCodeInternal, fmt.Sprintf("panic in SafeCallWithDefault: %v", r)).
					WithSeverity(ErrorSeverityCritical)
			}
		}
	}()
	
	return fn()
}
