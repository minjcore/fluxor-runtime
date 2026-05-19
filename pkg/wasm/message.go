package wasm

// WSMessage represents a WebSocket message
type WSMessage struct {
	Op      string            `json:"op"`      // publish, send, request, subscribe, unsubscribe, message
	Address string            `json:"address"` // EventBus address
	Body    interface{}       `json:"body"`    // Message body
	ID      string            `json:"id"`      // Request ID for request/reply
	Timeout int64             `json:"timeout"` // Timeout in milliseconds
	Error   string            `json:"error,omitempty"`
	Result  interface{}       `json:"result,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// Message represents a received message
type Message interface {
	Body() interface{}
	Headers() map[string]string
	DecodeBody(v interface{}) error
}

// messageImpl implements Message
type messageImpl struct {
	body    interface{}
	headers map[string]string
}

func (m *messageImpl) Body() interface{} {
	return m.body
}

func (m *messageImpl) Headers() map[string]string {
	return m.headers
}

func (m *messageImpl) DecodeBody(v interface{}) error {
	// In WASM, we'll use JSON encoding
	// This will be implemented using syscall/js
	return nil
}
