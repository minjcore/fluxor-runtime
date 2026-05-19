package aimodule

// EventBusError represents an error in the AI module
type EventBusError struct {
	Code    string
	Message string
}

func (e *EventBusError) Error() string {
	return e.Message
}
