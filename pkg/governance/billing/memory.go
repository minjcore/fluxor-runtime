package billing

import (
	"sync"
)

// MemoryRecorder stores usage events in memory (tests or small deployments).
type MemoryRecorder struct {
	mu     sync.Mutex
	events []UsageEvent
}

// NewMemoryRecorder creates an in-memory usage recorder.
func NewMemoryRecorder() *MemoryRecorder {
	return &MemoryRecorder{events: make([]UsageEvent, 0)}
}

// Record appends the event.
func (m *MemoryRecorder) Record(event UsageEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

// Events returns a copy of all recorded events.
func (m *MemoryRecorder) Events() []UsageEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]UsageEvent, len(m.events))
	copy(out, m.events)
	return out
}

// Clear removes all events (for tests).
func (m *MemoryRecorder) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = m.events[:0]
}
