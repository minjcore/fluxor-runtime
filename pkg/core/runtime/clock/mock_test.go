package clock

import (
	"testing"
	"time"
)

func TestMockClock_Now(t *testing.T) {
	initialTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewMockClock(initialTime)

	now := clock.Now()
	if !now.Equal(initialTime) {
		t.Errorf("Now() = %v, want %v", now, initialTime)
	}
}

func TestMockClock_Advance(t *testing.T) {
	initialTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewMockClock(initialTime)

	clock.Advance(1 * time.Hour)
	now := clock.Now()
	expected := initialTime.Add(1 * time.Hour)

	if !now.Equal(expected) {
		t.Errorf("After Advance(1h), Now() = %v, want %v", now, expected)
	}
}

func TestMockClock_SetTime(t *testing.T) {
	initialTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewMockClock(initialTime)

	newTime := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	clock.SetTime(newTime)

	now := clock.Now()
	if !now.Equal(newTime) {
		t.Errorf("After SetTime(), Now() = %v, want %v", now, newTime)
	}
}

func TestMockClock_Sleep(t *testing.T) {
	initialTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewMockClock(initialTime)

	clock.Sleep(1 * time.Hour)
	now := clock.Now()
	expected := initialTime.Add(1 * time.Hour)

	if !now.Equal(expected) {
		t.Errorf("After Sleep(1h), Now() = %v, want %v", now, expected)
	}
}

func TestMockClock_After(t *testing.T) {
	initialTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewMockClock(initialTime)

	ch := clock.After(1 * time.Hour)

	// Should not fire yet
	select {
	case <-ch:
		t.Error("After() fired before time advance")
	default:
		// Success
	}

	// Advance time and verify it fires
	clock.Advance(1 * time.Hour)

	select {
	case firedTime := <-ch:
		expected := initialTime.Add(1 * time.Hour)
		if !firedTime.Equal(expected) {
			t.Errorf("After() fired with time %v, want %v", firedTime, expected)
		}
	default:
		t.Error("After() did not fire after time advance")
	}
}

func TestMockClock_AfterFunc(t *testing.T) {
	initialTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewMockClock(initialTime)

	fired := make(chan bool, 1)
	timer := clock.AfterFunc(1*time.Hour, func() {
		fired <- true
	})

	// Should not fire yet
	select {
	case <-fired:
		t.Error("AfterFunc() fired before time advance")
	default:
		// Success
	}

	// Advance time and verify it fires
	clock.Advance(1 * time.Hour)

	select {
	case <-fired:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("AfterFunc() did not fire after time advance")
	}

	if !timer.Stop() {
		// Timer already fired, which is expected
	}
}

func TestMockClock_NewTimer(t *testing.T) {
	initialTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewMockClock(initialTime)

	timer := clock.NewTimer(1 * time.Hour)

	// Should not fire yet
	select {
	case <-timer.C():
		t.Error("NewTimer() fired before time advance")
	default:
		// Success
	}

	// Advance time and verify it fires
	clock.Advance(1 * time.Hour)

	select {
	case firedTime := <-timer.C():
		expected := initialTime.Add(1 * time.Hour)
		if !firedTime.Equal(expected) {
			t.Errorf("NewTimer() fired with time %v, want %v", firedTime, expected)
		}
	default:
		t.Error("NewTimer() did not fire after time advance")
	}
}

func TestMockClock_NewTicker(t *testing.T) {
	initialTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewMockClock(initialTime)

	ticker := clock.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// Advance by 3 hours and verify 3 ticks
	ticks := 0
	clock.Advance(3 * time.Hour)

	for i := 0; i < 3; i++ {
		select {
		case <-ticker.C():
			ticks++
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Ticker did not fire tick %d", i+1)
		}
	}

	if ticks != 3 {
		t.Errorf("Expected 3 ticks, got %d", ticks)
	}
}

func TestMockClock_Since(t *testing.T) {
	initialTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewMockClock(initialTime)

	past := clock.Now()
	clock.Advance(1 * time.Hour)
	elapsed := clock.Since(past)

	if elapsed != 1*time.Hour {
		t.Errorf("Since() returned %v, want %v", elapsed, 1*time.Hour)
	}
}

func TestMockClock_Until(t *testing.T) {
	initialTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewMockClock(initialTime)

	future := clock.Now().Add(1 * time.Hour)
	remaining := clock.Until(future)

	if remaining != 1*time.Hour {
		t.Errorf("Until() returned %v, want %v", remaining, 1*time.Hour)
	}

	// Advance time and verify it changes
	clock.Advance(30 * time.Minute)
	remaining = clock.Until(future)

	if remaining != 30*time.Minute {
		t.Errorf("After advance, Until() returned %v, want %v", remaining, 30*time.Minute)
	}
}

func TestMockTimer_Stop(t *testing.T) {
	initialTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewMockClock(initialTime)

	timer := clock.NewTimer(1 * time.Hour)

	if !timer.Stop() {
		t.Error("Stop() should return true for an active timer")
	}

	// Advance time and verify it doesn't fire
	clock.Advance(2 * time.Hour)

	select {
	case <-timer.C():
		t.Error("Timer fired after being stopped")
	default:
		// Success
	}
}

func TestMockTimer_Reset(t *testing.T) {
	initialTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewMockClock(initialTime)

	timer := clock.NewTimer(1 * time.Hour)

	// Reset with a different duration
	wasActive := timer.Reset(2 * time.Hour)
	if !wasActive {
		t.Error("Reset() should return true for an active timer")
	}

	// Advance by 1 hour - should not fire
	clock.Advance(1 * time.Hour)
	select {
	case <-timer.C():
		t.Error("Timer fired too early after Reset")
	default:
		// Success
	}

	// Advance by another hour - should fire now
	clock.Advance(1 * time.Hour)
	select {
	case <-timer.C():
		// Success
	default:
		t.Error("Timer did not fire after Reset duration")
	}
}

func TestMockTicker_Stop(t *testing.T) {
	initialTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewMockClock(initialTime)

	ticker := clock.NewTicker(1 * time.Hour)
	ticker.Stop()

	// Advance time and verify it doesn't fire
	clock.Advance(3 * time.Hour)

	select {
	case <-ticker.C():
		t.Error("Ticker fired after being stopped")
	default:
		// Success
	}
}

func TestMockClock_MultipleTimers(t *testing.T) {
	initialTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := NewMockClock(initialTime)

	timer1 := clock.NewTimer(1 * time.Hour)
	timer2 := clock.NewTimer(2 * time.Hour)
	timer3 := clock.NewTimer(30 * time.Minute)

	// Advance by 30 minutes - only timer3 should fire
	clock.Advance(30 * time.Minute)

	select {
	case <-timer3.C():
		// Success
	default:
		t.Error("Timer3 did not fire after 30 minutes")
	}

	select {
	case <-timer1.C():
		t.Error("Timer1 fired too early")
	case <-timer2.C():
		t.Error("Timer2 fired too early")
	default:
		// Success
	}

	// Advance by another 30 minutes - timer1 should fire
	clock.Advance(30 * time.Minute)

	select {
	case <-timer1.C():
		// Success
	default:
		t.Error("Timer1 did not fire after 1 hour")
	}

	// Advance by another hour - timer2 should fire
	clock.Advance(1 * time.Hour)

	select {
	case <-timer2.C():
		// Success
	default:
		t.Error("Timer2 did not fire after 2 hours")
	}
}
