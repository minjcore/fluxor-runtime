package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/protobuf"
	"google.golang.org/protobuf/proto"
)

// Message represents a message on the event bus
type Message interface {
	// Body returns the message body
	Body() interface{}

	// Headers returns the message headers
	Headers() map[string]string

	// ReplyAddress returns the reply address if this is a request message
	ReplyAddress() string

	// Reply sends a reply to this message
	Reply(body interface{}) error

	// DecodeBody decodes the message body into v
	DecodeBody(v interface{}) error

	// Fail indicates that processing failed
	Fail(failureCode int, message string) error
}

// message implements Message
type message struct {
	body         interface{}
	headers      map[string]string
	replyAddress string
	eventBus     EventBus
	mu           sync.RWMutex
}

func newMessage(body interface{}, headers map[string]string, replyAddress string, eventBus EventBus) Message {
	if headers == nil {
		headers = make(map[string]string)
	}
	return &message{
		body:         body,
		headers:      headers,
		replyAddress: replyAddress,
		eventBus:     eventBus,
	}
}

func (m *message) Body() interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.body
}

func (m *message) Headers() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]string)
	for k, v := range m.headers {
		result[k] = v
	}
	return result
}

func (m *message) ReplyAddress() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.replyAddress
}

func (m *message) Reply(body interface{}) error {
	if m.replyAddress == "" {
		return ErrNoReplyAddress
	}
	return m.eventBus.Send(m.replyAddress, body)
}

func (m *message) DecodeBody(v interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if data, ok := m.body.([]byte); ok {
		// Auto-detect: if target is proto.Message, decode as protobuf
		if _, ok := v.(proto.Message); ok {
			return protobuf.ProtobufDecode(data, v)
		}
		// Otherwise, decode as JSON
		return JSONDecode(data, v)
	}
	return fmt.Errorf("body is not []byte, got %T", m.body)
}

func (m *message) Fail(failureCode int, message string) error {
	// In a real implementation, this would send a failure response
	return m.Reply(map[string]interface{}{
		"failureCode": failureCode,
		"message":     message,
	})
}

// EventBus provides publish-subscribe and point-to-point messaging.
// Supports both JSON and gRPC/protobuf formats:
//   - If body implements proto.Message, it's automatically encoded as protobuf
//   - Otherwise, body is encoded as JSON
//   - If body is already []byte, it's passed through as-is
//
// Thread-safety: All methods are safe for concurrent use.
//
// Error handling patterns:
//   - Publish, Send, Request: return errors for invalid inputs or failures
//   - Consumer: PANICS on invalid address (fail-fast for programmer errors)
//
// The different error handling for Consumer is intentional:
//   - Invalid address in Consumer is a programming bug (should be caught in dev)
//   - Runtime errors in Publish/Send/Request are expected (network issues, etc.)
type EventBus interface {
	// Publish publishes a message to all handlers registered for the address.
	// Body is automatically encoded as protobuf (if proto.Message) or JSON if not already []byte.
	// Returns error if address is invalid or encoding fails.
	Publish(address string, body interface{}) error

	// Send sends a point-to-point message to one handler.
	// Body is automatically encoded as protobuf (if proto.Message) or JSON if not already []byte.
	// Returns error if address is invalid, no handlers registered, or encoding fails.
	Send(address string, body interface{}) error

	// Request sends a message and expects a reply within timeout.
	// Body is automatically encoded as protobuf (if proto.Message) or JSON if not already []byte.
	// Returns error if address is invalid, no handlers, timeout exceeded, or encoding fails.
	Request(address string, body interface{}, timeout time.Duration) (Message, error)

	// PublishWithContext publishes a message using the provided context for routing.
	// FloxID from context has highest priority for EventLoop routing.
	// Body is automatically encoded as protobuf (if proto.Message) or JSON if not already []byte.
	PublishWithContext(ctx context.Context, address string, body interface{}) error

	// SendWithContext sends a point-to-point message using the provided context for routing.
	// FloxID from context has highest priority for EventLoop routing.
	// Body is automatically encoded as protobuf (if proto.Message) or JSON if not already []byte.
	SendWithContext(ctx context.Context, address string, body interface{}, timeout time.Duration) (Message, error)

	// Consumer creates a consumer for the given address.
	//
	// IMPORTANT: This method PANICS if address is invalid (empty or too long).
	// This is intentional fail-fast behavior for programmer errors.
	// Invalid addresses should be caught during development, not at runtime.
	//
	// Usage pattern:
	//   consumer := eb.Consumer("my.address").Handler(func(ctx FluxorContext, msg Message) error {
	//       // handle message
	//       return nil
	//   })
	//   defer consumer.Unregister()
	Consumer(address string) Consumer

	// Close closes the event bus and releases all resources.
	// After Close, all other methods will fail.
	Close() error
}

// Producer produces messages to the EventBus (publish, send, request).
// EventBus implements Producer. Use Producer in components that only need to send messages
// (e.g. dependency injection with minimal surface, or testing with a mock producer).
type Producer interface {
	// Publish publishes a message to all handlers registered for the address.
	Publish(address string, body interface{}) error
	// PublishWithContext publishes a message using the provided context for routing.
	PublishWithContext(ctx context.Context, address string, body interface{}) error
	// Send sends a point-to-point message to one handler.
	Send(address string, body interface{}) error
	// SendWithContext sends a point-to-point message using the provided context for routing.
	SendWithContext(ctx context.Context, address string, body interface{}, timeout time.Duration) (Message, error)
	// Request sends a message and expects a reply within timeout.
	Request(address string, body interface{}, timeout time.Duration) (Message, error)
}

// Consumer represents a message consumer
type Consumer interface {
	// Handler sets the message handler
	Handler(handler MessageHandler) Consumer

	// Completion returns a channel that will be closed when the consumer is closed
	Completion() <-chan struct{}

	// Unregister unregisters the consumer
	Unregister() error
}

// MessageHandler handles incoming messages
type MessageHandler func(ctx FluxorContext, msg Message) error

// Errors
var (
	ErrNoReplyAddress = &EventBusError{Code: "NO_REPLY_ADDRESS", Message: "No reply address available"}
	ErrTimeout        = &EventBusError{Code: "TIMEOUT", Message: "Request timeout"}
)

// EventBusError represents an event bus error
type EventBusError struct {
	Code    string
	Message string
}

func (e *EventBusError) Error() string {
	return e.Message
}
