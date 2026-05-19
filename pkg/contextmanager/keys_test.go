package contextmanager

import (
	"context"
	"testing"
)

func TestKey_WithValue_Value(t *testing.T) {
	requestIDKey := NewKey[string]("request_id")
	parent := context.Background()

	ctx := WithValue(parent, requestIDKey, "abc-123")
	v, ok := Value(ctx, requestIDKey)
	if !ok || v != "abc-123" {
		t.Errorf("Value: got %q, %v; want \"abc-123\", true", v, ok)
	}
}

func TestKey_Value_NotSet(t *testing.T) {
	requestIDKey := NewKey[string]("request_id")
	parent := context.Background()

	_, ok := Value(parent, requestIDKey)
	if ok {
		t.Error("Value: want false when key not set")
	}
}

func TestKey_MustValue(t *testing.T) {
	requestIDKey := NewKey[string]("request_id")
	parent := context.Background()

	ctx := WithValue(parent, requestIDKey, "xyz")
	v := MustValue(ctx, requestIDKey)
	if v != "xyz" {
		t.Errorf("MustValue: got %q, want \"xyz\"", v)
	}

	// Not set returns zero value
	z := MustValue(parent, requestIDKey)
	if z != "" {
		t.Errorf("MustValue (unset): got %q, want \"\"", z)
	}
}

func TestKey_WithValue_PanicsOnNilParent(t *testing.T) {
	requestIDKey := NewKey[string]("request_id")
	defer func() {
		if r := recover(); r == nil {
			t.Error("WithValue(nil, key, val) should panic")
		}
	}()
	WithValue(nil, requestIDKey, "x")
}

func TestKey_Value_NilContext(t *testing.T) {
	requestIDKey := NewKey[string]("request_id")
	_, ok := Value(nil, requestIDKey)
	if ok {
		t.Error("Value(nil, key): want false")
	}
}
