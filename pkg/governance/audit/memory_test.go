package audit

import (
	"testing"
	"time"
)

func TestMemoryLogger_Log_and_Events(t *testing.T) {
	logger := NewMemoryLogger()
	ev := Event{
		ID:        "1",
		Timestamp: time.Now(),
		Actor:     "user1",
		Action:    "login",
		Resource:  "auth",
		Outcome:   "success",
	}
	if err := logger.Log(ev); err != nil {
		t.Fatalf("Log: %v", err)
	}
	events := logger.Events()
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	if events[0].Actor != "user1" || events[0].Action != "login" {
		t.Errorf("event = %+v", events[0])
	}
}

func TestMemoryLogger_Clear(t *testing.T) {
	logger := NewMemoryLogger()
	_ = logger.Log(Event{ID: "1", Actor: "a"})
	logger.Clear()
	if len(logger.Events()) != 0 {
		t.Errorf("after Clear, len(Events()) = %d, want 0", len(logger.Events()))
	}
}
