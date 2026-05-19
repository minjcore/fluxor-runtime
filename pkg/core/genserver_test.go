package core

import (
	"context"
	"testing"
	"time"
)

func TestNewGenServerVerticle(t *testing.T) {
	gs := NewGenServerVerticle("test", "initial")
	if gs == nil {
		t.Fatal("NewGenServerVerticle returned nil")
	}
	if gs.name != "test" {
		t.Errorf("name = %q, want test", gs.name)
	}
	if gs.initState != "initial" {
		t.Errorf("initState = %v, want initial", gs.initState)
	}
}

func TestGenServerVerticle_Call_NotStarted(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fctx := NewFluxorContext(ctx, gocmd)

	gs := NewGenServerVerticle("test", nil)
	_, err := gs.Call(fctx, "addr", "msg", time.Second)
	if err == nil {
		t.Error("Call when not started should fail")
	}
	err = gs.Cast(fctx, "addr", "msg")
	if err == nil {
		t.Error("Cast when not started should fail")
	}
}

func TestGenServerVerticle_RegisterCall_RegisterCast(t *testing.T) {
	gs := NewGenServerVerticle("test", 0)
	gs.RegisterCall("inc", func(ctx FluxorContext, state interface{}, msg interface{}) (interface{}, interface{}, error) {
		n := state.(int)
		return n + 1, n + 1, nil
	})
	gs.RegisterCast("tick", func(ctx FluxorContext, state interface{}, msg interface{}) interface{} {
		return state.(int) + 1
	})
	// Just ensure no panic; real behaviour tested in Call/Cast tests
}

func TestGenServerVerticle_Call_Success(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fctx := NewFluxorContext(ctx, gocmd)

	gs := NewGenServerVerticle("echo", nil)
	gs.RegisterCall("echo", func(ctx FluxorContext, state interface{}, msg interface{}) (interface{}, interface{}, error) {
		return state, msg, nil
	})
	if err := gs.Start(fctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer gs.Stop(fctx)

	reply, err := gs.Call(fctx, "echo", "hello", 2*time.Second)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if reply != "hello" {
		t.Errorf("reply = %v, want hello", reply)
	}
}

func TestGenServerVerticle_Call_NoHandler(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fctx := NewFluxorContext(ctx, gocmd)

	gs := NewGenServerVerticle("test", nil)
	if err := gs.Start(fctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer gs.Stop(fctx)

	_, err := gs.Call(fctx, "nonexistent", "msg", time.Second)
	if err == nil {
		t.Error("Call with no handler should fail")
	}
}

func TestGenServerVerticle_Cast_Success(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fctx := NewFluxorContext(ctx, gocmd)

	gs := NewGenServerVerticle("counter", 0)
	gs.RegisterCast("add", func(ctx FluxorContext, state interface{}, msg interface{}) interface{} {
		return state.(int) + msg.(int)
	})
	if err := gs.Start(fctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer gs.Stop(fctx)

	if err := gs.Cast(fctx, "add", 1); err != nil {
		t.Fatalf("Cast: %v", err)
	}
	if err := gs.Cast(fctx, "add", 2); err != nil {
		t.Fatalf("Cast: %v", err)
	}
	// State is updated inside event loop; we can't read it safely from outside without a Call.
	// So we use a Call to read state
	gs.RegisterCall("get", func(ctx FluxorContext, state interface{}, msg interface{}) (interface{}, interface{}, error) {
		return state, state, nil
	})
	reply, err := gs.Call(fctx, "get", nil, time.Second)
	if err != nil {
		t.Fatalf("Call get: %v", err)
	}
	if reply.(int) != 3 {
		t.Errorf("state = %v, want 3", reply)
	}
}

func TestGenServerVerticle_Call_Timeout(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fctx := NewFluxorContext(ctx, gocmd)

	gs := NewGenServerVerticle("slow", nil)
	gs.RegisterCall("slow", func(ctx FluxorContext, state interface{}, msg interface{}) (interface{}, interface{}, error) {
		time.Sleep(200 * time.Millisecond)
		return state, "ok", nil
	})
	if err := gs.Start(fctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer gs.Stop(fctx)

	_, err := gs.Call(fctx, "slow", nil, 50*time.Millisecond)
	if err == nil {
		t.Error("Call with short timeout should fail")
	}
}
