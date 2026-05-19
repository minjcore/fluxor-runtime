package managers

import "time"

// ComponentEvent represents a component lifecycle event
type ComponentEvent string

const (
	// ComponentStarted indicates a component has started
	ComponentStarted ComponentEvent = "started"
	// ComponentStopped indicates a component has stopped
	ComponentStopped ComponentEvent = "stopped"
	// ComponentHeartbeat indicates a heartbeat event
	ComponentHeartbeat ComponentEvent = "heartbeat"
	// ComponentHeartbeatMissed indicates a component missed its heartbeat
	ComponentHeartbeatMissed ComponentEvent = "heartbeat_missed"
)

// ComponentEventHandler handles component lifecycle events
// Car analogy: Managers receives signals from car parts (components)
type ComponentEventHandler func(componentName string, event ComponentEvent)

// HeartbeatEvent represents a heartbeat message published to EventBus
type HeartbeatEvent struct {
	Timestamp     time.Time `json:"timestamp"`
	ComponentName string    `json:"component_name,omitempty"` // Empty for Managers heartbeat
	ManagersAlive bool      `json:"managers_alive"`
}
