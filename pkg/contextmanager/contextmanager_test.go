package contextmanager

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWithScope(t *testing.T) {
	parent := context.Background()
	var gotCtx context.Context
	err := WithScope(parent, func(ctx context.Context) error {
		gotCtx = ctx
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	})
	if err != nil {
		t.Fatalf("WithScope: %v", err)
	}
	if gotCtx == nil {
		t.Fatal("fn did not receive context")
	}
	// After scope exits, context should be cancelled
	select {
	case <-gotCtx.Done():
		// expected
	default:
		t.Error("scoped context should be cancelled after fn returns")
	}
}

func TestWithScope_PropagatesError(t *testing.T) {
	want := errors.New("scope error")
	parent := context.Background()
	err := WithScope(parent, func(ctx context.Context) error {
		return want
	})
	if err != want {
		t.Errorf("WithScope: got %v, want %v", err, want)
	}
}

func TestWithScope_PanicsOnNilParent(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("WithScope(nil, fn) should panic")
		}
	}()
	_ = WithScope(nil, func(ctx context.Context) error { return nil })
}

func TestWithCancel(t *testing.T) {
	parent := context.Background()
	ctx, cancel := WithCancel(parent)
	if ctx == nil {
		t.Fatal("ctx is nil")
	}
	cancel()
	select {
	case <-ctx.Done():
		// expected
	case <-time.After(time.Second):
		t.Error("context not cancelled after cancel()")
	}
}

func TestWithCancel_PanicsOnNilParent(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("WithCancel(nil) should panic")
		}
	}()
	WithCancel(nil)
}

func TestWithTimeout(t *testing.T) {
	parent := context.Background()
	ctx, cancel := WithTimeout(parent, 10*time.Millisecond)
	defer cancel()
	select {
	case <-ctx.Done():
		// expected after timeout
	case <-time.After(50 * time.Millisecond):
		t.Error("context not cancelled after timeout")
	}
}

func TestWithTimeout_PanicsOnNilParent(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("WithTimeout(nil, d) should panic")
		}
	}()
	WithTimeout(nil, time.Second)
}

func TestWithTimeout_PanicsOnNonPositive(t *testing.T) {
	parent := context.Background()
	defer func() {
		if r := recover(); r == nil {
			t.Error("WithTimeout(parent, 0) should panic")
		}
	}()
	WithTimeout(parent, 0)
}

func TestRun(t *testing.T) {
	parent := context.Background()
	err := Run(parent, func(ctx context.Context) error {
		if ctx != parent {
			t.Errorf("Run: fn received wrong context")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRun_PropagatesError(t *testing.T) {
	want := errors.New("run error")
	err := Run(context.Background(), func(ctx context.Context) error {
		return want
	})
	if err != want {
		t.Errorf("Run: got %v, want %v", err, want)
	}
}

func TestRun_PanicsOnNilContext(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Run(nil, fn) should panic")
		}
	}()
	Run(nil, func(ctx context.Context) error { return nil })
}

func TestRun_PanicsOnNilFn(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Run(ctx, nil) should panic")
		}
	}()
	Run(context.Background(), nil)
}
