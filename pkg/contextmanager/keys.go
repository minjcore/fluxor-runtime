package contextmanager

import (
	"context"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// Key is a type-safe context key. Use distinct Key[T] types to avoid collisions.
//
// Example:
//
//	var requestIDKey = contextmanager.NewKey[string]("request_id")
//	ctx = contextmanager.WithValue(ctx, requestIDKey, "abc-123")
//	id, ok := contextmanager.Value(ctx, requestIDKey)
type Key[T any] struct {
	name string
}

// NewKey returns a new type-safe key with the given name (for debugging).
// The name is not used for equality; each Key[T] is unique by type and pointer.
func NewKey[T any](name string) Key[T] {
	return Key[T]{name: name}
}

// WithValue returns a copy of parent in which the value associated with key is val.
// Parent must be non-nil (fail-fast).
func WithValue[T any](parent context.Context, key Key[T], val T) context.Context {
	failfast.NotNil(parent, "parent context")
	return context.WithValue(parent, key, val)
}

// Value returns the value associated with key in ctx, or the zero value and false if not set.
func Value[T any](ctx context.Context, key Key[T]) (T, bool) {
	if ctx == nil {
		var zero T
		return zero, false
	}
	v := ctx.Value(key)
	if v == nil {
		var zero T
		return zero, false
	}
	t, ok := v.(T)
	return t, ok
}

// MustValue returns the value associated with key in ctx, or the zero value if not set.
// Use when the value is optional and zero is acceptable.
func MustValue[T any](ctx context.Context, key Key[T]) T {
	v, _ := Value(ctx, key)
	return v
}
