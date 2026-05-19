package buckey_storage

import (
	"context"
	"errors"
	"testing"
)

func TestSeatManager_Acquire_Release(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorage()
	mgr := NewSeatManager(s)

	scope := "global"
	limit := 2

	err := mgr.Acquire(ctx, scope, "user1", limit)
	if err != nil {
		t.Fatal(err)
	}
	err = mgr.Acquire(ctx, scope, "user2", limit)
	if err != nil {
		t.Fatal(err)
	}
	err = mgr.Acquire(ctx, scope, "user3", limit)
	if !errors.Is(err, ErrNoSeat) {
		t.Errorf("expected ErrNoSeat, got %v", err)
	}

	used, _ := mgr.Usage(ctx, scope)
	if used != 2 {
		t.Errorf("used %d, want 2", used)
	}

	_ = mgr.Release(ctx, scope, "user1")
	used, _ = mgr.Usage(ctx, scope)
	if used != 1 {
		t.Errorf("after release: used %d, want 1", used)
	}

	err = mgr.Acquire(ctx, scope, "user3", limit)
	if err != nil {
		t.Fatal(err)
	}
	used, _ = mgr.Usage(ctx, scope)
	if used != 2 {
		t.Errorf("used %d, want 2", used)
	}

	_ = mgr.Release(ctx, scope, "user2")
	_ = mgr.Release(ctx, scope, "user3")
	used, _ = mgr.Usage(ctx, scope)
	if used != 0 {
		t.Errorf("used %d, want 0", used)
	}
}

func TestSeatManager_AcquireIdempotent(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorage()
	mgr := NewSeatManager(s)

	err := mgr.Acquire(ctx, "org:1", "user1", 2)
	if err != nil {
		t.Fatal(err)
	}
	err = mgr.Acquire(ctx, "org:1", "user1", 2)
	if err != nil {
		t.Fatal(err)
	}
	used, _ := mgr.Usage(ctx, "org:1")
	if used != 1 {
		t.Errorf("same holder twice: used %d, want 1", used)
	}
}
