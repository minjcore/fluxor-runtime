package core

import "time"

// Publish is a convenience method to publish messages
func (bv *BaseVerticle) Publish(address string, body interface{}) error {
	if bv.eventBus == nil {
		return &EventBusError{Code: "NOT_STARTED", Message: "verticle not started"}
	}
	return bv.eventBus.Publish(address, body)
}

// Send is a convenience method to send messages
func (bv *BaseVerticle) Send(address string, body interface{}) error {
	if bv.eventBus == nil {
		return &EventBusError{Code: "NOT_STARTED", Message: "verticle not started"}
	}
	return bv.eventBus.Send(address, body)
}

// Request is a convenience method to send a request and wait for a reply
// This is the primary way to call services from a verticle
func (bv *BaseVerticle) Request(address string, body interface{}, timeout time.Duration) (Message, error) {
	if bv.eventBus == nil {
		return nil, &EventBusError{Code: "NOT_STARTED", Message: "verticle not started"}
	}
	return bv.eventBus.Request(address, body, timeout)
}
