package managers

import (
	"time"
)

// startHeartbeat starts the heartbeat ticker
// Internal method - use StartHeartbeat() for public API
func (m *Managers) startHeartbeat() error {
	m.heartbeatMu.Lock()
	defer m.heartbeatMu.Unlock()

	if m.heartbeatStarted {
		return nil // Already started
	}

	// Check if heartbeat is enabled
	m.mu.RLock()
	enabled := m.config.EnableHeartbeat
	interval := m.config.HeartbeatInterval
	m.mu.RUnlock()

	if !enabled {
		return nil // Heartbeat disabled
	}

	if interval <= 0 {
		interval = 10 * time.Second // Default interval
	}

	// Create ticker
	m.heartbeatTicker = time.NewTicker(interval)
	m.heartbeatStarted = true

	// Get GoCMD context for cancellation
	ctx := m.gocmd.Context()

	// Start heartbeat goroutine
	go func() {
		for {
			select {
			case <-m.heartbeatTicker.C:
				// Emit heartbeat event to handlers
				m.emitHeartbeatEvent()
				// Publish heartbeat to EventBus
				m.publishHeartbeatToEventBus()
				// Check for missed component heartbeats
				m.checkMissedHeartbeats()
			case <-m.heartbeatStop:
				return
			case <-ctx.Done():
				// Stop heartbeat when context is cancelled
				m.stopHeartbeat()
				return
			}
		}
	}()

	return nil
}

// stopHeartbeat stops the heartbeat ticker
// Internal method - use StopHeartbeat() for public API
func (m *Managers) stopHeartbeat() {
	m.heartbeatMu.Lock()
	defer m.heartbeatMu.Unlock()

	if !m.heartbeatStarted {
		return // Already stopped
	}

	if m.heartbeatTicker != nil {
		m.heartbeatTicker.Stop()
		m.heartbeatTicker = nil
	}

	close(m.heartbeatStop)
	m.heartbeatStop = make(chan struct{})
	m.heartbeatStarted = false
}

// sendHeartbeat updates the last heartbeat timestamp for a component
// Internal method - use SendHeartbeat() for public API
func (m *Managers) sendHeartbeat(componentName string) {
	m.heartbeatMu.Lock()
	defer m.heartbeatMu.Unlock()

	m.componentHeartbeats[componentName] = time.Now()
}

// checkMissedHeartbeats checks for components that missed their heartbeats
func (m *Managers) checkMissedHeartbeats() {
	m.heartbeatMu.RLock()
	componentHeartbeats := make(map[string]time.Time)
	for k, v := range m.componentHeartbeats {
		componentHeartbeats[k] = v
	}

	m.mu.RLock()
	interval := m.config.HeartbeatInterval
	m.mu.RUnlock()

	if interval <= 0 {
		interval = 10 * time.Second
	}

	threshold := 3 * interval // Miss heartbeat if no heartbeat for 3 intervals
	now := time.Now()
	m.heartbeatMu.RUnlock()

	// Check each component
	for componentName, lastHeartbeat := range componentHeartbeats {
		if now.Sub(lastHeartbeat) > threshold {
			// Component missed heartbeat
			m.emitEvent(componentName, ComponentHeartbeatMissed)
		}
	}
}

// emitHeartbeatEvent emits a heartbeat event to registered handlers
func (m *Managers) emitHeartbeatEvent() {
	// Emit heartbeat event for Managers itself (component name "managers")
	m.emitEvent("managers", ComponentHeartbeat)
}

// publishHeartbeatToEventBus publishes a heartbeat message to EventBus
func (m *Managers) publishHeartbeatToEventBus() {
	m.mu.RLock()
	address := m.config.HeartbeatEventBusAddress
	m.mu.RUnlock()

	if address == "" {
		address = "managers.heartbeat" // Default address
	}

	eventBus := m.EventBus()
	if eventBus == nil {
		return // No EventBus available
	}

	// Create heartbeat event
	event := HeartbeatEvent{
		Timestamp:     time.Now(),
		ComponentName: "", // Empty for Managers heartbeat
		ManagersAlive: true,
	}

	// Publish to EventBus (EventBus will auto-encode to JSON)
	_ = eventBus.Publish(address, event)
}
