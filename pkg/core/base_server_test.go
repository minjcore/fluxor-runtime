package core

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestBaseServer_Start_FailFast_RollbackStartedOnHookError(t *testing.T) {
	t.Parallel()

	gocmd := NewGoCMD(context.Background())
	bs := NewBaseServer("test", gocmd)
	bs.SetHooks(
		func() error { return errors.New("boom") },
		func() error { return nil },
	)

	if err := bs.Start(); err == nil {
		t.Fatalf("expected error")
	}
	if bs.IsStarted() {
		t.Fatalf("expected started=false after start hook error")
	}
}

func TestBaseServer_Start_BlocksButMarksStarted(t *testing.T) {
	gocmd := NewGoCMD(context.Background())
	bs := NewBaseServer("test", gocmd)

	release := make(chan struct{})
	var entered int64
	bs.SetHooks(
		func() error {
			atomic.AddInt64(&entered, 1)
			<-release
			return nil
		},
		func() error { return nil },
	)

	errCh := make(chan error, 1)
	go func() { errCh <- bs.Start() }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt64(&entered) == 1 && bs.IsStarted() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt64(&entered) != 1 {
		close(release)
		t.Fatalf("expected start hook to be entered")
	}
	if !bs.IsStarted() {
		close(release)
		t.Fatalf("expected IsStarted()=true while start hook is blocking")
	}

	close(release)

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("unexpected start error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("start did not return after unblocking hook")
	}
}

func TestBaseServer_StopBeforeStart_NoOp(t *testing.T) {
	t.Parallel()
	gocmd := NewGoCMD(context.Background())
	bs := NewBaseServer("test", gocmd)
	if err := bs.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if bs.IsStarted() || bs.IsStopped() {
		t.Fatalf("idle: started=%v stopped=%v state=%v", bs.IsStarted(), bs.IsStopped(), bs.State())
	}
}

func TestBaseServer_Context_RootAfterStart(t *testing.T) {
	gocmd := NewGoCMD(context.Background())
	bs := NewBaseServer("test", gocmd)
	ctxBefore := bs.Context()
	if ctxBefore == nil {
		t.Fatal("Context() returned nil")
	}
	bs.SetHooks(func() error { return nil }, func() error { return nil })
	if err := bs.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	ctxAfter := bs.Context()
	if ctxAfter == nil {
		t.Fatal("Context() nil after start")
	}
	_ = bs.Stop()
}

func TestBaseServer_SetHooksContext_ReceivesCancelableContext(t *testing.T) {
	gocmd := NewGoCMD(context.Background())
	bs := NewBaseServer("test", gocmd)
	var seen context.Context
	bs.SetHooksContext(
		func(ctx context.Context) error {
			seen = ctx
			return nil
		},
		func(ctx context.Context) error { return nil },
	)
	if err := bs.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if seen == nil {
		t.Fatal("start hook did not receive context")
	}
	if err := seen.Err(); err != nil {
		t.Fatalf("start ctx should not be done: %v", err)
	}
	_ = bs.Stop()
}
