package queue

import (
	"context"
	"time"
)

// Message represents a RabbitMQ message
type Message struct {
	// Body is the message body (will be JSON encoded if not []byte)
	Body interface{}

	// Headers are message headers
	Headers map[string]string

	// ContentType is the message content type (default: "application/json")
	ContentType string

	// DeliveryMode: 1 = non-persistent, 2 = persistent (default: 2)
	DeliveryMode uint8

	// Priority is message priority (0-255)
	Priority uint8

	// Expiration is message expiration time (e.g., "60000" for 60 seconds)
	Expiration string

	// MessageID is optional message identifier
	MessageID string

	// Timestamp is message timestamp
	Timestamp time.Time

	// ReplyTo is reply-to queue name
	ReplyTo string

	// CorrelationID is correlation identifier
	CorrelationID string
}

// Delivery represents a received message delivery
type Delivery struct {
	// Body is the raw message body
	Body []byte

	// Headers are message headers
	Headers map[string]string

	// ContentType is the message content type
	ContentType string

	// DeliveryTag is the delivery tag for acknowledgment
	DeliveryTag uint64

	// Exchange is the exchange name
	Exchange string

	// RoutingKey is the routing key
	RoutingKey string

	// MessageID is the message identifier
	MessageID string

	// Timestamp is message timestamp
	Timestamp time.Time

	// ReplyTo is reply-to queue name
	ReplyTo string

	// CorrelationID is correlation identifier
	CorrelationID string

	// Redelivered indicates if message was redelivered
	Redelivered bool
}

// DecodeBody decodes the message body into v (JSON)
func (d *Delivery) DecodeBody(v interface{}) error {
	return JSONDecode(d.Body, v)
}

// Publisher interface for publishing messages
type Publisher interface {
	// Publish publishes a message to an exchange
	Publish(ctx context.Context, exchange, routingKey string, msg Message) error

	// Close closes the publisher
	Close() error
}

// Consumer interface for consuming messages
type Consumer interface {
	// Consume starts consuming messages from a queue
	Consume(ctx context.Context, queue string, handler MessageHandler) error

	// Close closes the consumer
	Close() error
}

// MessageHandler handles incoming messages
type MessageHandler func(ctx context.Context, delivery *Delivery) error

// ReplyHandler handles RPC reply messages (similar to EventBus MessageHandler)
type ReplyHandler func(ctx context.Context, delivery *Delivery) error

// Error represents a queue error
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
