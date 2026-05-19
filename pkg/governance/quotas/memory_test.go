package quotas

import (
	"testing"
	"time"
)

func TestMemoryLimiter_Allow_no_refill(t *testing.T) {
	lim := NewMemoryLimiter(2, 0)
	key := "user1"

	allowed, rem := lim.Allow(key, false)
	if !allowed || rem != 2 {
		t.Errorf("first Allow(consume=false): allowed=%v rem=%d", allowed, rem)
	}
	allowed, rem = lim.Allow(key, true)
	if !allowed || rem != 1 {
		t.Errorf("Allow(consume=true): allowed=%v rem=%d", allowed, rem)
	}
	allowed, rem = lim.Allow(key, true)
	if !allowed || rem != 0 {
		t.Errorf("second consume: allowed=%v rem=%d", allowed, rem)
	}
	allowed, _ = lim.Allow(key, true)
	if allowed {
		t.Error("expected not allowed after limit exceeded")
	}
}

func TestMemoryLimiter_Reset(t *testing.T) {
	lim := NewMemoryLimiter(1, 0)
	key := "k"
	lim.Allow(key, true)
	allowed, _ := lim.Allow(key, false)
	if allowed {
		t.Error("expected not allowed before reset")
	}
	_ = lim.Reset(key)
	allowed, rem := lim.Allow(key, false)
	if !allowed || rem != 1 {
		t.Errorf("after Reset: allowed=%v rem=%d", allowed, rem)
	}
}

func TestMemoryLimiter_refill(t *testing.T) {
	lim := NewMemoryLimiter(2, 10*time.Millisecond)
	key := "k"
	lim.Allow(key, true)
	lim.Allow(key, true)
	allowed, _ := lim.Allow(key, false)
	if allowed {
		t.Error("expected not allowed when at limit")
	}
	time.Sleep(15 * time.Millisecond)
	allowed, rem := lim.Allow(key, false)
	if !allowed {
		t.Errorf("after refill: allowed=%v", allowed)
	}
	if rem < 1 {
		t.Errorf("remaining = %d", rem)
	}
}
