package core

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core/concurrency"
)

func TestNewBaseVerticle(t *testing.T) {
	v := NewBaseVerticle("test-verticle")
	if v == nil {
		t.Fatal("NewBaseVerticle() returned nil")
	}
	if v.Name() != "test-verticle" {
		t.Errorf("Name() = %v, want 'test-verticle'", v.Name())
	}
	if v.IsStarted() {
		t.Error("NewBaseVerticle() should create verticle that is not started")
	}
}

func TestBaseVerticle_Start_FailFast_NilContext(t *testing.T) {
	v := NewBaseVerticle("test")

	defer func() {
		if r := recover(); r == nil {
			t.Error("Start() should panic (fail-fast) with nil context")
		}
	}()

	v.Start(nil)
}

func TestBaseVerticle_Start_FailFast_DoubleStart(t *testing.T) {
	v := NewBaseVerticle("test")
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	err := v.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("First Start() failed: %v", err)
	}

	// Second start should fail-fast
	err = v.Start(fluxorCtx)
	if err == nil {
		t.Error("Start() should fail-fast when called twice")
	}
	if err != nil {
		if e, ok := err.(*EventBusError); ok {
			if e.Code != "ALREADY_STARTED" {
				t.Errorf("Error code = %v, want 'ALREADY_STARTED'", e.Code)
			}
		} else {
			t.Errorf("Expected EventBusError, got %T", err)
		}
	}
}

func TestBaseVerticle_Consumer_FailFast_NotStarted(t *testing.T) {
	v := NewBaseVerticle("test")

	defer func() {
		if r := recover(); r == nil {
			t.Error("Consumer() should panic (fail-fast) when verticle not started")
		}
	}()

	v.Consumer("test.address")
}

func TestBaseVerticle_Publish_FailFast_NotStarted(t *testing.T) {
	v := NewBaseVerticle("test")

	err := v.Publish("test.address", "test body")
	if err == nil {
		t.Error("Publish() should fail-fast when verticle not started")
	}
	if err != nil {
		if e, ok := err.(*EventBusError); ok {
			if e.Code != "NOT_STARTED" {
				t.Errorf("Error code = %v, want 'NOT_STARTED'", e.Code)
			}
		} else {
			t.Errorf("Expected EventBusError, got %T", err)
		}
	}
}

func TestBaseVerticle_Publish_FailFast_EmptyAddress(t *testing.T) {
	v := NewBaseVerticle("test")
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	err := v.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	err = v.Publish("", "test body")
	if err == nil {
		t.Error("Publish() should fail-fast with empty address")
	}
}

func TestBaseVerticle_Send_FailFast_NotStarted(t *testing.T) {
	v := NewBaseVerticle("test")

	err := v.Send("test.address", "test body")
	if err == nil {
		t.Error("Send() should fail-fast when verticle not started")
	}
	if err != nil {
		if e, ok := err.(*EventBusError); ok {
			if e.Code != "NOT_STARTED" {
				t.Errorf("Error code = %v, want 'NOT_STARTED'", e.Code)
			}
		} else {
			t.Errorf("Expected EventBusError, got %T", err)
		}
	}
}

func TestBaseVerticle_Send_FailFast_EmptyAddress(t *testing.T) {
	v := NewBaseVerticle("test")
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	err := v.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	err = v.Send("", "test body")
	if err == nil {
		t.Error("Send() should fail-fast with empty address")
	}
}

func TestBaseVerticle_RunOnEventLoop_FailFast_NotStarted(t *testing.T) {
	v := NewBaseVerticle("test")

	task := concurrency.TaskFunc(func(ctx context.Context) error { return nil })
	err := v.RunOnEventLoop(task)
	if err == nil {
		t.Error("RunOnEventLoop() should fail-fast when verticle not started")
	}
	if err != nil {
		if e, ok := err.(*EventBusError); ok {
			if e.Code != "NOT_STARTED" {
				t.Errorf("Error code = %v, want 'NOT_STARTED'", e.Code)
			}
		} else {
			t.Errorf("Expected EventBusError, got %T", err)
		}
	}
}

func TestBaseVerticle_RunOnEventLoop_FailFast_NilTask(t *testing.T) {
	v := NewBaseVerticle("test")
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	err := v.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	err = v.RunOnEventLoop(nil)
	if err == nil {
		t.Error("RunOnEventLoop() should fail-fast with nil task")
	}
}

func TestBaseVerticle_Context(t *testing.T) {
	v := NewBaseVerticle("test")
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	if v.Context() != nil {
		t.Error("Context() should return nil before Start()")
	}

	err := v.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if v.Context() != fluxorCtx {
		t.Errorf("Context() = %v, want %v", v.Context(), fluxorCtx)
	}
}

func TestBaseVerticle_EventBus(t *testing.T) {
	v := NewBaseVerticle("test")
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	if v.EventBus() != nil {
		t.Error("EventBus() should return nil before Start()")
	}

	err := v.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if v.EventBus() == nil {
		t.Error("EventBus() should not return nil after Start()")
	}
}

func TestBaseVerticle_IsStarted_IsStopped(t *testing.T) {
	v := NewBaseVerticle("test")
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	if v.IsStarted() {
		t.Error("IsStarted() should return false for new verticle")
	}
	if v.IsStopped() {
		t.Error("IsStopped() should return false for new verticle")
	}

	err := v.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if !v.IsStarted() {
		t.Error("IsStarted() should return true after Start()")
	}
	if v.IsStopped() {
		t.Error("IsStopped() should return false after Start()")
	}

	err = v.Stop(fluxorCtx)
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// Note: IsStarted() remains true after Stop() - it indicates Start() was called, not current state
	// IsStopped() indicates Stop() was called
	if !v.IsStopped() {
		t.Error("IsStopped() should return true after Stop()")
	}
}
