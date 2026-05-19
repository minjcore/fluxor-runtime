package failfast

import (
	"errors"
	"strings"
	"testing"
)

func TestErr(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Expected no panic, got: %v", r)
			}
		}()
		Err(nil)
	})

	t.Run("with error", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Expected panic, got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Fatalf("Expected error type, got: %T", r)
			}
			if err.Error() == "" {
				t.Error("Expected error message")
			}
		}()
		Err(errors.New("test error"))
	})
}

func TestIf(t *testing.T) {
	t.Run("condition true", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Expected no panic, got: %v", r)
			}
		}()
		If(true, "should not panic")
	})

	t.Run("condition false", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Expected panic, got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Fatalf("Expected error type, got: %T", r)
			}
			if err.Error() == "" {
				t.Error("Expected error message")
			}
		}()
		If(false, "test message")
	})

	t.Run("formatted message", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Expected panic, got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Fatalf("Expected error type, got: %T", r)
			}
			if !strings.Contains(err.Error(), "fail-fast: value is 42") {
				t.Errorf("Expected message to contain 'fail-fast: value is 42', got %q", err.Error())
			}
			if !strings.Contains(err.Error(), "failfast_test.go") {
				t.Error("Expected stack trace in panic")
			}
		}()
		If(false, "value is %d", 42)
	})
}

func TestNotNil(t *testing.T) {
	t.Run("not nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Expected no panic, got: %v", r)
			}
		}()
		val := "test"
		NotNil(&val, "val")
	})

	t.Run("nil pointer", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Expected panic, got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Fatalf("Expected error type, got: %T", r)
			}
			if !strings.Contains(err.Error(), "fail-fast: ptr is nil") {
				t.Errorf("Expected message to contain 'fail-fast: ptr is nil', got %q", err.Error())
			}
			if !strings.Contains(err.Error(), "failfast_test.go") {
				t.Error("Expected stack trace in panic")
			}
		}()
		var ptr *string
		NotNil(ptr, "ptr")
	})

	t.Run("nil interface", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Expected panic, got none")
			}
		}()
		var val interface{}
		NotNil(val, "val")
	})
}

func TestErrWithContext(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Expected no panic, got: %v", r)
			}
		}()
		ErrWithContext(nil, "init: %s", "eventbus")
	})

	t.Run("with error", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Expected panic, got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Fatalf("Expected error type, got: %T", r)
			}
			msg := err.Error()
			if !strings.Contains(msg, "init: eventbus") {
				t.Errorf("Expected context in message, got %q", msg)
			}
			if !strings.Contains(msg, "test error") {
				t.Errorf("Expected wrapped error in message, got %q", msg)
			}
		}()
		ErrWithContext(errors.New("test error"), "init: %s", "eventbus")
	})
}

func TestNotEmpty(t *testing.T) {
	t.Run("non-empty", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Expected no panic, got: %v", r)
			}
		}()
		NotEmpty("x", "name")
	})

	t.Run("empty", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Expected panic, got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Fatalf("Expected error type, got: %T", r)
			}
			if !strings.Contains(err.Error(), "name cannot be empty") {
				t.Errorf("Expected 'name cannot be empty', got %q", err.Error())
			}
		}()
		NotEmpty("", "name")
	})
}

func TestPositive(t *testing.T) {
	t.Run("positive", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Expected no panic, got: %v", r)
			}
		}()
		Positive(1, "size")
	})

	t.Run("zero", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Expected panic, got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Fatalf("Expected error type, got: %T", r)
			}
			if !strings.Contains(err.Error(), "size must be positive") {
				t.Errorf("Expected 'size must be positive', got %q", err.Error())
			}
		}()
		Positive(0, "size")
	})

	t.Run("negative", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Expected panic, got none")
			}
		}()
		Positive(-1, "size")
	})
}

func TestNonNegative(t *testing.T) {
	t.Run("zero", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Expected no panic, got: %v", r)
			}
		}()
		NonNegative(0, "capacity")
	})

	t.Run("positive", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Expected no panic, got: %v", r)
			}
		}()
		NonNegative(1, "capacity")
	})

	t.Run("negative", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Expected panic, got none")
			}
			err, ok := r.(error)
			if !ok {
				t.Fatalf("Expected error type, got: %T", r)
			}
			if !strings.Contains(err.Error(), "capacity must be non-negative") {
				t.Errorf("Expected 'capacity must be non-negative', got %q", err.Error())
			}
		}()
		NonNegative(-1, "capacity")
	})
}
