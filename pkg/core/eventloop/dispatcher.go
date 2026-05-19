package eventloop

import (
	"context"
	"fmt"
	"sync"
)

// Dispatcher routes events to event loops based on routing key
// Note: FloxID from context should be set in Event.Key or headers before dispatch
// EventBus level checks FloxID from context and sets it with highest priority
type Dispatcher struct {
	loops             []*EventLoop
	numLoops          int
	extractor         KeyExtractor
	mu                sync.RWMutex
	roundRobinCounter uint64 // For fallback when no key
}

// EnvelopeData represents envelope data for routing (to avoid import cycle)
type EnvelopeData struct {
	Topic string
	Key   string
	Data  []byte
	Meta  map[string]string
}

// GetRoutingKey extracts routing key from envelope data
// Priority: Envelope.Key > X-Route-Key > X-Flox-ID > X-User-ID > ...
// Note: FloxID from context should be set in Envelope.Key or Meta["X-Flox-ID"] at EventBus level
func (e *EnvelopeData) GetRoutingKey() string {
	// Priority 1: Explicit Key field in envelope (highest priority)
	// This is where FloxID from context should be set
	if e.Key != "" {
		return e.Key
	}
	if e.Meta == nil {
		return ""
	}

	// Priority 2: Headers (X-Route-Key > X-Flox-ID > X-User-ID > ...)
	keyOrder := []string{
		"X-Route-Key",  // Explicit routing key (highest priority in headers)
		"X-Flox-ID",    // Universal routing ID (aggregate ID, stream ID, entity ID, etc.)
		"X-User-ID",    // User-based routing
		"X-Session-ID", // Session-based routing
		"X-Request-ID", // Request-based routing (lowest priority)
	}

	for _, key := range keyOrder {
		if val, ok := e.Meta[key]; ok && val != "" {
			return val
		}
	}

	return ""
}

// NewDispatcher creates a new dispatcher
func NewDispatcher(loops []*EventLoop, extractor KeyExtractor) *Dispatcher {
	if extractor == nil {
		extractor = DefaultKeyExtractor
	}

	return &Dispatcher{
		loops:     loops,
		numLoops:  len(loops),
		extractor: extractor,
	}
}

// Dispatch routes an event to the appropriate event loop
// Priority: Event.Key (FloxID from context) > X-Route-Key > X-Flox-ID (header) > X-User-ID > ...
// Note: FloxID from context is set in Event.Key at EventBus level (highest priority)
func (d *Dispatcher) Dispatch(ctx context.Context, event *Event) error {
	if event == nil {
		return &EventLoopError{Code: "INVALID_EVENT", Message: "event cannot be nil"}
	}

	// Priority 1: Event.Key (set from FloxID in context at EventBus level - highest priority)
	var key string
	if event.Key != "" {
		key = event.Key
	}

	// Priority 2: Extract from headers if Event.Key not set
	if key == "" {
		key = d.extractor(event.Headers, event.Address, event.Body)
	}

	// Route to loop
	var loop *EventLoop
	if key != "" {
		// Hash-based routing: ensures same key always goes to same loop
		idx := RouteKey(key, d.numLoops)
		d.mu.RLock()
		if idx < len(d.loops) {
			loop = d.loops[idx]
		}
		d.mu.RUnlock()
	}

	// Fallback to round-robin if no key or invalid index
	if loop == nil {
		d.mu.Lock()
		// Safe conversion: modulo result is in [0, numLoops-1], which fits in int
		// Ensure numLoops is positive to avoid issues
		if d.numLoops <= 0 {
			d.mu.Unlock()
			return &EventLoopError{Code: "NO_LOOPS", Message: "no event loops available"}
		}
		modResult := d.roundRobinCounter % uint64(d.numLoops)
		// Convert to int safely - modResult is always < numLoops, so it fits in int
		// On 32-bit systems, clamp to int32 max to prevent overflow
		const maxInt32 = 1<<31 - 1
		var idx int
		if modResult > uint64(maxInt32) {
			// Clamp to maxInt32 to prevent overflow on 32-bit systems
			idx = maxInt32 % d.numLoops
		} else {
			idx = int(modResult)
		}
		d.roundRobinCounter++
		if idx < len(d.loops) {
			loop = d.loops[idx]
		}
		d.mu.Unlock()
	}

	if loop == nil {
		return &EventLoopError{Code: "NO_LOOPS", Message: "no event loops available"}
	}

	// Dispatch to selected loop
	return loop.Dispatch(event)
}

// DispatchByKey routes an event to a specific loop by key
func (d *Dispatcher) DispatchByKey(key string, event *Event) error {
	if event == nil {
		return &EventLoopError{Code: "INVALID_EVENT", Message: "event cannot be nil"}
	}

	idx := RouteKey(key, d.numLoops)
	d.mu.RLock()
	defer d.mu.RUnlock()

	if idx >= len(d.loops) {
		return &EventLoopError{Code: "INVALID_INDEX", Message: fmt.Sprintf("loop index %d out of range", idx)}
	}

	return d.loops[idx].Dispatch(event)
}

// Stats returns statistics for all loops
func (d *Dispatcher) Stats() []LoopMetrics {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := make([]LoopMetrics, 0, len(d.loops))
	for _, loop := range d.loops {
		stats = append(stats, loop.Stats())
	}
	return stats
}

// DispatchEnvelope routes an envelope to the appropriate event loop
// Priority: Envelope.Key > X-Route-Key > X-Flox-ID (header) > ...
// Note: FloxID from context should be set in Envelope.Key or Meta at EventBus level
func (d *Dispatcher) DispatchEnvelope(ctx context.Context, env *EnvelopeData) error {
	if env == nil {
		return &EventLoopError{Code: "INVALID_ENVELOPE", Message: "envelope cannot be nil"}
	}

	// Extract routing key from envelope
	// Envelope.Key has highest priority, then headers
	key := env.GetRoutingKey()

	// Convert envelope to event
	event := &Event{
		Key:     key,
		Address: env.Topic,
		Body:    env.Data,
		Headers: env.Meta,
		Handler: nil, // Will be set by the bus implementation
	}

	// Route to loop
	return d.Dispatch(ctx, event)
}

// Close closes all event loops
func (d *Dispatcher) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	var firstErr error
	for _, loop := range d.loops {
		if err := loop.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
