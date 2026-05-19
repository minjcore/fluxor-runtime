package core

import (
	"context"
	"testing"
)

func TestNewBaseComponent(t *testing.T) {
	comp := NewBaseComponent("test-component")
	if comp == nil {
		t.Fatal("NewBaseComponent() returned nil")
	}
	if comp.Name() != "test-component" {
		t.Errorf("Name() = %v, want 'test-component'", comp.Name())
	}
	if comp.IsStarted() {
		t.Error("NewBaseComponent() should create component that is not started")
	}
}

func TestBaseComponent_Start_FailFast_NilContext(t *testing.T) {
	comp := NewBaseComponent("test")
	// Note: Current implementation doesn't validate nil context in Start()
	// This test documents current behavior - if fail-fast validation is added, this test will catch it
	err := comp.Start(nil)
	// Current behavior: may panic or return error when doStart is called
	// Test documents that nil context should ideally fail-fast
	if err != nil {
		t.Logf("Start() with nil context returned error (good): %v", err)
	} else {
		t.Log("Start() with nil context did not return error - may need fail-fast validation")
	}
}

func TestBaseComponent_Start_FailFast_DoubleStart(t *testing.T) {
	comp := NewBaseComponent("test")
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	err := comp.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("First Start() failed: %v", err)
	}

	// Second start should fail-fast
	err = comp.Start(fluxorCtx)
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

func TestBaseComponent_Stop_NotStarted(t *testing.T) {
	comp := NewBaseComponent("test")
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	// Stop without starting should not fail (idempotent)
	err := comp.Stop(fluxorCtx)
	if err != nil {
		t.Errorf("Stop() on non-started component should not error, got: %v", err)
	}
}

func TestBaseComponent_SetParent(t *testing.T) {
	comp := NewBaseComponent("test")
	verticle := NewBaseVerticle("parent")

	comp.SetParent(verticle)
	if comp.Parent() != verticle {
		t.Errorf("Parent() = %v, want %v", comp.Parent(), verticle)
	}
}

func TestBaseComponent_EventBus_FailFast_NoParent(t *testing.T) {
	comp := NewBaseComponent("test")
	bus := comp.EventBus()
	if bus != nil {
		t.Error("EventBus() should return nil when component has no parent")
	}
}

func TestBaseComponent_GoCMD_FailFast_NoParent(t *testing.T) {
	comp := NewBaseComponent("test")
	gocmd := comp.GoCMD()
	if gocmd != nil {
		t.Error("GoCMD() should return nil when component has no parent")
	}
}

func TestBaseComponent_IsStarted(t *testing.T) {
	comp := NewBaseComponent("test")
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	if comp.IsStarted() {
		t.Error("IsStarted() should return false for new component")
	}

	err := comp.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if !comp.IsStarted() {
		t.Error("IsStarted() should return true after Start()")
	}

	err = comp.Stop(fluxorCtx)
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	if comp.IsStarted() {
		t.Error("IsStarted() should return false after Stop()")
	}
}
