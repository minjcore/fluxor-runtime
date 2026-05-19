package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/concurrency"
)

func TestSubmitBlockingFunc_NotStarted_FailsFuture(t *testing.T) {
	v := NewBaseVerticle("test")
	// Don't call Start() — worker pool is nil

	future := SubmitBlockingFunc(v, func() (string, error) {
		return "ok", nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := future.Await(ctx)
	if err == nil {
		t.Error("Await() should fail when verticle not started")
	}
	var ebErr *EventBusError
	if !errors.As(err, &ebErr) || ebErr.Code != "NOT_STARTED" {
		t.Errorf("expected NOT_STARTED EventBusError, got %v", err)
	}
}

func TestSubmitBlockingFunc_NilFunc_FailsFuture(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	v := NewBaseVerticle("test")
	if err := v.Start(fluxorCtx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	future := SubmitBlockingFunc[string](v, nil)

	awaitCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := future.Await(awaitCtx)
	if err == nil {
		t.Error("Await() should fail when fn is nil")
	}
	var ebErr *EventBusError
	if !errors.As(err, &ebErr) || ebErr.Code != "INVALID_FUNCTION" {
		t.Errorf("expected INVALID_FUNCTION EventBusError, got %v", err)
	}
}

func TestSubmitBlockingFunc_Success_AwaitReturnsResult(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	v := NewBaseVerticle("test")
	if err := v.Start(fluxorCtx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	want := "hello"
	future := SubmitBlockingFunc(v, func() (string, error) {
		return want, nil
	})

	awaitCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	got, err := future.Await(awaitCtx)
	if err != nil {
		t.Fatalf("Await() error: %v", err)
	}
	if got != want {
		t.Errorf("Await() = %q, want %q", got, want)
	}
}

func TestSubmitBlockingFunc_Error_AwaitReturnsError(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	v := NewBaseVerticle("test")
	if err := v.Start(fluxorCtx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	wantErr := errors.New("blocking task failed")
	future := SubmitBlockingFunc(v, func() (string, error) {
		return "", wantErr
	})

	awaitCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := future.Await(awaitCtx)
	if err == nil {
		t.Fatal("Await() should return error")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("Await() err = %v, want %v", err, wantErr)
	}
}

func TestSubmitBlockingFunc_OnSuccess_OnFailure_Called(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	v := NewBaseVerticle("test")
	if err := v.Start(fluxorCtx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Success path
	var successGot string
	futureOk := SubmitBlockingFunc(v, func() (string, error) {
		return "ok", nil
	})
	futureOk.OnSuccess(func(s string) { successGot = s })

	awaitCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, _ = futureOk.Await(awaitCtx)
	cancel()

	if successGot != "ok" {
		t.Errorf("OnSuccess: got %q, want 'ok'", successGot)
	}

	// Failure path
	var failureGot error
	wantErr := errors.New("fail")
	futureFail := SubmitBlockingFunc(v, func() (string, error) {
		return "", wantErr
	})
	futureFail.OnFailure(func(e error) { failureGot = e })

	awaitCtx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()

	_, _ = futureFail.Await(awaitCtx2)
	cancel2()

	if failureGot == nil || !errors.Is(failureGot, wantErr) {
		t.Errorf("OnFailure: got %v, want %v", failureGot, wantErr)
	}
}

// testBlockingVerticle embeds BaseVerticle like RTMPServerVerticle/stream_verticle.
// Used to functional-test SubmitBlockingFunc from a real verticle subtype.
type testBlockingVerticle struct {
	*BaseVerticle
	done chan struct{}
}

func newTestBlockingVerticle() *testBlockingVerticle {
	return &testBlockingVerticle{
		BaseVerticle: NewBaseVerticle("test-blocking"),
		done:         make(chan struct{}),
	}
}

// TestSubmitBlockingFunc_Functional_FromSubtypeVerticle is a functional test:
// a concrete verticle (embedding BaseVerticle) starts, submits blocking work via
// SubmitBlockingFunc(verticle.BaseVerticle, fn), and we verify the result and that
// the event loop stays responsive (not blocked by the blocking work).
func TestSubmitBlockingFunc_Functional_FromSubtypeVerticle(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	v := newTestBlockingVerticle()
	if err := v.Start(fluxorCtx); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// 1) Submit blocking work (simulates e.g. server Start()) from the subtype using .BaseVerticle
	blockingDone := make(chan struct{})
	var blockingResult string
	future := SubmitBlockingFunc(v.BaseVerticle, func() (string, error) {
		// Simulate blocking work (e.g. accept loop)
		time.Sleep(30 * time.Millisecond)
		close(blockingDone)
		return "blocking-done", nil
	})
	future.OnSuccess(func(s string) { blockingResult = s })

	// 2) While blocking work runs on worker pool, event loop should still process tasks
	eventLoopRan := make(chan struct{})
	err := v.RunOnEventLoop(concurrency.TaskFunc(func(context.Context) error {
		close(eventLoopRan)
		return nil
	}))
	if err != nil {
		t.Fatalf("RunOnEventLoop() failed: %v", err)
	}

	// Event loop task should complete quickly (not blocked by the 30ms blocking work)
	select {
	case <-eventLoopRan:
		// OK: event loop is responsive
	case <-time.After(100 * time.Millisecond):
		t.Fatal("event loop did not run within 100ms — blocking work may be running on event loop")
	}

	// 3) Await blocking result
	awaitCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got, err := future.Await(awaitCtx)
	if err != nil {
		t.Fatalf("Await() error: %v", err)
	}
	if got != "blocking-done" {
		t.Errorf("Await() = %q, want 'blocking-done'", got)
	}
	if blockingResult != "blocking-done" {
		t.Errorf("OnSuccess: got %q, want 'blocking-done'", blockingResult)
	}

	// 4) Blocking work should have run (channel closed)
	select {
	case <-blockingDone:
	default:
		t.Error("blocking fn did not run to completion")
	}
}
