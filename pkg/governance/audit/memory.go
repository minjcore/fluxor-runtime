package audit

import (
	"sync"
)

// MemoryLogger stores audit events in memory. Useful for tests and small deployments.
type MemoryLogger struct {
	mu     sync.Mutex
	events []Event
}

// NewMemoryLogger creates an in-memory audit logger.
func NewMemoryLogger() *MemoryLogger {
	return &MemoryLogger{events: make([]Event, 0)}
}

// Log appends the event to the in-memory slice.
func (m *MemoryLogger) Log(event Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

// Events returns a copy of all logged events.
func (m *MemoryLogger) Events() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Event, len(m.events))
	copy(out, m.events)
	return out
}

// Clear removes all events (for tests).
func (m *MemoryLogger) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = m.events[:0]
}
