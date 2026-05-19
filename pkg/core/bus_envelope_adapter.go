package core

import (
	"context"

	"github.com/fluxorio/fluxor/pkg/core/eventloop"
)

// EnvelopeToEventLoopData converts Envelope to eventloop.EnvelopeData
// This adapter bridges core.Envelope and eventloop.EnvelopeData to avoid import cycles
func EnvelopeToEventLoopData(env *Envelope) *eventloop.EnvelopeData {
	if env == nil {
		return nil
	}
	return &eventloop.EnvelopeData{
		Topic: env.Topic,
		Key:   env.GetRoutingKey(), // Extract key using priority
		Data:  env.Data,
		Meta:  env.Meta,
	}
}

// EventLoopGroupAdapter adapts EventLoopGroup to work with Envelope
// Implements DispatcherInterface
type EventLoopGroupAdapter struct {
	group *eventloop.EventLoopGroup
}

// NewEventLoopGroupAdapter creates an adapter for EventLoopGroup
func NewEventLoopGroupAdapter(group *eventloop.EventLoopGroup) *EventLoopGroupAdapter {
	return &EventLoopGroupAdapter{group: group}
}

// DispatchEnvelope dispatches an envelope to EventLoopGroup
// Implements DispatcherInterface
func (a *EventLoopGroupAdapter) DispatchEnvelope(ctx context.Context, env *Envelope) error {
	if a.group == nil {
		return &EventBusError{Code: "NO_EVENTLOOP_GROUP", Message: "EventLoopGroup not initialized"}
	}
	data := EnvelopeToEventLoopData(env)
	return a.group.DispatchEnvelope(ctx, data)
}
