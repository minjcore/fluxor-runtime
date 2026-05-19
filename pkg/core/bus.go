package core

import (
	"context"
	"sync"
	"time"
)

// SimpleBus is a simple in-memory bus implementation (legacy)
// Deprecated: Use EventBus with Envelope pattern instead
type SimpleBus struct {
	mu   sync.RWMutex
	subs map[string][]func(any)
}

func NewBus() *SimpleBus {
	return &SimpleBus{subs: make(map[string][]func(any))}
}

func (b *SimpleBus) Subscribe(topic string, handler func(any)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[topic] = append(b.subs[topic], handler)
}

func (b *SimpleBus) Publish(topic string, msg any) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if handlers, ok := b.subs[topic]; ok {
		for _, h := range handlers {
			// Fire and forget (trong thực tế có thể wrap goroutine)
			go h(msg)
		}
	}
}

// Bus represents a unified message bus interface with Envelope
// Both Internal Bus and NATS Bus should implement this interface
//
// Flow:
//
//	gRPC/NATS → Envelope → Dispatcher → EventLoop (hash(key) % N) → Internal Bus → Handler
type Bus interface {
	// Publish publishes an envelope to all subscribers
	// Envelope.Key is used for EventLoopGroup routing
	Publish(ctx context.Context, env *Envelope) error

	// Subscribe subscribes to a topic and receives envelopes
	// Returns unsubscribe function and error
	Subscribe(topic string, fn func(ctx context.Context, env *Envelope) error) (unsubscribe func() error, err error)

	// Send sends an envelope to one subscriber (point-to-point)
	// Envelope.Key is used for EventLoopGroup routing
	Send(ctx context.Context, env *Envelope) error

	// Request sends an envelope and expects a reply
	// Envelope.Key is used for EventLoopGroup routing
	Request(ctx context.Context, env *Envelope, timeout time.Duration) (*Envelope, error)

	// Close closes the bus
	Close() error
}

// DispatcherInterface defines the dispatcher contract for routing envelopes
type DispatcherInterface interface {
	// DispatchEnvelope routes an envelope to the appropriate event loop
	DispatchEnvelope(ctx context.Context, env *Envelope) error
}
