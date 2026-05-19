package billing

import (
	"testing"
	"time"
)

func TestMemoryRecorder_Record_and_Events(t *testing.T) {
	rec := NewMemoryRecorder()
	ev := UsageEvent{
		SubjectID: "user1",
		Resource:  "api_calls",
		Quantity:  1,
		Unit:      "request",
		Timestamp: time.Now(),
	}
	if err := rec.Record(ev); err != nil {
		t.Fatalf("Record: %v", err)
	}
	events := rec.Events()
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	if events[0].SubjectID != "user1" || events[0].Resource != "api_calls" {
		t.Errorf("event = %+v", events[0])
	}
}

func TestMemoryRecorder_Clear(t *testing.T) {
	rec := NewMemoryRecorder()
	_ = rec.Record(UsageEvent{SubjectID: "x", Resource: "y"})
	rec.Clear()
	if len(rec.Events()) != 0 {
		t.Errorf("after Clear, len(Events()) = %d, want 0", len(rec.Events()))
	}
}
