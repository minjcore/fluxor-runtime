package core

import (
	"sync"
	"sync/atomic"
)

// Bus is an in-process pub/sub bus.
//
// This is intentionally minimal and "fire-and-forget". For production use you may
// want backpressure, bounded queues, and error propagation.
type Bus struct {
	mu     sync.RWMutex
	nextID uint64
	subs   map[string]map[uint64]func(any)
}

func NewBus() *Bus {
	return &Bus{subs: make(map[string]map[uint64]func(any))}
}

// Subscribe registers a handler. It returns an unsubscribe function.
func (b *Bus) Subscribe(topic string, handler func(any)) (unsubscribe func()) {
	id := atomic.AddUint64(&b.nextID, 1)

	b.mu.Lock()
	if b.subs[topic] == nil {
		b.subs[topic] = make(map[uint64]func(any))
	}
	b.subs[topic][id] = handler
	b.mu.Unlock()

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		m := b.subs[topic]
		if m == nil {
			return
		}
		delete(m, id)
		if len(m) == 0 {
			delete(b.subs, topic)
		}
	}
}

func (b *Bus) Publish(topic string, msg any) {
	b.mu.RLock()
	m := b.subs[topic]
	handlers := make([]func(any), 0, len(m))
	for _, h := range m {
		handlers = append(handlers, h)
	}
	b.mu.RUnlock()

	for _, h := range handlers {
		// Fire-and-forget.
		go h(msg)
	}
}
