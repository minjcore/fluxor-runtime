package clock

import (
	"testing"
	"time"
)

func TestSystemClock_Now(t *testing.T) {
	clock := NewSystemClock()
	now := clock.Now()

	if now.IsZero() {
		t.Error("Now() should return a non-zero time")
	}

	// Verify it's reasonably close to actual time
	actualNow := time.Now()
	diff := actualNow.Sub(now)
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Second {
		t.Errorf("Now() returned time %v, which is more than 1 second away from actual time %v", now, actualNow)
	}
}

func TestSystemClock_Sleep(t *testing.T) {
	clock := NewSystemClock()
	start := time.Now()
	clock.Sleep(10 * time.Millisecond)
	duration := time.Since(start)

	if duration < 10*time.Millisecond {
		t.Errorf("Sleep(10ms) only slept for %v", duration)
	}
}

func TestSystemClock_After(t *testing.T) {
	clock := NewSystemClock()
	start := time.Now()
	ch := clock.After(10 * time.Millisecond)

	select {
	case <-ch:
		duration := time.Since(start)
		if duration < 10*time.Millisecond {
			t.Errorf("After(10ms) fired after only %v", duration)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("After(10ms) did not fire within 100ms")
	}
}

func TestSystemClock_AfterFunc(t *testing.T) {
	clock := NewSystemClock()
	done := make(chan bool, 1)

	start := time.Now()
	timer := clock.AfterFunc(10*time.Millisecond, func() {
		done <- true
	})

	select {
	case <-done:
		duration := time.Since(start)
		if duration < 10*time.Millisecond {
			t.Errorf("AfterFunc(10ms) fired after only %v", duration)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("AfterFunc(10ms) did not fire within 100ms")
	}

	if !timer.Stop() {
		// Timer already fired, which is expected
	}
}

func TestSystemClock_NewTimer(t *testing.T) {
	clock := NewSystemClock()
	start := time.Now()
	timer := clock.NewTimer(10 * time.Millisecond)

	select {
	case <-timer.C():
		duration := time.Since(start)
		if duration < 10*time.Millisecond {
			t.Errorf("NewTimer(10ms) fired after only %v", duration)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("NewTimer(10ms) did not fire within 100ms")
	}
}

func TestSystemClock_NewTicker(t *testing.T) {
	clock := NewSystemClock()
	ticker := clock.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	// Should fire at least once
	select {
	case <-ticker.C():
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("NewTicker(10ms) did not fire within 100ms")
	}
}

func TestSystemClock_Since(t *testing.T) {
	clock := NewSystemClock()
	start := clock.Now()
	clock.Sleep(10 * time.Millisecond)
	elapsed := clock.Since(start)

	if elapsed < 10*time.Millisecond {
		t.Errorf("Since() returned %v, expected at least 10ms", elapsed)
	}
}

func TestSystemClock_Until(t *testing.T) {
	clock := NewSystemClock()
	future := clock.Now().Add(100 * time.Millisecond)
	remaining := clock.Until(future)

	if remaining > 100*time.Millisecond || remaining < 0 {
		t.Errorf("Until() returned %v, expected approximately 100ms", remaining)
	}
}

func TestSystemTimer_Stop(t *testing.T) {
	clock := NewSystemClock()
	timer := clock.NewTimer(100 * time.Millisecond)

	if !timer.Stop() {
		t.Error("Stop() should return true for an active timer")
	}

	// Verify timer doesn't fire
	select {
	case <-timer.C():
		t.Error("Timer fired after being stopped")
	case <-time.After(200 * time.Millisecond):
		// Success - timer didn't fire
	}
}

func TestSystemTimer_Reset(t *testing.T) {
	clock := NewSystemClock()
	timer := clock.NewTimer(100 * time.Millisecond)

	// Stop the timer first
	if !timer.Stop() {
		t.Error("Stop() should return true")
	}

	// Reset with a shorter duration
	if timer.Reset(10 * time.Millisecond) {
		t.Error("Reset() should return false for a stopped timer")
	}

	// Verify timer fires with new duration
	start := time.Now()
	select {
	case <-timer.C():
		duration := time.Since(start)
		if duration < 10*time.Millisecond {
			t.Errorf("Reset timer fired after only %v", duration)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Reset timer did not fire")
	}
}
