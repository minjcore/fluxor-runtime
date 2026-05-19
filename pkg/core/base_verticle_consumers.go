package core

import (
	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// RegisterConsumer registers a consumer for automatic cleanup
// This is a convenience method for subclasses
func (bv *BaseVerticle) RegisterConsumer(consumer Consumer) {
	// Fail-fast: consumer cannot be nil
	failfast.NotNil(consumer, "consumer")
	bv.mu.Lock()
	defer bv.mu.Unlock()
	bv.consumers = append(bv.consumers, consumer)
}

// Consumer creates and registers a consumer for the given address
// Returns the consumer for further configuration
func (bv *BaseVerticle) Consumer(address string) Consumer {
	// Fail-fast: verticle must be started
	failfast.NotNil(bv.eventBus, "eventBus (verticle not started - cannot create consumer)")
	consumer := bv.eventBus.Consumer(address)
	bv.RegisterConsumer(consumer)
	return consumer
}
