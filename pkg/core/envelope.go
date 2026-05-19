package core

// Envelope represents a message envelope with routing information
// Used for unified message format across Internal Bus and NATS
type Envelope struct {
	// Topic is the event bus address/topic
	Topic string

	// Key is the routing key for EventLoopGroup dispatch
	// Priority: X-Route-Key > X-User-ID > X-Session-ID > X-Request-ID
	Key string

	// Data is the raw payload (protobuf/JSON/msgpack bytes)
	Data []byte

	// Meta contains metadata (headers, tracing info, etc.)
	Meta map[string]string
}

// NewEnvelope creates a new envelope
func NewEnvelope(topic string, key string, data []byte, meta map[string]string) *Envelope {
	if meta == nil {
		meta = make(map[string]string)
	}
	return &Envelope{
		Topic: topic,
		Key:   key,
		Data:  data,
		Meta:  meta,
	}
}

// GetRoutingKey extracts routing key from envelope
// Priority: Key field > Meta["X-Route-Key"] > Meta["X-User-ID"] > Meta["X-Session-ID"] > Meta["X-Request-ID"]
func (e *Envelope) GetRoutingKey() string {
	if e.Key != "" {
		return e.Key
	}
	if e.Meta == nil {
		return ""
	}

	// Priority order
	keyOrder := []string{
		"X-Route-Key",
		"X-User-ID",
		"X-Session-ID",
		"X-Request-ID",
	}

	for _, key := range keyOrder {
		if val, ok := e.Meta[key]; ok && val != "" {
			return val
		}
	}

	return ""
}

// SetRoutingKey sets the routing key in the envelope
func (e *Envelope) SetRoutingKey(key string) {
	e.Key = key
	if e.Meta == nil {
		e.Meta = make(map[string]string)
	}
	// Also set in meta for compatibility
	e.Meta["X-Route-Key"] = key
}

// GetMeta returns metadata value by key
func (e *Envelope) GetMeta(key string) string {
	if e.Meta == nil {
		return ""
	}
	return e.Meta[key]
}

// SetMeta sets metadata value
func (e *Envelope) SetMeta(key, value string) {
	if e.Meta == nil {
		e.Meta = make(map[string]string)
	}
	e.Meta[key] = value
}
