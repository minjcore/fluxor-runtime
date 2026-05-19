package fx

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestFluxor_KeepAndServe_BlocksUntilStop(t *testing.T) {
	ctx := context.Background()
	invoked := make(chan struct{})
	app, err := New(ctx, Invoke(NewInvoker(func(deps map[reflect.Type]interface{}) error {
		close(invoked)
		return nil
	})))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := app.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- app.KeepAndServe() }()

	select {
	case <-invoked:
	case <-time.After(2 * time.Second):
		t.Fatal("invoker not called")
	}

	// Stop should unblock KeepAndServe
	if err := app.Stop(); err != nil {
		t.Errorf("Stop: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("KeepAndServe: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("KeepAndServe did not return after Stop")
	}
}

func TestFluxor_NewAndStart(t *testing.T) {
	ctx := context.Background()
	var gotDeps bool
	app, err := New(ctx, Invoke(NewInvoker(func(deps map[reflect.Type]interface{}) error {
		gotDeps = deps != nil
		return nil
	})))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := app.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer app.Stop()
	if !gotDeps {
		t.Error("invoker did not receive deps")
	}
	if app.GoCMD() == nil {
		t.Error("GoCMD() is nil")
	}
}
